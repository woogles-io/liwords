// Package outbox sends emails and Discord posts that were written to the
// outbox table with the change that called for them.
//
// Writing the messages in the same transaction as the change means they are
// sent if and only if the change happened. Sending marks each message as
// soon as it goes out, so a send cut off by a deploy carries on where it
// stopped: nothing is lost, and the most that can go out twice is the one
// message in flight when a server died.
package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

// Channels a message goes out on.
const (
	// ChannelEmail is an email to Recipient.
	ChannelEmail = "email"
	// ChannelLeagueDiscord is a post to the league Discord channel.
	ChannelLeagueDiscord = "league_discord"
)

const (
	// maxAttempts is how many times a message is tried before it is given up.
	maxAttempts = 5
	// keepFor is how long sent and given-up messages stay in the table.
	// Senders must not write the same messages again after that long.
	keepFor = 30 * 24 * time.Hour
	// defaultGap is the wait between messages, as the league email loops
	// had: about one email every 2 s.
	defaultGap = 2 * time.Second
)

// A Message is one email or post. Key says what it is for, such as
// "league:<season>:started:<user>"; adding a message whose key is already
// in the outbox does nothing.
type Message struct {
	Key       string
	Channel   string
	Recipient string
	Subject   string
	Body      string
}

// Add writes msgs to the outbox, skipping any whose key is already there,
// and returns how many it added. Pass a q bound to a transaction to write
// them with the change that calls for them.
func Add(ctx context.Context, q *models.Queries, msgs []Message) (int64, error) {
	if len(msgs) == 0 {
		return 0, nil
	}
	p := models.AddOutboxMessagesParams{}
	for _, m := range msgs {
		if m.Key == "" || m.Channel == "" {
			return 0, fmt.Errorf("outbox: message without a key or channel: %q", m.Key)
		}
		p.DedupeKeys = append(p.DedupeKeys, m.Key)
		p.Channels = append(p.Channels, m.Channel)
		p.Recipients = append(p.Recipients, m.Recipient)
		p.Subjects = append(p.Subjects, m.Subject)
		p.Bodies = append(p.Bodies, m.Body)
	}
	return q.AddOutboxMessages(ctx, p)
}

// A SendFunc sends one message on its channel.
type SendFunc func(ctx context.Context, m Message) error

// A Sender sends what is in the outbox.
type Sender struct {
	q        *models.Queries
	channels map[string]SendFunc
	gap      time.Duration
}

// NewSender sends on the given channels, waiting gap between messages (0
// for the default, 2 s). A message on a channel not in channels fails, and
// is retried by a later run.
func NewSender(q *models.Queries, channels map[string]SendFunc, gap time.Duration) *Sender {
	if gap == 0 {
		gap = defaultGap
	}
	return &Sender{q: q, channels: channels, gap: gap}
}

// Send sends every unsent message, oldest first, marking each as it goes
// out. A message that fails is tried again by the next run, up to
// maxAttempts. It returns when there is nothing left to send, or ctx is
// done; it returns an error if any message failed.
func (s *Sender) Send(ctx context.Context) error {
	if n, err := s.q.DeleteOldOutboxMessages(ctx, models.DeleteOldOutboxMessagesParams{
		Before:      pgtype.Timestamptz{Time: time.Now().Add(-keepFor), Valid: true},
		MaxAttempts: maxAttempts,
	}); err != nil {
		return fmt.Errorf("delete old messages: %w", err)
	} else if n > 0 {
		log.Ctx(ctx).Info().Int64("deleted", n).Msg("outbox-deleted-old")
	}

	var after int64
	sent, failed := 0, 0
	for {
		row, err := s.q.NextOutboxMessage(ctx, models.NextOutboxMessageParams{
			AfterID: after, MaxAttempts: maxAttempts,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			break
		}
		if err != nil {
			return err
		}
		// A message that fails waits for the next run, rather than being
		// tried again at once.
		after = row.ID
		if sent+failed > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.gap):
			}
		}
		m := Message{Key: row.DedupeKey, Channel: row.Channel, Recipient: row.Recipient,
			Subject: row.Subject, Body: row.Body}
		sendErr := s.send(ctx, m)
		// Recorded even if ctx was cancelled meanwhile: a message that went
		// out and isn't marked goes out again.
		recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		if sendErr != nil {
			failed++
			log.Ctx(ctx).Err(sendErr).Str("key", m.Key).Int32("attempt", row.Attempts+1).Msg("outbox-send-failed")
			err = s.q.RecordOutboxMessageFailure(recordCtx, models.RecordOutboxMessageFailureParams{
				ID: row.ID, LastError: sendErr.Error(),
			})
		} else {
			sent++
			err = s.q.MarkOutboxMessageSent(recordCtx, row.ID)
		}
		cancel()
		if err != nil {
			return fmt.Errorf("record message %d: %w", row.ID, err)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	if sent+failed > 0 {
		log.Ctx(ctx).Info().Int("sent", sent).Int("failed", failed).Msg("outbox-sent")
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d messages failed", failed, sent+failed)
	}
	return nil
}

func (s *Sender) send(ctx context.Context, m Message) error {
	f, ok := s.channels[m.Channel]
	if !ok {
		return fmt.Errorf("no sender for channel %q", m.Channel)
	}
	return f(ctx, m)
}
