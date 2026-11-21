// Package chat holds chat history in Postgres. Chat is moving there from
// Redis streams; during the move, Mirror keeps Redis as the store every read
// comes from and copies each write to Postgres, and Backfill copies what
// Redis already holds.
package chat

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Mirror is a chat store that serves everything from another store (Redis)
// and copies each new and deleted message to Postgres. A failed copy is
// logged, not returned: Redis is still the record, and Backfill fills any
// gap.
type Mirror struct {
	user.ChatStore
	q *models.Queries
}

func NewMirror(primary user.ChatStore, q *models.Queries) *Mirror {
	return &Mirror{ChatStore: primary, q: q}
}

func (m *Mirror) AddChat(ctx context.Context, senderUsername, senderUID, msg,
	channel, channelFriendly string, regulateChat bool) (*pb.ChatMessage, error) {

	cm, err := m.ChatStore.AddChat(ctx, senderUsername, senderUID, msg, channel, channelFriendly, regulateChat)
	if err != nil {
		return nil, err
	}
	lid, err := parseLegacyID(cm.Id)
	if err == nil {
		_, err = m.q.AddChatMessages(ctx, messagesParams([]legacyMessage{{
			channel:  channel,
			senderID: senderUID,
			username: senderUsername,
			message:  msg,
			id:       lid,
			legacyID: cm.Id,
		}}))
	}
	if err != nil {
		log.Err(err).Str("channel", channel).Str("id", cm.Id).Msg("chat-mirror-add-failed")
	}
	return cm, nil
}

func (m *Mirror) DeleteChat(ctx context.Context, channel, msgID, deletedBy string) error {
	if err := m.ChatStore.DeleteChat(ctx, channel, msgID, deletedBy); err != nil {
		return err
	}
	_, err := m.q.DeleteChatMessageByLegacyID(ctx, models.DeleteChatMessageByLegacyIDParams{
		Channel:       channel,
		LegacyID:      pgtype.Text{String: msgID, Valid: true},
		DeletedByUuid: deletedBy,
	})
	if err != nil {
		log.Err(err).Str("channel", channel).Str("id", msgID).Msg("chat-mirror-delete-failed")
	}
	return nil
}

// A legacyMessage is a message as Redis holds it.
type legacyMessage struct {
	channel  string
	senderID string // UUID; "" for server lines and very old messages
	username string
	message  string
	id       legacyID
	legacyID string // the Redis stream ID, as Redis gave it
}

func messagesParams(msgs []legacyMessage) models.AddChatMessagesParams {
	p := models.AddChatMessagesParams{
		Ids:             make([]uuid.UUID, len(msgs)),
		Channels:        make([]string, len(msgs)),
		SenderUuids:     make([]string, len(msgs)),
		SenderUsernames: make([]string, len(msgs)),
		Messages:        make([]string, len(msgs)),
		CreatedAts:      make([]pgtype.Timestamptz, len(msgs)),
		LegacyIds:       make([]string, len(msgs)),
	}
	for i, m := range msgs {
		p.Ids[i] = m.id.messageID()
		p.Channels[i] = m.channel
		p.SenderUuids[i] = m.senderID
		p.SenderUsernames[i] = m.username
		p.Messages[i] = m.message
		p.CreatedAts[i] = pgtype.Timestamptz{Time: m.id.time(), Valid: true}
		p.LegacyIds[i] = m.legacyID
	}
	return p
}
