package broadcasts

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	NatsBroadcastChannelPrefix = "channel-broadcast-"
	pollerTickInterval         = 30 * time.Second
)

// StartPoller launches the background feed-polling goroutine.
// Call once from main after NATS is ready.
func (bs *BroadcastService) StartPoller(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(pollerTickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				bs.runPollerCycle(ctx)
			}
		}
	}()
	log.Info().Msg("broadcast-poller-started")
}

func (bs *BroadcastService) runPollerCycle(ctx context.Context) {
	broadcasts, err := bs.queries.GetBroadcastsForPolling(ctx)
	if err != nil {
		log.Err(err).Msg("broadcast-poller-get-broadcasts-failed")
		return
	}

	now := time.Now()
	for _, b := range broadcasts {
		if b.PollStartTime.Valid && now.Before(b.PollStartTime.Time) {
			continue
		}
		if b.PollEndTime.Valid && now.After(b.PollEndTime.Time) {
			continue
		}
		if b.LastPolledAt.Valid {
			nextPoll := b.LastPolledAt.Time.Add(time.Duration(b.PollIntervalSeconds) * time.Second)
			if now.Before(nextPoll) {
				continue
			}
		}
		if err := bs.pollBroadcast(ctx, b.ID, b.Slug, b.BroadcastUrl, b.BroadcastUrlFormat); err != nil {
			log.Err(err).Str("slug", b.Slug).Msg("broadcast-poll-failed")
		}
	}
}

// pollBroadcast fetches the feed for one broadcast, updates the cache, persists
// last_polled_at, and publishes a NATS update. Also used by TriggerPoll RPC.
func (bs *BroadcastService) pollBroadcast(ctx context.Context, id int32, slug, url, format string) error {
	divs, err := bs.fetchAndCacheAllDivisions(slug, url, format)
	if err != nil {
		return err
	}

	if err := bs.queries.UpdateBroadcastLastPolled(ctx, id); err != nil {
		log.Err(err).Str("slug", slug).Msg("broadcast-update-last-polled-failed")
	}

	if bs.natsConn != nil {
		publishBroadcastUpdate(bs.natsConn, slug, divs)
	}

	log.Debug().Str("slug", slug).Int("numDivisions", len(divs)).Msg("broadcast-polled")
	return nil
}

func fetchURL(url string) ([]byte, error) {
	resp, err := http.Get(url) //nolint:gosec // URL is admin-configured
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func publishBroadcastUpdate(nc *nats.Conn, slug string, divs map[string]*FeedData) {
	// Use the first (alphabetically) division's current round for the event.
	currentRound := int32(0)
	firstName := ""
	for name := range divs {
		if firstName == "" || name < firstName {
			firstName = name
		}
	}
	if fd, ok := divs[firstName]; ok {
		currentRound = int32(fd.CurrentRound)
	}
	evt := entity.WrapEvent(&ipc.BroadcastUpdatedEvent{
		Slug:         slug,
		CurrentRound: currentRound,
	}, ipc.MessageType_BROADCAST_UPDATED)

	bts, err := evt.Serialize()
	if err != nil {
		log.Err(err).Str("slug", slug).Msg("broadcast-event-serialize-failed")
		return
	}

	if err := nc.Publish(NatsBroadcastChannelPrefix+slug, bts); err != nil {
		log.Err(err).Str("slug", slug).Msg("broadcast-nats-publish-failed")
	}
}

