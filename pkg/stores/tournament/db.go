package tournament

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/rs/zerolog/log"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
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
	UUID              string `gorm:"uniqueIndex:,expression:lower(uuid)"`
	Name              string
	Description       string
	Directors         datatypes.JSON
	ExecutiveDirector string
	IsStarted         bool
	Divisions         datatypes.JSON
	// Slug
	// Slug              string `gorm:"uniqueIndex:,expression:lower(slug)"`
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

func (s *DBStore) Get(ctx context.Context, id string) (*entity.Tournament, error) {
	tm := &tournament{}
	ctxDB := s.db.WithContext(ctx)
	if result := ctxDB.Where("lower(uuid) = ?", strings.ToLower(id)).First(tm); result.Error != nil {
		return nil, result.Error
	}

	var divisions map[string]*entity.TournamentDivision
	err := json.Unmarshal(tm.Divisions, &divisions)
	if err != nil {
		return nil, err
	}

	var directors entity.TournamentPersons
	err = json.Unmarshal(tm.Directors, &directors)
	if err != nil {
		return nil, err
	}

	tme := &entity.Tournament{UUID: tm.UUID,
		Name:              tm.Name,
		Description:       tm.Description,
		Directors:         &directors,
		ExecutiveDirector: tm.ExecutiveDirector,
		IsStarted:         tm.IsStarted,
		Divisions:         divisions}

	return tme, nil
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
		Where("lower(uuid) = ?", strings.ToLower(tm.UUID)).Updates(dbt)

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

	divisions, err := json.Marshal(t.Divisions)
	if err != nil {
		return nil, err
	}

	dbt := &tournament{
		UUID:              t.UUID,
		Name:              t.Name,
		Description:       t.Description,
		Directors:         directors,
		ExecutiveDirector: t.ExecutiveDirector,
		IsStarted:         t.IsStarted,
		Divisions:         divisions}
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
