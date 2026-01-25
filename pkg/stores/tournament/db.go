package tournament

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/opentelemetry/tracing"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/game"
	tl "github.com/woogles-io/liwords/pkg/tournament"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	pb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

// DBStore is a postgres-backed store for games.
type DBStore struct {
	cfg                 *config.Config
	db                  *gorm.DB
	tournamentEventChan chan<- *entity.EventWrapper
	gameStore           *game.Cache
}

type tournament struct {
	gorm.Model
	UUID        string `gorm:"uniqueIndex"`
	Name        string
	Description string
	AliasOf     string
	Directors   datatypes.JSON
	IsStarted   *bool
	IsFinished  *bool
	Divisions   datatypes.JSON
	// Slug looks like /tournament/abcdef, /club/madison, /club/madison/2020-04-20
	Slug string `gorm:"uniqueIndex:,expression:lower(slug)"`
	// ExtraMeta contains some extra metadata for the tournament,
	// such as default board/tile style, disclaimer, default
	// club settings, and a possible password.
	ExtraMeta datatypes.JSON
	// Type is tournament, club, session, and maybe other things.
	Type string
	// Parent is a tournament parent ID.
	Parent             string     `gorm:"index"`
	ScheduledStartTime *time.Time `gorm:"default:null"`
	ScheduledEndTime   *time.Time `gorm:"default:null"`
	CreatedBy          *uint
}

type registrant struct {
	UserID       string `gorm:"uniqueIndex:idx_registrant;index:idx_user"`
	TournamentID string `gorm:"uniqueIndex:idx_registrant"`
	DivisionID   string `gorm:"uniqueIndex:idx_registrant"`
}

// NewDBStore creates a new DB store for tournament managers.
func NewDBStore(config *config.Config, gs *game.Cache) (*DBStore, error) {
	db, err := gorm.Open(postgres.Open(config.DBConnDSN), &gorm.Config{Logger: common.GormLogger})
	if err != nil {
		return nil, err
	}
	if err := db.Use(tracing.NewPlugin()); err != nil {
		return nil, err
	}
	return &DBStore{db: db, gameStore: gs, cfg: config}, nil
}

func (s *DBStore) dbObjToEntity(tm *tournament) (*entity.Tournament, error) {
	var divisions map[string]*entity.TournamentDivision
	err := json.Unmarshal(tm.Divisions, &divisions)
	if err != nil {
		return nil, err
	}

	for _, division := range divisions {
		if division.ManagerType == entity.ClassicTournamentType {
			var classicDivision tl.ClassicDivision
			err = json.Unmarshal(division.DivisionRawMessage, &classicDivision)
			if err != nil {
				return nil, err
			}
			division.DivisionManager = &classicDivision
			division.DivisionRawMessage = nil
		} else {
			return nil, fmt.Errorf("Unknown division manager type: %d", division.ManagerType)
		}
	}

	var directors ipc.TournamentPersons
	err = json.Unmarshal(tm.Directors, &directors)
	if err != nil {
		return nil, err
	}

	extraMeta := &entity.TournamentMeta{}
	err = json.Unmarshal(tm.ExtraMeta, extraMeta)
	if err != nil {
		// it's ok, don't error out; this tournament has no extra meta
	}

	tme := &entity.Tournament{UUID: tm.UUID,
		Name:               tm.Name,
		Description:        tm.Description,
		AliasOf:            tm.AliasOf,
		Directors:          &directors,
		IsStarted:          tm.IsStarted != nil && *tm.IsStarted,
		IsFinished:         tm.IsFinished != nil && *tm.IsFinished,
		Divisions:          divisions,
		ExtraMeta:          extraMeta,
		Type:               entity.CompetitionType(tm.Type),
		ParentID:           tm.Parent,
		Slug:               tm.Slug,
		ScheduledStartTime: tm.ScheduledStartTime,
		ScheduledEndTime:   tm.ScheduledEndTime,
	}
	log.Debug().Msg("return-full")

	return tme, nil
}

func (s *DBStore) Get(ctx context.Context, id string) (*entity.Tournament, error) {
	tm := &tournament{}
	ctxDB := s.db.WithContext(ctx)
	if result := ctxDB.Where("uuid = ?", id).First(tm); result.Error != nil {
		return nil, result.Error
	}

	return s.dbObjToEntity(tm)
}

func (s *DBStore) GetBySlug(ctx context.Context, slug string) (*entity.Tournament, error) {
	tm := &tournament{}
	ctxDB := s.db.WithContext(ctx)
	// Slug get should be case-insensitive
	if result := ctxDB.Where("lower(slug) = lower(?)", slug).First(tm); result.Error != nil {
		return nil, result.Error
	}
	return s.dbObjToEntity(tm)
}

func (s *DBStore) TournamentEventChan() chan<- *entity.EventWrapper {
	return s.tournamentEventChan
}

func (s *DBStore) Set(ctx context.Context, tm *entity.Tournament) error {
	dbt, err := s.toDBObj(tm)
	if err != nil {
		return err
	}
	log.Info().Str("tid", tm.UUID).Bool("finished", tm.IsFinished).Msg("db-set-tournament")

	ctxDB := s.db.WithContext(ctx)
	result := ctxDB.Model(&tournament{}).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("uuid = ?", tm.UUID).Updates(dbt)

	return result.Error
}

func (s *DBStore) Create(ctx context.Context, tm *entity.Tournament) error {
	dbt, err := s.toDBObj(tm)
	if err != nil {
		return err
	}
	log.Debug().Interface("dbt", dbt).Msg("dbt")
	ctxDB := s.db.WithContext(ctx)
	result := ctxDB.Create(dbt)
	return result.Error
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

func (s *DBStore) toDBObj(t *entity.Tournament) (*tournament, error) {

	directors, err := json.Marshal(t.Directors)
	if err != nil {
		return nil, err
	}

	for _, division := range t.Divisions {
		dmJSON, err := json.Marshal(division.DivisionManager)
		if err != nil {
			return nil, err
		}

		division.DivisionRawMessage = dmJSON
	}

	divisions, err := json.Marshal(t.Divisions)
	if err != nil {
		return nil, err
	}

	extraMeta, err := json.Marshal(t.ExtraMeta)
	if err != nil {
		return nil, err
	}
	createdBy := &(t.CreatedBy)
	if t.CreatedBy == 0 {
		createdBy = nil
	}

	dbt := &tournament{
		UUID:               t.UUID,
		Name:               t.Name,
		Description:        t.Description,
		AliasOf:            t.AliasOf,
		Directors:          directors,
		IsStarted:          &t.IsStarted,
		IsFinished:         &t.IsFinished,
		Divisions:          divisions,
		ExtraMeta:          extraMeta,
		Type:               string(t.Type),
		Parent:             t.ParentID,
		Slug:               t.Slug,
		ScheduledStartTime: t.ScheduledStartTime,
		ScheduledEndTime:   t.ScheduledEndTime,
		CreatedBy:          createdBy,
	}
	return dbt, nil
}

// SetTournamentEventChan sets the tournament event channel to the passed in channel.
func (s *DBStore) SetTournamentEventChan(c chan<- *entity.EventWrapper) {
	s.tournamentEventChan = c
}

func (s *DBStore) GetRecentGames(ctx context.Context, tourneyID string, numGames int, offset int) (*pb.RecentGamesResponse, error) {
	infos, err := s.gameStore.GetRecentTourneyGames(ctx, tourneyID, numGames, offset)
	if err != nil {
		return nil, err
	}

	evts := []*ipc.TournamentGameEndedEvent{}
	for _, info := range infos.GameInfo {

		var res1, res2 ipc.TournamentGameResult
		switch info.Winner {
		case -1:
			res1 = ipc.TournamentGameResult_DRAW
			res2 = ipc.TournamentGameResult_DRAW
		case 0:
			res1 = ipc.TournamentGameResult_WIN
			res2 = ipc.TournamentGameResult_LOSS
		case 1:
			res1 = ipc.TournamentGameResult_LOSS
			res2 = ipc.TournamentGameResult_WIN
		}
		if len(info.Scores) != 2 {
			log.Error().Str("tourneyID", tourneyID).Str("gameID", info.GameId).
				Msg("corrupted-recent-tourney-game")
			continue
		}
		players := []*ipc.TournamentGameEndedEvent_Player{
			{Username: info.Players[0].Nickname, Score: info.Scores[0], Result: res1},
			{Username: info.Players[1].Nickname, Score: info.Scores[1], Result: res2},
		}
		if info.Players[1].First {
			players[0], players[1] = players[1], players[0]
		}

		evt := &ipc.TournamentGameEndedEvent{
			Players:   players,
			GameId:    info.GameId,
			EndReason: info.GameEndReason,
			Time:      info.LastUpdate.Seconds,
			Round:     int32(info.TournamentRound),
			Division:  info.TournamentDivision,
			GameIndex: info.TournamentGameIndex,
		}
		evts = append(evts, evt)
	}
	return &pb.RecentGamesResponse{
		Games: evts,
	}, nil
}

func (s *DBStore) GetRecentClubSessions(ctx context.Context, id string, count int, offset int) (*pb.ClubSessionsResponse, error) {
	var sessions []*tournament
	ctxDB := s.db.WithContext(ctx)
	// Slug get should be case-insensitive
	if result := ctxDB.Limit(count).
		Offset(offset).
		Where("parent = ?", id).
		Order("created_at desc").Find(&sessions); result.Error != nil {
		return nil, result.Error
	}

	csrs := make([]*pb.ClubSessionResponse, len(sessions))
	for i, cs := range sessions {
		csrs[i] = &pb.ClubSessionResponse{
			TournamentId: cs.UUID,
			Slug:         cs.Slug,
		}
	}
	return &pb.ClubSessionsResponse{Sessions: csrs}, nil
}

func (s *DBStore) GetRecentAndUpcomingTournaments(ctx context.Context) ([]*entity.Tournament, error) {
	var tournaments []*tournament
	ctxDB := s.db.WithContext(ctx)
	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	oneWeekFromNow := time.Now().AddDate(0, 0, 7)

	result := ctxDB.Where(`
		(scheduled_start_time IS NOT NULL AND scheduled_start_time BETWEEN ? AND ?)
		OR
		(scheduled_end_time IS NOT NULL AND scheduled_end_time BETWEEN ? AND ?)
		`,
		oneWeekAgo, oneWeekFromNow, oneWeekAgo, oneWeekFromNow).
		Order("scheduled_start_time ASC").
		Find(&tournaments)

	if result.Error != nil {
		return nil, result.Error
	}

	tourneyList := make([]*entity.Tournament, len(tournaments))
	for i, t := range tournaments {
		t, err := s.dbObjToEntity(t)
		if err != nil {
			return nil, err
		}
		tourneyList[i] = t
	}

	return tourneyList, nil

}

func (s *DBStore) GetPastTournaments(ctx context.Context, limit int32) ([]*entity.Tournament, error) {
	var tournaments []*tournament
	ctxDB := s.db.WithContext(ctx)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	now := time.Now()

	query := ctxDB.Where(`
		scheduled_end_time IS NOT NULL
		AND scheduled_end_time BETWEEN ? AND ?
		`,
		thirtyDaysAgo, now).
		Order("scheduled_end_time DESC")

	if limit > 0 {
		query = query.Limit(int(limit))
	}

	result := query.Find(&tournaments)

	if result.Error != nil {
		return nil, result.Error
	}

	tourneyList := make([]*entity.Tournament, len(tournaments))
	for i, t := range tournaments {
		t, err := s.dbObjToEntity(t)
		if err != nil {
			return nil, err
		}
		tourneyList[i] = t
	}

	return tourneyList, nil
}

func (s *DBStore) ListAllIDs(ctx context.Context) ([]string, error) {
	var tids []struct{ UUID string }
	ctxDB := s.db.WithContext(ctx)

	result := ctxDB.Table("tournaments").Select("uuid").Order("created_at").Scan(&tids)
	ids := make([]string, len(tids))
	for idx, tid := range tids {
		ids[idx] = tid.UUID
	}
	return ids, result.Error
}

// FindTournamentByStreamKey uses UNIQUE index on stream_key for O(1) lookup
// Returns tournament UUID and full user ID in "uuid:username" format
// Returns empty strings if not found
func (s *DBStore) FindTournamentByStreamKey(ctx context.Context, streamKey string, streamType string) (tournamentID string, userID string, err error) {
	ctxDB := s.db.WithContext(ctx)
	var result struct {
		TournamentID string
		UserID       string
		Username     string
	}

	query := `
		SELECT tournament_id, user_id, username
		FROM monitoring_streams
		WHERE stream_key = ?
		LIMIT 1
	`

	if err := ctxDB.Raw(query, streamKey).Scan(&result).Error; err != nil {
		return "", "", err
	}

	if result.TournamentID == "" || result.UserID == "" {
		return "", "", nil
	}

	// Return full userID in "uuid:username" format
	return result.TournamentID, result.UserID + ":" + result.Username, nil
}

func (s *DBStore) AddRegistrants(ctx context.Context, tid string, userIDs []string, division string) error {

	ctxDB := s.db.WithContext(ctx)
	users := make([]*registrant, len(userIDs))
	idx := 0
	for _, uid := range userIDs {
		users[idx] = &registrant{
			UserID:       uid,
			TournamentID: tid,
			DivisionID:   division,
		}
		idx++
	}

	return ctxDB.Create(&users).Error
}

func (s *DBStore) RemoveRegistrants(ctx context.Context, tid string, userIDs []string, division string) error {
	ctxDB := s.db.WithContext(ctx)

	result := ctxDB.Delete(registrant{}, "user_id IN ? AND tournament_id = ? AND division_id = ?",
		userIDs, tid, division)
	return result.Error
}

func (s *DBStore) RemoveRegistrantsForTournament(ctx context.Context, tid string) error {
	ctxDB := s.db.WithContext(ctx)

	result := ctxDB.Delete(registrant{}, "tournament_id = ?", tid)
	return result.Error
}

// ActiveTournamentsFor returns a list of 2-tuples of tournament ID, division ID
// that this user is registered in - only for active tournaments (ones that have not finished).
func (s *DBStore) ActiveTournamentsFor(ctx context.Context, userID string) ([][2]string, error) {
	var registrants []*registrant

	ctxDB := s.db.WithContext(ctx)
	result := ctxDB.Raw(`
		select tournament_id, division_id from registrants
		inner join tournaments on tournament_id = tournaments.uuid
		where tournaments.is_finished is not TRUE
			and registrants.user_id = ?
		`, userID).Scan(&registrants)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected <= 0 {
		return nil, nil
	}
	log.Debug().Int64("num-active-tournaments", result.RowsAffected).Str("userID", userID).Msg("active-tournaments-for")
	ret := make([][2]string, result.RowsAffected)
	for idx, val := range registrants {
		ret[idx] = [2]string{val.TournamentID, val.DivisionID}
	}
	return ret, nil
}

// Monitoring streams methods - direct SQL operations

// InsertMonitoringStream inserts a new stream key for a user in a tournament
func (s *DBStore) InsertMonitoringStream(ctx context.Context, tid, uid, username, streamType, streamKey string) error {
	ctxDB := s.db.WithContext(ctx)
	query := `
		INSERT INTO monitoring_streams (tournament_id, user_id, username, stream_type, stream_key, status, created_at)
		VALUES (?, ?, ?, ?, ?, 0, NOW())
	`
	result := ctxDB.Exec(query, tid, uid, username, streamType, streamKey)
	return result.Error
}

// UpdateMonitoringStreamStatus updates the status and timestamp of a stream
// Can update by stream key OR by tournament_id + user_id + stream_type
func (s *DBStore) UpdateMonitoringStreamStatus(ctx context.Context, streamKey string, status int, timestamp int64) error {
	ctxDB := s.db.WithContext(ctx)
	query := `
		UPDATE monitoring_streams
		SET status = ?, status_timestamp = TO_TIMESTAMP(?)
		WHERE stream_key = ?
	`
	result := ctxDB.Exec(query, status, timestamp, streamKey)
	return result.Error
}

// UpdateMonitoringStreamStatusByUser updates the status and timestamp of a stream by user and type
func (s *DBStore) UpdateMonitoringStreamStatusByUser(ctx context.Context, tournamentID, userID, streamType string, status int, timestamp int64) error {
	ctxDB := s.db.WithContext(ctx)
	query := `
		UPDATE monitoring_streams
		SET status = ?, status_timestamp = TO_TIMESTAMP(?)
		WHERE tournament_id = ? AND user_id = ? AND stream_type = ?
	`
	result := ctxDB.Exec(query, status, timestamp, tournamentID, userID, streamType)
	return result.Error
}

// GetMonitoringStreams gets all monitoring streams for a tournament, grouped by user
// Returns a map of userID -> MonitoringData (combining camera and screenshot rows)
func (s *DBStore) GetMonitoringStreams(ctx context.Context, tournamentID string) (map[string]*ipc.MonitoringData, error) {
	ctxDB := s.db.WithContext(ctx)

	type streamRow struct {
		UserID          string
		Username        string
		StreamType      string
		StreamKey       string
		Status          int32
		StatusTimestamp *time.Time
	}

	var rows []streamRow
	query := `
		SELECT user_id, username, stream_type, stream_key, status, status_timestamp
		FROM monitoring_streams
		WHERE tournament_id = ?
		ORDER BY user_id, stream_type
	`

	if err := ctxDB.Raw(query, tournamentID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	// Group by user and combine camera + screenshot rows
	result := make(map[string]*ipc.MonitoringData)
	for _, row := range rows {
		data, exists := result[row.UserID]
		if !exists {
			data = &ipc.MonitoringData{
				UserId:   row.UserID,
				Username: row.Username,
			}
			result[row.UserID] = data
		}

		if row.StreamType == "camera" {
			data.CameraKey = row.StreamKey
			data.CameraStatus = ipc.StreamStatus(row.Status)
			if row.StatusTimestamp != nil {
				data.CameraTimestamp = timestamppb.New(*row.StatusTimestamp)
			}
		} else if row.StreamType == "screenshot" {
			data.ScreenshotKey = row.StreamKey
			data.ScreenshotStatus = ipc.StreamStatus(row.Status)
			if row.StatusTimestamp != nil {
				data.ScreenshotTimestamp = timestamppb.New(*row.StatusTimestamp)
			}
		}
	}

	return result, nil
}

// GetMonitoringStream gets monitoring data for a specific user in a tournament
func (s *DBStore) GetMonitoringStream(ctx context.Context, tournamentID, userID string) (*ipc.MonitoringData, error) {
	ctxDB := s.db.WithContext(ctx)

	type streamRow struct {
		UserID          string
		Username        string
		StreamType      string
		StreamKey       string
		Status          int32
		StatusTimestamp *time.Time
	}

	var rows []streamRow
	query := `
		SELECT user_id, username, stream_type, stream_key, status, status_timestamp
		FROM monitoring_streams
		WHERE tournament_id = ? AND user_id = ?
		ORDER BY stream_type
	`

	if err := ctxDB.Raw(query, tournamentID, userID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, nil
	}

	data := &ipc.MonitoringData{
		UserId:   userID,
		Username: rows[0].Username,
	}

	for _, row := range rows {
		if row.StreamType == "camera" {
			data.CameraKey = row.StreamKey
			data.CameraStatus = ipc.StreamStatus(row.Status)
			if row.StatusTimestamp != nil {
				data.CameraTimestamp = timestamppb.New(*row.StatusTimestamp)
			}
		} else if row.StreamType == "screenshot" {
			data.ScreenshotKey = row.StreamKey
			data.ScreenshotStatus = ipc.StreamStatus(row.Status)
			if row.StatusTimestamp != nil {
				data.ScreenshotTimestamp = timestamppb.New(*row.StatusTimestamp)
			}
		}
	}

	return data, nil
}

// DeleteMonitoringStreamsForTournament deletes all monitoring streams for a tournament
func (s *DBStore) DeleteMonitoringStreamsForTournament(ctx context.Context, tournamentID string) error {
	ctxDB := s.db.WithContext(ctx)
	result := ctxDB.Exec("DELETE FROM monitoring_streams WHERE tournament_id = ?", tournamentID)
	return result.Error
}

// GetActiveMonitoringStreams returns all streams with ACTIVE status (status = 2)
// Used by the polling service to check stream health
func (s *DBStore) GetActiveMonitoringStreams(ctx context.Context) ([]tl.ActiveMonitoringStream, error) {
	ctxDB := s.db.WithContext(ctx)
	var streams []tl.ActiveMonitoringStream

	query := `
		SELECT tournament_id, user_id, stream_type, stream_key
		FROM monitoring_streams
		WHERE status = 2
		ORDER BY tournament_id, user_id, stream_type
	`

	if err := ctxDB.Raw(query).Scan(&streams).Error; err != nil {
		return nil, err
	}

	return streams, nil
}
