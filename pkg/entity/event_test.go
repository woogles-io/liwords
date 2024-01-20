package entity

import (
	"testing"

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
