package sockets

import (
	"testing"

	"github.com/matryer/is"
)

// The realms a socket registers with are kept, unchanged, for its pongs to
// carry, after the hub has taken its working copy.
func TestRegisterRealmKeepsRealmsForPongs(t *testing.T) {
	is := is.New(t)
	c := &Client{}
	is.NoErr(registerRealm(c, "/", nil))
	is.Equal(c.pongRealms, []string{"lobby", "chat-lobby"})

	c.tempRealms[0] = "changed" // as the hub reuses it
	c.tempRealms = []string{}
	is.Equal(c.pongRealms, []string{"lobby", "chat-lobby"})
}
