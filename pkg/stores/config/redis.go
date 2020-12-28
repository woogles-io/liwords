package config

import (
	"context"

	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"
)

const (
	GamesDisabledKey = "config:games-disabled"
	FEHashKey        = "config:fe-hash"
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
