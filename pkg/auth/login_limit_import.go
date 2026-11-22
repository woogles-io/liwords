package auth

import (
	"context"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

// ImportKnownClientsFromRedis copies the browsers remembered in Redis
// (loginok:<username>:<client ID>, with their expiry) into
// login_known_clients, if that table is still empty. The API runs it on
// startup, so the first deploy carries them over; once it has, this and the
// Redis keys can go. Failure counts last minutes and aren't copied.
func ImportKnownClientsFromRedis(ctx context.Context, q *models.Queries, pool *redis.Pool) error {
	if pool == nil {
		return nil
	}
	n, err := q.CountKnownLoginClients(ctx)
	if err != nil || n > 0 {
		return err
	}
	conn := pool.Get()
	defer conn.Close()
	imported := 0
	cursor := "0"
	for {
		vals, err := redis.Values(conn.Do("SCAN", cursor, "MATCH", "loginok:*", "COUNT", 1000))
		if err != nil {
			return err
		}
		cursor, _ = redis.String(vals[0], nil)
		keys, _ := redis.Strings(vals[1], nil)
		for _, k := range keys {
			// The username can't hold a colon; the client ID is the rest.
			parts := strings.SplitN(strings.TrimPrefix(k, "loginok:"), ":", 2)
			if len(parts) != 2 || parts[1] == "" {
				continue
			}
			ttl, err := redis.Int(conn.Do("TTL", k))
			if err != nil {
				return err
			}
			if ttl <= 0 {
				continue
			}
			if err := q.AddKnownLoginClient(ctx, models.AddKnownLoginClientParams{
				Username:  parts[0],
				ClientID:  parts[1],
				ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Duration(ttl) * time.Second), Valid: true},
			}); err != nil {
				return err
			}
			imported++
		}
		if cursor == "0" {
			break
		}
	}
	log.Info().Int("imported", imported).Msg("known-login-clients-imported-from-redis")
	return nil
}
