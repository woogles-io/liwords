package game

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

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

	return &DBStore{
		cfg:       config,
		dbPool:    dbPool,
		userStore: userStore,
		queries:   models.New(dbPool),
	}, nil
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

	g, err := s.queries.GetLiveGame(ctx, common.ToPGTypeText(id))
	if err != nil {
		log.Err(err).Msg("error-get-game")
		return nil, err
	}
	
	// Check if the game has ended
	if g.GameEndReason.Valid && g.GameEndReason.Int32 != int32(pb.GameEndReason_NONE) {
		// Game has ended, check migration status
		if g.MigrationStatus.Valid && g.MigrationStatus.Int16 >= MigrationStatusMigrated {
			// Game has been migrated to past_games
			return s.getFromPastGames(ctx, g, true)
		} else {
			// Game ended but not yet migrated (legacy data)
			return s.inProgressGame(g, true)
		}
	} else {
		// Game is still in progress
		return s.inProgressGame(g, true)
	}
}

func (s *DBStore) inProgressGame(g models.Game, playTurns bool) (*entity.Game, error) {
	// convert to an entity.Game
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
		GameReq:        &entity.GameRequest{GameRequest: g.Request.GameRequest},
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
	
	// convert to an entity.Game
	entGame := &entity.Game{
		CreatedAt:      pastgame.CreatedAt.Time,
		GameEndReason:  pb.GameEndReason(pastgame.GameEndReason),
		GameReq:        &entity.GameRequest{GameRequest: pastgame.GameRequest.GameRequest},
		Stats:          &pastgame.Stats,
		Quickdata:      &pastgame.Quickdata,
		Type:           pb.GameType(pastgame.Type),
		TournamentData: pastgame.TournamentData, // could be null
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
			doc := &ipc.GameDocument{}
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

// Legacy method for backwards compatibility during migration
func (s *DBStore) pastGame(g models.Game, playTurns bool) (*entity.Game, error) {
	// This method is kept for legacy games that haven't been migrated yet
	// It extracts data from the existing games table
	return s.inProgressGame(g, playTurns)
}

// GetMetadata gets metadata about the game, but does not actually play the game.
func (s *DBStore) GetMetadata(ctx context.Context, id string) (*pb.GameInfoResponse, error) {

	g, err := s.queries.GetLiveGameMetadata(ctx, common.ToPGTypeText(id))
	if err != nil {
		if err == sql.ErrNoRows {
			// Game might be in past_games, let's check
			// First we need to get basic info from games table to know the created_at
			gameInfo, err := s.queries.GetLiveGame(ctx, common.ToPGTypeText(id))
			if err != nil {
				log.Err(err).Msg("error-get-game")
				return nil, err
			}
			
			// Now get from past_games
			pastMeta, err := s.queries.GetPastGameMetadata(ctx, models.GetPastGameMetadataParams{
				Gid:       id,
				CreatedAt: pgtype.Timestamptz{Time: gameInfo.CreatedAt.Time, Valid: true},
			})
			if err != nil {
				log.Err(err).Msg("error-get-past-game-metadata")
				return nil, err
			}
			
			// Get time control name
			timefmt := entity.TCRegular
			if pastMeta.GameRequest.GameRequest != nil {
				tc, _, err := entity.VariantFromGameReq(pastMeta.GameRequest.GameRequest)
				if err == nil {
					timefmt = tc
				}
			}
			
			winner := int32(-1)
			if pastMeta.WinnerIdx.Valid {
				winner = int32(pastMeta.WinnerIdx.Int16)
			}
			
			// Extract tournament info
			var tourneyID string
			if pastMeta.TournamentData != nil && pastMeta.TournamentData.Id != "" {
				tourneyID = pastMeta.TournamentData.Id
			}
			
			return &pb.GameInfoResponse{
				Players:         pastMeta.Quickdata.PlayerInfo,
				GameEndReason:   pb.GameEndReason(pastMeta.GameEndReason),
				Scores:          pastMeta.Quickdata.FinalScores,
				Winner:          winner,
				TimeControlName: string(timefmt),
				CreatedAt:       timestamppb.New(gameInfo.CreatedAt.Time),
				LastUpdate:      timestamppb.New(gameInfo.UpdatedAt.Time),
				GameId:          id,
				GameRequest:     pastMeta.GameRequest.GameRequest,
				TournamentId:    tourneyID,
				Type:            pb.GameType(pastMeta.Type),
			}, nil
		}
		log.Err(err).Msg("error-get-game")
		return nil, err
	}
	
	// Get time control name
	timefmt := entity.TCRegular
	if g.Request.GameRequest != nil {
		tc, _, err := entity.VariantFromGameReq(g.Request.GameRequest)
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
		GameRequest:     g.Request.GameRequest,
		TournamentId:    tourneyID,
		Type:            pb.GameType(g.Type.Int32),
	}, nil
}

// func (s *DBStore)

func (s *DBStore) GetRematchStreak(ctx context.Context, originalRequestId string) (*gs.StreakInfoResponse, error) {
	games, err := s.queries.GetRematchStreak(ctx, originalRequestId)
	if err != nil {
		return nil, err
	}

	resp := &gs.StreakInfoResponse{
		Streak: make([]*gs.StreakInfoResponse_SingleGameInfo, len(games)),
	}

	if len(games) == 0 {
		return resp, nil
	}

	for idx, g := range games {
		if idx == 0 {
			playersInfo := make([]*gs.StreakInfoResponse_PlayerInfo, len(g.Quickdata.PlayerInfo))
			for i, p := range g.Quickdata.PlayerInfo {
				playersInfo[i] = &gs.StreakInfoResponse_PlayerInfo{
					Nickname: p.Nickname,
					Uuid:     p.UserId,
				}
			}
			resp.PlayersInfo = playersInfo
		}
		
		winner := int32(-1)
		if g.WinnerIdx.Valid {
			winner = int32(g.WinnerIdx.Int16)
			
			// Handle player order mismatch
			if len(resp.PlayersInfo) > 0 && len(g.Quickdata.PlayerInfo) > 0 &&
				resp.PlayersInfo[0].Nickname != g.Quickdata.PlayerInfo[0].Nickname {
				if winner != -1 {
					winner = 1 - winner
				}
			}
		}
		
		resp.Streak[idx] = &gs.StreakInfoResponse_SingleGameInfo{
			GameId: g.Gid,
			Winner: winner,
		}
	}

	return resp, nil
}

func (s *DBStore) GetRecentGames(ctx context.Context, username string, numGames int, offset int) (*pb.GameInfoResponses, error) {
	if numGames > MaxRecentGames {
		return nil, errors.New("too many games")
	}

	games, err := s.queries.GetRecentGamesByUsername(ctx, models.GetRecentGamesByUsernameParams{
		Username:    username,
		OffsetGames: int32(offset),
		NumGames:    int32(numGames),
	})
	if err != nil {
		return nil, err
	}

	var responses []*pb.GameInfoResponse
	for _, g := range games {
		// Get time control name
		timefmt := entity.TCRegular
		if g.GameRequest.GameRequest != nil {
			tc, _, err := entity.VariantFromGameReq(g.GameRequest.GameRequest)
			if err == nil {
				timefmt = tc
			}
		}

		winner := int32(-1)
		if g.WinnerIdx.Valid {
			winner = int32(g.WinnerIdx.Int16)
		}

		info := &pb.GameInfoResponse{
			Players:         g.Quickdata.PlayerInfo,
			GameEndReason:   pb.GameEndReason(g.GameEndReason),
			Scores:          g.Quickdata.FinalScores,
			Winner:          winner,
			TimeControlName: string(timefmt),
			CreatedAt:       timestamppb.New(g.CreatedAt.Time),
			LastUpdate:      timestamppb.New(g.CreatedAt.Time), // Using created_at as proxy for last update
			GameId:          g.GameUuid,
			GameRequest:     g.GameRequest.GameRequest, // Access the underlying pb.GameRequest
			Type:            pb.GameType(g.GameType),
		}
		responses = append(responses, info)
	}

	return &pb.GameInfoResponses{GameInfo: responses}, nil
}

func (s *DBStore) GetRecentTourneyGames(ctx context.Context, tourneyID string, numGames int, offset int) (*pb.GameInfoResponses, error) {
	if numGames > MaxRecentGames {
		return nil, errors.New("too many games")
	}

	games, err := s.queries.GetRecentTourneyGames(ctx, models.GetRecentTourneyGamesParams{
		TourneyID:   tourneyID,
		OffsetGames: int32(offset),
		NumGames:    int32(numGames),
	})
	if err != nil {
		return nil, err
	}

	var responses []*pb.GameInfoResponse
	for _, g := range games {
		// Get time control name
		timefmt := entity.TCRegular
		if g.GameRequest.GameRequest != nil {
			tc, _, err := entity.VariantFromGameReq(g.GameRequest.GameRequest)
			if err == nil {
				timefmt = tc
			}
		}

		winner := int32(-1)
		if g.WinnerIdx.Valid {
			winner = int32(g.WinnerIdx.Int16)
		}

		// Extract tournament info
		var tDiv string
		var tRound int32
		var tGameIndex int32
		if g.TournamentData != nil {
			tDiv = g.TournamentData.Division
			tRound = int32(g.TournamentData.Round)
			tGameIndex = int32(g.TournamentData.GameIndex)
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
			GameRequest:         g.GameRequest.GameRequest, // Access the underlying pb.GameRequest
			TournamentDivision:  tDiv,
			TournamentRound:     tRound,
			TournamentGameIndex: tGameIndex,
			Type:                pb.GameType(g.Type),
		}
		responses = append(responses, info)
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
		Request:        *g.GameReq,
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
	// Check if game exists in games table
	_, err := s.queries.GetLiveGame(ctx, common.ToPGTypeText(id))
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Create saves a brand new entity to the database
func (s *DBStore) Create(ctx context.Context, g *entity.Game) error {
	hist, err := proto.Marshal(g.History())
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
		Request:        *g.GameReq,
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

	return s.queries.CreateRawGame(ctx, models.CreateRawGameParams{
		Uuid:          common.ToPGTypeText(g.Uid()),
		Request:       *g.GameReq,
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
				Players:   g.Quickdata.PlayerInfo,
				GameId:    g.Uuid.String,
				Type:      pb.GameType_NATIVE, // Default type for active games
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

	return int(readyRes.Int64), err
}

// TODO: Remove this GORM-based function

func (s *DBStore) Disconnect() {
	log.Warn().Msg("game-store-disconnect-not-implemented")
}

func (s *DBStore) CachedCount(ctx context.Context) int {
	return 0
}

func (s *DBStore) GetHistory(ctx context.Context, id string) (*macondopb.GameHistory, error) {
	bts, err := s.queries.GetHistory(ctx, common.ToPGTypeText(id))
	if err != nil {
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
