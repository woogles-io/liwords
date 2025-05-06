package game

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/opentelemetry/tracing"

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
)

// DBAndS3Store is a postgres and S3-backed store for games.
type DBAndS3Store struct {
	cfg     *config.Config
	db      *gorm.DB
	queries *models.Queries

	userStore       pkguser.Store
	s3Client        *s3.Client
	pastGamesBucket string

	// This reference is here so we can copy it to every game we pull
	// from the database.
	// All game events go down the same channel.
	gameEventChan chan<- *entity.EventWrapper
}

type game struct {
	gorm.Model
	UUID string `gorm:"type:varchar(24);index"`

	Type      pb.GameType
	Player0ID uint `gorm:"foreignKey;index"`
	// Player0   user.User

	Player1ID uint `gorm:"foreignKey;index"`
	// Player1   user.User

	ReadyFlag uint // When both players are ready, this game starts.

	Timers datatypes.JSON // A JSON blob containing the game timers.

	Started       bool
	GameEndReason int `gorm:"index"`
	WinnerIdx     int
	LoserIdx      int
	HistoryInS3   bool // Whether the history is in S3 or not.

	Quickdata datatypes.JSON // A JSON blob containing the game quickdata.

	// Protobuf representations of the game request and history.
	Request []byte
	History []byte
	// Meta Events (abort, adjourn, adjudicate, etc requests)
	MetaEvents datatypes.JSON

	Stats datatypes.JSON

	// This is purposefully not a foreign key. It can be empty/NULL for
	// most games.
	TournamentID   string `gorm:"index"`
	TournamentData datatypes.JSON
}

// NewDBStore creates a new DB store for games.
func NewDBAndS3Store(config *config.Config, userStore pkguser.Store, dbPool *pgxpool.Pool,
	s3Client *s3.Client, pastGamesBucket string) (*DBAndS3Store, error) {

	db, err := gorm.Open(postgres.Open(config.DBConnDSN), &gorm.Config{Logger: common.GormLogger})
	if err != nil {
		return nil, err
	}
	if err := db.Use(tracing.NewPlugin()); err != nil {
		return nil, err
	}
	// Note: We need to manually add the following index on production:
	// create index rematch_req_idx ON games using hash ((quickdata->>'o'));
	// I don't know how to do this with GORM. This makes the GetRematchStreak function
	// much faster.

	return &DBAndS3Store{db: db, cfg: config, userStore: userStore,
		queries: models.New(dbPool), s3Client: s3Client, pastGamesBucket: pastGamesBucket}, nil
}

// SetGameEventChan sets the game event channel to the passed in channel.
func (s *DBAndS3Store) SetGameEventChan(c chan<- *entity.EventWrapper) {
	s.gameEventChan = c
}

// GameEventChan returns the game event channel for all games.
func (s *DBAndS3Store) GameEventChan() chan<- *entity.EventWrapper {
	return s.gameEventChan
}

// Get creates an instantiated entity.Game from the database.
// This function should almost never be called during a live game.
// The db store should be wrapped with a cache.
// Only API nodes that have this game in its cache should respond to requests.
// XXX: The above comment is obsolete and we will likely redo the way we do caches in the future.
func (s *DBAndS3Store) Get(ctx context.Context, id string) (*entity.Game, error) {

	g, err := s.queries.GetGame(ctx, common.ToPGTypeText(id))
	if err != nil {
		log.Err(err).Msg("error-get-game")
		return nil, err
	}
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
		GameReq:        &g.Request, // move to g.GameRequest after JSON migration
		HistoryInS3:    g.HistoryInS3,
	}
	entGame.SetTimerModule(&entity.GameTimer{})

	hist := &macondopb.GameHistory{}

	if g.HistoryInS3 {
		result, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: &s.pastGamesBucket,
			Key:    &id,
		})
		if err != nil {
			log.Err(err).Msg("error-getting-history-from-s3")
			return nil, err
		}
		defer result.Body.Close()
		bts, err := io.ReadAll(result.Body)
		if err != nil {
			log.Err(err).Msg("error-reading-history-from-s3")
			return nil, err
		}
		doc := &ipc.GameDocument{}
		err = protojson.Unmarshal(bts, doc)
		if err != nil {
			log.Err(err).Msg("error-unmarshalling-gdoc-from-s3")
			return nil, err
		}

		hist, err = utilities.ToGameHistory(doc, s.cfg)
		if err != nil {
			log.Err(err).Msg("error-converting-gdoc-to-history")
			return nil, err
		}
		log.Debug().Interface("hist", hist).Msg("hist-convert")
	} else {
		// Then unmarshal the history and start a game from it.

		err = proto.Unmarshal(g.History, hist)
		if err != nil {
			return nil, err
		}
		log.Debug().Interface("hist", hist).Msg("hist-unmarshal")
	}
	lexicon := hist.Lexicon
	if lexicon == "" {
		// This can happen for some early games where we didn't migrate this.
		lexicon = g.Request.Lexicon
	}

	rules, err := macondogame.NewBasicGameRules(
		s.cfg.MacondoConfig(), lexicon, g.Request.Rules.BoardLayoutName,
		g.Request.Rules.LetterDistributionName, macondogame.CrossScoreOnly,
		macondogame.Variant(g.Request.Rules.VariantName))
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
func (s *DBAndS3Store) GetMetadata(ctx context.Context, id string) (*pb.GameInfoResponse, error) {
	g := &game{}

	result := s.db.Where("uuid = ?", id).First(g)
	if result.Error != nil {
		return nil, result.Error
	}

	return convertGameToInfoResponse(g)

}

func (s *DBAndS3Store) GetRematchStreak(ctx context.Context, originalRequestId string) (*gs.StreakInfoResponse, error) {
	games := []*game{}
	ctxDB := s.db.WithContext(ctx)
	result := ctxDB.Raw(`SELECT uuid, winner_idx, quickdata FROM games where quickdata->>'o' = ?
		AND game_end_reason not in (?, ?, ?) ORDER BY created_at desc`, originalRequestId,
		pb.GameEndReason_NONE, pb.GameEndReason_ABORTED, pb.GameEndReason_CANCELLED).Scan(&games)
	if result.Error != nil {
		return nil, result.Error
	}

	resp := &gs.StreakInfoResponse{
		Streak: make([]*gs.StreakInfoResponse_SingleGameInfo, len(games)),
	}

	if result.RowsAffected <= 0 {
		return resp, nil
	}

	for idx, g := range games {
		var mdata entity.Quickdata
		err := json.Unmarshal(g.Quickdata, &mdata)
		if err != nil {
			log.Debug().Err(err).Msg("convert-game-quickdata")
			// If it's empty or unconvertible don't quit. We need this
			// for backwards compatibility.
		}
		if idx == 0 {
			playersInfo := make([]*gs.StreakInfoResponse_PlayerInfo, len(mdata.PlayerInfo))
			for i, p := range mdata.PlayerInfo {
				playersInfo[i] = &gs.StreakInfoResponse_PlayerInfo{
					Nickname: p.Nickname,
					Uuid:     p.UserId,
				}
			}
			sort.Slice(playersInfo, func(i, j int) bool { return playersInfo[i].Nickname > playersInfo[j].Nickname })
			resp.PlayersInfo = playersInfo
		}
		winner := g.WinnerIdx
		if len(resp.PlayersInfo) > 0 && len(mdata.PlayerInfo) > 0 &&
			resp.PlayersInfo[0].Nickname != mdata.PlayerInfo[0].Nickname {

			if winner != -1 {
				winner = 1 - winner
			}
		}
		resp.Streak[idx] = &gs.StreakInfoResponse_SingleGameInfo{
			GameId: g.UUID,
			Winner: int32(winner),
		}
	}

	return resp, nil
}

func (s *DBAndS3Store) GetRecentGames(ctx context.Context, username string, numGames int, offset int) (*pb.GameInfoResponses, error) {
	if numGames > MaxRecentGames {
		return nil, errors.New("too many games")
	}
	ctxDB := s.db.WithContext(ctx)
	var games []*game

	if err := ctxDB.Transaction(func(tx *gorm.DB) error {

		var userId int64
		if results := tx.Raw(
			"select id from users where lower(username) = lower(?)",
			username).
			Scan(&userId); results.Error != nil {

			return results.Error
		} else if results.RowsAffected != 1 {
			// Note: With gorm, Scan does not return an error when the row is not found.
			// No users means no games.
			// There should already be a unique key on (lower(username)), so there cannot be multiple matches.
			return nil
		}

		// Note: The query now sorts by id. It used to sort by created_at, which was not indexed.
		// Note: A partial index may be helpful for the few players with the most number of completed games.
		// Note: This query only selects ids, to reduce the amount of work required by the db to paginate.
		var gameIds []int64
		if results := tx.Raw(
			`select id from games where (player0_id = ? or player1_id = ?)
			and game_end_reason not in (?, ?, ?) order by id desc limit ? offset ?`,
			userId, userId,
			pb.GameEndReason_NONE, pb.GameEndReason_ABORTED, pb.GameEndReason_CANCELLED, numGames, offset).
			Find(&gameIds); results.Error != nil {

			return results.Error
		} else if results.RowsAffected == 0 {
			// No game ids means no games.
			return nil
		}

		// convertGamesToInfoResponses does not need History.
		// This still reads each history, but then garbage-collects immediately.
		// The "correct" way is to manually list all surviving column names.
		if results := tx.Raw(
			"select *, null history from games where id in ? order by id desc",
			gameIds).
			Find(&games); results.Error != nil {

			return results.Error
		}

		return nil
	}, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	}); err != nil {
		// Note: REPEATABLE READ is correct for Postgres (other databases may require SERIALIZABLE to avoid phantom reads).
		// The default READ COMMITTED may return invalid rows if an update invalidates the row after the id has been chosen.
		log.Err(err).Str("username", username).Int("numGames", numGames).Int("offset", offset).Msg("get-recent-games")
		return nil, err
	}

	return convertGamesToInfoResponses(games)
}

func (s *DBAndS3Store) GetRecentTourneyGames(ctx context.Context, tourneyID string, numGames int, offset int) (*pb.GameInfoResponses, error) {
	if numGames > MaxRecentGames {
		return nil, errors.New("too many games")
	}
	ctxDB := s.db.WithContext(ctx)
	var games []*game

	if err := ctxDB.Transaction(func(tx *gorm.DB) error {

		// Note: This query only selects ids, to reduce the amount of work required by the db to paginate.
		var gameIds []int64
		if results := tx.Raw(
			`select id from games where tournament_id = ?
			and game_end_reason not in (?, ?, ?) order by updated_at desc limit ? offset ?`,
			tourneyID,
			pb.GameEndReason_NONE, pb.GameEndReason_ABORTED, pb.GameEndReason_CANCELLED, numGames, offset).
			Find(&gameIds); results.Error != nil {

			return results.Error
		} else if results.RowsAffected == 0 {
			// No game ids means no games.
			return nil
		}

		// convertGamesToInfoResponses does not need History.
		// This still reads each history, but then garbage-collects immediately.
		// The "correct" way is to manually list all surviving column names.
		if results := tx.Raw(
			"select *, null history from games where id in ? order by updated_at desc",
			gameIds).
			Find(&games); results.Error != nil {

			return results.Error
		}

		return nil
	}, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	}); err != nil {
		// Note: REPEATABLE READ is correct for Postgres (other databases may require SERIALIZABLE to avoid phantom reads).
		// The default READ COMMITTED may return invalid rows if an update invalidates the row after the id has been chosen.
		log.Err(err).Str("tourneyID", tourneyID).Int("numGames", numGames).Int("offset", offset).Msg("get-recent-tourney-games")
		return nil, err
	}

	return convertGamesToInfoResponses(games)
}

func convertGamesToInfoResponses(games []*game) (*pb.GameInfoResponses, error) {
	responses := []*pb.GameInfoResponse{}
	for _, g := range games {
		info, err := convertGameToInfoResponse(g)
		if err != nil {
			return nil, err
		}
		responses = append(responses, info)
	}
	return &pb.GameInfoResponses{GameInfo: responses}, nil
}

func convertGameToInfoResponse(g *game) (*pb.GameInfoResponse, error) {
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

	info := &pb.GameInfoResponse{
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
		Type:                g.Type,
	}
	return info, nil
}

// Set takes in a game entity that _already exists_ in the DB, and writes it to
// the database.
func (s *DBAndS3Store) Set(ctx context.Context, g *entity.Game) error {
	// s.db.LogMode(true)
	dbg, err := s.toDBObj(g)
	if err != nil {
		return err
	}
	if g.GameEndReason != pb.GameEndReason_NONE &&
		g.GameEndReason != pb.GameEndReason_CANCELLED &&
		!g.HistoryInS3 && s.s3Client != nil {
		// This game is over, one way or another. Save gdoc in S3.
		log.Info().Str("id", g.GameID()).Msg("uploading finished game to S3")
		gdoc, err := utilities.ToGameDocument(g, s.cfg)
		if err != nil {
			// this is bad!
			log.Err(err).Msg("error-converting-gdoc")
		} else {
			gdocbts, err := protojson.Marshal(gdoc)
			if err != nil {
				return err
			}
			gid := g.GameID()
			_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
				Bucket: &s.pastGamesBucket,
				Key:    &gid,
				Body:   bytes.NewReader(gdocbts),
			})
			if err != nil {
				log.Err(err).Msg("error-putting-gdoc-to-s3")
			} else {
				// only delete history from DB if we successfully
				// put it in S3.
				dbg.HistoryInS3 = true
				// Annoyingly we can't set it to null with gorm.
				dbg.History = []byte{'.'}
				g.HistoryInS3 = true
			}
		}

	}

	// result := s.db.Model(&game{}).Set("gorm:query_option", "FOR UPDATE").
	// 	Where("uuid = ?", g.GameID()).Update(dbg)
	// s.db.LogMode(false)

	// XXX: not sure this select for update is working. Might consider
	// moving to select for share??
	ctxDB := s.db.WithContext(ctx)
	result := ctxDB.Model(&game{}).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("uuid = ?", g.GameID()).Updates(dbg)

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (s *DBAndS3Store) Exists(ctx context.Context, id string) (bool, error) {

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
func (s *DBAndS3Store) Create(ctx context.Context, g *entity.Game) error {
	dbg, err := s.toDBObj(g)
	if err != nil {
		return err
	}
	log.Debug().Interface("dbg", dbg).Msg("dbg")
	ctxDB := s.db.WithContext(ctx)
	result := ctxDB.Create(dbg)
	return result.Error
}

func (s *DBAndS3Store) CreateRaw(ctx context.Context, g *entity.Game, gt pb.GameType) error {
	if gt == pb.GameType_NATIVE {
		return fmt.Errorf("this game already exists: %s", g.Uid())
	}
	ctxDB := s.db.WithContext(ctx)

	req, err := proto.Marshal(g.GameReq)
	if err != nil {
		return err
	}
	hist, err := proto.Marshal(g.History())
	if err != nil {
		return err
	}
	result := ctxDB.Exec(
		`insert into games(uuid, request, history, quickdata, timers,
			game_end_reason, type)
		values(?, ?, ?, ?, ?, ?, ?)`,
		g.Uid(), req, hist, g.Quickdata, g.Timers, g.GameEndReason, gt)
	return result.Error
}

func (s *DBAndS3Store) ListActive(ctx context.Context, tourneyID string) (*pb.GameInfoResponses, error) {
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

func (s *DBAndS3Store) Count(ctx context.Context) (int64, error) {
	var count int64
	result := s.db.Model(&game{}).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// List all game IDs, ordered by date played. Should not be used by anything
// other than debug or migration code when the db is still small.
func (s *DBAndS3Store) ListAllIDs(ctx context.Context) ([]string, error) {
	var gids []struct{ Uuid string }
	result := s.db.Table("games").Select("uuid").Order("created_at").Scan(&gids)

	ids := make([]string, len(gids))
	for idx, gid := range gids {
		ids[idx] = gid.Uuid
	}

	return ids, result.Error
}

func (s *DBAndS3Store) SetReady(ctx context.Context, gid string, pidx int) (int, error) {
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

func (s *DBAndS3Store) toDBObj(g *entity.Game) (*game, error) {
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
	mdata, err := json.Marshal(g.MetaEvents)
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
		MetaEvents:     mdata,
		Type:           g.Type,
	}
	if g.TournamentData != nil {
		dbg.TournamentID = g.TournamentData.Id
	}

	return dbg, nil
}

func (s *DBAndS3Store) Disconnect() {
	dbSQL, err := s.db.DB()
	if err == nil {
		log.Info().Msg("disconnecting SQL db")
		dbSQL.Close()
		return
	}
	log.Err(err).Msg("unable to disconnect")
}

func (s *DBAndS3Store) CachedCount(ctx context.Context) int {
	return 0
}

func (s *DBAndS3Store) GetHistory(ctx context.Context, id string) (*macondopb.GameHistory, error) {
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
	log.Debug().Interface("hist", hist).Msg("got-history")
	return hist, nil
}
