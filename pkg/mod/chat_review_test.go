package mod

import (
	"context"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/user"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestSummarizeChannel(t *testing.T) {
	is := is.New(t)
	msgs := []*ipc.ChatMessage{
		{Username: "cesar", UserId: "u1", Timestamp: 1000},
		{Username: "mina", UserId: "u2", Timestamp: 2000},
		{Username: "mina", UserId: "u2", Timestamp: 3000},
	}

	s, ok := summarizeChannel("chat.game.g1", msgs, "", "")
	is.True(ok)
	is.Equal(s.MessageCount, int32(3))
	is.Equal(s.FirstTimestamp, int64(1000))
	is.Equal(s.LastTimestamp, int64(3000))
	is.Equal(len(s.Senders), 2)
	is.Equal(s.Senders[0].Username, "mina")
	is.Equal(s.Senders[0].Count, int32(2))

	is.Equal(s.UserLastTimestamp, int64(0))

	s, ok = summarizeChannel("chat.game.g1", msgs, "u1", "Cesar")
	is.True(ok)
	is.Equal(s.UserLastTimestamp, int64(1000))
	// the user's own last message, not the channel's.
	s, _ = summarizeChannel("chat.game.g1", msgs, "u2", "mina")
	is.Equal(s.UserLastTimestamp, int64(3000))
	s, ok = summarizeChannel("chat.game.g1", msgs, "u3", "josh")
	is.True(!ok)
	is.Equal(s.UserLastTimestamp, int64(0))
	// a private chat names both parties, even one who never wrote.
	_, ok = summarizeChannel("chat.pm.u2_u3", msgs[1:], "u3", "josh")
	is.True(ok)
}

// oldChatsOnly is a chat store that can only list a channel's messages.
type oldChatsOnly struct {
	user.ChatStore
	msgs []*ipc.ChatMessage
}

func (s oldChatsOnly) OldChats(ctx context.Context, channel string, n int) ([]*ipc.ChatMessage, error) {
	return s.msgs, nil
}

func TestReviewChatsLeavesOutServerLines(t *testing.T) {
	is := is.New(t)
	said := []*ipc.ChatMessage{
		{Username: "cesar", UserId: "u1", Message: "gl", Timestamp: 1000},
		{Username: "mina", UserId: "u2", Message: "u2", Timestamp: 3000},
	}
	line := &ipc.ChatMessage{Message: "Play was challenged off the board! Returned tiles: AB", Timestamp: 2000}
	ms := NewModService(nil, oldChatsOnly{msgs: []*ipc.ChatMessage{said[0], line, said[1]}}, nil)

	msgs, err := ms.reviewChats(context.Background(), "chat.game.g1")
	is.NoErr(err)
	is.Equal(msgs, said)

	// a game chat holding nothing else reads as empty, so it is not listed
	ms = NewModService(nil, oldChatsOnly{msgs: []*ipc.ChatMessage{line}}, nil)
	msgs, err = ms.reviewChats(context.Background(), "chat.game.g1")
	is.NoErr(err)
	is.Equal(len(msgs), 0)
}

func TestPagePath(t *testing.T) {
	is := is.New(t)
	for in, want := range map[string]string{
		"https://woogles.io/club/philly/?x=1":  "/club/philly",
		"woogles.io/leagues/smoke#standings":   "/leagues/smoke",
		"/game/abc":                            "/game/abc",
		"/":                                    "/",
		"philly":                               "philly",
		"Philly Mondays":                       "Philly Mondays",
		"8f2c1b3e-0000-4000-8000-000000000000": "8f2c1b3e-0000-4000-8000-000000000000",
	} {
		is.Equal(pagePath(in), want)
	}
}
