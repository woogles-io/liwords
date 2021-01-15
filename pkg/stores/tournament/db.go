package tournament

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	tl "github.com/domino14/liwords/pkg/tournament"
	"github.com/domino14/liwords/rpc/api/proto/realtime"
	pb "github.com/domino14/liwords/rpc/api/proto/tournament_service"
)

// DBStore is a postgres-backed store for games.
type DBStore struct {
	cfg                 *config.Config
	db                  *gorm.DB
	tournamentEventChan chan<- *entity.EventWrapper
	gameStore           gameplay.GameStore
}

type tournament struct {
	gorm.Model
	UUID              string `gorm:"uniqueIndex"`
	Name              string
	Description       string
	AliasOf           string
	Directors         datatypes.JSON
	ExecutiveDirector string
	IsStarted         bool
	Divisions         datatypes.JSON
	// Slug looks like /tournament/abcdef, /club/madison, /club/madison/2020-04-20
	Slug string `gorm:"uniqueIndex:,expression:lower(slug)"`
	// DefaultSettings are mostly used for clubs. It's the default settings for
	// games in that club. It can be used for non-clubs as well, in perhaps
	// an advertisement or other such tournament page. (But in tournaments,
	// each division has their own settings).
	DefaultSettings datatypes.JSON
	// Type is tournament, club, session, and maybe other things.
	Type string
	// Parent is a tournament parent ID.
	Parent string `gorm:"index"`
}

// NewDBStore creates a new DB store for tournament managers.
func NewDBStore(config *config.Config, gs gameplay.GameStore) (*DBStore, error) {
	db, err := gorm.Open(postgres.Open(config.DBConnString), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&tournament{})
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
			log.Debug().Interface("division", division).Msg("unmarshalling")
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

	var directors realtime.TournamentPersons
	err = json.Unmarshal(tm.Directors, &directors)
	if err != nil {
		return nil, err
	}

	var defaultSettings *realtime.GameRequest
	err = json.Unmarshal(tm.DefaultSettings, defaultSettings)
	if err != nil {
		// it's ok, don't error out; this tournament has no default settings
	}

	tme := &entity.Tournament{UUID: tm.UUID,
		Name:              tm.Name,
		Description:       tm.Description,
		AliasOf:           tm.AliasOf,
		Directors:         &directors,
		ExecutiveDirector: tm.ExecutiveDirector,
		IsStarted:         tm.IsStarted,
		Divisions:         divisions,
		DefaultSettings:   defaultSettings,
		Type:              entity.CompetitionType(tm.Type),
		ParentID:          tm.Parent,
		Slug:              tm.Slug,
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

	defaultSettings, err := json.Marshal(t.DefaultSettings)
	if err != nil {
		// for now
		defaultSettings = []byte("{}")
	}

	dbt := &tournament{
		UUID:              t.UUID,
		Name:              t.Name,
		Description:       t.Description,
		AliasOf:           t.AliasOf,
		Directors:         directors,
		ExecutiveDirector: t.ExecutiveDirector,
		IsStarted:         t.IsStarted,
		Divisions:         divisions,
		DefaultSettings:   defaultSettings,
		Type:              string(t.Type),
		Parent:            t.ParentID,
		Slug:              t.Slug,
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

	evts := []*realtime.TournamentGameEndedEvent{}
	for _, info := range infos.GameInfo {

		var res1, res2 realtime.TournamentGameResult
		switch info.Winner {
		case -1:
			res1 = realtime.TournamentGameResult_DRAW
			res2 = realtime.TournamentGameResult_DRAW
		case 0:
			res1 = realtime.TournamentGameResult_WIN
			res2 = realtime.TournamentGameResult_LOSS
		case 1:
			res1 = realtime.TournamentGameResult_LOSS
			res2 = realtime.TournamentGameResult_WIN
		}
		if len(info.Scores) != 2 {
			log.Error().Str("tourneyID", tourneyID).Str("gameID", info.GameId).
				Msg("corrupted-recent-tourney-game")
			continue
		}
		players := []*realtime.TournamentGameEndedEvent_Player{
			{Username: info.Players[0].Nickname, Score: info.Scores[0], Result: res1},
			{Username: info.Players[1].Nickname, Score: info.Scores[1], Result: res2},
		}
		if info.Players[1].First {
			players[0], players[1] = players[1], players[0]
		}

		evt := &realtime.TournamentGameEndedEvent{
			Players:   players,
			GameId:    info.GameId,
			EndReason: info.GameEndReason,
			Time:      info.LastUpdate.Seconds,
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
