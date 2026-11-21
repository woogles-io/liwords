package chat

import (
	"bytes"
	"testing"

	"github.com/matryer/is"
)

// A message ID made from a Redis stream ID is a valid UUIDv7 carrying the
// stream's time, and IDs sort as the stream entries do: by time, then by
// sequence within a millisecond, whatever their hashed tail.
func TestMessageID(t *testing.T) {
	is := is.New(t)
	ids := []string{"1600000000000-0", "1600000000000-1", "1600000000000-7", "1600000000001-0", "1700000000000-0"}
	var prev []byte
	for _, s := range ids {
		l, err := parseLegacyID(s)
		is.NoErr(err)
		u := l.messageID("chat.lobby")
		is.Equal(int(u.Version()), 7)
		is.Equal(u.Variant().String(), "RFC4122")
		sec, nsec := u.Time().UnixTime()
		is.Equal(sec*1000+nsec/1e6, l.ms)
		if prev != nil {
			is.True(bytes.Compare(prev, u[:]) < 0)
		}
		prev = u[:]
	}

	// The same message always gets the same ID; the same stream ID in
	// another channel is another message.
	l, err := parseLegacyID("1700000000000-3")
	is.NoErr(err)
	is.Equal(l.messageID("chat.game.a"), l.messageID("chat.game.a"))
	is.True(l.messageID("chat.game.a") != l.messageID("chat.game.b"))

	_, err = parseLegacyID("nonsense")
	is.True(err != nil)
	_, err = parseLegacyID("12-x")
	is.True(err != nil)
}
