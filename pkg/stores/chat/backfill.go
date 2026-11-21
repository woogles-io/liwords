package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/gomodule/redigo/redis"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

// BackfillStats counts what Backfill did.
type BackfillStats struct {
	Channels int   // chat streams read
	Messages int   // messages read from them
	Inserted int64 // messages not already in Postgres
	Skipped  int   // entries that couldn't be read as messages
}

// Backfill copies every chat stream in Redis to Postgres, batch messages at
// a time. A message already there is left alone (its ID comes from its
// channel and stream ID), so it can run while Mirror is copying new
// messages, and can be re-run.
func Backfill(ctx context.Context, pool *redis.Pool, q *models.Queries, batch int) (BackfillStats, error) {
	var st BackfillStats
	conn := pool.Get()
	defer conn.Close()

	var pending []legacyMessage
	flush := func() error {
		if len(pending) == 0 {
			return nil
		}
		n, err := q.AddChatMessages(ctx, messagesParams(pending))
		if err != nil {
			return err
		}
		st.Inserted += n
		pending = pending[:0]
		return nil
	}

	cursor := "0"
	for {
		vals, err := redis.Values(conn.Do("SCAN", cursor, "MATCH", "chat:*", "COUNT", 1000))
		if err != nil {
			return st, err
		}
		cursor, err = redis.String(vals[0], nil)
		if err != nil {
			return st, err
		}
		keys, err := redis.Strings(vals[1], nil)
		if err != nil {
			return st, err
		}
		for _, key := range keys {
			// Streams are capped at about 500 entries, so one XRANGE reads a
			// whole channel.
			entries, err := redis.Values(conn.Do("XRANGE", key, "-", "+"))
			if err != nil {
				if strings.Contains(err.Error(), "WRONGTYPE") {
					continue
				}
				return st, fmt.Errorf("%s: %w", key, err)
			}
			st.Channels++
			channel := "chat." + strings.TrimPrefix(key, "chat:")
			for _, e := range entries {
				m, ok := parseEntry(channel, e)
				if !ok {
					st.Skipped++
					continue
				}
				st.Messages++
				pending = append(pending, m)
				if len(pending) >= batch {
					if err := flush(); err != nil {
						return st, err
					}
				}
			}
		}
		if cursor == "0" {
			break
		}
	}
	return st, flush()
}

// parseEntry reads one stream entry, [id, [field, value, ...]], with the
// fields username, message and (on all but very old messages) userID.
func parseEntry(channel string, e interface{}) (legacyMessage, bool) {
	parts, ok := e.([]interface{})
	if !ok || len(parts) != 2 {
		return legacyMessage{}, false
	}
	id, err := redis.String(parts[0], nil)
	if err != nil {
		return legacyMessage{}, false
	}
	lid, err := parseLegacyID(id)
	if err != nil {
		return legacyMessage{}, false
	}
	fields, err := redis.StringMap(parts[1], nil)
	if err != nil {
		return legacyMessage{}, false
	}
	msg, ok := fields["message"]
	if !ok {
		return legacyMessage{}, false
	}
	return legacyMessage{
		channel:  channel,
		senderID: fields["userID"],
		username: fields["username"],
		message:  msg,
		id:       lid,
	}, true
}
