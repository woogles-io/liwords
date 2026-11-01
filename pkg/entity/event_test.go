package entity

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/proto"
)

func TestDeserialize(t *testing.T) {
	is := is.New(t)

	for _, protocol := range []SerializationProtocol{EvtSerializationProtoWithHeader,
		EvtSerializationJSONWithHeader} {

		w := WrapEvent(&pb.ClientGameplayEvent{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         "foo123",
			PositionCoords: "G3",
			MachineLetters: []byte{1, 12, 13, 9, 2, 1, 18},
		}, pb.MessageType_CLIENT_GAMEPLAY_EVENT)
		w.SetSerializationProtocol(protocol)

		arr, err := w.Serialize()
		is.NoErr(err)

		ew, err := EventFromByteArray(arr)
		is.NoErr(err)
		is.Equal(w.Type, ew.Type)
		is.True(proto.Equal(w.Event, ew.Event))
	}
}

// frameWithHeader builds a frame by hand, in the two-byte form or the extended
// one, so readers are tested against the format rather than against Serialize.
func frameWithHeader(msgType pb.MessageType, body []byte, extended bool) []byte {
	var b bytes.Buffer
	if extended {
		binary.Write(&b, binary.BigEndian, uint16(ExtendedFrameLength))
		b.WriteByte(byte(msgType))
		binary.Write(&b, binary.BigEndian, uint32(len(body)+1))
	} else {
		binary.Write(&b, binary.BigEndian, uint16(len(body)+1))
		b.WriteByte(byte(msgType))
	}
	b.Write(body)
	return b.Bytes()
}

// A message too big for the two-byte length, in the extended form.
func TestDeserializeExtendedFrame(t *testing.T) {
	is := is.New(t)
	evt := &pb.GameHistoryRefresher{
		History: &macondopb.GameHistory{Uid: strings.Repeat("x", 100_000)},
	}
	body, err := proto.Marshal(evt)
	is.NoErr(err)
	is.True(len(body) > 0xFFFF)

	ew, err := EventFromByteArray(frameWithHeader(pb.MessageType_GAME_HISTORY_REFRESHER, body, true))
	is.NoErr(err)
	is.Equal(ew.Type, pb.MessageType_GAME_HISTORY_REFRESHER)
	is.True(proto.Equal(ew.Event, evt))
}

// Lengths from 32,768 to 65,534 fit two bytes but not a signed int16, which is
// what the length used to be read as. Making a slice of the negative result
// panicked.
func TestDeserializeFrameOver32K(t *testing.T) {
	is := is.New(t)
	evt := &pb.GameHistoryRefresher{
		History: &macondopb.GameHistory{Uid: strings.Repeat("x", 40_000)},
	}
	body, err := proto.Marshal(evt)
	is.NoErr(err)
	is.True(len(body) > 0x7FFF && len(body) < 0xFFFF)

	ew, err := EventFromByteArray(frameWithHeader(pb.MessageType_GAME_HISTORY_REFRESHER, body, false))
	is.NoErr(err)
	is.True(proto.Equal(ew.Event, evt))
}

func TestDeserializeTruncatedFrame(t *testing.T) {
	is := is.New(t)
	body, err := proto.Marshal(&pb.TimedOut{GameId: "abc", UserId: "def"})
	is.NoErr(err)
	for _, extended := range []bool{false, true} {
		frame := frameWithHeader(pb.MessageType_TIMED_OUT, body, extended)
		_, err = EventFromByteArray(frame[:len(frame)-2])
		is.True(err != nil)
	}
	_, err = EventFromByteArray([]byte{0xFF})
	is.True(err != nil)
}

// Serialize writes the two-byte header whenever the length fits, byte for byte
// what it wrote before the extended form existed, and the extended header
// only when it doesn't.
func TestSerializeHeaderForm(t *testing.T) {
	is := is.New(t)
	for _, tc := range []struct {
		uidLen   int
		extended bool
	}{
		{10, false},
		{40_000, false},
		{65_000, false},
		{70_000, true},
		{1_000_000, true},
	} {
		evt := &pb.GameHistoryRefresher{
			History: &macondopb.GameHistory{Uid: strings.Repeat("x", tc.uidLen)},
		}
		body, err := proto.Marshal(evt)
		is.NoErr(err)
		w := WrapEvent(evt, pb.MessageType_GAME_HISTORY_REFRESHER)
		arr, err := w.Serialize()
		is.NoErr(err)

		is.Equal(arr, frameWithHeader(pb.MessageType_GAME_HISTORY_REFRESHER, body, tc.extended))
		ew, err := EventFromByteArray(arr)
		is.NoErr(err)
		is.True(proto.Equal(ew.Event, evt))
	}
}

// The boundary: 65,533 bytes is the last length the two-byte form holds.
func TestFrameHeaderBoundary(t *testing.T) {
	is := is.New(t)
	for n, extended := range map[int]bool{65_533: false, 65_534: true} {
		var b bytes.Buffer
		writeFrameHeader(&b, n, 24)
		if extended {
			is.Equal(b.Bytes()[:2], []byte{0xFF, 0xFF})
			is.Equal(b.Len(), 7)
		} else {
			is.Equal(b.Len(), 3)
		}
		hdr := b.Bytes()
		msgType, got, err := readFrameHeader(bytes.NewReader(hdr))
		is.NoErr(err)
		is.Equal(msgType, byte(24))
		is.Equal(got, n)
	}
}
