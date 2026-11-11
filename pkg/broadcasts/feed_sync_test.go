package broadcasts

import (
	"context"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/nats-io/nats.go"

	pb "github.com/woogles-io/liwords/rpc/api/proto/broadcast_service"
)

func cachedFeedService(slug string, fetchedAt time.Time) *BroadcastService {
	bs := NewBroadcastService(nil, nil, nil, nil, nil)
	bs.feedCache[slug] = &feedCacheEntry{
		divisions: map[string]*FeedData{"A": {}},
		fetchedAt: fetchedAt,
		expiresAt: fetchedAt.Add(feedCacheTTL),
	}
	bs.statsCache.Add(slug, &pb.GetBroadcastGameStatsResponse{})
	return bs
}

func updateMsg(origin, slug string, polledAt time.Time) *nats.Msg {
	m := nats.NewMsg(NatsBroadcastSubjectPrefix + "uuid")
	m.Header.Set(headerOrigin, origin)
	m.Header.Set(headerSlug, slug)
	if !polledAt.IsZero() {
		m.Header.Set(headerPolledAt, polledAt.UTC().Format(time.RFC3339Nano))
	}
	return m
}

func TestRemoteBroadcastUpdates(t *testing.T) {
	fetched := time.Now().Add(-time.Minute)

	t.Run("another server polled since: feed and responses dropped", func(t *testing.T) {
		is := is.New(t)
		bs := cachedFeedService("nasc", fetched)
		bs.applyRemoteBroadcastEvent(updateMsg("other", "nasc", time.Now()))
		is.True(bs.peekCachedEntry("nasc") == nil)
		is.True(!bs.statsCache.Contains("nasc"))
	})

	t.Run("polled before our fetch: feed kept, responses dropped", func(t *testing.T) {
		is := is.New(t)
		bs := cachedFeedService("nasc", fetched)
		bs.applyRemoteBroadcastEvent(updateMsg("other", "nasc", fetched.Add(-time.Second)))
		is.True(bs.peekCachedEntry("nasc") != nil)
		is.True(!bs.statsCache.Contains("nasc"))
	})

	t.Run("games changed elsewhere (no poll): responses dropped only", func(t *testing.T) {
		is := is.New(t)
		bs := cachedFeedService("nasc", fetched)
		bs.applyRemoteBroadcastEvent(updateMsg("other", "nasc", time.Time{}))
		is.True(bs.peekCachedEntry("nasc") != nil)
		is.True(!bs.statsCache.Contains("nasc"))
	})

	t.Run("our own update, or one without headers: nothing dropped", func(t *testing.T) {
		is := is.New(t)
		bs := cachedFeedService("nasc", fetched)
		bs.applyRemoteBroadcastEvent(updateMsg(bs.instanceID, "nasc", time.Now()))
		bs.applyRemoteBroadcastEvent(&nats.Msg{Subject: NatsBroadcastSubjectPrefix + "uuid"})
		is.True(bs.peekCachedEntry("nasc") != nil)
		is.True(bs.statsCache.Contains("nasc"))
	})

	t.Run("a failed fetch is dropped whatever its time", func(t *testing.T) {
		is := is.New(t)
		bs := cachedFeedService("nasc", fetched)
		bs.feedCache["nasc"] = &feedCacheEntry{failed: true, expiresAt: time.Now().Add(feedFailureTTL)}
		bs.applyRemoteBroadcastEvent(updateMsg("other", "nasc", fetched.Add(-time.Hour)))
		_, ok := bs.feedCache["nasc"]
		is.True(!ok)
	})
}

// Two servers on a real NATS server, when one runs locally (CI has none):
// the one that polls keeps its fresh feed, the other drops its stale one.
func TestFeedSyncAcrossServers(t *testing.T) {
	is := is.New(t)
	nc, err := nats.Connect(nats.DefaultURL, nats.Timeout(time.Second))
	if err != nil {
		t.Skip("no local NATS server:", err)
	}
	defer nc.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slug := "feedsync-" + time.Now().Format("150405.000000")
	poller := cachedFeedService(slug, time.Now())
	other := cachedFeedService(slug, time.Now().Add(-time.Minute))
	for _, bs := range []*BroadcastService{poller, other} {
		bs.SetNatsConn(nc)
		bs.startFeedSync(ctx)
	}
	is.NoErr(nc.Flush())

	poller.publishBroadcastUpdate("uuid-"+slug, slug, map[string]*FeedData{"A": {}}, time.Now())
	is.NoErr(nc.Flush())
	deadline := time.Now().Add(2 * time.Second)
	for other.peekCachedEntry(slug) != nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	is.True(other.peekCachedEntry(slug) == nil)  // refetches on its next read
	is.True(poller.peekCachedEntry(slug) != nil) // kept what it just fetched
}
