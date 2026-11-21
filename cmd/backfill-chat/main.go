// backfill-chat copies the chat history held in Redis to Postgres
// (chat_messages). It is safe to re-run: messages already copied, by the
// API's dual write or an earlier run, are left alone. It reads REDIS_URL and
// the DB_* settings like the API.
//
//	go run ./cmd/backfill-chat [-batch 1000]
package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/chat"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

func main() {
	batch := flag.Int("batch", 1000, "messages per insert")
	flag.Parse()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := &config.Config{}
	cfg.Load(nil)
	ctx := context.Background()

	dbPool, err := pgxpool.New(ctx, cfg.DBConnDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("connecting to the database")
	}
	defer dbPool.Close()
	redisPool := &redis.Pool{
		MaxIdle:     1,
		IdleTimeout: time.Minute,
		Dial:        func() (redis.Conn, error) { return redis.DialURL(cfg.RedisURL) },
	}
	defer redisPool.Close()

	start := time.Now()
	st, err := chat.Backfill(ctx, redisPool, models.New(dbPool), *batch)
	ev := log.Info()
	if err != nil {
		ev = log.Error().Err(err)
	}
	ev.Int("channels", st.Channels).Int("messages", st.Messages).
		Int64("inserted", st.Inserted).Int("skipped", st.Skipped).
		Dur("took", time.Since(start)).Msg("backfill-chat")
	if err != nil {
		os.Exit(1)
	}
}
