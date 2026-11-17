package tournament

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/jobs"
	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Pairing COP rounds outside the tournament lock.
//
// COP takes seconds. Run inside a change, it held the tournament's lock the
// whole time: every other change to the tournament, in every division,
// waited for it, and a change waiting on another server would give up after
// LockTimeout. So with DeferCOPPairing on, a change that would pair a COP
// round leaves it unpaired instead, and saving the change writes a
// pair_round job for it in the same transaction: the change and the promise
// to pair the round commit together, or neither does.
//
// The job (PairRoundJob) loads the tournament without the lock, builds the
// COP request and runs COP. Then it takes the lock and builds the request
// again. If it is the same, it applies COP's answer, the way a director's
// Pair Round would. If not, something changed meanwhile (a score was
// amended), and it runs COP again on the new request. If the round no
// longer needs pairing (a director paired it by hand), it does nothing.
//
// A job cut off by a deploy runs again on the next server, from the start.
// COP is a pure function of the request, so that costs time, not
// correctness.

// PairRoundJobKind is the job kind that pairs a COP round.
const PairRoundJobKind = "pair_round"

// pairRoundAttempts is how many times a pair_round job tries before the
// directors are told the round couldn't be paired. COP failures are mostly
// about the request, which retrying doesn't change, so it is told soon.
const pairRoundAttempts = 3

// maxRequestChanges bounds how many times one run starts over because the
// division changed while COP ran.
const maxRequestChanges = 3

// pairRoundJobKey is the job key of a round: tournament/division/round, the
// round 0-based. Division names can hold '/', tournament IDs and rounds
// can't.
func pairRoundJobKey(tid, division string, round int) string {
	return tid + "/" + division + "/" + strconv.Itoa(round)
}

func parsePairRoundJobKey(key string) (tid, division string, round int, err error) {
	first := strings.Index(key, "/")
	last := strings.LastIndex(key, "/")
	if first < 0 || last <= first {
		return "", "", 0, fmt.Errorf("bad pair_round key %q", key)
	}
	round, err = strconv.Atoi(key[last+1:])
	if err != nil {
		return "", "", 0, fmt.Errorf("bad pair_round key %q: %w", key, err)
	}
	return key[:first], key[first+1 : last], round, nil
}

// EnqueuePendingPairings writes a pair_round job for each COP round the
// change left unpaired (TakePendingCOPRounds), with q, the change's
// transaction. It returns how many it wrote. A job already waiting for the
// round is left as it is: it builds its request when it runs.
func EnqueuePendingPairings(ctx context.Context, q *models.Queries, t *entity.Tournament) (int, error) {
	n := 0
	for name, div := range t.Divisions {
		if div.DivisionManager == nil {
			continue
		}
		for _, round := range div.DivisionManager.TakePendingCOPRounds() {
			if err := jobs.Enqueue(ctx, q, PairRoundJobKind, pairRoundJobKey(t.UUID, name, round), time.Now()); err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

// copOutcome is what applying COP's answer did.
type copOutcome int

const (
	copApplied   copOutcome = iota
	copNotNeeded            // the round no longer needs pairing
	copChanged              // the request changed while COP ran
)

// String is the outcome as the job span's cop.outcome records it.
func (o copOutcome) String() string {
	switch o {
	case copApplied:
		return "applied"
	case copNotNeeded:
		return "not_needed"
	case copChanged:
		return "changed"
	}
	return "unknown"
}

// PairRoundJob returns the handler of pair_round jobs.
func PairRoundJob(ts TournamentStore, pairer COPPairer) jobs.Handler {
	return func(ctx context.Context, job jobs.Job) error {
		tid, division, round, err := parsePairRoundJobKey(job.Key)
		if err != nil {
			return err
		}
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(attribute.String("tournament.id", tid),
			attribute.String("tournament.division", division), attribute.Int("tournament.round", round+1))
		for run := 1; run <= maxRequestChanges; run++ {
			span.SetAttributes(attribute.Int("cop.runs", run-1))
			t, err := ts.Get(ctx, tid)
			if err != nil {
				return err
			}
			cd, ok := copDivision(t, division)
			if !ok || t.IsFinished || !cd.needsCOPPairing(round) {
				span.SetAttributes(attribute.String("cop.outcome", "not_needed"))
				return nil
			}
			kept, err := cd.copKeptPlayers(round)
			if err != nil {
				return err
			}
			req, err := cd.copRequest(round, kept)
			if err != nil {
				return err
			}
			if req == nil {
				// Every player still to pair is in a kept pairing.
				span.SetAttributes(attribute.String("cop.outcome", "nothing_to_pair"))
				return nil
			}
			span.SetAttributes(attribute.Int("cop.runs", run))
			resp, err := runCOP(ctx, pairer, req, run)
			if err != nil {
				span.SetAttributes(attribute.String("cop.outcome", "cop_failed"))
				return err
			}
			outcome, err := applyCOPPairing(ctx, ts, tid, division, round, req, resp)
			if err != nil {
				return err
			}
			span.SetAttributes(attribute.String("cop.outcome", outcome.String()))
			if outcome != copChanged {
				return nil
			}
			span.AddEvent("request changed")
			log.Ctx(ctx).Info().Str("tid", tid).Str("division", division).Int("round", round+1).Msg("cop-request-changed")
		}
		return errors.New("the division kept changing while COP ran")
	}
}

// runCOP runs COP on req in a cop.pair span: where it ran, how big the
// request was, and COP's error code. In process, COP's seconds would
// otherwise show only as a gap in the job's trace.
func runCOP(ctx context.Context, pairer COPPairer, req *ipc.PairRequest, run int) (resp *ipc.PairResponse, err error) {
	ctx, span := tracer.Start(ctx, "cop.pair", trace.WithAttributes(
		attribute.String("cop.where", copWhere(pairer)),
		attribute.Int("cop.run", run),
		attribute.Int("cop.players", int(req.ValidPlayers)),
		attribute.Int("cop.all_players", int(req.AllPlayers)),
		attribute.Int("cop.division_sims", int(req.DivisionSims)),
		attribute.Int("cop.control_loss_sims", int(req.ControlLossSims))))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	resp, err = pairer.PairCOP(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("run COP: %w", err)
	}
	span.SetAttributes(attribute.String("cop.error_code", resp.ErrorCode.String()))
	if resp.ErrorCode != ipc.PairError_SUCCESS {
		return nil, fmt.Errorf("COP pairing failed: %s", resp.ErrorMessage)
	}
	return resp, nil
}

func copWhere(pairer COPPairer) string {
	switch pairer.(type) {
	case LocalCOPPairer, *LocalCOPPairer:
		return "local"
	case LambdaCOPPairer, *LambdaCOPPairer:
		return "lambda"
	}
	return "other"
}

// PairRoundGaveUp tells the directors that a round couldn't be paired, once
// its job has failed its last attempt. They can pair it again, or by hand.
func PairRoundGaveUp(ts TournamentStore) func(ctx context.Context, job jobs.Job, err error) {
	return func(ctx context.Context, job jobs.Job, err error) {
		tid, division, round, keyErr := parsePairRoundJobKey(job.Key)
		if keyErr != nil {
			return
		}
		t, getErr := ts.Get(ctx, tid)
		if getErr != nil {
			log.Ctx(ctx).Err(getErr).Str("tid", tid).Msg("pair-round-gave-up-load")
			return
		}
		notifyDirectors(ts, t, []string{entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_AUTO_PAIR_FAILED,
			t.Name, division, strconv.Itoa(round+1)).Error()})
	}
}

func copDivision(t *entity.Tournament, division string) (*ClassicDivision, bool) {
	div, ok := t.Divisions[division]
	if !ok || div.DivisionManager == nil {
		return nil, false
	}
	cd, ok := div.DivisionManager.(*ClassicDivision)
	return cd, ok
}

// applyCOPPairing pairs the round with COP's answer to req, under the
// tournament's lock, if the division would still send the same request.
// Then it does what a director's Pair Round does after pairing: pairs a
// lagged round that was waiting for this one, opens rounds for AutoStart,
// saves, and sends the pairings.
func applyCOPPairing(ctx context.Context, ts TournamentStore, tid, division string, round int,
	req *ipc.PairRequest, resp *ipc.PairResponse) (copOutcome, error) {

	t, lk, err := ts.LockForChange(ctx, tid)
	if err != nil {
		return 0, err
	}
	defer lk.Release()
	cd, ok := copDivision(t, division)
	if !ok || t.IsFinished || !cd.needsCOPPairing(round) {
		return copNotNeeded, nil
	}

	cd.copAnswer = &copAnswer{round: round, req: req, resp: resp}
	defer func() { cd.copAnswer = nil }()
	roundBefore := cd.GetCurrentRound()
	// The round holds what the change that left it kept, byes or whole
	// pairings, and anything paired by hand since: all of it is kept again,
	// as the job's request kept it.
	pairingsResp, err := cd.pairRoundKeeping(round, true, true)
	if err != nil {
		return 0, err
	}
	for _, r := range cd.pendingCOP {
		if r == round {
			// The request differed, so the answer wasn't used and the round
			// was left for a job again.
			return copChanged, nil
		}
	}
	laggedResp, err := cd.PairLaggedRoundAfter(round)
	if err != nil {
		log.Ctx(ctx).Err(err).Str("tid", tid).Str("division", division).Int("round", round+2).Msg("pair-lagged-round-after-cop")
	} else {
		pairingsResp.DivisionPairings = combinePairingsResponses(pairingsResp.DivisionPairings, laggedResp.DivisionPairings)
		pairingsResp.DivisionStandings = combineStandingsResponses(pairingsResp.DivisionStandings, laggedResp.DivisionStandings)
	}
	if cd.DivisionControls.AutoStart {
		cd.autoStartRounds()
	}
	if err := lk.Save(ctx); err != nil {
		return 0, err
	}

	pairingsResp.Id = tid
	pairingsResp.Division = division
	if err := SendTournamentMessage(ctx, ts, tid, entity.WrapEvent(pairingsResp, ipc.MessageType_TOURNAMENT_DIVISION_PAIRINGS_MESSAGE)); err != nil {
		return 0, err
	}
	for r := roundBefore + 1; r <= cd.GetCurrentRound(); r++ {
		if err := sendDivisionStart(ts, t.UUID, division, r); err != nil {
			return 0, err
		}
	}
	notifyDirectors(ts, t, autoPairingWarnings(t, division))
	log.Ctx(ctx).Info().Str("tid", tid).Str("division", division).Int("round", round+1).Msg("cop-pairing-applied")
	return copApplied, nil
}
