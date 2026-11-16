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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

var tracer = otel.Tracer("push")

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
// once; the sending happens in the background, traced as its own push.Send
// trace linked to the caller's span, with attrs on its root span.
func (s *Sender) Notify(ctx context.Context, userUUID string, payload []byte, attrs ...attribute.KeyValue) {
	link := trace.LinkFromContext(ctx)
	go func() {
		ctx, span := tracer.Start(context.Background(), "push.Send",
			trace.WithNewRoot(),
			trace.WithLinks(link),
			trace.WithAttributes(append(attrs, attribute.String("user.id", userUUID))...))
		defer span.End()
		s.send(ctx, span, userUUID, payload)
	}()
}

func (s *Sender) send(ctx context.Context, span trace.Span, userUUID string, payload []byte) {
	subs, err := s.queries.GetPushSubscriptions(ctx, userUUID)
	if err != nil {
		log.Err(err).Str("userID", userUUID).Msg("web-push-get-subscriptions")
		span.RecordError(err)
		span.SetStatus(codes.Error, "get subscriptions failed")
		return
	}
	counts := map[string]int{}
	for _, sub := range subs {
		sendCtx, done := s.startSend(ctx, sub.Endpoint)
		counts[s.sendOne(sendCtx, sub, payload)]++
		done()
	}
	span.SetAttributes(attribute.Int("push.devices", len(subs)))
	for _, o := range []string{outcomeSent, outcomeGone, outcomeRefused, outcomeFailed, outcomeSuperseded} {
		if counts[o] > 0 {
			span.SetAttributes(attribute.Int("push."+o, counts[o]))
		}
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

// recordSize is the size the library pads every encrypted body to. Its
// default, 4096, is over what Mozilla's push service takes: 4028 bytes, or
// 2744 for Firefox on Android, whose pushes it relays through FCM. Over that
// it answers 413, so every Firefox push failed. A turn is far smaller.
const recordSize = 2048

// Outcomes of one send, as the push.outcome span attribute and the counts on
// the push.Send span.
const (
	outcomeSent       = "sent"
	outcomeGone       = "gone"
	outcomeRefused    = "refused"
	outcomeFailed     = "failed"
	outcomeSuperseded = "superseded"
)

func (s *Sender) sendOne(ctx context.Context, sub models.GetPushSubscriptionsRow, payload []byte) string {
	// An endpoint's path is the subscription's secret, so only its host is
	// logged, and traced.
	host := ""
	if u, err := url.Parse(sub.Endpoint); err == nil {
		host = u.Host
	}
	ctx, span := tracer.Start(ctx, "push.SendOne",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("push.service", host)))
	defer span.End()
	outcome := s.post(ctx, span, host, sub, payload)
	span.SetAttributes(attribute.String("push.outcome", outcome))
	return outcome
}

func (s *Sender) post(ctx context.Context, span trace.Span, host string,
	sub models.GetPushSubscriptionsRow, payload []byte) string {
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
			RecordSize:      recordSize,
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
			return outcomeSuperseded
		}
		log.Err(err).Str("pushService", host).Msg("web-push-send")
		span.RecordError(err)
		span.SetStatus(codes.Error, "send failed")
		return outcomeFailed
	}
	defer resp.Body.Close()
	span.SetAttributes(attribute.Int("http.response.status_code", resp.StatusCode))
	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		// The browser dropped the subscription.
		if err := s.queries.DeletePushSubscription(ctx, sub.Endpoint); err != nil {
			log.Err(err).Str("pushService", host).Msg("web-push-delete-subscription")
		}
		return outcomeGone
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		log.Error().Int("status", resp.StatusCode).Str("pushService", host).Msg("web-push-refused")
		span.SetStatus(codes.Error, "push service refused")
		return outcomeRefused
	}
	return outcomeSent
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
