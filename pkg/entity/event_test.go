package entity

import (
	"testing"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/matryer/is"
	"google.golang.org/protobuf/proto"
)

func TestDeserialize(t *testing.T) {
	is := is.New(t)

	for _, protocol := range []string{"proto", "json"} {

		w := WrapEvent(&pb.ClientGameplayEvent{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			GameId:         "foo123",
			PositionCoords: "G3",
			Tiles:          "ALMIBAR",
		}, pb.MessageType_CLIENT_GAMEPLAY_EVENT, "foo")
		w.SetSerializationProtocol(protocol)

		arr, err := w.Serialize()
		is.NoErr(err)

		ew, err := EventFromByteArray(arr)
		is.NoErr(err)
		is.Equal(w.Type, ew.Type)
		is.True(proto.Equal(w.Event, ew.Event))
	}
}
