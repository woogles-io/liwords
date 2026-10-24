package redis

import (
	"context"
	"sort"
	"testing"

	"github.com/matryer/is"
)

func TestScanChannels(t *testing.T) {
	is := is.New(t)
	redisPool := newPool(RedisURL)
	cs := NewRedisChatStore(redisPool, nil, nil, nil)
	flushTestDB(redisPool)

	ctx := context.Background()
	for _, ch := range []string{"chat.game.g1", "chat.game.g2", "chat.gametv.g1", "chat.pm.u1_u2"} {
		_, err := cs.AddChat(ctx, "cesar", "u1", "hi "+ch, ch, ch, false)
		is.NoErr(err)
	}

	scanAll := func(prefix string) []string {
		var all []string
		cursor := ""
		for {
			chans, next, err := cs.ScanChannels(ctx, cursor, prefix, 1)
			is.NoErr(err)
			all = append(all, chans...)
			if next == "0" {
				break
			}
			cursor = next
		}
		sort.Strings(all)
		return all
	}

	is.Equal(scanAll("chat.game."), []string{"chat.game.g1", "chat.game.g2"})
	is.Equal(scanAll("chat.gametv."), []string{"chat.gametv.g1"})
	is.Equal(len(scanAll("")), 4)

	msgs, err := cs.OldChats(ctx, "chat.gametv.g1", 1000)
	is.NoErr(err)
	is.Equal(len(msgs), 1)
	is.Equal(msgs[0].Message, "hi chat.gametv.g1")
}
