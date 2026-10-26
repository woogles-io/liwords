package mod

import (
	"testing"

	"github.com/matryer/is"

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
