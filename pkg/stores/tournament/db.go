package tournament

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
)

// DBStore is a postgres-backed store for games.
type DBStore struct {
	cfg *config.Config
	db  *gorm.DB
}

type tournament struct {
	gorm.Model
	UUID string `gorm:"type:varchar(24);index"`

	Name              string
	Description       string
	StartTime         time.Time
	Directors         datatypes.JSON
	Type              entity.TournamentType
	TournamentManager datatypes.JSON

	Controls datatypes.JSON
	Players  datatypes.JSON
}

// NewDBStore creates a new DB store for tournament managers.
func NewDBStore(config *config.Config) (*DBStore, error) {
	db, err := gorm.Open(postgres.Open(config.DBConnString), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&tournament{})
	return &DBStore{db: db, cfg: config}, nil
}

func (s *DBStore) Get(ctx context.Context, id string) (*entity.Tournament, error) {
	tm := &tournament{}
	ctxDB := s.db.WithContext(ctx)
	if result := ctxDB.Where("uuid = ?", tm.UUID).First(tm); result.Error != nil {
		return nil, result.Error
	}

	// Have to fix this a bit for Arena mode
	tc, err := entity.TournamentClassicUnserialize(tm.TournamentManager)
	if err != nil {
		return nil, err
	}

	var controls realtime.GameRequest
	err = json.Unmarshal(tm.Controls, &controls)
	if err != nil {
		return nil, err
	}

	var players entity.TournamentPersons
	err = json.Unmarshal(tm.Players, &players)
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
		StartTime:         tm.StartTime,
		Directors:         &directors,
		Type:              tm.Type,
		Controls:          &controls,
		Players:           &players,
		TournamentManager: tc}

	return tme, nil
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
	tm, err := t.TournamentManager.Serialize()
	if err != nil {
		return nil, err
	}

	controls, err := json.Marshal(t.Controls)
	if err != nil {
		return nil, err
	}

	directors, err := json.Marshal(t.Directors)
	if err != nil {
		return nil, err
	}

	players, err := json.Marshal(t.Players)
	if err != nil {
		return nil, err
	}

	dbt := &tournament{
		UUID:              t.UUID,
		Name:              t.Name,
		Description:       t.Description,
		StartTime:         t.StartTime,
		Directors:         directors,
		Type:              t.Type,
		Controls:          controls,
		Players:           players,
		TournamentManager: tm}
	return dbt, nil
}
