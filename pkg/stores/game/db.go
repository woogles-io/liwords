package game

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"

	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	pkguser "github.com/woogles-io/liwords/pkg/user"
	gs "github.com/woogles-io/liwords/rpc/api/proto/game_service"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	MaxRecentGames = 1000
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
	g, err := s.queries.GetGame(ctx, common.ToPGTypeText(id))
	if err != nil {
		log.Err(err).Msg("error-get-game")
		return nil, err
	}

	// Convert to an entity.Game
	gamereq := &pb.GameRequest{}
	if len(g.Request) == 0 {
		return nil, fmt.Errorf("game %s has no request data", id)
	}
	err = proto.Unmarshal(g.Request, gamereq)
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
		GameReq:        &entity.GameRequest{GameRequest: gamereq},
	}
	entGame.SetTimerModule(&entity.GameTimer{})

	// Then unmarshal the history and start a game from it.
	hist := &macondopb.GameHistory{}
	err = proto.Unmarshal(g.History, hist)
	if err != nil {
		return nil, err
	}
	log.Debug().Interface("hist", hist).Msg("hist-unmarshal")

	lexicon := hist.Lexicon
	if lexicon == "" {
		// This can happen for some early games where we didn't migrate this.
		lexicon = gamereq.Lexicon
	}

	rules, err := macondogame.NewBasicGameRules(
		s.cfg.MacondoConfig(), lexicon, gamereq.Rules.BoardLayoutName,
		gamereq.Rules.LetterDistributionName, macondogame.CrossScoreOnly,
		macondogame.Variant(gamereq.Rules.VariantName))
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

// GetMetadata gets metadata about the game, but does not actually play the game.
func (s *DBStore) GetMetadata(ctx context.Context, id string) (*pb.GameInfoResponse, error) {
	g, err := s.queries.GetGameMetadata(ctx, common.ToPGTypeText(id))
	if err != nil {
		return nil, err
	}

	mdata := g.Quickdata

	gamereq := &pb.GameRequest{}
	if len(g.Request) == 0 {
		return nil, fmt.Errorf("game %s has no request data", id)
	}
	err = proto.Unmarshal(g.Request, gamereq)
	if err != nil {
		return nil, err
	}

	timefmt, _, err := entity.VariantFromGameReq(gamereq)
	if err != nil {
		return nil, err
	}

	trdata := g.TournamentData
	tDiv := trdata.Division
	tRound := trdata.Round
	tGameIndex := trdata.GameIndex
	tid := trdata.Id

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

	info := &pb.GameInfoResponse{
		Players:             mdata.PlayerInfo,
		GameEndReason:       gameEndReason,
		Scores:              mdata.FinalScores,
		Winner:              winner,
		TimeControlName:     string(timefmt),
		CreatedAt:           timestamppb.New(g.CreatedAt.Time),
		LastUpdate:          timestamppb.New(g.UpdatedAt.Time),
		GameId:              g.Uuid.String,
		TournamentId:        tid,
		GameRequest:         gamereq,
		TournamentDivision:  tDiv,
		TournamentRound:     int32(tRound),
		TournamentGameIndex: int32(tGameIndex),
		Type:                gameType,
	}
	return info, nil
}

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
		// Use quickdata from each individual game (like original GORM implementation)
		mdata := g.Quickdata
		
		if idx == 0 {
			// Establish consistent player ordering from first game, sorted by nickname descending
			playersInfo := make([]*gs.StreakInfoResponse_PlayerInfo, len(mdata.PlayerInfo))
			for i, p := range mdata.PlayerInfo {
				playersInfo[i] = &gs.StreakInfoResponse_PlayerInfo{
					Nickname: p.Nickname,
					Uuid:     p.UserId,
				}
			}
			// Sort by nickname descending (like original)
			sort.Slice(playersInfo, func(i, j int) bool { return playersInfo[i].Nickname > playersInfo[j].Nickname })
			resp.PlayersInfo = playersInfo
		}
		
		winner := int32(-1)
		if g.WinnerIdx.Valid {
			winner = g.WinnerIdx.Int32
		}
		
		// Normalize winner index like original implementation
		if len(resp.PlayersInfo) > 0 && len(mdata.PlayerInfo) > 0 &&
			resp.PlayersInfo[0].Nickname != mdata.PlayerInfo[0].Nickname {
			
			if winner != -1 {
				winner = 1 - winner  // Flip the winner index
			}
		}
		
		resp.Streak[idx] = &gs.StreakInfoResponse_SingleGameInfo{
			GameId: g.Uuid.String,
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
		NumGames:    int32(numGames),
		OffsetGames: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	responses := []*pb.GameInfoResponse{}
	for _, g := range games {
		// Convert directly from GetRecentGamesRow
		mdata := g.Quickdata

		gamereq := &pb.GameRequest{}
		if len(g.Request) == 0 {
			return nil, fmt.Errorf("game has no request data")
		}
		err := proto.Unmarshal(g.Request, gamereq)
		if err != nil {
			return nil, err
		}

		timefmt, _, err := entity.VariantFromGameReq(gamereq)
		if err != nil {
			return nil, err
		}

		trdata := g.TournamentData
		tDiv := trdata.Division
		tRound := trdata.Round
		tGameIndex := trdata.GameIndex
		tid := trdata.Id

		winner := int32(-1)
		if g.WinnerIdx.Valid {
			winner = g.WinnerIdx.Int32
		}

		endReason := pb.GameEndReason_NONE
		if g.GameEndReason.Valid {
			endReason = pb.GameEndReason(g.GameEndReason.Int32)
		}

		gType := pb.GameType_NATIVE
		if g.Type.Valid {
			gType = pb.GameType(g.Type.Int32)
		}

		info := &pb.GameInfoResponse{
			Players:             mdata.PlayerInfo,
			GameEndReason:       endReason,
			Scores:              mdata.FinalScores,
			Winner:              winner,
			TimeControlName:     string(timefmt),
			CreatedAt:           timestamppb.New(g.CreatedAt.Time),
			LastUpdate:          timestamppb.New(g.UpdatedAt.Time),
			GameId:              g.Uuid.String,
			TournamentId:        tid,
			GameRequest:         gamereq,
			TournamentDivision:  tDiv,
			TournamentRound:     int32(tRound),
			TournamentGameIndex: int32(tGameIndex),
			Type:                gType,
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
		NumGames:    int32(numGames),
		OffsetGames: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	responses := []*pb.GameInfoResponse{}
	for _, g := range games {
		// Convert directly from GetRecentGamesRow
		mdata := g.Quickdata

		gamereq := &pb.GameRequest{}
		if len(g.Request) == 0 {
			return nil, fmt.Errorf("game has no request data")
		}
		err := proto.Unmarshal(g.Request, gamereq)
		if err != nil {
			return nil, err
		}

		timefmt, _, err := entity.VariantFromGameReq(gamereq)
		if err != nil {
			return nil, err
		}

		trdata := g.TournamentData
		tDiv := trdata.Division
		tRound := trdata.Round
		tGameIndex := trdata.GameIndex
		tid := trdata.Id

		winner := int32(-1)
		if g.WinnerIdx.Valid {
			winner = g.WinnerIdx.Int32
		}

		endReason := pb.GameEndReason_NONE
		if g.GameEndReason.Valid {
			endReason = pb.GameEndReason(g.GameEndReason.Int32)
		}

		gType := pb.GameType_NATIVE
		if g.Type.Valid {
			gType = pb.GameType(g.Type.Int32)
		}

		info := &pb.GameInfoResponse{
			Players:             mdata.PlayerInfo,
			GameEndReason:       endReason,
			Scores:              mdata.FinalScores,
			Winner:              winner,
			TimeControlName:     string(timefmt),
			CreatedAt:           timestamppb.New(g.CreatedAt.Time),
			LastUpdate:          timestamppb.New(g.UpdatedAt.Time),
			GameId:              g.Uuid.String,
			TournamentId:        tid,
			GameRequest:         gamereq,
			TournamentDivision:  tDiv,
			TournamentRound:     int32(tRound),
			TournamentGameIndex: int32(tGameIndex),
			Type:                gType,
		}
		responses = append(responses, info)
	}

	return &pb.GameInfoResponses{GameInfo: responses}, nil
}


// Set takes in a game entity that _already exists_ in the DB, and writes it to
// the database.
func (s *DBStore) Set(ctx context.Context, g *entity.Game) error {
	hist, err := proto.Marshal(g.History())
	if err != nil {
		return err
	}

	reqBytes, err := proto.Marshal(g.GameReq.GameRequest)
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
		Request:        reqBytes,
		History:        hist,
		Stats:          safeDerefStats(g.Stats),
		Quickdata:      safeDerefQuickdata(g.Quickdata),
		TournamentData: safeDerefTournamentData(g.TournamentData),
		TournamentID:   tourneyID,
		ReadyFlag:      pgtype.Int8{Int64: 0, Valid: true}, // Default to 0
		MetaEvents:     safeDerefMetaEvents(g.MetaEvents),
		Uuid:           common.ToPGTypeText(g.GameID()),
	})
}

func (s *DBStore) Exists(ctx context.Context, id string) (bool, error) {
	exists, err := s.queries.GameExists(ctx, common.ToPGTypeText(id))
	if err != nil {
		log.Err(err).Msg("error-checking-game-exists")
		return false, err
	}
	return exists, nil
}

// Create saves a brand new entity to the database
func (s *DBStore) Create(ctx context.Context, g *entity.Game) error {
	hist, err := proto.Marshal(g.History())
	if err != nil {
		return err
	}

	reqBytes, err := proto.Marshal(g.GameReq.GameRequest)
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
		Request:        reqBytes,
		History:        hist,
		Stats:          safeDerefStats(g.Stats),
		Quickdata:      safeDerefQuickdata(g.Quickdata),
		TournamentData: safeDerefTournamentData(g.TournamentData),
		TournamentID:   tourneyID,
		ReadyFlag:      pgtype.Int8{Int64: 0, Valid: true}, // Default to 0
		MetaEvents:     safeDerefMetaEvents(g.MetaEvents),
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

	reqBytes, err := proto.Marshal(g.GameReq.GameRequest)
	if err != nil {
		return err
	}

	return s.queries.CreateRawGame(ctx, models.CreateRawGameParams{
		Uuid:          common.ToPGTypeText(g.Uid()),
		Request:       reqBytes,
		History:       hist,
		Quickdata:     safeDerefQuickdata(g.Quickdata),
		Timers:        g.Timers,
		GameEndReason: pgtype.Int4{Int32: int32(g.GameEndReason), Valid: true},
		Type:          pgtype.Int4{Int32: int32(gt), Valid: true},
	})
}

func (s *DBStore) ListActive(ctx context.Context, tourneyID string) (*pb.GameInfoResponses, error) {
	var responses []*pb.GameInfoResponse

	if tourneyID != "" {
		games, err := s.queries.ListActiveTournamentGames(ctx, tourneyID)
		if err != nil {
			return nil, err
		}
		for _, g := range games {
			mdata := g.Quickdata
			
			// Unmarshal the GameRequest from stored bytes
			gamereq := &pb.GameRequest{}
			err := proto.Unmarshal(g.Request, gamereq)
			if err != nil {
				return nil, err
			}
			
			info := &pb.GameInfoResponse{
				Players:     mdata.PlayerInfo,
				GameId:      g.Uuid.String,
				GameRequest: gamereq,
				Type:        pb.GameType_NATIVE, // Default type for active games
			}
			responses = append(responses, info)
		}
	} else {
		games, err := s.queries.ListActiveGames(ctx)
		if err != nil {
			return nil, err
		}
		for _, g := range games {
			mdata := g.Quickdata
			
			// Unmarshal the GameRequest from stored bytes
			gamereq := &pb.GameRequest{}
			err := proto.Unmarshal(g.Request, gamereq)
			if err != nil {
				return nil, err
			}
			
			info := &pb.GameInfoResponse{
				Players:     mdata.PlayerInfo,
				GameId:      g.Uuid.String,
				GameRequest: gamereq,
				Type:        pb.GameType_NATIVE, // Default type for active games
			}
			responses = append(responses, info)
		}
	}

	return &pb.GameInfoResponses{GameInfo: responses}, nil
}

func (s *DBStore) Count(ctx context.Context) (int64, error) {
	count, err := s.queries.GameCount(ctx)
	if err != nil {
		return 0, err
	}
	return count, nil
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
	readyFlag, err := s.queries.SetReady(ctx, models.SetReadyParams{
		PlayerIdx: int32(pidx),
		Uuid:      common.ToPGTypeText(gid),
	})

	if err != nil {
		// Handle case where no rows are updated (game doesn't exist or already ready)
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		log.Err(err).Int("playerIdx", pidx).Str("gid", gid).Msg("setting-ready")
		return 0, err
	}

	log.Debug().Int("playerIdx", pidx).Str("gid", gid).Int("readyFlag", int(readyFlag.Int64)).Msg("player-set-ready")
	return int(readyFlag.Int64), nil
}

func (s *DBStore) Disconnect() {
	// pgxpool handles connection pooling, just log that we're "disconnecting"
	log.Info().Msg("game-store-disconnect-called")
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

// Helper functions to safely dereference potentially nil entity pointers
func safeDerefStats(stats *entity.Stats) entity.Stats {
	if stats != nil {
		return *stats
	}
	return entity.Stats{} // Return zero value if nil
}

func safeDerefQuickdata(quickdata *entity.Quickdata) entity.Quickdata {
	if quickdata != nil {
		return *quickdata
	}
	return entity.Quickdata{} // Return zero value if nil
}

func safeDerefTournamentData(tournamentData *entity.TournamentData) entity.TournamentData {
	if tournamentData != nil {
		return *tournamentData
	}
	return entity.TournamentData{} // Return zero value if nil
}

func safeDerefMetaEvents(metaEvents *entity.MetaEventData) entity.MetaEventData {
	if metaEvents != nil {
		return *metaEvents
	}
	return entity.MetaEventData{} // Return zero value if nil
}