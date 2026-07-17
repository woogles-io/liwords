package broadcasts

import (
	"testing"

	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func opponentTestDoc() *ipc.GameDocument {
	return &ipc.GameDocument{
		Players: []*ipc.GameDocument_MinimalPlayerInfo{
			{Nickname: "alice", RealName: "Alice Smith", UserId: "u-alice"},
			{Nickname: "bob", RealName: "Bob Jones", UserId: "u-bob"},
		},
	}
}

func TestOpponentName(t *testing.T) {
	doc := opponentTestDoc()

	t.Run("resolves by user ID, either side", func(t *testing.T) {
		if got := opponentName(doc, "u-alice", "alice"); got != "Bob Jones" {
			t.Errorf("opponentName(tracked=alice) = %q, want %q", got, "Bob Jones")
		}
		if got := opponentName(doc, "u-bob", "bob"); got != "Alice Smith" {
			t.Errorf("opponentName(tracked=bob) = %q, want %q", got, "Alice Smith")
		}
	})

	t.Run("falls back to username match when user ID is absent or unmatched", func(t *testing.T) {
		if got := opponentName(doc, "", "alice"); got != "Bob Jones" {
			t.Errorf("opponentName(no uuid) = %q, want %q", got, "Bob Jones")
		}
		if got := opponentName(doc, "u-someone-else", "bob"); got != "Alice Smith" {
			t.Errorf("opponentName(unmatched uuid) = %q, want %q", got, "Alice Smith")
		}
	})

	t.Run("username match is case-insensitive against RealName or Nickname", func(t *testing.T) {
		if got := opponentName(doc, "", "ALICE"); got != "Bob Jones" {
			t.Errorf("opponentName(uppercase nickname) = %q, want %q", got, "Bob Jones")
		}
		if got := opponentName(doc, "", "bob jones"); got != "Alice Smith" {
			t.Errorf("opponentName(realname match) = %q, want %q", got, "Alice Smith")
		}
	})

	t.Run("no match returns empty string", func(t *testing.T) {
		if got := opponentName(doc, "", "carol"); got != "" {
			t.Errorf("opponentName(no match) = %q, want empty string", got)
		}
	})

	t.Run("non-2-player game returns empty string", func(t *testing.T) {
		single := &ipc.GameDocument{
			Players: []*ipc.GameDocument_MinimalPlayerInfo{
				{Nickname: "alice", UserId: "u-alice"},
			},
		}
		if got := opponentName(single, "u-alice", "alice"); got != "" {
			t.Errorf("opponentName(1 player) = %q, want empty string", got)
		}
	})
}
