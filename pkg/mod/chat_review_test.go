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

	_, ok = summarizeChannel("chat.game.g1", msgs, "u1", "Cesar")
	is.True(ok)
	_, ok = summarizeChannel("chat.game.g1", msgs, "u3", "josh")
	is.True(!ok)
	// a private chat names both parties, even one who never wrote.
	_, ok = summarizeChannel("chat.pm.u2_u3", msgs[1:], "u3", "josh")
	is.True(ok)
}
