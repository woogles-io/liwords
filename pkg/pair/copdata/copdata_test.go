package copdata_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/matryer/is"
	pkgcopdata "github.com/woogles-io/liwords/pkg/pair/copdata"
	pairtestutils "github.com/woogles-io/liwords/pkg/pair/testutils"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestCOPPrecompData(t *testing.T) {
	is := is.New(t)

	var logsb strings.Builder
	var req *ipc.PairRequest
	var copdata *pkgcopdata.PrecompData

	// Empty division
	req = pairtestutils.CreateDefaultPairRequest()
	copdata = pkgcopdata.GetPrecompData(req, &logsb)
	for _, count := range copdata.PairingCounts {
		is.Equal(count, 0)
	}
	for _, count := range copdata.RepeatCounts {
		is.Equal(count, 0)
	}
	for _, rank := range copdata.HighestRankAbsolutely {
		is.Equal(rank, 0)
	}
	for _, rank := range copdata.HighestRankHopefully {
		is.Equal(rank, 0)
	}
	for _, group := range copdata.GibsonGroups {
		is.Equal(group, 0)
	}
	is.Equal(copdata.HighestControlLossRankIdx, -1)

	// Need to test:
	// 1st gibsonized
	// 3rd gibsonized with cl
	// 1st and 4th gibsonized
	// 1sta and 4th and 8th gibsonized
	// no one gibsonized, no cl,
	// no one gibsonized, cl, no dc
	// no one gibsonized, cl, dc
	// no improvement to factor

	req = pairtestutils.CreateDefaultOddPairRequest()
	_ = pkgcopdata.GetPrecompData(req, &logsb)
	req = pairtestutils.CreateLakeGeorgeAfterRound13PairRequest()
	_ = pkgcopdata.GetPrecompData(req, &logsb)
	req = pairtestutils.CreateAlbanyCSWAfterRound24PairRequest()
	_ = pkgcopdata.GetPrecompData(req, &logsb)
	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	_ = pkgcopdata.GetPrecompData(req, &logsb)
	req = pairtestutils.CreateAlbany4thGibsonizedAfterRound25PairRequest()
	_ = pkgcopdata.GetPrecompData(req, &logsb)
	req = pairtestutils.CreateAlbany1stAnd4thGibsonizedAfterRound25PairRequest()
	_ = pkgcopdata.GetPrecompData(req, &logsb)
	req = pairtestutils.CreateAlbany1stAnd4thAnd8thGibsonizedAfterRound25PairRequest()
	_ = pkgcopdata.GetPrecompData(req, &logsb)
	req = pairtestutils.CreateAlbanyjuly4th2024AfterRound21PairRequest()
	_ = pkgcopdata.GetPrecompData(req, &logsb)
	req = pairtestutils.CreateBellevilleCSWAfterRound12PairRequest()
	_ = pkgcopdata.GetPrecompData(req, &logsb)

	fmt.Println(logsb.String())
}
