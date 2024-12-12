package cop_lambda

import (
	"context"
	"testing"

	"github.com/matryer/is"
	pairtestutils "github.com/woogles-io/liwords/pkg/pair/testutils"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/proto"
)

func TestHandleRequest(t *testing.T) {
	is := is.New(t)
	req := pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Seed = 1
	req.Rounds = 26
	req.ClassPrizes = []int32{2}
	// Create class B
	req.PlayerClasses[14] = 1
	req.PlayerClasses[13] = 1
	req.PlayerClasses[17] = 1
	req.PlayerClasses[21] = 1
	req.PlayerClasses[23] = 1
	req.PlayerClasses[19] = 1
	req.PlayerClasses[20] = 1
	ctx := context.Background()
	pairRequestByes, err := proto.Marshal(req)
	is.NoErr(err)
	evt := LambdaInvokeInput{
		PairRequestBytes: pairRequestByes,
	}
	JSONResponse, err := HandleRequest(ctx, evt)
	is.NoErr(err)
	pairResponse := &pb.PairResponse{}
	err = proto.Unmarshal([]byte(JSONResponse), pairResponse)
	is.NoErr(err)
	// Expect the normal KOTH casher pairings:
	is.Equal(pairResponse.Pairings[0], int32(1))
	is.Equal(pairResponse.Pairings[1], int32(0))
	is.Equal(pairResponse.Pairings[2], int32(9))
	is.Equal(pairResponse.Pairings[9], int32(2))
	is.Equal(pairResponse.Pairings[12], int32(15))
	is.Equal(pairResponse.Pairings[15], int32(12))
	// Expect class B KOTH pairings for 2 class prizes:
	is.Equal(pairResponse.Pairings[14], int32(13))
	is.Equal(pairResponse.Pairings[13], int32(14))
	is.Equal(pairResponse.Pairings[17], int32(21))
	is.Equal(pairResponse.Pairings[21], int32(17))

}
