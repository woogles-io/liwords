package tournament

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/game"
	"github.com/woogles-io/liwords/pkg/stores/models"
	tl "github.com/woogles-io/liwords/pkg/tournament"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	pb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

// DBStore is a postgres-backed store for tournaments.
type DBStore struct {
	dbPool              *pgxpool.Pool
	queries             *models.Queries
	tournamentEventChan chan<- *entity.EventWrapper
	gameStore           *game.Cache
}

// NewDBStore creates a new DB store for tournament managers.
func NewDBStore(p *pgxpool.Pool, gs *game.Cache) (*DBStore, error) {
	return &DBStore{dbPool: p, queries: models.New(p), gameStore: gs}, nil
}

// storedDivision is a division as the divisions column holds it. Classic is
// the only manager type, so the manager is decoded in the same pass as the
// rest instead of being kept as raw JSON and decoded again, and encoded the
// same way. The second pass made decoding about 30% slower and allocated a
// second copy of every division's JSON.
type storedDivision struct {
	ManagerType entity.TournamentType `json:"mgrType"`
	Classic     *tl.ClassicDivision   `json:"json"`
}

func optTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func toTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func text(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

func (s *DBStore) dbObjToEntity(tm *models.Tournament) (*entity.Tournament, error) {
	var stored map[string]*storedDivision
	err := json.Unmarshal(tm.Divisions, &stored)
	if err != nil {
		return nil, err
	}

	divisions := make(map[string]*entity.TournamentDivision, len(stored))
	for name, division := range stored {
		if division.ManagerType != entity.ClassicTournamentType {
			return nil, fmt.Errorf("Unknown division manager type: %d", division.ManagerType)
		}
		if division.Classic == nil {
			division.Classic = &tl.ClassicDivision{}
		}
		divisions[name] = &entity.TournamentDivision{
			ManagerType:     division.ManagerType,
			DivisionManager: division.Classic,
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

	tme := &entity.Tournament{UUID: tm.Uuid.String,
		Name:               tm.Name.String,
		Description:        tm.Description.String,
		AliasOf:            tm.AliasOf.String,
		Directors:          &directors,
		IsStarted:          tm.IsStarted.Valid && tm.IsStarted.Bool,
		IsFinished:         tm.IsFinished.Valid && tm.IsFinished.Bool,
		Divisions:          divisions,
		ExtraMeta:          extraMeta,
		Type:               entity.CompetitionType(tm.Type.String),
		ParentID:           tm.Parent.String,
		Slug:               tm.Slug.String,
		ScheduledStartTime: optTime(tm.ScheduledStartTime),
		ScheduledEndTime:   optTime(tm.ScheduledEndTime),
	}
	log.Debug().Msg("return-full")

	return tme, nil
}

func (s *DBStore) Get(ctx context.Context, id string) (*entity.Tournament, error) {
	tm, err := s.queries.GetTournamentByUUID(ctx, text(id))
	if err != nil {
		return nil, err
	}
	return s.dbObjToEntity(&tm)
}

// PrivateAnalysis reports whether a tournament hides players' racks from
// observers. It reads that one setting instead of loading the tournament, so
// the move path doesn't decode every division to get a boolean.
func (s *DBStore) PrivateAnalysis(ctx context.Context, id string) (bool, error) {
	return s.queries.GetTournamentPrivateAnalysis(ctx, text(id))
}

func (s *DBStore) GetBySlug(ctx context.Context, slug string) (*entity.Tournament, error) {
	tm, err := s.queries.GetTournamentBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return s.dbObjToEntity(&tm)
}

func (s *DBStore) TournamentEventChan() chan<- *entity.EventWrapper {
	return s.tournamentEventChan
}

// Set writes every column the entity holds, including empty and nil ones, so
// clearing a description or a scheduled time is saved. (Gorm's Updates
// skipped zero values, so those changes reached the cache but not the
// database.)
func (s *DBStore) Set(ctx context.Context, tm *entity.Tournament) error {
	dbt, err := s.toDBObj(tm)
	if err != nil {
		return err
	}
	log.Info().Str("tid", tm.UUID).Bool("finished", tm.IsFinished).Msg("db-set-tournament")

	return s.queries.UpdateTournament(ctx, models.UpdateTournamentParams{
		Name:               dbt.Name,
		Description:        dbt.Description,
		AliasOf:            dbt.AliasOf,
		Directors:          dbt.Directors,
		IsStarted:          dbt.IsStarted,
		IsFinished:         dbt.IsFinished,
		Divisions:          dbt.Divisions,
		ExtraMeta:          dbt.ExtraMeta,
		Type:               dbt.Type,
		Parent:             dbt.Parent,
		Slug:               dbt.Slug,
		ScheduledStartTime: dbt.ScheduledStartTime,
		ScheduledEndTime:   dbt.ScheduledEndTime,
		Uuid:               dbt.Uuid,
	})
}

func (s *DBStore) Create(ctx context.Context, tm *entity.Tournament) error {
	dbt, err := s.toDBObj(tm)
	if err != nil {
		return err
	}
	return s.queries.CreateTournament(ctx, *dbt)
}

func (s *DBStore) Disconnect() {
	s.dbPool.Close()
}

func (s *DBStore) toDBObj(t *entity.Tournament) (*models.CreateTournamentParams, error) {

	directors, err := json.Marshal(t.Directors)
	if err != nil {
		return nil, err
	}

	stored := make(map[string]*storedDivision, len(t.Divisions))
	for name, division := range t.Divisions {
		classic, ok := division.DivisionManager.(*tl.ClassicDivision)
		if !ok {
			return nil, fmt.Errorf("unknown division manager %T in division %s", division.DivisionManager, name)
		}
		stored[name] = &storedDivision{ManagerType: division.ManagerType, Classic: classic}
	}

	divisions, err := json.Marshal(stored)
	if err != nil {
		return nil, err
	}

	extraMeta, err := json.Marshal(t.ExtraMeta)
	if err != nil {
		return nil, err
	}
	var createdBy pgtype.Int4
	if t.CreatedBy != 0 {
		createdBy = pgtype.Int4{Int32: int32(t.CreatedBy), Valid: true}
	}

	return &models.CreateTournamentParams{
		Uuid:               text(t.UUID),
		Name:               text(t.Name),
		Description:        text(t.Description),
		AliasOf:            text(t.AliasOf),
		Directors:          directors,
		IsStarted:          pgtype.Bool{Bool: t.IsStarted, Valid: true},
		IsFinished:         pgtype.Bool{Bool: t.IsFinished, Valid: true},
		Divisions:          divisions,
		ExtraMeta:          extraMeta,
		Type:               text(string(t.Type)),
		Parent:             text(t.ParentID),
		Slug:               text(t.Slug),
		ScheduledStartTime: toTimestamptz(t.ScheduledStartTime),
		ScheduledEndTime:   toTimestamptz(t.ScheduledEndTime),
		CreatedBy:          createdBy,
	}, nil
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
			{Username: info.Players[0].Nickname, UserId: info.Players[0].UserId, Score: info.Scores[0], Result: res1},
			{Username: info.Players[1].Nickname, UserId: info.Players[1].UserId, Score: info.Scores[1], Result: res2},
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
	sessions, err := s.queries.ListClubSessions(ctx, models.ListClubSessionsParams{
		Parent: text(id),
		Lim:    int32(count),
		Off:    int32(offset),
	})
	if err != nil {
		return nil, err
	}

	csrs := make([]*pb.ClubSessionResponse, len(sessions))
	for i, cs := range sessions {
		csrs[i] = &pb.ClubSessionResponse{
			TournamentId: cs.Uuid.String,
			Slug:         cs.Slug.String,
		}
	}
	return &pb.ClubSessionsResponse{Sessions: csrs}, nil
}

// listingToEntities decodes rows from the listing queries, whose divisions
// are stripped of pairings and standings (see db/queries/tournaments.sql).
func (s *DBStore) listingToEntities(rows []models.Tournament) ([]*entity.Tournament, error) {
	tourneyList := make([]*entity.Tournament, len(rows))
	for i := range rows {
		t, err := s.dbObjToEntity(&rows[i])
		if err != nil {
			return nil, err
		}
		tourneyList[i] = t
	}
	return tourneyList, nil
}

func (s *DBStore) GetRecentAndUpcomingTournaments(ctx context.Context) ([]*entity.Tournament, error) {
	now := time.Now()
	rows, err := s.queries.ListRecentAndUpcomingTournaments(ctx, models.ListRecentAndUpcomingTournamentsParams{
		Since: pgtype.Timestamptz{Time: now.AddDate(0, 0, -7), Valid: true},
		Until: pgtype.Timestamptz{Time: now.AddDate(0, 0, 7), Valid: true},
	})
	if err != nil {
		return nil, err
	}
	tms := make([]models.Tournament, len(rows))
	for i, r := range rows {
		tms[i] = models.Tournament(r)
	}
	return s.listingToEntities(tms)
}

func (s *DBStore) GetPastTournaments(ctx context.Context, limit int32) ([]*entity.Tournament, error) {
	now := time.Now()
	var lim pgtype.Int4
	if limit > 0 {
		lim = pgtype.Int4{Int32: limit, Valid: true}
	}
	rows, err := s.queries.ListPastTournaments(ctx, models.ListPastTournamentsParams{
		Since: pgtype.Timestamptz{Time: now.AddDate(0, 0, -30), Valid: true},
		Until: pgtype.Timestamptz{Time: now, Valid: true},
		Lim:   lim,
	})
	if err != nil {
		return nil, err
	}
	tms := make([]models.Tournament, len(rows))
	for i, r := range rows {
		tms[i] = models.Tournament(r)
	}
	return s.listingToEntities(tms)
}

func (s *DBStore) ListAllIDs(ctx context.Context) ([]string, error) {
	tids, err := s.queries.ListAllTournamentIDs(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(tids))
	for idx, tid := range tids {
		ids[idx] = tid.String
	}
	return ids, nil
}

func (s *DBStore) CountRecentTournamentsByUser(ctx context.Context, userID uint) (int64, error) {
	return s.queries.CountTournamentsCreatedBySince(ctx, models.CountTournamentsCreatedBySinceParams{
		CreatedBy: pgtype.Int4{Int32: int32(userID), Valid: true},
		Since:     pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -7), Valid: true},
	})
}

func (s *DBStore) GetTournamentsByDirector(ctx context.Context, userID uint) ([]*entity.Tournament, error) {
	rows, err := s.queries.ListDirectedTournamentsForListing(ctx, int32(userID))
	if err != nil {
		return nil, err
	}
	tms := make([]models.Tournament, len(rows))
	for i, r := range rows {
		tms[i] = models.Tournament(r)
	}
	return s.listingToEntities(tms)
}

// FindTournamentByStreamKey uses UNIQUE index on stream_key for O(1) lookup
// Returns tournament UUID and full user ID in "uuid:username" format
// Returns empty strings if not found
func (s *DBStore) FindTournamentByStreamKey(ctx context.Context, streamKey string, streamType string) (tournamentID string, userID string, err error) {
	result, err := s.queries.FindMonitoringStreamByKey(ctx, streamKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}

	if result.TournamentID == "" || result.UserID == "" {
		return "", "", nil
	}

	// Return full userID in "uuid:username" format
	return result.TournamentID, result.UserID + ":" + result.Username, nil
}

func (s *DBStore) AddRegistrants(ctx context.Context, tid string, userIDs []string, division string) error {
	return s.queries.AddRegistrants(ctx, models.AddRegistrantsParams{
		UserIds:      userIDs,
		TournamentID: text(tid),
		DivisionID:   text(division),
	})
}

func (s *DBStore) RemoveRegistrants(ctx context.Context, tid string, userIDs []string, division string) error {
	return s.queries.RemoveRegistrants(ctx, models.RemoveRegistrantsParams{
		UserIds:      userIDs,
		TournamentID: text(tid),
		DivisionID:   text(division),
	})
}

func (s *DBStore) RemoveRegistrantsForTournament(ctx context.Context, tid string) error {
	return s.queries.RemoveRegistrantsForTournament(ctx, text(tid))
}

// ActiveTournamentsFor returns a list of 2-tuples of tournament ID, division ID
// that this user is registered in - only for active tournaments (ones that have not finished).
func (s *DBStore) ActiveTournamentsFor(ctx context.Context, userID string) ([][2]string, error) {
	registrants, err := s.queries.ActiveTournamentsForUser(ctx, text(userID))
	if err != nil {
		return nil, err
	}
	if len(registrants) == 0 {
		return nil, nil
	}
	log.Debug().Int("num-active-tournaments", len(registrants)).Str("userID", userID).Msg("active-tournaments-for")
	ret := make([][2]string, len(registrants))
	for idx, val := range registrants {
		ret[idx] = [2]string{val.TournamentID.String, val.DivisionID.String}
	}
	return ret, nil
}

// Monitoring streams methods - direct SQL operations

// InsertMonitoringStream inserts a new stream key for a user in a tournament
func (s *DBStore) InsertMonitoringStream(ctx context.Context, tid, uid, username, streamType, streamKey string) error {
	return s.queries.InsertMonitoringStream(ctx, models.InsertMonitoringStreamParams{
		TournamentID: tid,
		UserID:       uid,
		Username:     username,
		StreamType:   streamType,
		StreamKey:    streamKey,
	})
}

// UpdateMonitoringStreamStatus updates the status and timestamp of a stream
// Can update by stream key OR by tournament_id + user_id + stream_type
func (s *DBStore) UpdateMonitoringStreamStatus(ctx context.Context, streamKey string, status int, timestamp int64) error {
	return s.queries.UpdateMonitoringStreamStatus(ctx, models.UpdateMonitoringStreamStatusParams{
		Status:          int32(status),
		StatusTimestamp: timestamp,
		StreamKey:       streamKey,
	})
}

// UpdateMonitoringStreamStatusByUser updates the status and timestamp of a stream by user and type
func (s *DBStore) UpdateMonitoringStreamStatusByUser(ctx context.Context, tournamentID, userID, streamType string, status int, timestamp int64) error {
	return s.queries.UpdateMonitoringStreamStatusByUser(ctx, models.UpdateMonitoringStreamStatusByUserParams{
		Status:          int32(status),
		StatusTimestamp: timestamp,
		TournamentID:    tournamentID,
		UserID:          userID,
		StreamType:      streamType,
	})
}

// addStreamRow fills in the camera or screenshot half of a user's
// monitoring data from one monitoring_streams row.
func addStreamRow(data *ipc.MonitoringData, streamType, streamKey string, status int32, ts pgtype.Timestamp) {
	var timestamp *timestamppb.Timestamp
	if ts.Valid {
		timestamp = timestamppb.New(ts.Time)
	}
	switch streamType {
	case "camera":
		data.CameraKey = streamKey
		data.CameraStatus = ipc.StreamStatus(status)
		data.CameraTimestamp = timestamp
	case "screenshot":
		data.ScreenshotKey = streamKey
		data.ScreenshotStatus = ipc.StreamStatus(status)
		data.ScreenshotTimestamp = timestamp
	}
}

// GetMonitoringStreams gets all monitoring streams for a tournament, grouped by user
// Returns a map of userID -> MonitoringData (combining camera and screenshot rows)
func (s *DBStore) GetMonitoringStreams(ctx context.Context, tournamentID string) (map[string]*ipc.MonitoringData, error) {
	rows, err := s.queries.GetMonitoringStreams(ctx, tournamentID)
	if err != nil {
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
		addStreamRow(data, row.StreamType, row.StreamKey, row.Status, row.StatusTimestamp)
	}

	return result, nil
}

// GetMonitoringStream gets monitoring data for a specific user in a tournament
func (s *DBStore) GetMonitoringStream(ctx context.Context, tournamentID, userID string) (*ipc.MonitoringData, error) {
	rows, err := s.queries.GetMonitoringStreamsForUser(ctx, models.GetMonitoringStreamsForUserParams{
		TournamentID: tournamentID,
		UserID:       userID,
	})
	if err != nil {
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
		addStreamRow(data, row.StreamType, row.StreamKey, row.Status, row.StatusTimestamp)
	}

	return data, nil
}

// DeleteMonitoringStreamsForTournament deletes all monitoring streams for a tournament
func (s *DBStore) DeleteMonitoringStreamsForTournament(ctx context.Context, tournamentID string) error {
	return s.queries.DeleteMonitoringStreamsForTournament(ctx, tournamentID)
}

// GetActiveMonitoringStreams returns all streams with ACTIVE status (status = 2)
// Used by the polling service to check stream health
func (s *DBStore) GetActiveMonitoringStreams(ctx context.Context) ([]tl.ActiveMonitoringStream, error) {
	rows, err := s.queries.GetActiveMonitoringStreams(ctx)
	if err != nil {
		return nil, err
	}
	streams := make([]tl.ActiveMonitoringStream, len(rows))
	for i, r := range rows {
		streams[i] = tl.ActiveMonitoringStream(r)
	}
	return streams, nil
}
