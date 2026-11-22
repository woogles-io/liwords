package config

import (
	"context"
	"errors"

	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

// ImportFromRedis copies the settings that were kept in Redis into
// config_settings, unless they already have a value there. The API runs it
// on startup, so the first deploy carries the settings over; once that has
// happened everywhere, this and the Redis keys can go.
func ImportFromRedis(ctx context.Context, s *DBStore, pool *redis.Pool) error {
	if pool == nil {
		// Tests and tools that run without Redis have nothing to import.
		return nil
	}
	conn := pool.Get()
	defer conn.Close()
	for _, key := range []string{GamesDisabledKey, FEHashKey, AnnouncementsKey} {
		v, err := redis.String(conn.Do("GET", key))
		if errors.Is(err, redis.ErrNil) {
			continue
		} else if err != nil {
			return err
		}
		n, err := s.q.AddConfigSettingIfMissing(ctx, models.AddConfigSettingIfMissingParams{Key: key, Value: v})
		if err != nil {
			return err
		}
		if n > 0 {
			log.Info().Str("key", key).Msg("config-setting-imported-from-redis")
		}
	}
	return nil
}
