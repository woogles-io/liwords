package config

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"

	pb "github.com/domino14/liwords/rpc/api/proto/config_service"
)

const (
	GamesDisabledKey = "config:games-disabled"
	FEHashKey        = "config:fe-hash"
	// Front-page announcements. Some of these could later be dynamically
	// updated from a blog or something.
	AnnouncementsKey = "config:static-announcements"
)

type RedisConfigStore struct {
	redisPool *redis.Pool
}

func NewRedisConfigStore(r *redis.Pool) *RedisConfigStore {
	return &RedisConfigStore{redisPool: r}
}

func (s *RedisConfigStore) SetGamesEnabled(ctx context.Context, enabled bool) error {
	conn := s.redisPool.Get()
	defer conn.Close()

	var val string
	// Opposites: val 0 is enabled (the key is games-disabled).
	// We do this as we want to default to games being enabled if for
	// some reason the key is missing.
	if enabled {
		val = "0"
	} else {
		val = "1"
	}

	_, err := conn.Do("SET", GamesDisabledKey, val)
	return err
}

func (s *RedisConfigStore) GamesEnabled(ctx context.Context) (bool, error) {
	conn := s.redisPool.Get()
	defer conn.Close()
	val, err := redis.Int(conn.Do("GET", GamesDisabledKey))
	if err != nil {
		// If the key is missing, or if there's some other error,
		// games are enabled by default. Log the error, though
		log.Err(err).Msg("get-games-enabled")
		return true, nil
	}
	// disabled == 0 means enabled:
	return val == 0, nil
}

func (s *RedisConfigStore) SetFEHash(ctx context.Context, hash string) error {
	conn := s.redisPool.Get()
	defer conn.Close()

	_, err := conn.Do("SET", FEHashKey, hash)
	return err
}

func (s *RedisConfigStore) FEHash(ctx context.Context) (string, error) {
	conn := s.redisPool.Get()
	defer conn.Close()
	return redis.String(conn.Do("GET", FEHashKey))
}

func (s *RedisConfigStore) SetAnnouncements(ctx context.Context, announcements []*pb.Announcement) error {
	conn := s.redisPool.Get()
	defer conn.Close()

	bts, err := json.Marshal(announcements)
	if err != nil {
		return err
	}

	_, err = conn.Do("SET", AnnouncementsKey, string(bts))
	return err
}

func (s *RedisConfigStore) GetAnnouncements(ctx context.Context) ([]*pb.Announcement, error) {
	conn := s.redisPool.Get()
	defer conn.Close()

	a, err := redis.String(conn.Do("GET", AnnouncementsKey))
	if err != nil {
		a = "[]"
	}

	var announcements []*pb.Announcement

	err = json.Unmarshal([]byte(a), &announcements)
	if err != nil {
		return nil, err
	}

	return announcements, nil
}

func (s *RedisConfigStore) SetAnnouncement(ctx context.Context, linkSearchString string, announcement *pb.Announcement) error {
	conn := s.redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("WATCH", AnnouncementsKey)
	if err != nil {
		return err
	}

	annos, err := s.GetAnnouncements(ctx)
	if err != nil {
		return err
	}
	var anno *pb.Announcement
	var idx int
	found := false
	for idx, anno = range annos {
		if strings.Contains(anno.Link, linkSearchString) {
			found = true
			break
		}
	}
	if !found {
		return errors.New("link search string not found in announcements")
	}
	if annos[idx].Body == announcement.Body && annos[idx].Link == announcement.Link &&
		annos[idx].Title == announcement.Title {

		log.Debug().Msg("posts-match-not-replacing")
		_, err = conn.Do("UNWATCH")
		return err
	}

	annos[idx] = announcement

	_, err = conn.Do("MULTI")
	if err != nil {
		return err
	}

	err = s.SetAnnouncements(ctx, annos)
	if err != nil {
		return err
	}
	_, err = conn.Do("EXEC")
	if err != nil {
		return err
	}
	return nil
}
