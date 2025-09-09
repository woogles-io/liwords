package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/entity/utilities"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"

	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	pkguser "github.com/woogles-io/liwords/pkg/user"
	gs "github.com/woogles-io/liwords/rpc/api/proto/game_service"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// GameRequest utility functions for handling both proto and protojson formats

// ParseGameRequest parses GameRequest from bytes, trying proto format first, then protojson
func ParseGameRequest(data []byte) (*pb.GameRequest, error) {
	if len(data) == 0 {
		return &pb.GameRequest{}, nil
	}

	gr := &pb.GameRequest{}

	// Try proto format first (binary data from live games)
	err := proto.Unmarshal(data, gr)
	if err == nil {
		return gr, nil
	}

	// Fall back to protojson format (from past games)
	err = protojson.Unmarshal(data, gr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GameRequest as both proto and protojson: %w", err)
	}

	return gr, nil
}

// MarshalGameRequestAsProto marshals GameRequest as binary proto for live games table
func MarshalGameRequestAsProto(gr *pb.GameRequest) ([]byte, error) {
	if gr == nil {
		return nil, fmt.Errorf("GameRequest is nil")
	}
	return proto.Marshal(gr)
}

// MarshalGameRequestAsJSON marshals GameRequest as protojson for past games table
func MarshalGameRequestAsJSON(gr *pb.GameRequest) ([]byte, error) {
	if gr == nil {
		return nil, fmt.Errorf("GameRequest is nil")
	}
	return protojson.Marshal(gr)
}

const (
	MaxRecentGames = 1000

	// Migration status constants
	MigrationStatusNotMigrated = 0
	MigrationStatusMigrated    = 1
	MigrationStatusCleaned     = 2
)

// DBStore is a postgres-backed store for games.
type DBStore struct {
	cfg     *config.Config
	dbPool  *pgxpool.Pool
	queries *models.Queries

	userStore pkguser.Store

	// This reference is here so we can copy it to every game we pull
	// from the database.
	// All game events go down the same channel.
	gameEventChan chan<- *entity.EventWrapper

	// Feature flag to control whether to use past_games table
	// When false, uses old queries against games table
	// When true, uses new queries against past_games/game_players tables
	usePastGamesTable bool
}

// type game struct {
// 	gorm.Model
// 	UUID string `gorm:"type:varchar(24);index"`

// 	Type      pb.GameType
// 	Player0ID uint `gorm:"foreignKey;index"`
// 	// Player0   user.User

// 	Player1ID uint `gorm:"foreignKey;index"`
// 	// Player1   user.User

// 	ReadyFlag uint // When both players are ready, this game starts.

// 	Timers datatypes.JSON // A JSON blob containing the game timers.

// 	Started       bool
// 	GameEndReason int `gorm:"index"`
// 	WinnerIdx     int
// 	LoserIdx      int

// 	Quickdata datatypes.JSON // A JSON blob containing the game quickdata.

// 	// Protobuf representations of the game request and history.
// 	Request []byte
// 	History []byte
// 	// Meta Events (abort, adjourn, adjudicate, etc requests)
// 	MetaEvents datatypes.JSON

// 	Stats datatypes.JSON

// 	// This is purposefully not a foreign key. It can be empty/NULL for
// 	// most games.
// 	TournamentID   string `gorm:"index"`
// 	TournamentData datatypes.JSON
// }

// NewDBStore creates a new DB store for games.
func NewDBStore(config *config.Config, userStore pkguser.Store, dbPool *pgxpool.Pool) (*DBStore, error) {
	// Note: We need to manually add the following index on production:
	// create index rematch_req_idx ON games using hash ((quickdata->>'o'));

	// Check environment variable for feature flag
	// Default to false (use old queries) for backward compatibility
	usePastGames := false
	if envVal := os.Getenv("USE_PAST_GAMES_TABLE"); envVal == "true" {
		usePastGames = true
		log.Info().Bool("use_past_games", usePastGames).Msg("past-games-table-feature-flag")
	}

	return &DBStore{
		cfg:               config,
		dbPool:            dbPool,
		userStore:         userStore,
		queries:           models.New(dbPool),
		usePastGamesTable: usePastGames,
	}, nil
}

// SetUsePastGamesTable allows runtime configuration of the feature flag
func (s *DBStore) SetUsePastGamesTable(use bool) {
	s.usePastGamesTable = use
	log.Info().Bool("use_past_games", use).Msg("updated-past-games-table-feature-flag")
}

// SetGameEventChan sets the game event channel to the passed in channel.
func (s *DBStore) SetGameEventChan(c chan<- *entity.EventWrapper) {
	s.gameEventChan = c
}

// GameEventChan returns the game event channel for all games.
func (s *DBStore) GameEventChan() chan<- *entity.EventWrapper {
	return s.gameEventChan
}

// Get creates an instantiated entity.Game from the database.
// This function should almost never be called during a live game.
// The db store should be wrapped with a cache.
// Only API nodes that have this game in its cache should respond to requests.
// XXX: The above comment is obsolete and we will likely redo the way we do caches in the future.
func (s *DBStore) Get(ctx context.Context, id string) (*entity.Game, error) {
	// First get basic info to check migration status without unmarshaling nullable fields
	basicInfo, err := s.queries.GetGameBasicInfo(ctx, common.ToPGTypeText(id))
	if err != nil {
		log.Err(err).Msg("error-get-game-basic-info")
		return nil, err
	}

	// Check if the game has ended
	if basicInfo.GameEndReason.Valid && basicInfo.GameEndReason.Int32 != int32(pb.GameEndReason_NONE) {
		// Game has ended, check migration status
		if basicInfo.MigrationStatus.Valid && basicInfo.MigrationStatus.Int16 >= MigrationStatusMigrated {
			// Game has been migrated to past_games, fetch from there
			// We need to create a minimal Game struct to pass to getFromPastGames
			g := models.Game{
				ID:        basicInfo.ID,
				Uuid:      basicInfo.Uuid,
				CreatedAt: basicInfo.CreatedAt,
				UpdatedAt: basicInfo.UpdatedAt,
			}
			return s.getFromPastGames(ctx, g, true)
		} else {
			// Game ended but not yet migrated (legacy data)
			// Try to get full data - this should work for non-migrated games
			fullGame, err := s.queries.GetGameFullData(ctx, common.ToPGTypeText(id))
			if err != nil {
				log.Err(err).Msg("error-get-game-full-data")
				return nil, err
			}
			return s.inProgressGame(fullGame, true)
		}
	} else {
		// Game is still in progress, get full data
		fullGame, err := s.queries.GetGameFullData(ctx, common.ToPGTypeText(id))
		if err != nil {
			log.Err(err).Msg("error-get-game-full-data-in-progress")
			return nil, err
		}
		return s.inProgressGame(fullGame, true)
	}
}

func (s *DBStore) inProgressGame(g models.Game, playTurns bool) (*entity.Game, error) {
	// convert to an entity.Game
	gr, err := ParseGameRequest(g.Request)
	if err != nil {
		return nil, err
	}
	entGame := &entity.Game{
		Started:        g.Started.Bool,
		Timers:         g.Timers,
		GameEndReason:  pb.GameEndReason(g.GameEndReason.Int32),
		WinnerIdx:      int(g.WinnerIdx.Int32),
		LoserIdx:       int(g.LoserIdx.Int32),
		ChangeHook:     s.gameEventChan,
		PlayerDBIDs:    [2]uint{uint(g.Player0ID.Int32), uint(g.Player1ID.Int32)},
		Stats:          &g.Stats,
		MetaEvents:     &g.MetaEvents,
		Quickdata:      &g.Quickdata,
		CreatedAt:      g.CreatedAt.Time,
		Type:           pb.GameType(g.Type.Int32),
		DBID:           uint(g.ID),
		TournamentData: &g.TournamentData,
		GameReq:        &entity.GameRequest{GameRequest: gr},
	}
	entGame.SetTimerModule(&entity.GameTimer{})
	if playTurns {
		// Then unmarshal the history and start a game from it.
		hist := &macondopb.GameHistory{}
		err := proto.Unmarshal(g.History, hist)
		if err != nil {
			return nil, err
		}
		log.Debug().Interface("hist", hist).Msg("hist-unmarshal")
		return s.playHistory(entGame, hist)
	}
	return entGame, nil
}

func (s *DBStore) playHistory(entGame *entity.Game, hist *macondopb.GameHistory) (*entity.Game, error) {
	lexicon := hist.Lexicon
	if lexicon == "" {
		// This can happen for some early games where we didn't migrate this.
		lexicon = entGame.GameReq.Lexicon
	}

	rules, err := macondogame.NewBasicGameRules(
		s.cfg.MacondoConfig(), lexicon, entGame.GameReq.Rules.BoardLayoutName,
		entGame.GameReq.Rules.LetterDistributionName, macondogame.CrossScoreOnly,
		macondogame.Variant(entGame.GameReq.Rules.VariantName))
	if err != nil {
		return nil, err
	}

	// There's a chance the game is over, so we want to get that state before
	// the following function modifies it.
	histPlayState := hist.GetPlayState()
	// We also want to back up the challenge rule.
	histChallRule := hist.GetChallengeRule()
	// Temporarily set the challenge rule to SINGLE if it was VOID.
	// We want to avoid situations where dictionaries may have been mistakenly
	// updated in place to make some words phonies.
	// (See RD28, which did not stay constant over time)
	if histChallRule == macondopb.ChallengeRule_VOID {
		hist.ChallengeRule = macondopb.ChallengeRule_SINGLE
	}
	log.Debug().Interface("old-play-state", histPlayState).Msg("play-state-loading-hist")

	// This function modifies the history. (XXX it probably shouldn't)
	// It modifies the play state as it plays the game from the beginning.
	mcg, err := macondogame.NewFromHistory(hist, rules, len(hist.Events))
	if err != nil {
		return nil, err
	}
	// XXX: We should probably move this to `NewFromHistory`:
	mcg.SetBackupMode(macondogame.InteractiveGameplayMode)
	// Note: we don't need to set the stack length here, as NewFromHistory
	// above does it.

	entGame.Game = *mcg
	log.Debug().Interface("history", entGame.History()).Msg("from-state")
	// Finally, restore the play state from the passed-in history. This
	// might immediately end the game (for example, the game could have timed
	// out, but the NewFromHistory function doesn't actually handle that).
	// We could consider changing NewFromHistory, but we want it to be as
	// flexible as possible for things like analysis mode.
	entGame.SetPlaying(histPlayState)
	entGame.History().ChallengeRule = histChallRule
	entGame.History().PlayState = histPlayState

	return entGame, nil
}

func (s *DBStore) getFromPastGames(ctx context.Context, g models.Game, playTurns bool) (*entity.Game, error) {
	gid := g.Uuid.String
	createdAt := g.CreatedAt.Time

	pastgame, err := s.queries.GetPastGame(ctx, models.GetPastGameParams{
		Gid:       gid,
		CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
	})
	if err != nil {
		log.Err(err).Msg("error-get-past-game")
		return nil, err
	}

	// Get player IDs from game_players table
	players, err := s.queries.GetGamePlayers(ctx, gid)
	if err != nil {
		log.Err(err).Msg("error-get-game-players")
		return nil, err
	}

	// Map players by index
	var playerDBIDs [2]uint
	for _, p := range players {
		if p.PlayerIndex >= 0 && p.PlayerIndex < 2 {
			playerDBIDs[p.PlayerIndex] = uint(p.PlayerID)
		}
	}
	// Get GameRequest and TournamentData from game_metadata
	metadata, err := s.queries.GetGameMetadata(ctx, gid)
	if err != nil {
		log.Err(err).Msg("error-get-game-metadata")
		return nil, err
	}

	gr, err := ParseGameRequest(metadata.GameRequest)
	if err != nil {
		log.Err(err).Msg("error-unmarshalling-game-request")
		return nil, err
	}

	// Parse tournament data from metadata
	var tournamentData *entity.TournamentData
	if metadata.TournamentData != nil {
		var td entity.TournamentData
		if err := json.Unmarshal(metadata.TournamentData, &td); err == nil {
			tournamentData = &td
		}
	}
	if tournamentData == nil {
		tournamentData = &entity.TournamentData{}
	}

	// convert to an entity.Game
	entGame := &entity.Game{
		CreatedAt:      pastgame.CreatedAt.Time,
		GameEndReason:  pb.GameEndReason(pastgame.GameEndReason),
		GameReq:        &entity.GameRequest{GameRequest: gr},
		Stats:          &pastgame.Stats,
		Quickdata:      &pastgame.Quickdata,
		Type:           pb.GameType(pastgame.Type),
		TournamentData: tournamentData,
		PlayerDBIDs:    playerDBIDs,
		ChangeHook:     s.gameEventChan,
		DBID:           uint(g.ID), // Keep the original game ID
	}

	entGame.SetTimerModule(&entity.GameTimer{})

	winnerIdx := pastgame.WinnerIdx
	if winnerIdx.Valid {
		entGame.WinnerIdx = int(winnerIdx.Int16)
		switch entGame.WinnerIdx {
		case 0:
			entGame.LoserIdx = 1
		case 1:
			entGame.LoserIdx = 0
		case -1:
			entGame.LoserIdx = -1
		default:
			log.Err(fmt.Errorf("invalid winner index: %d", entGame.WinnerIdx)).Msg("invalid-winner-index")
			return nil, fmt.Errorf("invalid winner index: %d", entGame.WinnerIdx)
		}
	}

	if playTurns {
		docbts := pastgame.GameDocument
		if docbts != nil {
			doc := &pb.GameDocument{}
			err = protojson.Unmarshal(docbts, doc)
			if err != nil {
				log.Err(err).Msg("error-unmarshalling-game-document")
				return nil, err
			}
			gh, err := utilities.ToGameHistory(doc, s.cfg)
			if err != nil {
				log.Err(err).Msg("error-converting-game-document")
				return nil, err
			}
			return s.playHistory(entGame, gh)
		}
		return nil, fmt.Errorf("game document is nil")
	}

	return entGame, nil
}

// GetMetadata gets metadata about the game, but does not actually play the game.
func (s *DBStore) GetMetadata(ctx context.Context, id string) (*pb.GameInfoResponse, error) {
	// First get basic info to check migration status without unmarshaling nullable fields
	basicInfo, err := s.queries.GetGameBasicInfo(ctx, common.ToPGTypeText(id))
	if err != nil {
		log.Err(err).Msg("error-get-game-basic-info-in-get-metadata")
		return nil, err
	}

	// Check if the game has ended and been migrated
	if basicInfo.GameEndReason.Valid && basicInfo.GameEndReason.Int32 != int32(pb.GameEndReason_NONE) {
		// Game has ended, check migration status
		if basicInfo.MigrationStatus.Valid && basicInfo.MigrationStatus.Int16 >= MigrationStatusMigrated {
			// Game has been migrated to past_games, fetch from there directly
			pastMeta, err := s.queries.GetPastGameMetadata(ctx, models.GetPastGameMetadataParams{
				Gid:       id,
				CreatedAt: pgtype.Timestamptz{Time: basicInfo.CreatedAt.Time, Valid: true},
			})
			if err != nil {
				log.Err(err).Msg("error-get-past-game-metadata-migrated")
				return nil, err
			}

			gr, err := ParseGameRequest(pastMeta.GameRequest)
			if err != nil {
				log.Err(err).Msg("error-unmarshalling-game-request")
				return nil, err
			}

			// Get time control name
			timefmt := entity.TCRegular
			if gr != nil {
				tc, _, err := entity.VariantFromGameReq(gr)
				if err == nil {
					timefmt = tc
				}
			}

			winner := int32(-1)
			if pastMeta.WinnerIdx.Valid {
				winner = int32(pastMeta.WinnerIdx.Int16)
			}

			// Parse tournament info from JSONB
			var tourneyID string
			if pastMeta.TournamentData != nil {
				var td entity.TournamentData
				if err := json.Unmarshal(pastMeta.TournamentData, &td); err == nil && td.Id != "" {
					tourneyID = td.Id
				}
			}

			return &pb.GameInfoResponse{
				Players:         pastMeta.Quickdata.PlayerInfo,
				GameEndReason:   pb.GameEndReason(pastMeta.GameEndReason),
				Scores:          pastMeta.Quickdata.FinalScores,
				Winner:          winner,
				TimeControlName: string(timefmt),
				CreatedAt:       timestamppb.New(basicInfo.CreatedAt.Time),
				LastUpdate:      timestamppb.New(basicInfo.UpdatedAt.Time),
				GameId:          id,
				GameRequest:     gr,
				TournamentId:    tourneyID,
				Type:            pb.GameType(pastMeta.Type),
			}, nil
		}
	}

	// Game is either in progress or ended but not migrated yet
	// Try to get metadata from games table
	g, err := s.queries.GetLiveGameMetadata(ctx, common.ToPGTypeText(id))
	if err != nil {
		// If this fails for an ended game, it might be a legacy migration without proper status
		// (migrated to past_games but migration_status not set), so try past_games as fallback
		if err == pgx.ErrNoRows || basicInfo.GameEndReason.Valid && basicInfo.GameEndReason.Int32 != int32(pb.GameEndReason_NONE) {
			// Try to get from past_games as fallback for legacy migrations
			pastMeta, err := s.queries.GetPastGameMetadata(ctx, models.GetPastGameMetadataParams{
				Gid:       id,
				CreatedAt: pgtype.Timestamptz{Time: basicInfo.CreatedAt.Time, Valid: true},
			})
			if err != nil {
				log.Err(err).Msg("error-get-past-game-metadata-fallback")
				return nil, err
			}

			gr, err := ParseGameRequest(pastMeta.GameRequest)
			if err != nil {
				log.Err(err).Msg("error-unmarshalling-game-request")
				return nil, err
			}

			// Get time control name
			timefmt := entity.TCRegular
			if gr != nil {
				tc, _, err := entity.VariantFromGameReq(gr)
				if err == nil {
					timefmt = tc
				}
			}

			winner := int32(-1)
			if pastMeta.WinnerIdx.Valid {
				winner = int32(pastMeta.WinnerIdx.Int16)
			}

			// Parse tournament info from JSONB
			var tourneyID string
			if pastMeta.TournamentData != nil {
				var td entity.TournamentData
				if err := json.Unmarshal(pastMeta.TournamentData, &td); err == nil && td.Id != "" {
					tourneyID = td.Id
				}
			}

			return &pb.GameInfoResponse{
				Players:         pastMeta.Quickdata.PlayerInfo,
				GameEndReason:   pb.GameEndReason(pastMeta.GameEndReason),
				Scores:          pastMeta.Quickdata.FinalScores,
				Winner:          winner,
				TimeControlName: string(timefmt),
				CreatedAt:       timestamppb.New(basicInfo.CreatedAt.Time),
				LastUpdate:      timestamppb.New(basicInfo.UpdatedAt.Time),
				GameId:          id,
				GameRequest:     gr,
				TournamentId:    tourneyID,
				Type:            pb.GameType(pastMeta.Type),
			}, nil
		}
		log.Err(err).Msg("error-get-live-game-metadata")
		return nil, err
	}

	// Successfully got metadata from games table
	// Note that the game request is stored as proto in the current games
	// table, but as protojson in past games table. We will likely migrate
	// the current games table to use protojson as well in the future.
	gr, err := ParseGameRequest(g.Request)
	if err != nil {
		log.Err(err).Msg("error-unmarshalling-game-request")
		return nil, err
	}

	// Get time control name
	timefmt := entity.TCRegular
	if gr != nil {
		tc, _, err := entity.VariantFromGameReq(gr)
		if err == nil {
			timefmt = tc
		}
	}

	// Extract tournament info
	var tourneyID string
	if g.TournamentData.Id != "" {
		tourneyID = g.TournamentData.Id
	}

	return &pb.GameInfoResponse{
		Players:         g.Quickdata.PlayerInfo,
		GameEndReason:   pb.GameEndReason(g.GameEndReason.Int32),
		Scores:          g.Quickdata.FinalScores,
		Winner:          int32(g.WinnerIdx.Int32),
		TimeControlName: string(timefmt),
		CreatedAt:       timestamppb.New(g.CreatedAt.Time),
		LastUpdate:      timestamppb.New(g.UpdatedAt.Time),
		GameId:          g.Uuid.String,
		GameRequest:     gr,
		TournamentId:    tourneyID,
		Type:            pb.GameType(g.Type.Int32),
	}, nil
}

// func (s *DBStore)

func (s *DBStore) GetRematchStreak(ctx context.Context, originalRequestId string) (*gs.StreakInfoResponse, error) {
	resp := &gs.StreakInfoResponse{}

	if s.usePastGamesTable {
		// New path: use game_players table
		games, err := s.queries.GetRematchStreak(ctx, originalRequestId)
		if err != nil {
			return nil, err
		}

		resp.Streak = make([]*gs.StreakInfoResponse_SingleGameInfo, len(games))

		if len(games) == 0 {
			return resp, nil
		}

		// Get player info from the first game from past_games table
		firstGameID := games[0].Gid

		// Get the created_at timestamp from games table to query past_games
		gameRow, err := s.queries.GetGameBasicInfo(ctx, common.ToPGTypeText(firstGameID))
		if err != nil {
			return nil, fmt.Errorf("failed to get game for streak: %w", err)
		}

		pastGame, err := s.queries.GetPastGame(ctx, models.GetPastGameParams{
			Gid:       firstGameID,
			CreatedAt: pgtype.Timestamptz{Time: gameRow.CreatedAt.Time, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get past game for streak: %w", err)
		}

		// Extract player info from quickdata
		if len(pastGame.Quickdata.PlayerInfo) >= 2 {
			resp.PlayersInfo = []*gs.StreakInfoResponse_PlayerInfo{
				{Nickname: pastGame.Quickdata.PlayerInfo[0].Nickname, Uuid: pastGame.Quickdata.PlayerInfo[0].UserId},
				{Nickname: pastGame.Quickdata.PlayerInfo[1].Nickname, Uuid: pastGame.Quickdata.PlayerInfo[1].UserId},
			}
		}

		for idx, g := range games {
			resp.Streak[idx] = &gs.StreakInfoResponse_SingleGameInfo{
				GameId: g.Gid,
				Winner: g.WinnerIdx,
			}
		}
	} else {
		// Old path: use games table directly
		games, err := s.queries.GetRematchStreakOld(ctx, originalRequestId)
		if err != nil {
			return nil, err
		}

		resp.Streak = make([]*gs.StreakInfoResponse_SingleGameInfo, len(games))

		if len(games) == 0 {
			return resp, nil
		}

		// Get player info from the first game
		firstGameID := games[0].Gid.String
		firstGame, err := s.queries.GetGameFullData(ctx, common.ToPGTypeText(firstGameID))
		if err != nil {
			return nil, fmt.Errorf("failed to get first game for streak: %w", err)
		}

		// Extract player info from quickdata
		if len(firstGame.Quickdata.PlayerInfo) >= 2 {
			resp.PlayersInfo = []*gs.StreakInfoResponse_PlayerInfo{
				{Nickname: firstGame.Quickdata.PlayerInfo[0].Nickname, Uuid: firstGame.Quickdata.PlayerInfo[0].UserId},
				{Nickname: firstGame.Quickdata.PlayerInfo[1].Nickname, Uuid: firstGame.Quickdata.PlayerInfo[1].UserId},
			}
		}

		for idx, g := range games {
			winner := int32(-1)
			if g.WinnerIdx.Valid {
				winner = g.WinnerIdx.Int32
			}
			resp.Streak[idx] = &gs.StreakInfoResponse_SingleGameInfo{
				GameId: g.Gid.String,
				Winner: winner,
			}
		}
	}

	return resp, nil
}

func (s *DBStore) GetRecentGames(ctx context.Context, username string, numGames int, offset int) (*pb.GameInfoResponses, error) {
	if numGames > MaxRecentGames {
		return nil, errors.New("too many games")
	}

	var responses []*pb.GameInfoResponse

	if s.usePastGamesTable {
		// New path: use game_players and past_games tables
		games, err := s.queries.GetRecentGamesByUsername(ctx, models.GetRecentGamesByUsernameParams{
			Username:    username,
			OffsetGames: int32(offset),
			NumGames:    int32(numGames),
		})
		if err != nil {
			return nil, err
		}

		for _, g := range games {
			// Parse the GameRequest from bytes
			gameRequest, err := ParseGameRequest(g.GameRequest)
			if err != nil {
				log.Err(err).Msg("error-parsing-game-request")
				continue // Skip this game if we can't parse its request
			}

			// Get time control name
			timefmt := entity.TCRegular
			if gameRequest != nil {
				tc, _, err := entity.VariantFromGameReq(gameRequest)
				if err == nil {
					timefmt = tc
				}
			}

			winner := int32(-1)
			if g.WinnerIdx.Valid {
				winner = int32(g.WinnerIdx.Int16)
			}

			// Parse tournament info from JSONB
			var tourneyID string
			var tDiv string
			var tRound int32
			var tGameIndex int32
			if g.TournamentData != nil {
				var td entity.TournamentData
				if err := json.Unmarshal(g.TournamentData, &td); err == nil && td.Id != "" {
					tourneyID = td.Id
					tDiv = td.Division
					tRound = int32(td.Round)
					tGameIndex = int32(td.GameIndex)
				}
			}

			info := &pb.GameInfoResponse{
				Players:             g.Quickdata.PlayerInfo,
				GameEndReason:       pb.GameEndReason(g.GameEndReason),
				Scores:              g.Quickdata.FinalScores,
				Winner:              winner,
				TimeControlName:     string(timefmt),
				CreatedAt:           timestamppb.New(g.CreatedAt.Time),
				LastUpdate:          timestamppb.New(g.CreatedAt.Time), // Using created_at as proxy for last update
				GameId:              g.GameUuid,
				GameRequest:         gameRequest,
				Type:                pb.GameType(g.GameType),
				TournamentId:        tourneyID,
				TournamentDivision:  tDiv,
				TournamentRound:     tRound,
				TournamentGameIndex: tGameIndex,
			}
			responses = append(responses, info)
		}
	} else {
		// Old path: use games table directly
		games, err := s.queries.GetRecentGamesByUsernameOld(ctx, models.GetRecentGamesByUsernameOldParams{
			Username:    pgtype.Text{String: username, Valid: true},
			OffsetGames: int32(offset),
			NumGames:    int32(numGames),
		})
		if err != nil {
			return nil, err
		}

		for _, g := range games {
			// Parse the GameRequest from bytes
			gameRequest, err := ParseGameRequest(g.GameRequest)
			if err != nil {
				log.Err(err).Msg("error-parsing-game-request")
				continue // Skip this game if we can't parse its request
			}

			// Get time control name
			timefmt := entity.TCRegular
			if gameRequest != nil {
				tc, _, err := entity.VariantFromGameReq(gameRequest)
				if err == nil {
					timefmt = tc
				}
			}

			winner := int32(-1)
			if g.WinnerIdx.Valid {
				winner = g.WinnerIdx.Int32
			}

			gameEndReason := pb.GameEndReason_NONE
			if g.GameEndReason.Valid {
				gameEndReason = pb.GameEndReason(g.GameEndReason.Int32)
			}

			gameType := pb.GameType_NATIVE
			if g.GameType.Valid {
				gameType = pb.GameType(g.GameType.Int32)
			}

			info := &pb.GameInfoResponse{
				Players:         g.Quickdata.PlayerInfo,
				GameEndReason:   gameEndReason,
				Scores:          g.Quickdata.FinalScores,
				Winner:          winner,
				TimeControlName: string(timefmt),
				CreatedAt:       timestamppb.New(g.CreatedAt.Time),
				LastUpdate:      timestamppb.New(g.CreatedAt.Time), // Using created_at as proxy for last update
				GameId:          g.GameUuid.String,
				GameRequest:     gameRequest,
				Type:            gameType,
			}
			responses = append(responses, info)
		}
	}

	return &pb.GameInfoResponses{GameInfo: responses}, nil
}

func (s *DBStore) GetRecentTourneyGames(ctx context.Context, tourneyID string, numGames int, offset int) (*pb.GameInfoResponses, error) {
	if numGames > MaxRecentGames {
		return nil, errors.New("too many games")
	}

	var responses []*pb.GameInfoResponse

	if s.usePastGamesTable {
		// New path: use past_games table
		games, err := s.queries.GetRecentTourneyGames(ctx, models.GetRecentTourneyGamesParams{
			TourneyID:   tourneyID,
			OffsetGames: int32(offset),
			NumGames:    int32(numGames),
		})
		if err != nil {
			return nil, err
		}

		for _, g := range games {
			// Parse the GameRequest from bytes
			gameRequest, err := ParseGameRequest(g.GameRequest)
			if err != nil {
				log.Err(err).Msg("error-parsing-game-request")
				continue // Skip this game if we can't parse its request
			}

			// Get time control name
			timefmt := entity.TCRegular
			if gameRequest != nil {
				tc, _, err := entity.VariantFromGameReq(gameRequest)
				if err == nil {
					timefmt = tc
				}
			}

			winner := int32(-1)
			if g.WinnerIdx.Valid {
				winner = int32(g.WinnerIdx.Int16)
			}

			// Parse tournament info from JSONB
			var tDiv string
			var tRound int32
			var tGameIndex int32
			if g.TournamentData != nil {
				var td entity.TournamentData
				if err := json.Unmarshal(g.TournamentData, &td); err == nil {
					tDiv = td.Division
					tRound = int32(td.Round)
					tGameIndex = int32(td.GameIndex)
				}
			}

			info := &pb.GameInfoResponse{
				Players:             g.Quickdata.PlayerInfo,
				GameEndReason:       pb.GameEndReason(g.GameEndReason),
				Scores:              g.Quickdata.FinalScores,
				Winner:              winner,
				TimeControlName:     string(timefmt),
				CreatedAt:           timestamppb.New(g.CreatedAt.Time),
				LastUpdate:          timestamppb.New(g.CreatedAt.Time),
				GameId:              g.Gid,
				TournamentId:        tourneyID,
				GameRequest:         gameRequest,
				TournamentDivision:  tDiv,
				TournamentRound:     tRound,
				TournamentGameIndex: tGameIndex,
				Type:                pb.GameType(g.Type),
			}
			responses = append(responses, info)
		}
	} else {
		// Old path: use games table directly
		games, err := s.queries.GetRecentTourneyGamesOld(ctx, models.GetRecentTourneyGamesOldParams{
			TourneyID:   tourneyID,
			OffsetGames: int32(offset),
			NumGames:    int32(numGames),
		})
		if err != nil {
			return nil, err
		}

		for _, g := range games {
			// Parse the GameRequest from bytes
			gameRequest, err := ParseGameRequest(g.GameRequest)
			if err != nil {
				log.Err(err).Msg("error-parsing-game-request")
				continue // Skip this game if we can't parse its request
			}

			// Get time control name
			timefmt := entity.TCRegular
			if gameRequest != nil {
				tc, _, err := entity.VariantFromGameReq(gameRequest)
				if err == nil {
					timefmt = tc
				}
			}

			winner := int32(-1)
			if g.WinnerIdx.Valid {
				winner = g.WinnerIdx.Int32
			}

			gameEndReason := pb.GameEndReason_NONE
			if g.GameEndReason.Valid {
				gameEndReason = pb.GameEndReason(g.GameEndReason.Int32)
			}

			gameType := pb.GameType_NATIVE
			if g.Type.Valid {
				gameType = pb.GameType(g.Type.Int32)
			}

			// Extract tournament info
			var tDiv string
			var tRound int32
			var tGameIndex int32
			if g.TournamentData.Id != "" {
				tDiv = g.TournamentData.Division
				tRound = int32(g.TournamentData.Round)
				tGameIndex = int32(g.TournamentData.GameIndex)
			}

			info := &pb.GameInfoResponse{
				Players:             g.Quickdata.PlayerInfo,
				GameEndReason:       gameEndReason,
				Scores:              g.Quickdata.FinalScores,
				Winner:              winner,
				TimeControlName:     string(timefmt),
				CreatedAt:           timestamppb.New(g.CreatedAt.Time),
				LastUpdate:          timestamppb.New(g.CreatedAt.Time),
				GameId:              g.Gid.String,
				TournamentId:        tourneyID,
				GameRequest:         gameRequest,
				TournamentDivision:  tDiv,
				TournamentRound:     tRound,
				TournamentGameIndex: tGameIndex,
				Type:                gameType,
			}
			responses = append(responses, info)
		}
	}

	return &pb.GameInfoResponses{GameInfo: responses}, nil
}

// TODO: Remove these GORM-based functions once migrated to sqlc

// Set takes in a game entity that _already exists_ in the DB, and writes it to
// the database.
func (s *DBStore) Set(ctx context.Context, g *entity.Game) error {
	hist, err := proto.Marshal(g.History())
	if err != nil {
		return err
	}

	// Marshal GameRequest as proto for live games table
	requestBytes, err := MarshalGameRequestAsProto(g.GameReq.GameRequest)
	if err != nil {
		return err
	}

	var tourneyID pgtype.Text
	if g.TournamentData != nil && g.TournamentData.Id != "" {
		tourneyID = pgtype.Text{String: g.TournamentData.Id, Valid: true}
	}

	return s.queries.UpdateGame(ctx, models.UpdateGameParams{
		UpdatedAt:      pgtype.Timestamptz{Time: g.CreatedAt, Valid: true}, // Use CreatedAt as proxy
		Player0ID:      pgtype.Int4{Int32: int32(g.PlayerDBIDs[0]), Valid: true},
		Player1ID:      pgtype.Int4{Int32: int32(g.PlayerDBIDs[1]), Valid: true},
		Timers:         g.Timers,
		Started:        pgtype.Bool{Bool: g.Started, Valid: true},
		GameEndReason:  pgtype.Int4{Int32: int32(g.GameEndReason), Valid: true},
		WinnerIdx:      pgtype.Int4{Int32: int32(g.WinnerIdx), Valid: true},
		LoserIdx:       pgtype.Int4{Int32: int32(g.LoserIdx), Valid: true},
		Request:        requestBytes,
		History:        hist,
		Stats:          *g.Stats,
		Quickdata:      *g.Quickdata,
		TournamentData: *g.TournamentData,
		TournamentID:   tourneyID,
		ReadyFlag:      pgtype.Int8{Int64: 0, Valid: true}, // Default to 0
		MetaEvents:     *g.MetaEvents,
		Uuid:           common.ToPGTypeText(g.GameID()),
	})
}

func (s *DBStore) Exists(ctx context.Context, id string) (bool, error) {
	// Check if game exists in games table. Note that we only need to check this
	// table because we don't migrate the ID to the partitioned past games table.
	exists, err := s.queries.GameExists(ctx, common.ToPGTypeText(id))
	if err != nil {
		log.Err(err).Msg("error-checking-game-exists")
		return false, err
	}
	if !exists {
		log.Debug().Str("game_id", id).Msg("game-not-found-in-live-games")
		return false, nil
	}
	return true, nil
}

// Create saves a brand new entity to the database
func (s *DBStore) Create(ctx context.Context, g *entity.Game) error {
	hist, err := proto.Marshal(g.History())
	if err != nil {
		return err
	}

	// Marshal GameRequest as proto for live games table
	requestBytes, err := MarshalGameRequestAsProto(g.GameReq.GameRequest)
	if err != nil {
		return err
	}

	var tourneyID pgtype.Text
	if g.TournamentData != nil && g.TournamentData.Id != "" {
		tourneyID = pgtype.Text{String: g.TournamentData.Id, Valid: true}
	}

	return s.queries.CreateGame(ctx, models.CreateGameParams{
		CreatedAt:      pgtype.Timestamptz{Time: g.CreatedAt, Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Time: g.CreatedAt, Valid: true},
		Uuid:           common.ToPGTypeText(g.GameID()),
		Player0ID:      pgtype.Int4{Int32: int32(g.PlayerDBIDs[0]), Valid: true},
		Player1ID:      pgtype.Int4{Int32: int32(g.PlayerDBIDs[1]), Valid: true},
		Timers:         g.Timers,
		Started:        pgtype.Bool{Bool: g.Started, Valid: true},
		GameEndReason:  pgtype.Int4{Int32: int32(g.GameEndReason), Valid: true},
		WinnerIdx:      pgtype.Int4{Int32: int32(g.WinnerIdx), Valid: true},
		LoserIdx:       pgtype.Int4{Int32: int32(g.LoserIdx), Valid: true},
		Request:        requestBytes,
		History:        hist,
		Stats:          *g.Stats,
		Quickdata:      *g.Quickdata,
		TournamentData: *g.TournamentData,
		TournamentID:   tourneyID,
		ReadyFlag:      pgtype.Int8{Int64: 0, Valid: true}, // Default to 0
		MetaEvents:     *g.MetaEvents,
		Type:           pgtype.Int4{Int32: int32(g.Type), Valid: true},
	})
}

func (s *DBStore) CreateRaw(ctx context.Context, g *entity.Game, gt pb.GameType) error {
	if gt == pb.GameType_NATIVE {
		return fmt.Errorf("this game already exists: %s", g.Uid())
	}

	hist, err := proto.Marshal(g.History())
	if err != nil {
		return err
	}

	// Marshal GameRequest as proto for live games table
	requestBytes, err := MarshalGameRequestAsProto(g.GameReq.GameRequest)
	if err != nil {
		return err
	}

	return s.queries.CreateRawGame(ctx, models.CreateRawGameParams{
		Uuid:          common.ToPGTypeText(g.Uid()),
		Request:       requestBytes,
		History:       hist,
		Quickdata:     *g.Quickdata,
		Timers:        g.Timers,
		GameEndReason: pgtype.Int4{Int32: int32(g.GameEndReason), Valid: true},
		Type:          pgtype.Int4{Int32: int32(gt), Valid: true},
	})
}

func (s *DBStore) ListActive(ctx context.Context, tourneyID string) (*pb.GameInfoResponses, error) {
	var responses []*pb.GameInfoResponse

	if tourneyID != "" {
		games, err := s.queries.ListActiveTournamentGames(ctx, common.ToPGTypeText(tourneyID))
		if err != nil {
			return nil, err
		}
		for _, g := range games {
			info := &pb.GameInfoResponse{
				Players: g.Quickdata.PlayerInfo,
				GameId:  g.Uuid.String,
				Type:    pb.GameType_NATIVE, // Default type for active games
			}
			responses = append(responses, info)
		}
	} else {
		games, err := s.queries.ListActiveGames(ctx)
		if err != nil {
			return nil, err
		}
		for _, g := range games {
			info := &pb.GameInfoResponse{
				Players: g.Quickdata.PlayerInfo,
				GameId:  g.Uuid.String,
				Type:    pb.GameType_NATIVE, // Default type for active games
			}
			responses = append(responses, info)
		}
	}

	return &pb.GameInfoResponses{GameInfo: responses}, nil
}

// List all game IDs, ordered by date played. Should not be used by anything
// other than debug or migration code when the db is still small.
func (s *DBStore) ListAllIDs(ctx context.Context) ([]string, error) {

	ids, err := s.queries.ListAllIDs(ctx)
	if err != nil {
		log.Err(err).Msg("error-listing-all-ids")
		return nil, err
	}
	gameIDs := make([]string, len(ids))
	for i, id := range ids {
		gameIDs[i] = id.String
	}
	return gameIDs, nil
}

func (s *DBStore) SetReady(ctx context.Context, gid string, pidx int) (int, error) {
	readyRes, err := s.queries.SetReady(ctx, models.SetReadyParams{
		PlayerIdx: int32(pidx),
		Uuid:      common.ToPGTypeText(gid),
	})

	if err != nil {
		// Only error now is if game doesn't exist
		log.Err(err).Int("playerIdx", pidx).Str("gid", gid).Msg("setting-ready")
		return 0, err
	}

	log.Debug().Int("playerIdx", pidx).Str("gid", gid).Int("readyFlag", int(readyRes.Int64)).Msg("player-set-ready")
	return int(readyRes.Int64), nil
}

// TODO: Remove this GORM-based function

func (s *DBStore) Disconnect() {
	log.Warn().Msg("game-store-disconnect-not-implemented")
}

func (s *DBStore) CachedCount(ctx context.Context) int {
	return 0
}

func (s *DBStore) GetHistory(ctx context.Context, id string) (*macondopb.GameHistory, error) {
	// First check if the game has been migrated
	basicInfo, err := s.queries.GetGameBasicInfo(ctx, common.ToPGTypeText(id))
	if err != nil {
		log.Err(err).Msg("error-get-game-basic-info-in-get-history")
		return nil, err
	}

	// Check if the game has been migrated to past_games
	if basicInfo.MigrationStatus.Valid && basicInfo.MigrationStatus.Int16 >= MigrationStatusMigrated {
		// Game has been migrated, get from past_games
		pastGame, err := s.queries.GetPastGame(ctx, models.GetPastGameParams{
			Gid:       id,
			CreatedAt: pgtype.Timestamptz{Time: basicInfo.CreatedAt.Time, Valid: true},
		})
		if err != nil {
			log.Err(err).Msg("error-get-past-game-in-get-history")
			return nil, err
		}

		// Parse the game document
		doc := &pb.GameDocument{}
		err = protojson.Unmarshal(pastGame.GameDocument, doc)
		if err != nil {
			log.Err(err).Msg("error-unmarshalling-game-document-in-get-history")
			return nil, err
		}

		// Convert to game history
		gh, err := utilities.ToGameHistory(doc, s.cfg)
		if err != nil {
			log.Err(err).Msg("error-converting-game-document-to-history")
			return nil, err
		}
		log.Debug().Interface("hist", gh).Msg("got-history-from-past-games")
		return gh, nil
	}

	// Game not migrated, try to get from games table
	bts, err := s.queries.GetHistory(ctx, common.ToPGTypeText(id))
	if err != nil {
		// If this fails, it might be a legacy migration without proper status
		// Try to get from past_games as fallback
		if basicInfo.GameEndReason.Valid && basicInfo.GameEndReason.Int32 != int32(pb.GameEndReason_NONE) {
			pastGame, err := s.queries.GetPastGame(ctx, models.GetPastGameParams{
				Gid:       id,
				CreatedAt: pgtype.Timestamptz{Time: basicInfo.CreatedAt.Time, Valid: true},
			})
			if err != nil {
				log.Err(err).Msg("error-get-past-game-fallback-in-get-history")
				return nil, err
			}

			// Parse the game document
			doc := &pb.GameDocument{}
			err = protojson.Unmarshal(pastGame.GameDocument, doc)
			if err != nil {
				log.Err(err).Msg("error-unmarshalling-game-document-fallback")
				return nil, err
			}

			// Convert to game history
			gh, err := utilities.ToGameHistory(doc, s.cfg)
			if err != nil {
				log.Err(err).Msg("error-converting-game-document-to-history-fallback")
				return nil, err
			}
			log.Debug().Interface("hist", gh).Msg("got-history-from-past-games-fallback")
			return gh, nil
		}
		log.Err(err).Msg("error-get-history")
		return nil, err
	}

	hist := &macondopb.GameHistory{}
	err = proto.Unmarshal(bts, hist)
	if err != nil {
		return nil, err
	}
	log.Debug().Interface("hist", hist).Msg("got-history")
	return hist, nil
}
