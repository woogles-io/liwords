package analysis

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	pb "github.com/woogles-io/liwords/rpc/api/proto/analysis_service"
)

func TestGamesAnalysisStatusCap(t *testing.T) {
	s := &AnalysisService{}
	_, err := s.GetGamesAnalysisStatus(context.Background(), connect.NewRequest(&pb.GetGamesAnalysisStatusRequest{
		GameIds: make([]string, maxAnalysisStatusGames+1),
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("got %v", err)
	}
}
