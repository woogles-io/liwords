package tournament

import (
	"context"

	"github.com/rs/zerolog/log"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	pkguser "github.com/domino14/liwords/pkg/user"
)

// DBStore is a postgres-backed store for games.
type DBStore struct {
	cfg *config.Config
	db  *gorm.DB
}

type tournamentmanager struct {
	gorm.Model
	UUID string `gorm:"type:varchar(24);index"`

	Name       string
	Directors  []string
	Type       entity.TournamentType
	Tournament datatypes.JSON
}

// NewDBStore creates a new DB store for tournament managers.
func NewDBStore(config *config.Config, userStore pkguser.Store) (*DBStore, error) {
	db, err := gorm.Open(postgres.Open(config.DBConnString), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&tournamentmanager{})
	return &DBStore{db: db, cfg: config}, nil
}

func (s *DBStore) Get(ctx context.Context, id string) (*entity.TournamentManager, error) {
	tm := &tournamentmanager{}
	ctxDB := s.db.WithContext(ctx)
	if result := ctxDB.Where("uuid = ?", tm.UUID).First(tm); result.Error != nil {
		return nil, result.Error
	}

	// Have to fix this a bit for Arena mode
	tc, err := entity.TournamentClassicUnserialize(tm.Tournament)
	if err != nil {
		return nil, err
	}
	tme := &entity.TournamentManager{UUID: tm.UUID,
		Name:       tm.Name,
		Directors:  tm.Directors,
		Type:       tm.Type,
		Tournament: tc}

	return tme, nil
}

func (s *DBStore) Set(ctx context.Context, tm *entity.TournamentManager) error {
	dbt, err := s.toDBObj(tm)
	if err != nil {
		return err
	}

	ctxDB := s.db.WithContext(ctx)
	result := ctxDB.Model(&tournamentmanager{}).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("uuid = ?", tm.UUID).Updates(dbt)

	return result.Error
}

func (s *DBStore) Create(ctx context.Context, tm *entity.TournamentManager) error {
	dbt, err := s.toDBObj(tm)
	if err != nil {
		return err
	}
	log.Debug().Interface("dbt", dbt).Msg("dbt")
	ctxDB := s.db.WithContext(ctx)
	result := ctxDB.Create(dbt)
	return result.Error
}

func (s *DBStore) toDBObj(tm *entity.TournamentManager) (*tournamentmanager, error) {
	tournament, err := tm.Tournament.Serialize()
	if err != nil {
		return nil, err
	}
	dbt := &tournamentmanager{
		UUID:       tm.UUID,
		Name:       tm.Name,
		Directors:  tm.Directors,
		Type:       tm.Type,
		Tournament: tournament}
	return dbt, nil
}
