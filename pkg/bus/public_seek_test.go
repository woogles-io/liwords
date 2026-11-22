package bus

import (
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Seeks go out without the connection IDs they were made from and sent to,
// and the stored seek keeps them.
func TestSeeksSentWithoutConnIDs(t *testing.T) {
	is := is.New(t)
	sr := &pb.SeekRequest{SeekerConnectionId: "seekerconn", ReceiverConnectionId: "receiverconn",
		User: &pb.MatchUser{UserId: "u1"}}

	out, err := sanitize(nil, nil, nil, entity.WrapEvent(sr, pb.MessageType_SEEK_REQUEST), "u2")
	is.NoErr(err)
	got := out.Event.(*pb.SeekRequest)
	is.Equal(got.SeekerConnectionId, "")
	is.Equal(got.ReceiverConnectionId, "")
	is.Equal(got.User.UserId, "u1")

	out, err = sanitize(nil, nil, nil, entity.WrapEvent(&pb.SeekRequests{Requests: []*pb.SeekRequest{sr}},
		pb.MessageType_SEEK_REQUESTS), "u2")
	is.NoErr(err)
	is.Equal(out.Event.(*pb.SeekRequests).Requests[0].SeekerConnectionId, "")
	is.Equal(sr.SeekerConnectionId, "seekerconn")
}
