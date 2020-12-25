package club

import (
	"context"
	"encoding/json"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
)

type DBStore struct {
	cfg *config.Config
	db  *gorm.DB

	clubEventChan chan<- *entity.EventWrapper
}

type club struct {
	gorm.Model
	UUID        string `gorm:"uniqueIndex"`
	Name        string
	Description string
	Slug        string `gorm:"uniqueIndex:,expression:lower(slug)"`

	Directors         datatypes.JSON
	ExecutiveDirector string

	DefaultSettings datatypes.JSON
}

func NewDBStore(config *config.Config) (*DBStore, error) {
	db, err := gorm.Open(postgres.Open(config.DBConnString), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&club{})
	return &DBStore{db: db}, nil
}

func (s *DBStore) Get(ctx context.Context, id string) (*entity.Club, error) {
	cm := &club{}
	ctxDB := s.db.WithContext(ctx)
	if result := ctxDB.Where("uuid = ?", id).First(cm); result.Error != nil {
		return nil, result.Error
	}

	var directors entity.TournamentPersons
	err := json.Unmarshal(cm.Directors, &directors)
	if err != nil {
		return nil, err
	}

	var settings *realtime.GameRequest
	err = json.Unmarshal(cm.DefaultSettings, settings)
	if err != nil {
		return nil, err
	}

	cme := &entity.Club{
		UUID:              cm.UUID,
		Name:              cm.Name,
		Slug:              cm.Slug,
		Description:       cm.Description,
		Directors:         &directors,
		ExecutiveDirector: cm.ExecutiveDirector,
		DefaultSettings:   settings}
	return cme, nil
}
