package soughtgame

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/rs/zerolog/log"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
)

const (
	typeSeek  = "seek"
	typeMatch = "match"
)

type DBStore struct {
	cfg *config.Config
	db  *gorm.DB
}

type soughtgame struct {
	CreatedAt time.Time
	UUID      string `gorm:"index"`
	Seeker    string `gorm:"index"`

	Type   string // seek or match
	ConnID string `gorm:"index"`
	// Only for match requests
	Receiver string `gorm:"index"`

	Request datatypes.JSON
}

func NewDBStore(config *config.Config) (*DBStore, error) {
	db, err := gorm.Open(postgres.Open(config.DBConnString), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&soughtgame{})
	return &DBStore{db: db, cfg: config}, nil
}

func (s *DBStore) sgFromDBObj(g *soughtgame) (*entity.SoughtGame, error) {
	var err error

	switch g.Type {
	case typeSeek:
		sr := &pb.SeekRequest{}
		err = json.Unmarshal(g.Request, sr)
		if err != nil {
			return nil, err
		}
		return entity.NewSoughtGame(sr), nil
	case typeMatch:
		mr := &pb.MatchRequest{}
		err = json.Unmarshal(g.Request, mr)
		if err != nil {
			return nil, err
		}
		return entity.NewMatchRequest(mr), nil
	}
	log.Error().Str("seekType", g.Type).Str("id", g.UUID).Msg("unexpected-seek-type")
	return nil, errors.New("unknown error getting seek or match")
}

// Get gets the sought game with the given ID.
func (s *DBStore) Get(ctx context.Context, id string) (*entity.SoughtGame, error) {
	g := &soughtgame{}
	ctxDB := s.db.WithContext(ctx)
	if result := ctxDB.Where("uuid = ?", id).First(g); result.Error != nil {
		return nil, result.Error
	}
	return s.sgFromDBObj(g)
}

// GetByConnID gets the sought game with the given socket connection ID.
func (s *DBStore) GetByConnID(ctx context.Context, connID string) (*entity.SoughtGame, error) {
	g := &soughtgame{}
	ctxDB := s.db.WithContext(ctx)
	if result := ctxDB.Where("conn_id = ?", connID).First(g); result.Error != nil {
		return nil, result.Error
	}
	return s.sgFromDBObj(g)
}

func (s *DBStore) getBySeekerID(ctx context.Context, seekerID string) (*entity.SoughtGame, error) {
	g := &soughtgame{}
	ctxDB := s.db.WithContext(ctx)
	if result := ctxDB.Where("seeker = ?", seekerID).First(g); result.Error != nil {
		return nil, result.Error
	}
	return s.sgFromDBObj(g)
}

// Set sets the sought-game in the database.
func (s *DBStore) Set(ctx context.Context, game *entity.SoughtGame) error {
	var bts []byte
	var sgtype string
	var err error
	if game.Type() == entity.TypeSeek {
		bts, err = json.Marshal(game.SeekRequest)
		sgtype = typeSeek
	} else if game.Type() == entity.TypeMatch {
		bts, err = json.Marshal(game.MatchRequest)
		sgtype = typeMatch
	}
	if err != nil {
		return err
	}
	dbg := &soughtgame{
		UUID:    game.ID(),
		ConnID:  game.ConnID(),
		Seeker:  game.Seeker(),
		Type:    sgtype,
		Request: bts,
	}
	if game.Type() == entity.TypeMatch {
		dbg.Receiver = game.MatchRequest.ReceivingUser.UserId
	}
	ctxDB := s.db.WithContext(ctx)
	result := ctxDB.Create(dbg)
	return result.Error
}

func (s *DBStore) deleteSoughtGame(ctx context.Context, id string) error {
	ctxDB := s.db.WithContext(ctx)
	result := ctxDB.Where("uuid = ?", id).Delete(&soughtgame{})
	return result.Error
}

func (s *DBStore) Delete(ctx context.Context, id string) error {
	return s.deleteSoughtGame(ctx, id)
}

// DeleteForUser deletes the game by seeker ID.
func (s *DBStore) DeleteForUser(ctx context.Context, userID string) (*entity.SoughtGame, error) {
	sg, err := s.getBySeekerID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return sg, s.deleteSoughtGame(ctx, sg.ID())
}

// DeleteForConnID deletes the game by connection ID
func (s *DBStore) DeleteForConnID(ctx context.Context, connID string) (*entity.SoughtGame, error) {
	sg, err := s.GetByConnID(ctx, connID)
	if err != nil {
		return nil, err
	}
	return sg, s.deleteSoughtGame(ctx, sg.ID())
}

// ListOpenSeeks lists all open seek requests
func (s *DBStore) ListOpenSeeks(ctx context.Context) ([]*entity.SoughtGame, error) {

	var games []soughtgame
	var err error

	ctxDB := s.db.WithContext(ctx)
	if result := ctxDB.Table("soughtgames").
		Select("request").
		Where("type = ?", typeSeek).Scan(&games); result.Error != nil {

		return nil, result.Error
	}
	entGames := make([]*entity.SoughtGame, len(games))
	for idx, g := range games {
		entGames[idx], err = s.sgFromDBObj(&g)
		if err != nil {
			return nil, err
		}
	}
	return entGames, nil
}

// ListOpenMatches lists all open match requests for receiverID, in tourneyID (optional)
func (s *DBStore) ListOpenMatches(ctx context.Context, receiverID, tourneyID string) ([]*entity.SoughtGame, error) {
	var games []soughtgame

	ctxDB := s.db.WithContext(ctx)
	if result := ctxDB.Table("soughtgames").
		Select("request").
		Where("receiver = ? AND type = ?", receiverID, typeMatch).Scan(&games); result.Error != nil {

		return nil, result.Error
	}
	entGames := []*entity.SoughtGame{}
	for _, g := range games {
		sg, err := s.sgFromDBObj(&g)
		if err != nil {
			return nil, err
		}
		// XXX: We can probably encode this condition in the query above as
		// we're using JSONB:
		if tourneyID != "" && sg.MatchRequest.TournamentId != tourneyID {
			continue
		}
		entGames = append(entGames, sg)
	}
	return entGames, nil
}

// ExistsForUser returns true if the user already has an outstanding seek request.
func (s *DBStore) ExistsForUser(ctx context.Context, userID string) (bool, error) {
	ctxDB := s.db.WithContext(ctx)
	var count int64
	if result := ctxDB.Model(&soughtgame{}).Where("seeker = ?", userID).Count(&count); result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// UserMatchedBy returns true if there is an open seek request from matcher for user
func (s *DBStore) UserMatchedBy(ctx context.Context, userID, matcher string) (bool, error) {

	ctxDB := s.db.WithContext(ctx)
	var count int64

	if result := ctxDB.Model(&soughtgame{}).
		Where("receiver = ? AND seeker = ? AND type = ?", userID, matcher, typeMatch).
		Count(&count); result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}
