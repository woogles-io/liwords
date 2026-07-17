package broadcasts

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	NatsBroadcastSubjectPrefix = "broadcasts."
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
		if err := bs.pollBroadcast(ctx, b.ID, b.Uuid.String(), b.Slug, b.BroadcastUrl, b.BroadcastUrlFormat); err != nil {
			log.Err(err).Str("slug", b.Slug).Msg("broadcast-poll-failed")
		}
	}
}

// pollBroadcast fetches the feed for one broadcast, updates the cache, persists
// last_polled_at, and publishes a NATS update. Also used by TriggerPoll RPC.
func (bs *BroadcastService) pollBroadcast(ctx context.Context, id int32, broadcastUUID, slug, url, format string) error {
	divs, err := bs.fetchAndCacheAllDivisions(slug, url, format)
	if err != nil {
		return err
	}

	// Bust RPC response caches so the next viewer request picks up new feed scores.
	bs.invalidateSlugCaches(slug)

	if err := bs.queries.UpdateBroadcastLastPolled(ctx, id); err != nil {
		log.Err(err).Str("slug", slug).Msg("broadcast-update-last-polled-failed")
	}

	if bs.natsConn != nil {
		publishBroadcastUpdate(bs.natsConn, broadcastUUID, slug, divs)
	}

	log.Debug().Str("slug", slug).Int("numDivisions", len(divs)).Msg("broadcast-polled")
	return nil
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
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // verified manually via VerifyConnection below
			VerifyConnection:   verifyWithAIAFallback,
		},
	},
}

func fetchURL(url string) ([]byte, error) {
	resp, err := broadcastHTTPClient.Get(url) //nolint:gosec // URL is admin-configured
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func publishBroadcastUpdate(nc *nats.Conn, broadcastUUID, slug string, divs map[string]*FeedData) {
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

	if err := nc.Publish(NatsBroadcastSubjectPrefix+broadcastUUID, bts); err != nil {
		log.Err(err).Str("slug", slug).Msg("broadcast-nats-publish-failed")
	}
}
