package chat

import (
	"bytes"
	"testing"

	"github.com/matryer/is"
)

// A message ID made from a Redis stream ID is a valid UUIDv7 carrying the
// stream's time, and IDs sort as the stream entries do: by time, then by
// sequence within a millisecond.
func TestMessageID(t *testing.T) {
	is := is.New(t)
	ids := []string{"1600000000000-0", "1600000000000-1", "1600000000000-7", "1600000000001-0", "1700000000000-0"}
	var prev []byte
	for _, s := range ids {
		l, err := parseLegacyID(s)
		is.NoErr(err)
		for i := 0; i < 20; i++ { // the random tail never reorders them
			u := l.messageID()
			is.Equal(int(u.Version()), 7)
			is.Equal(u.Variant().String(), "RFC4122")
			sec, nsec := u.Time().UnixTime()
			is.Equal(sec*1000+nsec/1e6, l.ms)
			if prev != nil {
				is.True(bytes.Compare(prev, u[:]) < 0)
			}
			if i == 19 {
				prev = append([]byte(nil), u[:]...)
			}
		}
	}

	_, err := parseLegacyID("nonsense")
	is.True(err != nil)
	_, err = parseLegacyID("12-x")
	is.True(err != nil)
}
