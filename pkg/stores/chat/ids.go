package chat

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Message IDs are UUIDv7s: 48 bits of Unix milliseconds, then randomness.
// Ordering by ID is ordering by time, so an ID alone marks a place in a
// channel's history.

// legacyID is a Redis stream ID, "<ms>-<seq>".
type legacyID struct {
	ms  int64
	seq uint64
}

func parseLegacyID(s string) (legacyID, error) {
	msPart, seqPart, ok := strings.Cut(s, "-")
	if !ok {
		return legacyID{}, fmt.Errorf("bad stream ID %q", s)
	}
	ms, err := strconv.ParseInt(msPart, 10, 64)
	if err != nil {
		return legacyID{}, fmt.Errorf("bad stream ID %q: %w", s, err)
	}
	seq, err := strconv.ParseUint(seqPart, 10, 64)
	if err != nil {
		return legacyID{}, fmt.Errorf("bad stream ID %q: %w", s, err)
	}
	return legacyID{ms: ms, seq: seq}, nil
}

func (l legacyID) time() time.Time { return time.UnixMilli(l.ms) }

// messageID is the ID of a message first stored in Redis in channel: a
// UUIDv7 with the message's Redis time and, in the 12 bits UUIDv7 leaves for
// ordering within a millisecond, its stream sequence number, so messages
// copied at any time sort where they were sent, in Redis's order. The other
// 62 bits come from a hash of the channel and stream ID: a message always
// gets the same ID, so copying it twice is a no-op, and its Redis ID finds
// it. (A stream ID is unique within its channel.)
func (l legacyID) messageID(channel string) uuid.UUID {
	var u uuid.UUID
	var ms [8]byte
	binary.BigEndian.PutUint64(ms[:], uint64(l.ms))
	copy(u[0:6], ms[2:8])
	seq := l.seq
	if seq > 0xfff {
		seq = 0xfff
	}
	u[6] = 0x70 | byte(seq>>8) // version 7
	u[7] = byte(seq)
	h := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d-%d", channel, l.ms, l.seq)))
	copy(u[8:], h[:8])
	u[8] = u[8]&0x3f | 0x80 // RFC 9562 variant
	return u
}
