package entity

import (
	"testing"

	pb "github.com/domino14/crosswords/rpc/api/proto"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"
	"google.golang.org/protobuf/proto"
)

func TestDeserialize(t *testing.T) {
	is := is.New(t)

	for _, protocol := range []string{"proto", "json"} {

		w := NewEventWrapper("UserGameplayEvent", &pb.UserGameplayEvent{
			Event: &macondopb.GameEvent{
				Nickname:   "cesitar",
				Cumulative: 75,
				Rack:       "ALMIBAR",
				Position:   "G3",
				Type:       macondopb.GameEvent_TILE_PLACEMENT_MOVE,
			},
			NewRack:       "DOGS",
			TimeRemaining: 12345,
		})
		w.SetSerializationProtocol(protocol)

		arr, err := w.Serialize()
		is.NoErr(err)
		ew, err := EventFromByteArray(arr)
		is.NoErr(err)
		is.Equal(w.Name, ew.Name)
		is.True(proto.Equal(w.Event, ew.Event))
	}
}
