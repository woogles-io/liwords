package broadcasts

// Keeping each API server's broadcast caches current.
//
// The poller runs on every server, but last_polled_at (shared, in the
// database) decides which one polls a feed. That server refreshes its feed
// cache and drops its RPC response caches; until this, the others did
// neither, so their viewers and OBS overlays could show scores up to
// feedCacheTTL (30 minutes) old.
//
// Every broadcast update already goes out on NATS (broadcasts.<uuid>) for the
// socket server. Each server now also listens to them, and on one from
// another server drops what it has cached for that broadcast, so its next
// read fetches the feed again. The update carries no feed data, only headers
// saying who sent it, for which broadcast, and when the feed was fetched; the
// socket server reads only the message body.

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

const (
	headerOrigin   = "Liwords-Origin"
	headerSlug     = "Liwords-Broadcast-Slug"
	headerPolledAt = "Liwords-Feed-Polled-At"
)

// publishBroadcastEvent publishes a serialized broadcast event, with the
// headers other servers use to refresh their caches. polledAt is when the
// feed was fetched, for an update that comes from a poll, or zero.
func (bs *BroadcastService) publishBroadcastEvent(broadcastUUID, slug string, data []byte, polledAt time.Time) error {
	msg := nats.NewMsg(NatsBroadcastSubjectPrefix + broadcastUUID)
	msg.Data = data
	if bs.natsConn.HeadersSupported() {
		msg.Header.Set(headerOrigin, bs.instanceID)
		msg.Header.Set(headerSlug, slug)
		if !polledAt.IsZero() {
			msg.Header.Set(headerPolledAt, polledAt.UTC().Format(time.RFC3339Nano))
		}
	}
	return bs.natsConn.PublishMsg(msg)
}

// applyRemoteBroadcastEvent drops this server's cached responses for the
// broadcast an update from another server is about, and its cached feed if
// that is older than the poll the update reports. It is safe to call more
// than once for one update (both the cache listener and OBS streams do).
func (bs *BroadcastService) applyRemoteBroadcastEvent(msg *nats.Msg) {
	if bs == nil || msg.Header == nil || msg.Header.Get(headerOrigin) == bs.instanceID {
		return
	}
	slug := msg.Header.Get(headerSlug)
	if slug == "" {
		return
	}
	bs.invalidateSlugCaches(slug)
	if v := msg.Header.Get(headerPolledAt); v != "" {
		polledAt, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			log.Err(err).Str("slug", slug).Msg("broadcast-polled-at-header")
			return
		}
		bs.dropFeedOlderThan(slug, polledAt)
	}
}

// dropFeedOlderThan removes the slug's cached feed if it was fetched before
// t, or its last fetch failed.
func (bs *BroadcastService) dropFeedOlderThan(slug string, t time.Time) {
	bs.feedCacheMu.Lock()
	defer bs.feedCacheMu.Unlock()
	if e, ok := bs.feedCache[slug]; ok && (e.failed || e.fetchedAt.Before(t)) {
		delete(bs.feedCache, slug)
	}
}

// startFeedSync listens for other servers' broadcast updates until ctx ends.
func (bs *BroadcastService) startFeedSync(ctx context.Context) {
	sub, err := bs.natsConn.Subscribe(NatsBroadcastSubjectPrefix+">", bs.applyRemoteBroadcastEvent)
	if err != nil {
		log.Err(err).Msg("broadcast-feed-sync-subscribe-failed")
		return
	}
	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
	}()
}
