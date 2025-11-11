package game

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
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
	"github.com/woogles-io/liwords/pkg/cwgame/board"
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

	// timerModuleCreator creates timer modules for games loaded from the database
	timerModuleCreator TimerModuleCreator
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
		// Default timer module creator
		timerModuleCreator: func() entity.Nower {
			return &entity.GameTimer{}
		},
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

// SetTimerModuleCreator sets the function used to create timer modules for games.
func (s *DBStore) SetTimerModuleCreator(creator TimerModuleCreator) {
	s.timerModuleCreator = creator
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
	gamereq := &g.GameRequest

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
		GameReq:        gamereq,
	}
	if s.timerModuleCreator != nil {
		entGame.SetTimerModule(s.timerModuleCreator())
		log.Info().Int64("Now", entGame.TimerModule().Now()).Msg("set timer module from creator")
	} else {
		// Fallback to default if creator is nil
		entGame.SetTimerModule(&entity.GameTimer{})
	}

	// Populate league-related fields if they exist
	if g.LeagueID.Valid {
		leagueID := uuid.UUID(g.LeagueID.Bytes)
		entGame.LeagueID = &leagueID
	}
	if g.SeasonID.Valid {
		seasonID := uuid.UUID(g.SeasonID.Bytes)
		entGame.SeasonID = &seasonID
	}
	if g.LeagueDivisionID.Valid {
		divisionID := uuid.UUID(g.LeagueDivisionID.Bytes)
		entGame.LeagueDivisionID = &divisionID
	}

	// Then unmarshal the history and start a game from it.
	hist := &macondopb.GameHistory{}
	err = proto.Unmarshal(g.History, hist)
	if err != nil {
		return nil, err
	}
	log.Debug().Interface("hist", hist).Msg("hist-unmarshal")

	// Check if history has no players. This can indicate a null history, like
	// for an annotated game, which should not be handled by this code path.
	if len(hist.Players) == 0 {
		return nil, fmt.Errorf("game %s has no players", id)
	}

	lexicon := hist.Lexicon
	if lexicon == "" {
		// This can happen for some early games where we didn't migrate this.
		lexicon = gamereq.Lexicon
	}

	// Handle cases where Rules might be nil for old games
	// XXX: this really shouldn't happen but i don't want to crash
	var boardLayoutName, letterDistributionName, variantName string
	if gamereq.Rules != nil {
		boardLayoutName = gamereq.Rules.BoardLayoutName
		letterDistributionName = gamereq.Rules.LetterDistributionName
		variantName = gamereq.Rules.VariantName
	} else {
		// Use sensible defaults for old games without Rules
		boardLayoutName = board.CrosswordGameLayout
		letterDistributionName = "english"
		variantName = "classic"
	}

	rules, err := macondogame.NewBasicGameRules(
		s.cfg.MacondoConfig(), lexicon, boardLayoutName,
		letterDistributionName, macondogame.CrossScoreOnly,
		macondogame.Variant(variantName))
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
	log.Debug().Str("id", id).Msg("get-metadata-dbstore")
	g, err := s.queries.GetGameMetadata(ctx, common.ToPGTypeText(id))
	if err != nil {
		return nil, err
	}

	mdata := g.Quickdata

	gamereq := &g.GameRequest

	timefmt := safeTimeControlFromGameReq(gamereq, id)

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
		GameRequest:         gamereq.GameRequest,
		TournamentDivision:  tDiv,
		TournamentRound:     int32(tRound),
		TournamentGameIndex: int32(tGameIndex),
		Type:                gameType,
	}

	// Populate league fields if present
	if g.LeagueID.Valid {
		leagueUUID := uuid.UUID(g.LeagueID.Bytes)
		info.LeagueId = leagueUUID.String()

		// Fetch league slug from database
		league, err := s.queries.GetLeagueByUUID(ctx, leagueUUID)
		if err == nil {
			info.LeagueSlug = league.Slug
		} else {
			log.Warn().Err(err).Str("league_id", leagueUUID.String()).Msg("failed to fetch league slug")
		}
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
				winner = 1 - winner // Flip the winner index
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

		gamereq := &g.GameRequest

		timefmt := safeTimeControlFromGameReq(gamereq, g.Uuid.String)

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
			GameRequest:         gamereq.GameRequest,
			TournamentDivision:  tDiv,
			TournamentRound:     int32(tRound),
			TournamentGameIndex: int32(tGameIndex),
			Type:                gType,
		}

		// Populate league fields if present
		if g.LeagueID.Valid {
			leagueUUID := uuid.UUID(g.LeagueID.Bytes)
			info.LeagueId = leagueUUID.String()

			league, err := s.queries.GetLeagueByUUID(ctx, leagueUUID)
			if err == nil {
				info.LeagueSlug = league.Slug
			}
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
		// Convert directly from GetRecentTourneyGamesRow
		mdata := g.Quickdata

		gamereq := &g.GameRequest

		timefmt := safeTimeControlFromGameReq(gamereq, g.Uuid.String)

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
			GameRequest:         gamereq.GameRequest,
			TournamentDivision:  tDiv,
			TournamentRound:     int32(tRound),
			TournamentGameIndex: int32(tGameIndex),
			Type:                gType,
		}
		responses = append(responses, info)
	}

	return &pb.GameInfoResponses{GameInfo: responses}, nil
}

func (s *DBStore) GetRecentCorrespondenceGames(ctx context.Context, username string, numGames int) (*pb.GameInfoResponses, error) {
	if numGames > MaxRecentGames {
		return nil, errors.New("too many games")
	}

	games, err := s.queries.GetRecentCorrespondenceGamesByUsername(ctx, models.GetRecentCorrespondenceGamesByUsernameParams{
		Username: username,
		NumGames: int32(numGames),
	})
	if err != nil {
		return nil, err
	}

	responses := []*pb.GameInfoResponse{}
	for _, g := range games {
		// Convert directly from GetRecentCorrespondenceGamesRow
		mdata := g.Quickdata

		gamereq := &g.GameRequest

		timefmt := safeTimeControlFromGameReq(gamereq, g.Uuid.String)

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
			GameRequest:         gamereq.GameRequest,
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

	var tourneyID pgtype.Text
	if g.TournamentData != nil && g.TournamentData.Id != "" {
		tourneyID = pgtype.Text{String: g.TournamentData.Id, Valid: true}
	}

	var leagueID pgtype.UUID
	if g.LeagueID != nil {
		leagueID = pgtype.UUID{Bytes: *g.LeagueID, Valid: true}
	}

	var seasonID pgtype.UUID
	if g.SeasonID != nil {
		seasonID = pgtype.UUID{Bytes: *g.SeasonID, Valid: true}
	}

	var leagueDivisionID pgtype.UUID
	if g.LeagueDivisionID != nil {
		leagueDivisionID = pgtype.UUID{Bytes: *g.LeagueDivisionID, Valid: true}
	}

	return s.queries.UpdateGame(ctx, models.UpdateGameParams{
		UpdatedAt:        pgtype.Timestamptz{Time: time.UnixMilli(g.TimerModule().Now()), Valid: true},
		Player0ID:        pgtype.Int4{Int32: int32(g.PlayerDBIDs[0]), Valid: true},
		Player1ID:        pgtype.Int4{Int32: int32(g.PlayerDBIDs[1]), Valid: true},
		Timers:           g.Timers,
		Started:          pgtype.Bool{Bool: g.Started, Valid: true},
		GameEndReason:    pgtype.Int4{Int32: int32(g.GameEndReason), Valid: true},
		WinnerIdx:        pgtype.Int4{Int32: int32(g.WinnerIdx), Valid: true},
		LoserIdx:         pgtype.Int4{Int32: int32(g.LoserIdx), Valid: true},
		History:          hist,
		Stats:            safeDerefStats(g.Stats),
		Quickdata:        safeDerefQuickdata(g.Quickdata),
		TournamentData:   safeDerefTournamentData(g.TournamentData),
		TournamentID:     tourneyID,
		ReadyFlag:        pgtype.Int8{Int64: 0, Valid: true}, // Default to 0
		MetaEvents:       safeDerefMetaEvents(g.MetaEvents),
		Uuid:             common.ToPGTypeText(g.GameID()),
		GameRequest:      safeDerefGameRequest(g.GameReq),
		PlayerOnTurn:     pgtype.Int4{Int32: int32(g.PlayerOnTurn()), Valid: true},
		LeagueID:         leagueID,
		SeasonID:         seasonID,
		LeagueDivisionID: leagueDivisionID,
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

	var tourneyID pgtype.Text
	if g.TournamentData != nil && g.TournamentData.Id != "" {
		tourneyID = pgtype.Text{String: g.TournamentData.Id, Valid: true}
	}

	var leagueID pgtype.UUID
	if g.LeagueID != nil {
		leagueID = pgtype.UUID{Bytes: *g.LeagueID, Valid: true}
	}

	var seasonID pgtype.UUID
	if g.SeasonID != nil {
		seasonID = pgtype.UUID{Bytes: *g.SeasonID, Valid: true}
	}

	var leagueDivisionID pgtype.UUID
	if g.LeagueDivisionID != nil {
		leagueDivisionID = pgtype.UUID{Bytes: *g.LeagueDivisionID, Valid: true}
	}

	return s.queries.CreateGame(ctx, models.CreateGameParams{
		CreatedAt:        pgtype.Timestamptz{Time: g.CreatedAt, Valid: true},
		UpdatedAt:        pgtype.Timestamptz{Time: g.CreatedAt, Valid: true},
		Uuid:             common.ToPGTypeText(g.GameID()),
		Player0ID:        pgtype.Int4{Int32: int32(g.PlayerDBIDs[0]), Valid: true},
		Player1ID:        pgtype.Int4{Int32: int32(g.PlayerDBIDs[1]), Valid: true},
		Timers:           g.Timers,
		Started:          pgtype.Bool{Bool: g.Started, Valid: true},
		GameEndReason:    pgtype.Int4{Int32: int32(g.GameEndReason), Valid: true},
		WinnerIdx:        pgtype.Int4{Int32: int32(g.WinnerIdx), Valid: true},
		LoserIdx:         pgtype.Int4{Int32: int32(g.LoserIdx), Valid: true},
		History:          hist,
		Stats:            safeDerefStats(g.Stats),
		Quickdata:        safeDerefQuickdata(g.Quickdata),
		TournamentData:   safeDerefTournamentData(g.TournamentData),
		TournamentID:     tourneyID,
		ReadyFlag:        pgtype.Int8{Int64: 0, Valid: true}, // Default to 0
		MetaEvents:       safeDerefMetaEvents(g.MetaEvents),
		Type:             pgtype.Int4{Int32: int32(g.Type), Valid: true},
		GameRequest:      safeDerefGameRequest(g.GameReq),
		PlayerOnTurn:     pgtype.Int4{Int32: 0, Valid: true}, // First player starts
		LeagueID:         leagueID,
		SeasonID:         seasonID,
		LeagueDivisionID: leagueDivisionID,
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
		History:       hist,
		Quickdata:     safeDerefQuickdata(g.Quickdata),
		Timers:        g.Timers,
		GameEndReason: pgtype.Int4{Int32: int32(g.GameEndReason), Valid: true},
		Type:          pgtype.Int4{Int32: int32(gt), Valid: true},
		GameRequest:   safeDerefGameRequest(g.GameReq),
	})
}

// ListActive lists all active games, except for correspondence games.
func (s *DBStore) ListActive(ctx context.Context, tourneyID string, bust bool) (*pb.GameInfoResponses, error) {
	var responses []*pb.GameInfoResponse

	if tourneyID != "" {
		games, err := s.queries.ListActiveTournamentGames(ctx, tourneyID)
		if err != nil {
			return nil, err
		}
		for _, g := range games {
			mdata := g.Quickdata
			trdata := g.TournamentData

			// Get the GameRequest from the entity
			gamereq := &g.GameRequest

			info := &pb.GameInfoResponse{
				Players:             mdata.PlayerInfo,
				GameId:              g.Uuid.String,
				GameRequest:         gamereq.GameRequest,
				Type:                pb.GameType_NATIVE, // Default type for active games
				TournamentId:        trdata.Id,
				TournamentDivision:  trdata.Division,
				TournamentRound:     int32(trdata.Round),
				TournamentGameIndex: int32(trdata.GameIndex),
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
			trdata := g.TournamentData

			// Get the GameRequest from the entity
			gamereq := &g.GameRequest

			info := &pb.GameInfoResponse{
				Players:             mdata.PlayerInfo,
				GameId:              g.Uuid.String,
				GameRequest:         gamereq.GameRequest,
				Type:                pb.GameType_NATIVE, // Default type for active games
				TournamentId:        trdata.Id,
				TournamentDivision:  trdata.Division,
				TournamentRound:     int32(trdata.Round),
				TournamentGameIndex: int32(trdata.GameIndex),
			}
			responses = append(responses, info)
		}
	}
	log.Debug().Int("num-active", len(responses)).Msg("list-active")
	return &pb.GameInfoResponses{GameInfo: responses}, nil
}

// ListActiveCorrespondence lists all active correspondence games.
func (s *DBStore) ListActiveCorrespondence(ctx context.Context) (*pb.GameInfoResponses, error) {
	var responses []*pb.GameInfoResponse

	games, err := s.ListActiveCorrespondenceRaw(ctx)
	if err != nil {
		return nil, err
	}

	for _, g := range games {
		mdata := g.Quickdata
		trdata := g.TournamentData

		// Get the GameRequest from the entity
		gamereq := &g.GameRequest

		info := &pb.GameInfoResponse{
			Players:             mdata.PlayerInfo,
			GameId:              g.Uuid.String,
			GameRequest:         gamereq.GameRequest,
			Type:                pb.GameType_NATIVE,
			TournamentId:        trdata.Id,
			TournamentDivision:  trdata.Division,
			TournamentRound:     int32(trdata.Round),
			TournamentGameIndex: int32(trdata.GameIndex),
			LastUpdate:          timestamppb.New(g.UpdatedAt.Time),
		}
		if g.PlayerOnTurn.Valid {
			playerOnTurn := uint32(g.PlayerOnTurn.Int32)
			info.PlayerOnTurn = &playerOnTurn
		}

		// Populate league fields if present
		if g.LeagueID.Valid {
			leagueUUID := uuid.UUID(g.LeagueID.Bytes)
			info.LeagueId = leagueUUID.String()

			// Fetch league slug from database
			league, err := s.queries.GetLeagueByUUID(ctx, leagueUUID)
			if err == nil {
				info.LeagueSlug = league.Slug
			} else {
				log.Warn().Err(err).Str("league_id", leagueUUID.String()).Msg("failed to fetch league slug")
			}
		}

		responses = append(responses, info)
	}
	log.Debug().Int("num-correspondence", len(responses)).Msg("list-active-correspondence")
	return &pb.GameInfoResponses{GameInfo: responses}, nil
}

// ListActiveCorrespondenceForUser lists active correspondence games for a specific user.
func (s *DBStore) ListActiveCorrespondenceForUser(ctx context.Context, userID string) (*pb.GameInfoResponses, error) {
	var responses []*pb.GameInfoResponse

	games, err := s.queries.ListActiveCorrespondenceGamesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, g := range games {
		mdata := g.Quickdata
		trdata := g.TournamentData

		// Get the GameRequest from the entity
		gamereq := &g.GameRequest

		info := &pb.GameInfoResponse{
			Players:             mdata.PlayerInfo,
			GameId:              g.Uuid.String,
			GameRequest:         gamereq.GameRequest,
			Type:                pb.GameType_NATIVE,
			TournamentId:        trdata.Id,
			TournamentDivision:  trdata.Division,
			TournamentRound:     int32(trdata.Round),
			TournamentGameIndex: int32(trdata.GameIndex),
			LastUpdate:          timestamppb.New(g.UpdatedAt.Time),
		}
		if g.PlayerOnTurn.Valid {
			playerOnTurn := uint32(g.PlayerOnTurn.Int32)
			info.PlayerOnTurn = &playerOnTurn
		}

		// Populate league fields if present
		if g.LeagueID.Valid {
			leagueUUID := uuid.UUID(g.LeagueID.Bytes)
			info.LeagueId = leagueUUID.String()

			// Fetch league slug from database
			league, err := s.queries.GetLeagueByUUID(ctx, leagueUUID)
			if err == nil {
				info.LeagueSlug = league.Slug
			} else {
				log.Warn().Err(err).Str("league_id", leagueUUID.String()).Msg("failed to fetch league slug")
			}
		}

		responses = append(responses, info)
	}
	log.Debug().Int("num-correspondence", len(responses)).Str("user", userID).Msg("list-active-correspondence-for-user")
	return &pb.GameInfoResponses{GameInfo: responses}, nil
}

// ListActiveCorrespondenceForUserAndLeague lists active correspondence games for a specific user in a specific league.
func (s *DBStore) ListActiveCorrespondenceForUserAndLeague(ctx context.Context, leagueID uuid.UUID, userID string) (*pb.GameInfoResponses, error) {
	var responses []*pb.GameInfoResponse

	params := models.ListActiveCorrespondenceGamesForUserAndLeagueParams{
		LeagueID: leagueID,
		UserUuid: userID,
	}

	games, err := s.queries.ListActiveCorrespondenceGamesForUserAndLeague(ctx, params)
	if err != nil {
		return nil, err
	}

	// Fetch league slug once
	league, err := s.queries.GetLeagueByUUID(ctx, leagueID)
	if err != nil {
		log.Warn().Err(err).Str("league_id", leagueID.String()).Msg("failed to fetch league slug")
		return nil, err
	}

	for _, g := range games {
		info := &pb.GameInfoResponse{
			Players:     g.Quickdata.PlayerInfo,
			GameId:      g.Uuid.String,
			GameRequest: g.GameRequest.GameRequest,
			Type:        pb.GameType_NATIVE,
			LastUpdate:  timestamppb.New(g.UpdatedAt.Time),
			LeagueId:    leagueID.String(),
			LeagueSlug:  league.Slug,
		}

		if g.PlayerOnTurn.Valid {
			playerOnTurn := uint32(g.PlayerOnTurn.Int32)
			info.PlayerOnTurn = &playerOnTurn
		}

		responses = append(responses, info)
	}
	log.Debug().Int("num-correspondence", len(responses)).Str("user", userID).Str("league", leagueID.String()).Msg("list-active-correspondence-for-user-and-league")
	return &pb.GameInfoResponses{GameInfo: responses}, nil
}

// ListActiveCorrespondenceRaw returns raw DB rows with timer data.
// This is used internally by ListActiveCorrespondence and the adjudication process.
func (s *DBStore) ListActiveCorrespondenceRaw(ctx context.Context) ([]models.ListActiveCorrespondenceGamesRow, error) {
	games, err := s.queries.ListActiveCorrespondenceGames(ctx)
	if err != nil {
		return nil, err
	}
	return games, nil
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

// Helper function to safely get time control from GameRequest
func safeTimeControlFromGameReq(gamereq *entity.GameRequest, gameID string) entity.TimeControl {
	if gamereq.GameRequest == nil || gamereq.GameRequest.Rules == nil {
		log.Warn().Str("game_id", gameID).Msg("game has incomplete GameRequest data, using default time control")
		return "Unknown"
	}

	timefmt, _, err := entity.VariantFromGameReq(gamereq.GameRequest)
	if err != nil {
		log.Warn().Err(err).Str("game_id", gameID).Msg("error getting variant from GameRequest, using default")
		return "Unknown"
	}

	return timefmt
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

func safeDerefGameRequest(g *entity.GameRequest) entity.GameRequest {
	if g == nil {
		return entity.GameRequest{}
	}
	return *g
}

// InsertGamePlayers creates entries in the game_players table for a completed game
func (s *DBStore) InsertGamePlayers(ctx context.Context, g *entity.Game) error {
	// Only insert for completed games (not ongoing or cancelled)
	// CANCELLED games never started, so we don't track them
	// ABORTED games are tracked since they represent actual gameplay
	if g.GameEndReason == pb.GameEndReason_NONE || g.GameEndReason == pb.GameEndReason_CANCELLED {
		return nil
	}

	// Extract original request ID for rematch tracking
	var originalRequestID pgtype.Text
	if g.Quickdata != nil && g.Quickdata.OriginalRequestId != "" {
		originalRequestID = pgtype.Text{String: g.Quickdata.OriginalRequestId, Valid: true}
	}

	// Get scores directly from the game object
	player0Score := int32(g.PointsFor(0))
	player1Score := int32(g.PointsFor(1))

	// Determine win status for each player
	var player0Won, player1Won pgtype.Bool
	// For ABORTED games, don't set any winner regardless of WinnerIdx value
	if g.GameEndReason != pb.GameEndReason_ABORTED {
		switch g.WinnerIdx {
		case 0:
			player0Won = pgtype.Bool{Bool: true, Valid: true}
			player1Won = pgtype.Bool{Bool: false, Valid: true}
		case 1:
			player0Won = pgtype.Bool{Bool: false, Valid: true}
			player1Won = pgtype.Bool{Bool: true, Valid: true}
		}
	}
	// If WinnerIdx is -1 (tie) or game was ABORTED, both remain null

	return s.queries.InsertGamePlayers(ctx, models.InsertGamePlayersParams{
		GameUuid:          g.GameID(),
		Player0ID:         int32(g.PlayerDBIDs[0]),
		Player1ID:         int32(g.PlayerDBIDs[1]),
		Player0Score:      player0Score,
		Player1Score:      player1Score,
		Player0Won:        player0Won,
		Player1Won:        player1Won,
		GameEndReason:     int16(g.GameEndReason),
		CreatedAt:         pgtype.Timestamptz{Time: g.CreatedAt, Valid: true},
		GameType:          int16(g.Type),
		OriginalRequestID: originalRequestID,
	})
}
