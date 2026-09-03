package game

// Keeping ongoing_games in step with a live game.
//
// The table exists so that "which games are in progress" is a scan of a few
// thousand rows rather than a filter over 12M. It holds no position: whose turn
// it is and what phase a game is in are functions of the event log, and the two
// columns here that name them are for finding rows, not for answering the
// question. See the migration.
//
// Written inside Set's transaction, with the games row, deliberately. A listing
// that disagrees with the game it names is worse than no listing, and the only
// way to be sure they agree is to write them together. The cost is that a
// failure here fails the move, which is why it is behind a flag until it has
// been watched.

import (
	"context"
	"fmt"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// syncOngoingGame upserts or removes this game's ongoing_games row.
//
// A finished game is not ongoing, so ending one deletes the row -- in the same
// transaction that records the outcome, so a game can never be absent from
// `games` results and present in the listings at the same time.
func syncOngoingGame(ctx context.Context, q *models.Queries, g *entity.Game) error {
	if g.Playing() == macondopb.PlayState_GAME_OVER || g.GameEndReason != pb.GameEndReason_NONE {
		return q.DeleteOngoingGame(ctx, g.GameID())
	}
	params, err := ongoingParams(g)
	if err != nil {
		return err
	}
	return q.UpsertOngoingGame(ctx, params)
}

// ongoingParams projects a live game onto its row.
func ongoingParams(g *entity.Game) (models.UpsertOngoingGameParams, error) {
	var p models.UpsertOngoingGameParams

	// player0_id and player1_id are NOT NULL REFERENCES users(id). An unset id
	// would fail the foreign key, and because this runs inside Set's
	// transaction that would abort the whole save -- so refuse here, where it
	// is a returned error and not a poisoned transaction.
	if g.PlayerDBIDs[0] == 0 || g.PlayerDBIDs[1] == 0 {
		return p, fmt.Errorf("game: %s has no player database ids", g.GameID())
	}

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
		GameUuid: g.GameID(),

		// Derived from the game, and only so the listings can filter on them.
		// Nothing reads these back into a game; DBStore.Get rebuilds a position
		// by replaying the event log.
		PlayState: int16(g.Playing()),
		OnTurn:    pgtype.Int2{Int16: int16(g.PlayerOnTurn()), Valid: true},

		Lexicon:            lexicon,
		LetterDistribution: letterDist,
		BoardLayout:        boardLayout,
		Variant:            variant,
		ChallengeRule:      challengeRule,

		Player0ID: int32(g.PlayerDBIDs[0]),
		Player1ID: int32(g.PlayerDBIDs[1]),

		Timers:     timersJSON,
		MetaEvents: metaJSON,
		// Readiness is owned by SetReady, which ORs a bit into the row; the
		// upsert must not clobber it, and the query leaves it alone on
		// conflict. This value therefore only applies to the first insert.
		ReadyFlag: 0,
		Started:   g.Started,

		GameMode:         gameMode,
		GameType:         gameType,
		TournamentID:     tourneyID,
		LeagueID:         leagueID,
		SeasonID:         seasonID,
		LeagueDivisionID: divisionID,
	}, nil
}
