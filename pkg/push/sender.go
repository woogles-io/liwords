package push

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

var (
	errBadKeys = errors.New("a push subscription needs its p256dh and auth keys")
	errNoUser  = errors.New("no such user")
)

// Queries are the subscription queries the sender uses.
type Queries interface {
	UpsertPushSubscription(ctx context.Context, arg models.UpsertPushSubscriptionParams) (int64, error)
	GetPushSubscriptions(ctx context.Context, userUuid string) ([]models.GetPushSubscriptionsRow, error)
	DeleteUserPushSubscription(ctx context.Context, arg models.DeleteUserPushSubscriptionParams) (int64, error)
	DeletePushSubscription(ctx context.Context, endpoint string) error
}

// Sender keeps users' push subscriptions and sends to them.
type Sender struct {
	settings *Settings
	queries  Queries
	client   *http.Client

	// inFlight holds, by endpoint, how to cancel the send to that device
	// still under way. A newer push to a device supersedes the older one, so
	// each device has at most one send in flight.
	mu       sync.Mutex
	inFlight map[string]*flight
}

// flight is one send to one device.
type flight struct {
	cancel context.CancelFunc
}

// NewSender returns a sender for settings that NewSettings accepted.
func NewSender(settings *Settings, queries Queries) *Sender {
	return &Sender{
		settings: settings,
		queries:  queries,
		client:   newClient(),
		inFlight: map[string]*flight{},
	}
}

// PublicKey is the key a browser subscribes with.
func (s *Sender) PublicKey() string {
	return s.settings.PublicKey
}

// Subscribe keeps a browser's subscription for a user.
func (s *Sender) Subscribe(ctx context.Context, userUUID, endpoint, p256dh, auth string) error {
	if err := checkEndpoint(endpoint); err != nil {
		return err
	}
	if !validKeys(p256dh, auth) {
		return errBadKeys
	}
	n, err := s.queries.UpsertPushSubscription(ctx, models.UpsertPushSubscriptionParams{
		Endpoint: endpoint,
		P256dh:   p256dh,
		Auth:     auth,
		UserUuid: userUUID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return errNoUser
	}
	return nil
}

// Unsubscribe forgets a browser's subscription for a user.
func (s *Sender) Unsubscribe(ctx context.Context, userUUID, endpoint string) error {
	_, err := s.queries.DeleteUserPushSubscription(ctx, models.DeleteUserPushSubscriptionParams{
		Endpoint: endpoint,
		UserUuid: userUUID,
	})
	return err
}

// Notify sends payload to every browser the user subscribed. It returns at
// once; the sending happens in the background.
func (s *Sender) Notify(userUUID string, payload []byte) {
	go s.send(context.Background(), userUUID, payload)
}

func (s *Sender) send(ctx context.Context, userUUID string, payload []byte) {
	subs, err := s.queries.GetPushSubscriptions(ctx, userUUID)
	if err != nil {
		log.Err(err).Str("userID", userUUID).Msg("web-push-get-subscriptions")
		return
	}
	for _, sub := range subs {
		sendCtx, done := s.startSend(ctx, sub.Endpoint)
		s.sendOne(sendCtx, sub, payload)
		done()
	}
}

// startSend cancels any earlier send to endpoint still in flight and returns
// the context of this one, and done, which forgets this send unless a newer
// one has already taken its place.
func (s *Sender) startSend(ctx context.Context, endpoint string) (context.Context, func()) {
	ctx, cancel := context.WithCancel(ctx)
	this := &flight{cancel: cancel}
	s.mu.Lock()
	if earlier, ok := s.inFlight[endpoint]; ok {
		earlier.cancel()
	}
	s.inFlight[endpoint] = this
	s.mu.Unlock()
	return ctx, func() {
		s.mu.Lock()
		if s.inFlight[endpoint] == this {
			delete(s.inFlight, endpoint)
		}
		s.mu.Unlock()
		cancel()
	}
}

func (s *Sender) sendOne(ctx context.Context, sub models.GetPushSubscriptionsRow, payload []byte) {
	// An endpoint's path is the subscription's secret, so only its host is
	// logged.
	host := ""
	if u, err := url.Parse(sub.Endpoint); err == nil {
		host = u.Host
	}
	// The push library writes its padding into the message's spare capacity,
	// so every send gets its own copy.
	message := slices.Clone(payload)
	resp, err := webpush.SendNotificationWithContext(ctx, message,
		&webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys:     webpush.Keys{Auth: sub.Auth, P256dh: sub.P256dh},
		},
		&webpush.Options{
			HTTPClient:      s.client,
			Subscriber:      s.subscriber(),
			TTL:             int(s.settings.TTL / time.Second),
			VAPIDPublicKey:  s.settings.PublicKey,
			VAPIDPrivateKey: s.settings.PrivateKey,
		})
	if err != nil {
		// The client's errors quote the whole endpoint, so only the cause is
		// logged.
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		if errors.Is(err, context.Canceled) {
			// A newer push to this device took its place.
			log.Debug().Str("pushService", host).Msg("web-push-superseded")
			return
		}
		log.Err(err).Str("pushService", host).Msg("web-push-send")
		return
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		// The browser dropped the subscription.
		if err := s.queries.DeletePushSubscription(ctx, sub.Endpoint); err != nil {
			log.Err(err).Str("pushService", host).Msg("web-push-delete-subscription")
		}
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		log.Error().Int("status", resp.StatusCode).Str("pushService", host).Msg("web-push-refused")
	}
}

// subscriber is the subject as the push library wants it: the library adds
// "mailto:" to anything that is not an https URL.
func (s *Sender) subscriber() string {
	if address, ok := strings.CutPrefix(s.settings.Subject, "mailto:"); ok {
		return address
	}
	return s.settings.Subject
}

// validKeys checks a subscription's keys as the push library will read
// them: p256dh a P-256 public key, auth the 16-byte secret of RFC 8291.
func validKeys(p256dh, auth string) bool {
	k, err := decodeSubscriptionKey(p256dh)
	if err != nil {
		return false
	}
	if _, err := ecdh.P256().NewPublicKey(k); err != nil {
		return false
	}
	a, err := decodeSubscriptionKey(auth)
	return err == nil && len(a) == 16
}

// decodeSubscriptionKey reads standard or URL base64, padded or not, as the
// push library does.
func decodeSubscriptionKey(key string) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(key)
	if rem := len(key) % 4; rem != 0 {
		b.WriteString(strings.Repeat("=", 4-rem))
	}
	if k, err := base64.StdEncoding.DecodeString(b.String()); err == nil {
		return k, nil
	}
	return base64.URLEncoding.DecodeString(b.String())
}
