// Package chat stores chat in Postgres: messages (chat_messages), kept
// indefinitely and soft-deleted by moderators, and how far each user has
// read each channel (chat_reads). A private chat's two people are its
// members; other channels belong to their pages and have no members.
package chat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	upb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

const (
	pmPrefix = "chat.pm."

	// A regulated message must come this long after the sender's previous
	// one, and must not repeat it within repeatWindow.
	messageCooldown = 5 * time.Second
	repeatWindow    = 30 * time.Minute

	defaultPage = 100
)

var (
	ErrTooFast  = errors.New("you cannot send messages that quickly")
	ErrRepeated = errors.New("you cannot send a message identical to the previous one")
)

// DBStore is the chat store.
type DBStore struct {
	pool      *pgxpool.Pool
	q         *models.Queries
	eventChan chan *entity.EventWrapper
}

var _ user.ChatStore = (*DBStore)(nil)

func NewDBStore(pool *pgxpool.Pool) *DBStore {
	return &DBStore{pool: pool, q: models.New(pool)}
}

func (s *DBStore) SetEventChan(c chan *entity.EventWrapper) { s.eventChan = c }
func (s *DBStore) EventChan() chan *entity.EventWrapper     { return s.eventChan }

// ParseMessageID reads a message ID: a UUID, or a Redis stream ID that a
// client loaded before chat moved to Postgres.
func ParseMessageID(channel, s string) (uuid.UUID, error) {
	if id, err := uuid.Parse(s); err == nil {
		return id, nil
	}
	lid, err := parseLegacyID(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("bad message ID %q", s)
	}
	return lid.messageID(channel), nil
}

// PrivateChatUsers returns the UUIDs of a private chat's two people.
func PrivateChatUsers(channel string) ([]string, bool) {
	if !strings.HasPrefix(channel, pmPrefix) {
		return nil, false
	}
	users := strings.Split(strings.TrimPrefix(channel, pmPrefix), "_")
	if len(users) != 2 || users[0] == "" || users[1] == "" {
		return nil, false
	}
	return users, true
}

func (s *DBStore) AddChat(ctx context.Context, senderUsername, senderUID, msg, channel string,
	regulateChat bool) (*pb.ChatMessage, error) {

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	sec, nsec := id.Time().UnixTime()
	at := time.Unix(sec, nsec)

	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if regulateChat && senderUID != "" {
			if err := q.LockChatSender(ctx, senderUID); err != nil {
				return err
			}
			last, err := q.LastRegulatedChat(ctx, senderUID)
			if err == nil {
				since := at.Sub(last.CreatedAt.Time)
				if since < repeatWindow && last.Message == msg {
					return ErrRepeated
				}
				if since < messageCooldown {
					return ErrTooFast
				}
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}
		_, err := q.AddChatMessages(ctx, models.AddChatMessagesParams{
			Ids:             []uuid.UUID{id},
			Channels:        []string{channel},
			SenderUuids:     []string{senderUID},
			SenderUsernames: []string{senderUsername},
			Messages:        []string{msg},
			CreatedAts:      []pgtype.Timestamptz{{Time: at, Valid: true}},
		})
		if err != nil {
			return err
		}
		if users, ok := PrivateChatUsers(channel); ok {
			if err := q.AddChatMembers(ctx, models.AddChatMembersParams{Channel: channel, UserUuids: users}); err != nil {
				return err
			}
		}
		if senderUID != "" {
			// The sender has read what they sent.
			_, err = q.MarkChatRead(ctx, models.MarkChatReadParams{UserUuid: senderUID, Channel: channel, MessageID: pgtype.UUID{Bytes: id, Valid: true}})
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return &pb.ChatMessage{
		Username:  senderUsername,
		UserId:    senderUID,
		Channel:   channel,
		Message:   msg,
		Timestamp: at.UnixMilli(),
		Id:        id.String(),
	}, nil
}

func (s *DBStore) OldChats(ctx context.Context, channel string, n int) ([]*pb.ChatMessage, error) {
	page, err := s.History(ctx, channel, user.ChatQuery{Limit: n})
	if err != nil {
		return nil, err
	}
	return page.Messages, nil
}

type chatRow struct {
	ID             uuid.UUID
	SenderUsername string
	Message        string
	CreatedAt      pgtype.Timestamptz
	SenderUuid     string
}

func toMessage(channel string, r chatRow) *pb.ChatMessage {
	return &pb.ChatMessage{
		Username:  r.SenderUsername,
		UserId:    r.SenderUuid,
		Channel:   channel,
		Message:   r.Message,
		Timestamp: r.CreatedAt.Time.UnixMilli(),
		Id:        r.ID.String(),
	}
}

// older returns up to n messages older than before (or the newest, if
// before is nil), oldest first, and whether there are more.
func (s *DBStore) older(ctx context.Context, channel string, before *uuid.UUID, n int) ([]*pb.ChatMessage, bool, error) {
	p := models.ChatMessagesBeforeParams{Channel: channel, Lim: int32(n + 1)}
	if before != nil {
		p.Before = pgtype.UUID{Bytes: *before, Valid: true}
	}
	rows, err := s.q.ChatMessagesBefore(ctx, p)
	if err != nil {
		return nil, false, err
	}
	more := len(rows) > n
	if more {
		rows = rows[:n]
	}
	msgs := make([]*pb.ChatMessage, len(rows))
	for i, r := range rows {
		msgs[len(rows)-1-i] = toMessage(channel, chatRow(r))
	}
	return msgs, more, nil
}

// newer returns up to n messages newer than after, oldest first, and
// whether there are more.
func (s *DBStore) newer(ctx context.Context, channel string, after uuid.UUID, n int) ([]*pb.ChatMessage, bool, error) {
	rows, err := s.q.ChatMessagesAfter(ctx, models.ChatMessagesAfterParams{Channel: channel, After: after, Lim: int32(n + 1)})
	if err != nil {
		return nil, false, err
	}
	more := len(rows) > n
	if more {
		rows = rows[:n]
	}
	msgs := make([]*pb.ChatMessage, len(rows))
	for i, r := range rows {
		msgs[i] = toMessage(channel, chatRow(r))
	}
	return msgs, more, nil
}

func (s *DBStore) History(ctx context.Context, channel string, cq user.ChatQuery) (*user.ChatPage, error) {
	limit := cq.Limit
	if limit <= 0 {
		limit = defaultPage
	}
	page := &user.ChatPage{}
	var err error
	switch {
	case cq.Before != "":
		id, perr := ParseMessageID(channel, cq.Before)
		if perr != nil {
			return nil, perr
		}
		page.Messages, page.HasOlder, err = s.older(ctx, channel, &id, limit)
		page.HasNewer = true
	case cq.After != "":
		id, perr := ParseMessageID(channel, cq.After)
		if perr != nil {
			return nil, perr
		}
		page.Messages, page.HasNewer, err = s.newer(ctx, channel, id, limit)
		page.HasOlder = true
	case cq.Around != "":
		id, perr := ParseMessageID(channel, cq.Around)
		if perr != nil {
			return nil, perr
		}
		var older, newer []*pb.ChatMessage
		older, page.HasOlder, err = s.older(ctx, channel, &id, limit/2)
		if err != nil {
			return nil, err
		}
		page.Messages = older
		rest := limit - len(older)
		if m, gerr := s.GetChat(ctx, channel, id.String()); gerr == nil {
			page.Messages = append(page.Messages, m)
			rest--
		}
		newer, page.HasNewer, err = s.newer(ctx, channel, id, max(rest, 0))
		page.Messages = append(page.Messages, newer...)
	case cq.FromStart:
		// Every message ID sorts after the nil UUID.
		page.Messages, page.HasNewer, err = s.newer(ctx, channel, uuid.Nil, limit)
	default:
		page.Messages, page.HasOlder, err = s.older(ctx, channel, nil, limit)
	}
	if err != nil {
		return nil, err
	}
	return page, nil
}

func (s *DBStore) GetChat(ctx context.Context, channel, msgID string) (*pb.ChatMessage, error) {
	id, err := ParseMessageID(channel, msgID)
	if err != nil {
		return nil, err
	}
	r, err := s.q.GetChatMessage(ctx, models.GetChatMessageParams{ID: id, Channel: channel})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("no such message id")
	} else if err != nil {
		return nil, err
	}
	return toMessage(channel, chatRow(r)), nil
}

func (s *DBStore) DeleteChat(ctx context.Context, channel, msgID, deletedBy string) error {
	id, err := ParseMessageID(channel, msgID)
	if err != nil {
		return err
	}
	n, err := s.q.DeleteChatMessage(ctx, models.DeleteChatMessageParams{ID: id, Channel: channel, DeletedByUuid: deletedBy})
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("zero chats deleted")
	}
	return nil
}

func (s *DBStore) MarkRead(ctx context.Context, uid, channel, msgID string) error {
	id, err := ParseMessageID(channel, msgID)
	if err != nil {
		return err
	}
	_, err = s.q.MarkChatRead(ctx, models.MarkChatReadParams{UserUuid: uid, Channel: channel, MessageID: pgtype.UUID{Bytes: id, Valid: true}})
	return err
}

func (s *DBStore) LastRead(ctx context.Context, uid, channel string) (string, bool, error) {
	id, err := s.q.GetChatRead(ctx, models.GetChatReadParams{UserUuid: uid, Channel: channel})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	} else if err != nil {
		return "", false, err
	} else if !id.Valid {
		return "", true, nil
	}
	return uuid.UUID(id.Bytes).String(), true, nil
}

func (s *DBStore) LatestChannels(ctx context.Context, count int, before, uid string) (*upb.ActiveChatChannels, error) {
	p := models.ListPrivateChatsParams{UserUuid: uid, Lim: int32(count + 1)}
	if before != "" {
		id, err := uuid.Parse(before)
		if err != nil {
			return nil, fmt.Errorf("bad chat ID %q", before)
		}
		p.BeforeID = pgtype.UUID{Bytes: id, Valid: true}
	}
	rows, err := s.q.ListPrivateChats(ctx, p)
	if err != nil {
		return nil, err
	}
	resp := &upb.ActiveChatChannels{}
	if len(rows) > count {
		rows, resp.HasMore = rows[:count], true
	}
	resp.Channels = make([]*upb.ActiveChatChannels_Channel, len(rows))
	for i, r := range rows {
		names := slices.Clone(r.Usernames)
		slices.Sort(names)
		unread := !r.LastFromUser &&
			(!r.LastReadMessageID.Valid || bytes.Compare(r.LastID[:], r.LastReadMessageID.Bytes[:]) > 0)
		resp.Channels[i] = &upb.ActiveChatChannels_Channel{
			Name:          r.Channel,
			DisplayName:   "pm:" + strings.Join(names, ":"),
			LastUpdate:    r.LastAt.Time.Unix(),
			LastMessage:   r.LastMessage,
			HasUpdate:     unread,
			LastMessageId: r.LastID.String(),
		}
	}
	if before == "" {
		resp.UnreadChannels, err = s.q.ListUnreadPrivateChats(ctx, models.ListUnreadPrivateChatsParams{UserUuid: uid, Lim: maxUnreadChannels})
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// maxUnreadChannels is the most unread private chats LatestChannels lists.
const maxUnreadChannels = 100

// ScanChannels returns up to count channels whose names start with prefix
// (all chat channels if it is ""), in name order from cursor, and the cursor
// for the next call: "0" when there are no more. An empty or "0" cursor
// starts from the beginning.
func (s *DBStore) ScanChannels(ctx context.Context, cursor, prefix string, count int) ([]string, string, error) {
	if cursor == "0" {
		cursor = ""
	}
	if prefix == "" {
		prefix = "chat."
	}
	chans, err := s.q.ScanChatChannels(ctx, models.ScanChatChannelsParams{Prefix: prefix, After: cursor, Lim: int32(count)})
	if err != nil {
		return nil, "", err
	}
	next := "0"
	if len(chans) == count && count > 0 {
		next = chans[len(chans)-1]
	}
	return chans, next, nil
}
