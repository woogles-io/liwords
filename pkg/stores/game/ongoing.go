package game

// Writing the live position to ongoing_games.
//
// This is phase 3 of the storage migration (docs/mikado/liwords_referee.md).
// The governing rule is that macondo stays authoritative: nothing here is read
// back for a gameplay decision, and nothing here may fail a move. Every path
// below logs and returns nil, so a bug in the snapshot code costs a log line
// rather than a game.
//
// The write hangs off DBStore.Set rather than its twelve call sites. Set
// already receives the *entity.Game, which embeds the macondo game, so the
// snapshot can be derived in one place that nobody has to remember to call.
// That also makes the migration in-place: the statement is an upsert, so a game
// already in progress when this deploys acquires its row on its next save, and
// a game never touched again never needs converting.

import (
	"context"
	"sync/atomic"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/xwordbridge"
	"github.com/woogles-io/liwords/pkg/xwordgame"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// maybeWriteOngoingGame derives an xwordgame snapshot of g and, depending on
// configuration, compares it against macondo and persists it.
//
// It never returns an error and never panics. Callers are on the move path, so
// a snapshot problem must not become a player-visible failure -- and "returns
// no error" is not the same promise as "cannot bring down the request", which
// is why the recover is here rather than trusted to nil checks upstream.
func (s *DBStore) maybeWriteOngoingGame(ctx context.Context, g *entity.Game) {
	cfg, err := config.Ctx(ctx)
	if err != nil || (!cfg.ShadowXwordState && !cfg.WriteXwordState) {
		return
	}
	log := zerolog.Ctx(ctx)

	// Read the id up front, and defensively: a game malformed enough to panic
	// the code below is malformed enough to panic GameID(), and a panic raised
	// inside a recover handler is not recovered -- it takes the process down.
	// The first version of this did exactly that.
	gid := safeGameID(g)

	defer func() {
		if r := recover(); r != nil {
			// Deliberately swallowed. This is shadow work; macondo remains
			// authoritative and the move has already been written.
			logBounded(log.Error().Interface("panic", r).Str("gameID", gid),
				&panicCount, "ongoing-game-panic")
		}
	}()

	// A finished game is not ongoing. Removing the row is what ends its life
	// here; the archive path reads game_turns, not this table.
	if g.Game.Playing() == macondopb.PlayState_GAME_OVER {
		if cfg.WriteXwordState {
			if err := s.queries.DeleteOngoingGame(ctx, gid); err != nil {
				logBounded(log.Err(err).Str("gameID", gid),
					&deleteErrCount, "ongoing-game-delete-error")
			}
		}
		return
	}

	state, err := xwordbridge.StateFromGame(&g.Game)
	if err != nil {
		// Expected for games this package cannot yet represent, and for a board
		// read mid-transposition. Neither is a reason to fail a move.
		logBounded(log.Warn().Err(err).Str("gameID", gid),
			&snapshotSkipCount, "ongoing-game-snapshot-skipped")
		return
	}

	if cfg.ShadowXwordState {
		s.checkSnapshot(ctx, g, state)
	}
	if !cfg.WriteXwordState {
		return
	}

	params, err := upsertParamsFor(g, state)
	if err != nil {
		logBounded(log.Warn().Err(err).Str("gameID", gid),
			&paramsSkipCount, "ongoing-game-params-skipped")
		return
	}
	if err := s.queries.UpsertOngoingGame(ctx, params); err != nil {
		logBounded(log.Err(err).Str("gameID", gid),
			&upsertErrCount, "ongoing-game-upsert-error")
	}
}

// checkSnapshot runs the assertions that only real traffic can exercise:
// production sees board sizes, distributions, variants and game shapes that no
// test enumerates, and this is where a conversion or serialization bug shows up
// on a real game rather than a synthetic one.
//
// All three checks are cheap -- tens of microseconds against a move that
// already costs a database round trip -- and all three are read-only. A failure
// is logged at ERROR and changes nothing.
func (s *DBStore) checkSnapshot(ctx context.Context, g *entity.Game, state *xwordgame.State) {
	log := zerolog.Ctx(ctx)
	gid := g.GameID()

	if err := state.Validate(); err != nil {
		logBounded(log.Error().Err(err).Str("gameID", gid),
			&invalidStateCount, "xword-shadow-invalid-state")
		return
	}

	// The invariant that matters most: every tile in the distribution accounted
	// for exactly once, in the bag, on a rack, or on the board. This is what
	// catches a corrupt position at the moment it is created rather than after
	// it has been played on for weeks.
	if ld := s.letterDistributionFor(g); ld != nil {
		if err := state.ValidateTileConservation(ld); err != nil {
			logBounded(log.Error().Err(err).Str("gameID", gid),
				&notConservedCount, "xword-shadow-tiles-not-conserved")
		}
	}

	// A snapshot that encodes fine but decodes wrong looks perfect until it
	// makes a round trip, which is precisely what persisting it will do.
	var decoded xwordgame.State
	if err := decoded.Decode(state.Encode()); err != nil {
		logBounded(log.Error().Err(err).Str("gameID", gid),
			&decodeFailCount, "xword-shadow-decode-failed")
		return
	}
	if !decoded.Equal(state) {
		logBounded(log.Error().Str("gameID", gid).
			Uint64("digest", state.Digest()).
			Uint64("decodedDigest", decoded.Digest()),
			&roundTripCount, "xword-shadow-round-trip-mismatch")
	}
}

// letterDistributionFor resolves a game's letter distribution. word-golib
// caches these, so this is a map lookup after the first game of each kind.
// Returns nil rather than an error: a missing distribution means one check is
// skipped, not that anything is wrong with the game.
func (s *DBStore) letterDistributionFor(g *entity.Game) *tilemapping.LetterDistribution {
	if s == nil || s.cfg == nil || s.cfg.MacondoConfig() == nil {
		return nil
	}
	name := "english"
	if req := g.GameReq; req != nil && req.GameRequest != nil &&
		req.Rules != nil && req.Rules.LetterDistributionName != "" {
		name = req.Rules.LetterDistributionName
	}
	ld, err := tilemapping.GetDistribution(s.cfg.MacondoConfig().WGLConfig(), name)
	if err != nil {
		return nil
	}
	return ld
}

// safeGameID reads a game's id without trusting it not to panic.
func safeGameID(g *entity.Game) (id string) {
	defer func() {
		if recover() != nil {
			id = "<unreadable>"
		}
	}()
	if g == nil {
		return "<nil>"
	}
	return g.GameID()
}

// Counters for the bounded logger below. A fixed set of package-level atomics,
// not a map: nothing here grows with the number of games, and each process
// counting its own occurrences is exactly what is wanted from logs.
var (
	panicCount        atomic.Uint64
	snapshotSkipCount atomic.Uint64
	paramsSkipCount   atomic.Uint64
	upsertErrCount    atomic.Uint64
	deleteErrCount    atomic.Uint64
	invalidStateCount atomic.Uint64
	notConservedCount atomic.Uint64
	decodeFailCount   atomic.Uint64
	roundTripCount    atomic.Uint64
)

const (
	// logBurst is how many of each kind to log before thinning out.
	logBurst = 50
	// logEvery is the sampling interval after the burst.
	logEvery = 500
)

// logBounded emits an event with a running count, thinning out after the first
// logBurst so a systemic failure cannot flood the logs.
//
// The point of shadow mode is to find out what breaks on real traffic, so the
// first occurrences are all logged. But a bug that fires on every move would
// otherwise emit one line per move across every game on the server, and the
// resulting volume would be its own incident. The count rides on every line, so
// the scale is visible even when the lines are not.
func logBounded(e *zerolog.Event, counter *atomic.Uint64, msg string) {
	n := counter.Add(1)
	if n > logBurst && n%logEvery != 0 {
		return
	}
	e.Uint64("occurrence", n).Msg(msg)
}

// upsertParamsFor builds the row for a live game.
func upsertParamsFor(g *entity.Game, state *xwordgame.State) (models.UpsertOngoingGameParams, error) {
	var p models.UpsertOngoingGameParams

	hist := g.History()
	req := g.GameReq

	lexicon := ""
	if hist != nil {
		lexicon = hist.Lexicon
	}
	boardLayout, letterDist, variant := entity.CrosswordGame, "english", "classic"
	var challengeRule int16
	gameMode, gameType := int16(0), int16(0)
	if req != nil && req.GameRequest != nil {
		if lexicon == "" {
			lexicon = req.Lexicon
		}
		if req.Rules != nil {
			if req.Rules.BoardLayoutName != "" {
				boardLayout = req.Rules.BoardLayoutName
			}
			if req.Rules.LetterDistributionName != "" {
				letterDist = req.Rules.LetterDistributionName
			}
			if req.Rules.VariantName != "" {
				variant = req.Rules.VariantName
			}
		}
		challengeRule = int16(req.ChallengeRule)
		gameMode = int16(req.GameMode)
	}
	if hist != nil {
		// The history's challenge rule is the one the game is actually being
		// played under; the request is only the opening ask.
		challengeRule = int16(hist.ChallengeRule)
	}
	if g.Type != pb.GameType_NATIVE {
		gameType = int16(g.Type)
	}

	timers, err := g.Timers.Value()
	if err != nil {
		return p, err
	}
	timersJSON, _ := timers.([]byte)

	var metaJSON []byte
	if g.MetaEvents != nil {
		if v, err := g.MetaEvents.Value(); err == nil {
			metaJSON, _ = v.([]byte)
		}
	}

	var tourneyID pgtype.Text
	if g.TournamentData != nil && g.TournamentData.Id != "" {
		tourneyID = pgtype.Text{String: g.TournamentData.Id, Valid: true}
	}
	var leagueID, seasonID, divisionID pgtype.UUID
	if g.LeagueID != nil {
		leagueID = pgtype.UUID{Bytes: *g.LeagueID, Valid: true}
	}
	if g.SeasonID != nil {
		seasonID = pgtype.UUID{Bytes: *g.SeasonID, Valid: true}
	}
	if g.LeagueDivisionID != nil {
		divisionID = pgtype.UUID{Bytes: *g.LeagueDivisionID, Valid: true}
	}

	return models.UpsertOngoingGameParams{
		GameUuid:  g.GameID(),
		State:     state.Encode(),
		PlayState: int16(state.PlayState),
		OnTurn:    pgtype.Int2{Int16: int16(state.OnTurn), Valid: true},

		Lexicon:            lexicon,
		LetterDistribution: letterDist,
		BoardLayout:        boardLayout,
		Variant:            variant,
		ChallengeRule:      challengeRule,

		Player0ID: int32(g.PlayerDBIDs[0]),
		Player1ID: int32(g.PlayerDBIDs[1]),

		Timers:     timersJSON,
		MetaEvents: metaJSON,
		ReadyFlag:  0,
		Started:    g.Started,

		GameMode:         gameMode,
		GameType:         gameType,
		TournamentID:     tourneyID,
		LeagueID:         leagueID,
		SeasonID:         seasonID,
		LeagueDivisionID: divisionID,

		// Non-empty exactly when the last move was a tile placement, which is
		// the only move a challenge can act on. The statement uses this to
		// decide whether to rotate the outgoing position into prev_state.
		HasChallengeablePlay: len(state.LastWordsFormed) > 0,
	}, nil
}
