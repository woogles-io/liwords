package broadcasts

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	NatsBroadcastSubjectPrefix = "broadcasts."
	// NatsUserSlotSubjectPrefix is published when a user re-points one of their
	// stream slots, so live OBS streams rebind to the new target broadcast.
	NatsUserSlotSubjectPrefix = "obs.userslot."
	pollerTickInterval        = 30 * time.Second
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

// maxPollsPerCycle bounds one cycle's polls, so a backlog (after downtime,
// say) is spread over a few cycles.
const maxPollsPerCycle = 20

// runPollerCycle polls every broadcast that is due, claiming each one so that
// no other server polls it in the same interval.
func (bs *BroadcastService) runPollerCycle(ctx context.Context) {
	for range maxPollsPerCycle {
		if ctx.Err() != nil {
			return
		}
		b, err := bs.queries.ClaimBroadcastForPolling(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			return
		}
		if err != nil {
			log.Err(err).Msg("broadcast-poller-claim-failed")
			return
		}
		if _, err := bs.pollBroadcast(ctx, b.ID, b.Uuid.String(), b.Slug, b.BroadcastUrl, b.BroadcastUrlFormat); err != nil {
			log.Err(err).Str("slug", b.Slug).Msg("broadcast-poll-failed")
		}
	}
}

// pollBroadcast fetches one broadcast's feed and stores it. When it changed,
// viewers are told over NATS. A failed fetch is recorded and leaves the
// stored feed in place. Also used by the TriggerPoll RPC.
func (bs *BroadcastService) pollBroadcast(ctx context.Context, id int32, broadcastUUID, slug, url, format string) (changed bool, err error) {
	ctx, span := otel.Tracer("broadcast-poller").Start(ctx, "broadcast.poll",
		trace.WithAttributes(attribute.String("broadcast.slug", slug)))
	defer func() {
		span.SetAttributes(attribute.Bool("feed.changed", changed))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			if rerr := bs.queries.RecordBroadcastFeedError(ctx, models.RecordBroadcastFeedErrorParams{
				BroadcastID: id,
				LastError:   err.Error(),
			}); rerr != nil {
				log.Err(rerr).Str("slug", slug).Msg("broadcast-record-feed-error-failed")
			}
		}
		span.End()
	}()

	raw, err := fetchURL(ctx, url)
	if err != nil {
		return false, err
	}
	span.SetAttributes(attribute.Int("feed.bytes", len(raw)))
	hash := feedHash(format, raw)

	n, err := bs.queries.MarkBroadcastFeedUnchanged(ctx, models.MarkBroadcastFeedUnchangedParams{
		BroadcastID: id,
		ContentHash: hash,
	})
	if err != nil {
		return false, err
	}
	if n == 0 {
		// Parse before storing, so a feed we can't read never replaces one
		// we can.
		divs, err := parseFeed(raw, format)
		if err != nil {
			return false, fmt.Errorf("parse feed: %w", err)
		}
		gz, err := gzipBytes(raw)
		if err != nil {
			return false, err
		}
		f := newParsedFeed(hash, divs)
		currentRound := 0
		if fd := f.division(""); fd != nil {
			currentRound = fd.CurrentRound
		}
		if err := bs.queries.SaveBroadcastFeed(ctx, models.SaveBroadcastFeedParams{
			BroadcastID:  id,
			Format:       format,
			RawGz:        gz,
			ContentHash:  hash,
			CurrentRound: int32(currentRound),
		}); err != nil {
			return false, err
		}
		span.SetAttributes(attribute.Int("feed.gz_bytes", len(gz)))
		bs.parsedFeeds.Add(slug, f)
		changed = true
		if bs.natsConn != nil {
			publishBroadcastUpdate(bs.natsConn, broadcastUUID, slug, int32(currentRound))
		}
	}

	// Older servers schedule polls from last_polled_at.
	if err := bs.queries.UpdateBroadcastLastPolled(ctx, id); err != nil {
		log.Err(err).Str("slug", slug).Msg("broadcast-update-last-polled-failed")
	}
	log.Debug().Str("slug", slug).Bool("changed", changed).Msg("broadcast-polled")
	return changed, nil
}

// aiaCertCache caches intermediate certificates fetched via the Authority
// Information Access "CA Issuers" URL, keyed by that URL.
var aiaCertCache sync.Map // string -> *x509.Certificate

// fetchIssuerCert downloads and parses the issuer certificate served at an
// Authority Information Access "CA Issuers" URL, caching the result.
func fetchIssuerCert(url string) (*x509.Certificate, error) {
	if v, ok := aiaCertCache.Load(url); ok {
		return v.(*x509.Certificate), nil
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url) //nolint:gosec // fixed AIA URL taken from a verified leaf cert
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(body)
	if err != nil {
		return nil, fmt.Errorf("parse issuer cert from %s: %w", url, err)
	}
	aiaCertCache.Store(url, cert)
	return cert, nil
}

// verifyWithAIAFallback verifies the server's certificate chain, fetching
// missing intermediate certificates via the leaf's Authority Information
// Access extension when the server doesn't send a full chain (as browsers
// do automatically but Go's TLS stack does not). This works around broadcast
// feed hosts, such as event.scrabbleplayers.org, that omit their
// intermediate CA cert from the TLS handshake.
func verifyWithAIAFallback(cs tls.ConnectionState) error {
	if len(cs.PeerCertificates) == 0 {
		return fmt.Errorf("no peer certificates presented")
	}
	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		roots = x509.NewCertPool()
	}
	intermediates := x509.NewCertPool()
	for _, c := range cs.PeerCertificates[1:] {
		intermediates.AddCert(c)
	}

	leaf := cs.PeerCertificates[0]
	opts := x509.VerifyOptions{
		DNSName:       cs.ServerName,
		Roots:         roots,
		Intermediates: intermediates,
	}
	_, verifyErr := leaf.Verify(opts)

	// Walk up the chain, fetching each missing issuer, until verification
	// succeeds, fails for a reason other than a missing issuer, or we run
	// out of AIA URLs to follow.
	cert := leaf
	for range 5 {
		if verifyErr == nil {
			return nil
		}
		if _, ok := verifyErr.(x509.UnknownAuthorityError); !ok {
			return verifyErr
		}
		if len(cert.IssuingCertificateURL) == 0 {
			return verifyErr
		}
		issuer, fetchErr := fetchIssuerCert(cert.IssuingCertificateURL[0])
		if fetchErr != nil {
			return verifyErr
		}
		intermediates.AddCert(issuer)
		cert = issuer
		_, verifyErr = leaf.Verify(opts)
	}
	return verifyErr
}

var broadcastHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: otelhttp.NewTransport(&http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // verified manually via VerifyConnection below
			VerifyConnection:   verifyWithAIAFallback,
		},
	}),
}

func fetchURL(ctx context.Context, feedURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cacheBustedURL(feedURL, time.Now()), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := broadcastHTTPClient.Do(req) //nolint:gosec // URL is admin-configured
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("feed fetch returned %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// cacheBustedURL appends a unique query parameter so CDNs in front of
// director-hosted feeds can't serve a stale copy. Some (e.g. Hostinger's) cache
// tourney.js for days, ignore a Cache-Control request header, and hold separate
// copies per cache node, so polls would flip between current and hours-old
// results. Existing query parameters are kept verbatim.
func cacheBustedURL(feedURL string, now time.Time) string {
	u, err := url.Parse(feedURL)
	if err != nil {
		return feedURL
	}
	bust := "_=" + strconv.FormatInt(now.UnixNano(), 10)
	if u.RawQuery == "" {
		u.RawQuery = bust
	} else {
		u.RawQuery += "&" + bust
	}
	return u.String()
}

func publishBroadcastUpdate(nc *nats.Conn, broadcastUUID, slug string, currentRound int32) {
	evt := entity.WrapEvent(&ipc.BroadcastUpdatedEvent{
		Slug:         slug,
		CurrentRound: currentRound,
	}, ipc.MessageType_BROADCAST_UPDATED)

	bts, err := evt.Serialize()
	if err != nil {
		log.Err(err).Str("slug", slug).Msg("broadcast-event-serialize-failed")
		return
	}

	if err := nc.Publish(NatsBroadcastSubjectPrefix+broadcastUUID, bts); err != nil {
		log.Err(err).Str("slug", slug).Msg("broadcast-nats-publish-failed")
	}
}
