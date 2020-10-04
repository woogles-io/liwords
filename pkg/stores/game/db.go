package game

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/rs/zerolog/log"

	"google.golang.org/protobuf/proto"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"

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

	Player0ID uint
	Player0   user.User

	Player1ID uint
	Player1   user.User

	Timers postgres.Jsonb // A JSON blob containing the game timers.

	Started       bool
	GameEndReason int `gorm:"index"`
	WinnerIdx     int
	LoserIdx      int

	Quickdata postgres.Jsonb // A JSON blob containing the game quickdata.

	// Protobuf representations of the game request and history.
	Request []byte
	History []byte

	Stats postgres.Jsonb
}

// NewDBStore creates a new DB store for games.
func NewDBStore(config *config.Config, userStore pkguser.Store) (*DBStore, error) {
	db, err := gorm.Open("postgres", config.DBConnString)
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&game{})
	db.Model(&game{}).AddForeignKey("player0_id", "users(id)", "RESTRICT", "RESTRICT")
	db.Model(&game{}).AddForeignKey("player1_id", "users(id)", "RESTRICT", "RESTRICT")
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

	if result := s.db.Where("uuid = ?", id).First(g); result.Error != nil {
		return nil, result.Error
	}

	var tdata entity.Timers
	err := json.Unmarshal(g.Timers.RawMessage, &tdata)
	if err != nil {
		return nil, err
	}

	var sdata *entity.Stats
	err = json.Unmarshal(g.Stats.RawMessage, sdata)
	if err != nil {
		// it could be that the stats are empty, so don't worry.
	}

	return FromState(tdata, g.Started, g.GameEndReason, g.Player0ID, g.Player1ID,
		g.WinnerIdx, g.LoserIdx, g.Request, g.History, sdata, s.gameEventChan, s.cfg)
}

// Similar to get but does not unmarshal the stats and timers and does
// not play the game
func (s *DBStore) GetQuickdata(ctx context.Context, id string) (*entity.Game, error) {
	g := &game{}

	if result := s.db.Where("uuid = ?", id).First(g); result.Error != nil {
		return nil, result.Error
	}

	return FromStateQuickdata(g.Started, g.GameEndReason, g.Player0ID, g.Player1ID,
		g.WinnerIdx, g.LoserIdx, g.Request, g.History, s.gameEventChan, s.cfg)
}

func (s *DBStore) GetRematchStreak(ctx context.Context, originalRequestId string) (*gs.GameInfoResponses, error) {
	games := []*game{}
	if results := s.db.Where("quickdata->>'s' = ?", originalRequestId).Order("updated_at desc").Find(games); results.Error != nil {
		return nil, results.Error
	}
	return convertGamesToInfoResponses(games)
}

func (s *DBStore) GetRecentGames(ctx context.Context, playerId string, numGames int, offset int) (*gs.GameInfoResponses, error) {
	games := []*game{}
	if results := s.db.Limit(numGames).Offset(offset).Where("uuid = ?", playerId).Order("updated_at desc").Find(games); results.Error != nil {
		return nil, results.Error
	}
	return convertGamesToInfoResponses(games)
}

func convertGamesToInfoResponses(games []*game) (*gs.GameInfoResponses, error) {
	responses := []*gs.GameInfoResponse{}
	for _, g := range games {
		var mdata entity.Quickdata
		err := json.Unmarshal(g.Quickdata.RawMessage, &mdata)
		if err != nil {
			return nil, err
		}

		playerInfo := []*gs.PlayerInfo{
			&gs.PlayerInfo{UserId: g.Player0.UUID, Nickname: g.Player0.Username},
			&gs.PlayerInfo{UserId: g.Player1.UUID, Nickname: g.Player1.Username},
		}

		info := &gs.GameInfoResponse{
			Players:       playerInfo,
			GameEndReason: pb.GameEndReason(g.GameEndReason),
			Scores:        mdata.FinalScores,
			Winner:        int32(g.WinnerIdx),
			UpdatedAt:     g.UpdatedAt.Unix()}
		responses = append(responses, info)
	}
	return &gs.GameInfoResponses{GameInfo: responses}, nil
}

// FromState returns an entity.Game from a DB State.
func FromState(timers entity.Timers, Started bool,
	GameEndReason int, p0id, p1id uint, WinnerIdx, LoserIdx int, reqBytes, histBytes []byte,
	stats *entity.Stats,
	gameEventChan chan<- *entity.EventWrapper, cfg *config.Config) (*entity.Game, error) {

	g := &entity.Game{
		Started:       Started,
		Timers:        timers,
		GameEndReason: pb.GameEndReason(GameEndReason),
		WinnerIdx:     WinnerIdx,
		LoserIdx:      LoserIdx,
		ChangeHook:    gameEventChan,
		PlayerDBIDs:   [2]uint{p0id, p1id},
		Stats:         stats,
	}
	g.SetTimerModule(&entity.GameTimer{})

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

	dist, err := alphabet.LoadLetterDistribution(&cfg.MacondoConfig, req.Rules.LetterDistributionName)
	if err != nil {
		return nil, err
	}

	lexicon := hist.Lexicon
	if lexicon == "" {
		// This can happen for some early games where we didn't migrate this.
		lexicon = req.Lexicon
	}

	gd, err := gaddag.LoadFromCache(&cfg.MacondoConfig, lexicon)
	if err != nil {
		return nil, err
	}

	rules := macondogame.NewGameRules(
		&cfg.MacondoConfig, dist, board.MakeBoard(bd),
		&gaddag.Lexicon{GenericDawg: gd},
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

// FromStateQuickdata returns an entity.Game from a DB State.
// It does not load the letter distribution, gaddag, or game rules.
// It does not play the game.
func FromStateQuickdata(Started bool,
	GameEndReason int, p0id, p1id uint, WinnerIdx, LoserIdx int, reqBytes, histBytes []byte,
	gameEventChan chan<- *entity.EventWrapper, cfg *config.Config) (*entity.Game, error) {

	g := &entity.Game{
		Started:       Started,
		GameEndReason: pb.GameEndReason(GameEndReason),
		WinnerIdx:     WinnerIdx,
		LoserIdx:      LoserIdx,
		ChangeHook:    gameEventChan,
		PlayerDBIDs:   [2]uint{p0id, p1id},
	}
	g.SetTimerModule(&entity.GameTimer{})

	// Now copy the request
	req := &pb.GameRequest{}
	err := proto.Unmarshal(reqBytes, req)
	if err != nil {
		return nil, err
	}
	g.GameReq = req
	log.Debug().Interface("req", req).Msg("req-unmarshal-quickdata")
	// Then unmarshal the history and start a game from it.
	hist := &macondopb.GameHistory{}
	err = proto.Unmarshal(histBytes, hist)
	if err != nil {
		return nil, err
	}
	log.Info().Interface("hist", hist).Msg("hist-unmarshal-quickdata")

	return g, nil
}

// Set takes in a game entity that _already exists_ in the DB, and writes it to
// the database.
func (s *DBStore) Set(ctx context.Context, g *entity.Game) error {
	// s.db.LogMode(true)
	dbg, err := s.toDBObj(ctx, g)
	if err != nil {
		return err
	}
	th := &macondopb.GameHistory{}
	err = proto.Unmarshal(dbg.History, th)
	if err != nil {
		return err
	}

	result := s.db.Model(&game{}).Set("gorm:query_option", "FOR UPDATE").
		Where("uuid = ?", g.GameID()).Update(dbg)
	// s.db.LogMode(false)

	return result.Error
}

// Create saves a brand new entity to the database
func (s *DBStore) Create(ctx context.Context, g *entity.Game) error {
	dbg, err := s.toDBObj(ctx, g)
	if err != nil {
		return err
	}
	result := s.db.Create(dbg)
	return result.Error
}

func (s *DBStore) ListActive(ctx context.Context) ([]*pb.GameMeta, error) {
	var games []activeGame

	// Create query manually
	result := s.db.Table("games").Select(
		"u0.username as p0_username, u1.username as p1_username, "+
			"p0.ratings as p0_ratings, p1.ratings as p1_ratings, "+
			"games.request as request, games.uuid ").
		Joins("JOIN users as u0  ON u0.id = games.player0_id").
		Joins("JOIN users as u1  ON u1.id = games.player1_id").
		Joins("JOIN profiles as p0 on p0.user_id = games.player0_id").
		Joins("JOIN profiles as p1 on p1.user_id = games.player1_id").
		Where("games.game_end_reason = ?", 0 /* ongoing games only*/).
		Order("games.id").
		Scan(&games)

	if result.Error != nil {
		return nil, result.Error
	}

	gamesMeta := make([]*pb.GameMeta, len(games))
	// This function looks kinda slow; should benchmark.
	var err error
	for idx, g := range games {
		gamesMeta[idx], err = g.ToGameMeta()
		if err != nil {
			return nil, err
		}
	}

	return gamesMeta, nil
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

func (s *DBStore) toDBObj(ctx context.Context, g *entity.Game) (*game, error) {
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

	dbg := &game{
		UUID:          g.GameID(),
		Player0ID:     g.PlayerDBIDs[0],
		Player1ID:     g.PlayerDBIDs[1],
		Timers:        postgres.Jsonb{RawMessage: timers},
		Stats:         postgres.Jsonb{RawMessage: stats},
		Quickdata:     postgres.Jsonb{RawMessage: quickdata},
		Started:       g.Started,
		GameEndReason: int(g.GameEndReason),
		WinnerIdx:     g.WinnerIdx,
		LoserIdx:      g.LoserIdx,
		Request:       req,
		History:       hist,
	}
	return dbg, nil
}

func (s *DBStore) Disconnect() {
	s.db.Close()
}

type activeGame struct {
	P0Username string
	P1Username string
	P0Ratings  postgres.Jsonb
	P1Ratings  postgres.Jsonb
	Request    []byte
	Uuid       string
}

func (a *activeGame) ToGameMeta() (*pb.GameMeta, error) {
	req := &pb.GameRequest{}
	err := proto.Unmarshal(a.Request, req)
	if err != nil {
		return nil, err
	}

	timefmt, variant, err := entity.VariantFromGameReq(req)
	ratingKey := entity.ToVariantKey(req.Lexicon, variant, timefmt)

	var p0data entity.Ratings
	err = json.Unmarshal(a.P0Ratings.RawMessage, &p0data)
	if err != nil {
		log.Err(err).Msg("unmarshal-p0-rating")
	}
	var p1data entity.Ratings
	err = json.Unmarshal(a.P1Ratings.RawMessage, &p1data)
	if err != nil {
		log.Err(err).Msg("unmarshal-p0-rating")
	}
	// Don't quit if we can't unmarshal ratings.

	p0Rating := entity.RelevantRating(p0data, ratingKey)
	p1Rating := entity.RelevantRating(p1data, ratingKey)

	players := []*pb.GameMeta_UserMeta{
		{RelevantRating: p0Rating, DisplayName: a.P0Username},
		{RelevantRating: p1Rating, DisplayName: a.P1Username}}

	return &pb.GameMeta{Users: players, GameRequest: req, Id: a.Uuid}, nil
}
