package tournament

import (
	"errors"
	"testing"

	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestCOPAnswers(t *testing.T) {
	is := is.New(t)
	calls := 0
	c := &copAnswers{run: func(req *pb.PairRequest) *pb.PairResponse {
		calls++
		return &pb.PairResponse{Pairings: []int32{1, 0}, GibsonizedPlayers: []bool{false, false}}
	}}
	req := &pb.PairRequest{PlayerNames: []string{"a", "b"}, Seed: 7}

	// Without an answer, pairing is deferred and the request kept.
	_, err := c.pair(req)
	is.True(errors.Is(err, errCOPDeferred))
	is.Equal(len(c.missed), 1)
	is.Equal(calls, 0)

	c.answerMissed()
	is.Equal(calls, 1)
	is.Equal(len(c.missed), 0)

	// An equal request gets the answer, as a copy the caller may keep.
	resp, err := c.pair(&pb.PairRequest{PlayerNames: []string{"a", "b"}, Seed: 7})
	is.NoErr(err)
	is.Equal(resp.Pairings, []int32{1, 0})
	resp.GibsonizedPlayers[0] = true
	again, err := c.pair(req)
	is.NoErr(err)
	is.Equal(again.GibsonizedPlayers[0], false)
	is.Equal(calls, 1)

	// A different request misses, unless computing is allowed.
	_, err = c.pair(&pb.PairRequest{PlayerNames: []string{"a", "b"}, Seed: 8})
	is.True(errors.Is(err, errCOPDeferred))
	c.missed = nil
	c.compute = true
	_, err = c.pair(&pb.PairRequest{PlayerNames: []string{"a", "b"}, Seed: 8})
	is.NoErr(err)
	is.Equal(calls, 2)

	// With none attached, COP runs as it always did.
	var none *copAnswers
	resp, err = none.pair(&pb.PairRequest{})
	is.NoErr(err)
	is.True(resp != nil)
}
