package tournament

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
)

// DBStore is a postgres-backed store for games.
type DBStore struct {
	cfg *config.Config
	db  *gorm.DB
}

type tournament struct {
	gorm.Model
	UUID string `gorm:"type:varchar(24);index"`

	Name        string
	Description string
	Directors   datatypes.JSON
	Type        entity.TournamentType
	IsStarted   bool
	Divisions   datatypes.JSON
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
		Name:        tm.Name,
		Description: tm.Description,
		Directors:   &directors,
		IsStarted:   tm.IsStarted,
		Divisions:   divisions}

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

	directors, err := json.Marshal(t.Directors)
	if err != nil {
		return nil, err
	}

	divisions, err := json.Marshal(t.Divisions)
	if err != nil {
		return nil, err
	}

	dbt := &tournament{
		UUID:        t.UUID,
		Name:        t.Name,
		Description: t.Description,
		Directors:   directors,
		IsStarted:   t.IsStarted,
		Divisions:   divisions}
	return dbt, nil
}
