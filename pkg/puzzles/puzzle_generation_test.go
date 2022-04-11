package puzzles

import (
	"context"
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/rpc/api/proto/puzzle_service"
	pb "github.com/domino14/liwords/rpc/api/proto/puzzle_service"
	"github.com/matryer/is"
)

var DefaultPuzzleGenerationJobRequest = &pb.PuzzleGenerationJobRequest{
	BotVsBot:               true,
	Lexicon:                "CSW21",
	LetterDistribution:     "english",
	SqlOffset:              0,
	GameConsiderationLimit: 1000,
	GameCreationLimit:      100,
	Request: &macondopb.PuzzleGenerationRequest{
		Buckets: []*macondopb.PuzzleBucket{
			{
				Size:     1000,
				Includes: []macondopb.PuzzleTag{macondopb.PuzzleTag_EQUITY},
				Excludes: []macondopb.PuzzleTag{},
			},
		},
	},
}

func TestPuzzleGeneration(t *testing.T) {
	is := is.New(t)
	db, ps, us, gs, _, _ := RecreateDB()
	pgrjReq := proto.Clone(DefaultPuzzleGenerationJobRequest).(*puzzle_service.PuzzleGenerationJobRequest)
	cfg := &config.Config{}
	// Only load config from environment variables:
	cfg.Load(nil)
	cfg.MacondoConfig.DefaultLexicon = pgrjReq.Lexicon
	cfg.MacondoConfig.DefaultLetterDistribution = pgrjReq.LetterDistribution
	ctx := context.Background()
	err := Generate(ctx, cfg, db, gs, ps, pgrjReq, true)
	is.NoErr(err)

	us.Disconnect()
	gs.Disconnect()
	ps.Disconnect()
	db.Close()
}
