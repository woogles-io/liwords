package game

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/rs/zerolog/log"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stores/user"
	pkguser "github.com/domino14/liwords/pkg/user"
	gs "github.com/domino14/liwords/rpc/api/proto/game_service"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/domino14/macondo/alphabet"
	"github.com/domino14/macondo/board"
	"github.com/domino14/macondo/cross_set"
	"github.com/domino14/macondo/gaddag"
	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

var (
	ErrGameAlreadyOver = errors.New("game is already over")
)

const (
	MaxRecentGames = 20
)

// DBStore is a postgres-backed store for games.
type DBStore struct {
	cfg *config.Config
	db  *gorm.DB

	userStore pkguser.Store

	// This reference is here so we can copy it to every game we pull
	// from the database.
	// All game events go down the same channel.
	gameEventChan chan<- *entity.EventWrapper
}

type game struct {
	gorm.Model
	UUID string `gorm:"type:varchar(24);index"`

	Player0ID uint `gorm:"foreignKey"`
	Player0   user.User

	Player1ID uint `gorm:"foreignKey"`
	Player1   user.User

	ReadyFlag uint // When both players are ready, this game starts.

	Timers datatypes.JSON // A JSON blob containing the game timers.

	Started       bool
	GameEndReason int `gorm:"index"`
	WinnerIdx     int
	LoserIdx      int

	Quickdata datatypes.JSON // A JSON blob containing the game quickdata.

	// Protobuf representations of the game request and history.
	Request []byte
	History []byte

	Stats datatypes.JSON

	// This is purposefully not a foreign key. It can be empty/NULL for
	// most games.
	TournamentID   string `gorm:"index"`
	TournamentData datatypes.JSON
}

// NewDBStore creates a new DB store for games.
func NewDBStore(config *config.Config, userStore pkguser.Store) (*DBStore, error) {

	db, err := gorm.Open(postgres.Open(config.DBConnString), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&game{})
	// Note: We need to manually add the following index on production:
	// create index rematch_req_idx ON games using hash ((quickdata->>'o'));
	// I don't know how to do this with GORM. This makes the GetRematchStreak function
	// much faster.

	return &DBStore{db: db, cfg: config, userStore: userStore}, nil
}

// SetGameEventChan sets the game event channel to the passed in channel.
func (s *DBStore) SetGameEventChan(c chan<- *entity.EventWrapper) {
	s.gameEventChan = c
}

// Get creates an instantiated entity.Game from the database.
// This function should almost never be called during a live game.
// The db store should be wrapped with a cache.
// Only API nodes that have this game in its cache should respond to requests.
func (s *DBStore) Get(ctx context.Context, id string) (*entity.Game, error) {
	g := &game{}
	ctxDB := s.db.WithContext(ctx)
	if result := ctxDB.Where("uuid = ?", id).First(g); result.Error != nil {
		return nil, result.Error
	}

	var tdata entity.Timers
	err := json.Unmarshal(g.Timers, &tdata)
	if err != nil {
		return nil, err
	}

	var sdata entity.Stats
	err = json.Unmarshal(g.Stats, &sdata)
	if err != nil {
		// it could be that the stats are empty, so don't worry.
		// return nil, err
	}

	var qdata entity.Quickdata
	err = json.Unmarshal(g.Quickdata, &qdata)
	if err != nil {
		return nil, err
	}

	entGame, err := fromState(tdata, &qdata, g.Started, g.GameEndReason, g.Player0ID, g.Player1ID,
		g.WinnerIdx, g.LoserIdx, g.Request, g.History, &sdata, s.gameEventChan, s.cfg, g.CreatedAt)
	if err != nil {
		return nil, err
	}

	var trdata entity.TournamentData
	err = json.Unmarshal(g.TournamentData, &trdata)
	if err == nil {
		// however, it's ok for a game to not have tournament data
		entGame.TournamentData = &trdata
		entGame.TournamentData.Id = g.TournamentID
	}
	return entGame, nil
}

// GetMetadata gets metadata about the game, but does not actually play the game.
func (s *DBStore) GetMetadata(ctx context.Context, id string) (*gs.GameInfoResponse, error) {
	g := &game{}

	result := s.db.Where("uuid = ?", id).First(g)
	if result.Error != nil {
		return nil, result.Error
	}

	return convertGameToInfoResponse(g)

}

func (s *DBStore) GetRematchStreak(ctx context.Context, originalRequestId string) (*gs.StreakInfoResponse, error) {
	games := []*game{}
	if results := s.db.
		Where("quickdata->>'o' = ? AND game_end_reason != 0", originalRequestId).
		Order("created_at desc").
		Find(&games); results.Error != nil {
		return nil, results.Error
	}

	resp := &gs.StreakInfoResponse{
		Streak: make([]*gs.StreakInfoResponse_SingleGameInfo, len(games)),
	}

	for idx, g := range games {
		var mdata entity.Quickdata
		err := json.Unmarshal(g.Quickdata, &mdata)
		if err != nil {
			log.Debug().Err(err).Msg("convert-game-quickdata")
			// If it's empty or unconvertible don't quit. We need this
			// for backwards compatibility.
		}
		players := make([]string, len(mdata.PlayerInfo))
		for i, p := range mdata.PlayerInfo {
			players[i] = p.Nickname
		}
		resp.Streak[idx] = &gs.StreakInfoResponse_SingleGameInfo{
			GameId:  g.UUID,
			Winner:  int32(g.WinnerIdx),
			Players: players,
		}
	}

	return resp, nil
}

func (s *DBStore) GetRecentGames(ctx context.Context, username string, numGames int, offset int) (*gs.GameInfoResponses, error) {
	if numGames > MaxRecentGames {
		return nil, errors.New("too many games")
	}
	ctxDB := s.db.WithContext(ctx)
	var games []*game
	if results := ctxDB.Limit(numGames).
		Offset(offset).
		Joins("JOIN users as u0  ON u0.id = games.player0_id").
		Joins("JOIN users as u1  ON u1.id = games.player1_id").
		Where("(lower(u0.username) = lower(?) OR lower(u1.username) = lower(?)) AND game_end_reason NOT IN (?, ?, ?)",
			username, username, pb.GameEndReason_NONE, pb.GameEndReason_ABORTED, pb.GameEndReason_CANCELLED).
		Order("created_at desc").
		Find(&games); results.Error != nil {
		return nil, results.Error
	}
	return convertGamesToInfoResponses(games)
}

func (s *DBStore) GetRecentTourneyGames(ctx context.Context, tourneyID string, numGames int, offset int) (*gs.GameInfoResponses, error) {
	if numGames > MaxRecentGames {
		return nil, errors.New("too many games")
	}
	ctxDB := s.db.WithContext(ctx)
	var games []*game
	if results := ctxDB.Limit(numGames).
		Offset(offset).
		// Basically, everything except for 0 (ongoing), 5 (aborted) or 7 (cancelled)
		Where("tournament_id = ? AND game_end_reason NOT IN (?, ?, ?)", tourneyID,
			pb.GameEndReason_NONE, pb.GameEndReason_ABORTED, pb.GameEndReason_CANCELLED).
		Order("updated_at desc").
		Find(&games); results.Error != nil {
		return nil, results.Error
	}
	return convertGamesToInfoResponses(games)
}

func convertGamesToInfoResponses(games []*game) (*gs.GameInfoResponses, error) {
	responses := []*gs.GameInfoResponse{}
	for _, g := range games {
		info, err := convertGameToInfoResponse(g)
		if err != nil {
			return nil, err
		}
		responses = append(responses, info)
	}
	return &gs.GameInfoResponses{GameInfo: responses}, nil
}

func convertGameToInfoResponse(g *game) (*gs.GameInfoResponse, error) {
	var mdata entity.Quickdata

	err := json.Unmarshal(g.Quickdata, &mdata)
	if err != nil {
		log.Debug().Err(err).Msg("convert-game-quickdata")
		// If it's empty or unconvertible don't quit. We need this
		// for backwards compatibility.
	}

	gamereq := &pb.GameRequest{}
	err = proto.Unmarshal(g.Request, gamereq)
	if err != nil {
		return nil, err
	}
	timefmt, _, err := entity.VariantFromGameReq(gamereq)
	if err != nil {
		return nil, err
	}

	var trdata entity.TournamentData
	tDiv := ""
	tRound := 0
	tGameIndex := 0
	tid := ""

	err = json.Unmarshal(g.TournamentData, &trdata)
	if err == nil {
		tDiv = trdata.Division
		tRound = trdata.Round
		tGameIndex = trdata.GameIndex
		tid = trdata.Id
	}

	info := &gs.GameInfoResponse{
		Players:             mdata.PlayerInfo,
		GameEndReason:       pb.GameEndReason(g.GameEndReason),
		Scores:              mdata.FinalScores,
		Winner:              int32(g.WinnerIdx),
		TimeControlName:     string(timefmt),
		CreatedAt:           timestamppb.New(g.CreatedAt),
		LastUpdate:          timestamppb.New(g.UpdatedAt),
		GameId:              g.UUID,
		TournamentId:        tid,
		GameRequest:         gamereq,
		TournamentDivision:  tDiv,
		TournamentRound:     int32(tRound),
		TournamentGameIndex: int32(tGameIndex),
	}
	return info, nil
}

// fromState returns an entity.Game from a DB State.
func fromState(timers entity.Timers, qdata *entity.Quickdata, Started bool,
	GameEndReason int, p0id, p1id uint, WinnerIdx, LoserIdx int, reqBytes, histBytes []byte,
	stats *entity.Stats,
	gameEventChan chan<- *entity.EventWrapper, cfg *config.Config, createdAt time.Time) (*entity.Game, error) {

	g := &entity.Game{
		Started:       Started,
		Timers:        timers,
		GameEndReason: pb.GameEndReason(GameEndReason),
		WinnerIdx:     WinnerIdx,
		LoserIdx:      LoserIdx,
		ChangeHook:    gameEventChan,
		PlayerDBIDs:   [2]uint{p0id, p1id},
		Stats:         stats,
		Quickdata:     qdata,
		CreatedAt:     createdAt,
	}
	if timers.Nower == "FakeNower" {
		// For test purposes
		g.SetTimerModule(&entity.FakeNower{})
	} else {
		// Pretty much anytime on production.
		g.SetTimerModule(&entity.GameTimer{})
	}

	// Now copy the request
	req := &pb.GameRequest{}
	err := proto.Unmarshal(reqBytes, req)
	if err != nil {
		return nil, err
	}
	g.GameReq = req
	log.Debug().Interface("req", req).Msg("req-unmarshal")
	// Then unmarshal the history and start a game from it.
	hist := &macondopb.GameHistory{}
	err = proto.Unmarshal(histBytes, hist)
	if err != nil {
		return nil, err
	}
	log.Info().Interface("hist", hist).Msg("hist-unmarshal")

	var bd []string
	switch req.Rules.BoardLayoutName {
	case entity.CrosswordGame:
		bd = board.CrosswordGameBoard
	default:
		return nil, errors.New("unsupported board layout")
	}

	dist, err := alphabet.Get(&cfg.MacondoConfig, req.Rules.LetterDistributionName)
	if err != nil {
		return nil, err
	}

	lexicon := hist.Lexicon
	if lexicon == "" {
		// This can happen for some early games where we didn't migrate this.
		lexicon = req.Lexicon
	}

	dawg, err := gaddag.GetDawg(&cfg.MacondoConfig, lexicon)
	if err != nil {
		return nil, err
	}

	rules := macondogame.NewGameRules(
		&cfg.MacondoConfig, dist, board.MakeBoard(bd),
		&gaddag.Lexicon{GenericDawg: dawg},
		cross_set.CrossScoreOnlyGenerator{Dist: dist})

	if err != nil {
		return nil, err
	}

	// There's a chance the game is over, so we want to get that state before
	// the following function modifies it.
	histPlayState := hist.GetPlayState()
	log.Debug().Interface("old-play-state", histPlayState).Msg("play-state-loading-hist")
	// This function modifies the history. (XXX it probably shouldn't)
	// It modifies the play state as it plays the game from the beginning.
	mcg, err := macondogame.NewFromHistory(hist, rules, len(hist.Events))
	if err != nil {
		return nil, err
	}
	// XXX: We should probably move this to `NewFromHistory`:
	mcg.SetBackupMode(macondogame.InteractiveGameplayMode)
	g.Game = *mcg
	log.Debug().Interface("history", g.History()).Msg("from-state")
	// Finally, restore the play state from the passed-in history. This
	// might immediately end the game (for example, the game could have timed
	// out, but the NewFromHistory function doesn't actually handle that).
	// We could consider changing NewFromHistory, but we want it to be as
	// flexible as possible for things like analysis mode.
	g.SetPlaying(histPlayState)
	g.History().PlayState = histPlayState
	return g, nil
}

// Set takes in a game entity that _already exists_ in the DB, and writes it to
// the database.
func (s *DBStore) Set(ctx context.Context, g *entity.Game) error {
	// s.db.LogMode(true)
	dbg, err := s.toDBObj(g)
	if err != nil {
		return err
	}
	th := &macondopb.GameHistory{}
	err = proto.Unmarshal(dbg.History, th)
	if err != nil {
		return err
	}

	// result := s.db.Model(&game{}).Set("gorm:query_option", "FOR UPDATE").
	// 	Where("uuid = ?", g.GameID()).Update(dbg)
	// s.db.LogMode(false)

	// XXX: not sure this select for update is working. Might consider
	// moving to select for share??
	ctxDB := s.db.WithContext(ctx)
	result := ctxDB.Model(&game{}).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("uuid = ?", g.GameID()).Updates(dbg)

	return result.Error
}

func (s *DBStore) Exists(ctx context.Context, id string) (bool, error) {

	var count int64
	result := s.db.Model(&game{}).Where("uuid = ?", id).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	if count > 1 {
		return true, errors.New("unexpected duplicate ids")
	}
	return count == 1, nil
}

// Create saves a brand new entity to the database
func (s *DBStore) Create(ctx context.Context, g *entity.Game) error {
	dbg, err := s.toDBObj(g)
	if err != nil {
		return err
	}
	log.Debug().Interface("dbg", dbg).Msg("dbg")
	ctxDB := s.db.WithContext(ctx)
	result := ctxDB.Create(dbg)
	return result.Error
}

func (s *DBStore) ListActive(ctx context.Context, tourneyID string) (*gs.GameInfoResponses, error) {
	var games []*game

	ctxDB := s.db.WithContext(ctx)
	query := ctxDB.Table("games").Select("quickdata, request, uuid, started, tournament_data").
		Where("games.game_end_reason = ?", 0 /* ongoing games only*/)

	if tourneyID != "" {
		query = query.Where("games.tournament_id = ?", tourneyID)
	}

	result := query.Order("games.id").Scan(&games)

	if result.Error != nil {
		return nil, result.Error
	}

	return convertGamesToInfoResponses(games)
}

func (s *DBStore) Count(ctx context.Context) (int64, error) {
	var count int64
	result := s.db.Model(&game{}).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// List all game IDs, ordered by date played. Should not be used by anything
// other than debug or migration code when the db is still small.
func (s *DBStore) ListAllIDs(ctx context.Context) ([]string, error) {
	var gids []struct{ Uuid string }
	result := s.db.Table("games").Select("uuid").Order("created_at").Scan(&gids)

	ids := make([]string, len(gids))
	for idx, gid := range gids {
		ids[idx] = gid.Uuid
	}

	return ids, result.Error
}

func (s *DBStore) SetReady(ctx context.Context, gid string, pidx int) (int, error) {
	var rf struct {
		ReadyFlag int
	}
	ctxDB := s.db.WithContext(ctx)

	// If the game is already ready and this gets called again, this function
	// returns 0 rows, which means rf.ReadyFlag == 0 and the game won't start again.
	result := ctxDB.Raw(`update games set ready_flag = ready_flag | (1 << ?) where uuid = ?
		and ready_flag & (1 << ?) = 0 returning ready_flag`, pidx, gid, pidx).Scan(&rf)

	return rf.ReadyFlag, result.Error
}

// SetGameEndReason updates the endgame reason in an atomic fashion. It
// will return an error if the endgame reason was already set.
func (s *DBStore) SetGameEndReason(ctx context.Context, g *entity.Game) error {
	ctxDB := s.db.WithContext(ctx)

	result := ctxDB.Raw(`update games set game_end_reason = ? where uuid = ?
		and game_end_reason = ?`, g.GameEndReason, pb.GameEndReason_NONE)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrGameAlreadyOver
	}

	return nil
}

func (s *DBStore) toDBObj(g *entity.Game) (*game, error) {
	timers, err := json.Marshal(g.Timers)
	if err != nil {
		return nil, err
	}
	stats, err := json.Marshal(g.Stats)
	if err != nil {
		return nil, err
	}
	quickdata, err := json.Marshal(g.Quickdata)
	if err != nil {
		return nil, err
	}
	req, err := proto.Marshal(g.GameReq)
	if err != nil {
		return nil, err
	}
	hist, err := proto.Marshal(g.History())
	if err != nil {
		return nil, err
	}

	tourneydata, err := json.Marshal(g.TournamentData)
	if err != nil {
		return nil, err
	}

	dbg := &game{
		UUID:           g.GameID(),
		Player0ID:      g.PlayerDBIDs[0],
		Player1ID:      g.PlayerDBIDs[1],
		Timers:         timers,
		Stats:          stats,
		Quickdata:      quickdata,
		Started:        g.Started,
		GameEndReason:  int(g.GameEndReason),
		WinnerIdx:      g.WinnerIdx,
		LoserIdx:       g.LoserIdx,
		Request:        req,
		History:        hist,
		TournamentData: tourneydata,
	}
	if g.TournamentData != nil {
		dbg.TournamentID = g.TournamentData.Id
	}

	return dbg, nil
}

func (s *DBStore) Disconnect() {
	dbSQL, err := s.db.DB()
	if err == nil {
		log.Info().Msg("disconnecting SQL db")
		dbSQL.Close()
		return
	}
	log.Err(err).Msg("unable to disconnect")
}

func (s *DBStore) CachedCount(ctx context.Context) int {
	return 0
}

func (s *DBStore) GetHistory(ctx context.Context, id string) (*macondopb.GameHistory, error) {
	g := &game{}

	ctxDB := s.db.WithContext(ctx)
	if result := ctxDB.Select("history").Where("uuid = ?", id).First(g); result.Error != nil {
		return nil, result.Error
	}

	hist := &macondopb.GameHistory{}
	err := proto.Unmarshal(g.History, hist)
	if err != nil {
		return nil, err
	}

	return hist, nil
}
