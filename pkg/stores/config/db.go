package config

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/config_service"
)

// The settings' keys in config_settings, the same as they were in Redis.
const (
	GamesDisabledKey = "config:games-disabled"
	FEHashKey        = "config:fe-hash"
	// Front-page announcements. Some of these could later be dynamically
	// updated from a blog or something.
	AnnouncementsKey = "config:static-announcements"
	// RatingSystemKey picks the rating system for finished games: "1", "2"
	// or "paused" (games end without changing ratings). Missing means "1".
	// See docs/ratings_plan.md.
	RatingSystemKey = "config:rating-system"
)

// DBStore keeps the site settings in Postgres.
type DBStore struct {
	pool *pgxpool.Pool
	q    *models.Queries
}

func NewDBStore(pool *pgxpool.Pool) *DBStore {
	return &DBStore{pool: pool, q: models.New(pool)}
}

// get is the setting's value, and false if it has none.
func (s *DBStore) get(ctx context.Context, key string) (string, bool, error) {
	v, err := s.q.GetConfigSetting(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	} else if err != nil {
		return "", false, err
	}
	return v, true, nil
}

func (s *DBStore) set(ctx context.Context, key, value string) error {
	return s.q.SetConfigSetting(ctx, models.SetConfigSettingParams{Key: key, Value: value})
}

func (s *DBStore) SetGamesEnabled(ctx context.Context, enabled bool) error {
	// Kept as games-disabled, so that a missing value means games are on.
	val := "1"
	if enabled {
		val = "0"
	}
	return s.set(ctx, GamesDisabledKey, val)
}

// GamesEnabled is true unless games were disabled. If the setting can't be
// read, games stay enabled.
func (s *DBStore) GamesEnabled(ctx context.Context) (bool, error) {
	v, ok, err := s.get(ctx, GamesDisabledKey)
	if err != nil {
		log.Err(err).Msg("get-games-enabled")
		return true, nil
	}
	return !ok || v != "1", nil
}

// RatingSystemSetting is the rating-system setting; "1" if it was never set.
func (s *DBStore) RatingSystemSetting(ctx context.Context) (string, error) {
	v, ok, err := s.get(ctx, RatingSystemKey)
	if err != nil || !ok {
		return "1", err
	}
	return v, nil
}

func (s *DBStore) SetRatingSystemSetting(ctx context.Context, value string) error {
	return s.set(ctx, RatingSystemKey, value)
}

func (s *DBStore) SetFEHash(ctx context.Context, hash string) error {
	return s.set(ctx, FEHashKey, hash)
}

// FEHash is the front end's build hash; "" if none was set.
func (s *DBStore) FEHash(ctx context.Context) (string, error) {
	v, _, err := s.get(ctx, FEHashKey)
	return v, err
}

func (s *DBStore) SetAnnouncements(ctx context.Context, announcements []*pb.Announcement) error {
	bts, err := json.Marshal(announcements)
	if err != nil {
		return err
	}
	return s.set(ctx, AnnouncementsKey, string(bts))
}

func (s *DBStore) GetAnnouncements(ctx context.Context) ([]*pb.Announcement, error) {
	v, ok, err := s.get(ctx, AnnouncementsKey)
	if err != nil {
		return nil, err
	}
	return parseAnnouncements(v, ok)
}

func parseAnnouncements(v string, ok bool) ([]*pb.Announcement, error) {
	if !ok {
		v = "[]"
	}
	var announcements []*pb.Announcement
	if err := json.Unmarshal([]byte(v), &announcements); err != nil {
		return nil, err
	}
	return announcements, nil
}

// SetAnnouncement puts announcement in place of the first unpinned one whose
// link contains linkSearchString (see replacementSlot). The announcements are
// locked meanwhile, so two changes can't overwrite each other.
func (s *DBStore) SetAnnouncement(ctx context.Context, linkSearchString string, announcement *pb.Announcement) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.q.WithTx(tx)
	if _, err := q.AddConfigSettingIfMissing(ctx, models.AddConfigSettingIfMissingParams{
		Key: AnnouncementsKey, Value: "[]"}); err != nil {
		return err
	}
	v, err := q.LockConfigSetting(ctx, AnnouncementsKey)
	if err != nil {
		return err
	}
	annos, err := parseAnnouncements(v, true)
	if err != nil {
		return err
	}
	idx, replace, err := replacementSlot(annos, linkSearchString, announcement)
	if err != nil {
		return err
	}
	if !replace {
		log.Debug().Msg("posts-match-not-replacing")
		return nil
	}
	annos[idx] = announcement
	bts, err := json.Marshal(annos)
	if err != nil {
		return err
	}
	if err := q.SetConfigSetting(ctx, models.SetConfigSettingParams{
		Key: AnnouncementsKey, Value: string(bts)}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
