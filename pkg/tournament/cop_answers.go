package tournament

// Running COP without holding the tournament.
//
// COP pairs a round by simulating the rest of the tournament, which takes
// seconds (about 5 on two cores at the default 100,000 simulations, mid
// event). It used to run inside the change that triggered it, holding the
// tournament's lock (pkg/stores/tournament/lock.go): when the last result of
// a round came in, every other change to the tournament, in every division,
// waited for it, and with more than one server a change waiting on another
// server could give up after the lock timeout and lose a result.
//
// So a change that may pair with COP goes through lockForChangeWithCOP:
//
//  1. Lock, load, and apply the change, with COP not allowed to compute. If
//     pairing needs a COP answer it doesn't have, it records COP's request
//     and fails, and the change is abandoned: nothing was saved or sent.
//  2. Release the lock, and run COP for the recorded requests, holding
//     nothing.
//  3. Lock, load and apply again. Pairing builds the same request from the
//     same state and gets its answer. If the tournament changed in the
//     meantime (another result came in), the request differs and the answer
//     isn't used: back to 2.
//
// An answer is only ever used for a request equal to the one it answered, so
// a change applied this way pairs exactly as if COP had run inside it. The
// last attempt lets COP compute under the lock, as before, so a tournament
// that keeps changing still gets paired.

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/pair/cop"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// copAttempts bounds the applications of one change. The last one computes
// COP under the lock if it must.
const copAttempts = 3

// runCOP is COP; tests wrap it.
var runCOP = cop.COPPair

// errCOPDeferred is how pairing fails when it needs a COP answer that isn't
// ready. The change that hit it is abandoned and applied again.
var errCOPDeferred = errors.New("COP pairing deferred until it runs outside the tournament lock")

// copAnswers holds COP's answers for one change, across its attempts.
type copAnswers struct {
	answers []copAnswer
	// missed are the requests asked for since the last takeMissed that had
	// no answer.
	missed []*pb.PairRequest
	// compute lets pair run COP itself on a miss, under the lock.
	compute bool
	run func(*pb.PairRequest) *pb.PairResponse
}

type copAnswer struct {
	req  *pb.PairRequest
	resp *pb.PairResponse
}

func newCOPAnswers() *copAnswers {
	return &copAnswers{run: runCOP}
}

// pair returns COP's answer to req. With no copAnswers attached, COP runs
// right away, as it always did.
func (c *copAnswers) pair(req *pb.PairRequest) (*pb.PairResponse, error) {
	if c == nil {
		return cop.COPPair(req), nil
	}
	for _, a := range c.answers {
		if proto.Equal(a.req, req) {
			return proto.Clone(a.resp).(*pb.PairResponse), nil
		}
	}
	if !c.compute {
		c.missed = append(c.missed, proto.Clone(req).(*pb.PairRequest))
		return nil, errCOPDeferred
	}
	resp := c.run(req)
	c.answers = append(c.answers, copAnswer{req: proto.Clone(req).(*pb.PairRequest), resp: resp})
	return proto.Clone(resp).(*pb.PairResponse), nil
}

// answerMissed runs COP for the requests that had no answer.
func (c *copAnswers) answerMissed() {
	for _, req := range c.missed {
		c.answers = append(c.answers, copAnswer{req: req, resp: c.run(req)})
	}
	c.missed = nil
}

// attach makes pairing in every division of t use c.
func (c *copAnswers) attach(t *entity.Tournament) {
	for _, d := range t.Divisions {
		if cd, ok := d.DivisionManager.(*ClassicDivision); ok {
			cd.cop = c
		}
	}
}

// lockForChangeWithCOP is LockForChange for a change that may pair rounds
// with COP (see the top of this file). apply makes the change to the loaded
// tournament, in memory only: it may run more than once, each time on a
// fresh load, and only the last run is kept. Its error is returned with the
// lock still held, like everything else, so the caller always Releases.
func lockForChangeWithCOP(ctx context.Context, ts TournamentStore, id string,
	apply func(t *entity.Tournament) error) (*entity.Tournament, TournamentLock, error) {
	return lockForChangeWith(ctx, ts, id, newCOPAnswers(), apply)
}

func lockForChangeWith(ctx context.Context, ts TournamentStore, id string, answers *copAnswers,
	apply func(t *entity.Tournament) error) (*entity.Tournament, TournamentLock, error) {
	span := trace.SpanFromContext(ctx)
	for attempt := 1; ; attempt++ {
		answers.compute = attempt == copAttempts
		t, lk, err := ts.LockForChange(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		answers.attach(t)
		err = applyHolding(lk, t, apply)
		if len(answers.missed) == 0 {
			if attempt > 1 {
				span.SetAttributes(attribute.Int("cop.attempts", attempt))
			}
			return t, lk, err
		}
		// Abandon this attempt and pair outside the lock.
		lk.Release()
		span.AddEvent("cop-outside-lock", trace.WithAttributes(
			attribute.Int("cop.requests", len(answers.missed)), attribute.Int("cop.attempt", attempt)))
		log.Info().Str("tid", id).Int("requests", len(answers.missed)).Int("attempt", attempt).
			Msg("cop-pairing-outside-lock")
		answers.answerMissed()
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
	}
}

// applyHolding runs apply while lk is held. If apply panics, lk is released
// on the way out: the caller's deferred Release isn't in place yet, and a
// recovered panic (the bus recovers its handlers') must not leave the
// tournament locked.
func applyHolding(lk TournamentLock, t *entity.Tournament, apply func(*entity.Tournament) error) error {
	returned := false
	defer func() {
		if !returned {
			lk.Release()
		}
	}()
	err := apply(t)
	returned = true
	return err
}
