package tournament

import pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"

// SetCOPRunForTest makes changes run f as COP, and returns a function that
// puts COP back.
func SetCOPRunForTest(f func(*pb.PairRequest) *pb.PairResponse) func() {
	old := runCOP
	runCOP = f
	return func() { runCOP = old }
}

// LockForChangeWithCOPForTest is lockForChangeWithCOP.
var LockForChangeWithCOPForTest = lockForChangeWithCOP
