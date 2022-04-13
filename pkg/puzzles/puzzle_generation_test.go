package puzzles

import (
	"context"
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/liwords/pkg/config"
	pb "github.com/domino14/liwords/rpc/api/proto/puzzle_service"
	"github.com/matryer/is"
)

var DefaultPuzzleGenerationJobRequest = &pb.PuzzleGenerationJobRequest{
	BotVsBot:               true,
	Lexicon:                "CSW21",
	LetterDistribution:     "english",
	SqlOffset:              0,
	GameConsiderationLimit: 1000000,
	GameCreationLimit:      100,
	Request: &macondopb.PuzzleGenerationRequest{
		Buckets: []*macondopb.PuzzleBucket{
			{
				Size:     50,
				Includes: []macondopb.PuzzleTag{macondopb.PuzzleTag_EQUITY},
				Excludes: []macondopb.PuzzleTag{},
			},
		},
	},
}

func TestPuzzleGeneration(t *testing.T) {
	is := is.New(t)
	db, ps, us, gs, _, _ := RecreateDB()
	cfg := &config.Config{}
	cfg.Load(nil)
	cfg.MacondoConfig.DefaultLexicon = DefaultPuzzleGenerationJobRequest.Lexicon
	cfg.MacondoConfig.DefaultLetterDistribution = DefaultPuzzleGenerationJobRequest.LetterDistribution
	ctx := context.Background()

	// A fulfilled request
	pgrjReq := proto.Clone(DefaultPuzzleGenerationJobRequest).(*pb.PuzzleGenerationJobRequest)
	genId, err := Generate(ctx, cfg, db, gs, ps, pgrjReq)
	is.NoErr(err)

	_, _, _, fulfilledOption, errorStatusOption, totalPuzzles, _, _, err := GetJobInfo(ctx, ps, genId)
	is.NoErr(err)
	is.True(*fulfilledOption)
	is.Equal(errorStatusOption, nil)
	is.Equal(totalPuzzles, 50)

	// An unfulfilled request
	pgrjReq = proto.Clone(DefaultPuzzleGenerationJobRequest).(*pb.PuzzleGenerationJobRequest)
	pgrjReq.GameCreationLimit = 10
	genId, err = Generate(ctx, cfg, db, gs, ps, pgrjReq)
	is.NoErr(err)
	_, _, _, fulfilledOption, errorStatusOption, _, totalGames, _, err := GetJobInfo(ctx, ps, genId)
	is.NoErr(err)
	is.True(!*fulfilledOption)
	is.Equal(errorStatusOption, nil)
	is.Equal(totalGames, 10)

	// An error
	pgrjReq = proto.Clone(DefaultPuzzleGenerationJobRequest).(*pb.PuzzleGenerationJobRequest)
	pgrjReq.Request = nil
	genId, err = Generate(ctx, cfg, db, gs, ps, pgrjReq)
	is.NoErr(err)
	_, _, _, fulfilledOption, errorStatusOption, totalPuzzles, totalGames, _, err = GetJobInfo(ctx, ps, genId)
	is.NoErr(err)
	is.Equal(*fulfilledOption, false)
	is.Equal(*errorStatusOption, "puzzle generation request is nil")
	is.Equal(totalGames, 0)
	is.Equal(totalPuzzles, 0)

	// Multiple buckets, request fulfilled
	pgrjReq = proto.Clone(DefaultPuzzleGenerationJobRequest).(*pb.PuzzleGenerationJobRequest)
	pgrjReq.GameCreationLimit = 1000
	pgrjReq.Request.Buckets = append(pgrjReq.Request.Buckets, []*macondopb.PuzzleBucket{
		{
			Size:     50,
			Includes: []macondopb.PuzzleTag{macondopb.PuzzleTag_BINGO},
			Excludes: []macondopb.PuzzleTag{},
		},
		{
			Size:     50,
			Includes: []macondopb.PuzzleTag{macondopb.PuzzleTag_CEL_ONLY},
			Excludes: []macondopb.PuzzleTag{},
		},
	}...)
	genId, err = Generate(ctx, cfg, db, gs, ps, pgrjReq)
	is.NoErr(err)
	_, _, _, fulfilledOption, errorStatusOption, totalPuzzles, _, breakdowns, err := GetJobInfo(ctx, ps, genId)
	is.NoErr(err)
	is.Equal(*fulfilledOption, true)
	is.Equal(errorStatusOption, nil)
	is.Equal(totalPuzzles, 150)
	for _, bd := range breakdowns {
		is.Equal(bd[1], 50)
	}

	pgrjReq = proto.Clone(DefaultPuzzleGenerationJobRequest).(*pb.PuzzleGenerationJobRequest)
	pgrjReq.GameCreationLimit = 100000
	pgrjReq.Request.Buckets = append(pgrjReq.Request.Buckets, []*macondopb.PuzzleBucket{
		{
			Size:     25,
			Includes: []macondopb.PuzzleTag{macondopb.PuzzleTag_BINGO},
			Excludes: []macondopb.PuzzleTag{},
		},
		{
			Size:     10,
			Includes: []macondopb.PuzzleTag{macondopb.PuzzleTag_CEL_ONLY},
			Excludes: []macondopb.PuzzleTag{},
		},
		{
			Size:     50,
			Includes: []macondopb.PuzzleTag{macondopb.PuzzleTag_ONLY_BINGO},
			Excludes: []macondopb.PuzzleTag{},
		},
		{
			Size:     2,
			Includes: []macondopb.PuzzleTag{macondopb.PuzzleTag_POWER_TILE},
			Excludes: []macondopb.PuzzleTag{},
		},
	}...)
	genId, err = Generate(ctx, cfg, db, gs, ps, pgrjReq)
	is.NoErr(err)
	_, _, _, fulfilledOption, errorStatusOption, totalPuzzles, _, breakdowns, err = GetJobInfo(ctx, ps, genId)
	is.NoErr(err)
	is.Equal(*fulfilledOption, true)
	is.Equal(errorStatusOption, nil)
	is.Equal(totalPuzzles, 137)
	is.Equal(breakdowns[0][1], 50)
	is.Equal(breakdowns[1][1], 25)
	is.Equal(breakdowns[2][1], 10)
	is.Equal(breakdowns[3][1], 50)
	is.Equal(breakdowns[4][1], 2)

	// Multitple buckets, request unfulfilled
	pgrjReq = proto.Clone(DefaultPuzzleGenerationJobRequest).(*pb.PuzzleGenerationJobRequest)
	pgrjReq.GameCreationLimit = 20
	pgrjReq.Request.Buckets = append(pgrjReq.Request.Buckets, []*macondopb.PuzzleBucket{
		{
			Size:     25,
			Includes: []macondopb.PuzzleTag{macondopb.PuzzleTag_BINGO},
			Excludes: []macondopb.PuzzleTag{},
		},
		{
			Size:     10,
			Includes: []macondopb.PuzzleTag{macondopb.PuzzleTag_CEL_ONLY},
			Excludes: []macondopb.PuzzleTag{},
		},
		{
			Size:     50,
			Includes: []macondopb.PuzzleTag{macondopb.PuzzleTag_BINGO, macondopb.PuzzleTag_EQUITY},
			Excludes: []macondopb.PuzzleTag{},
		},
		{
			Size:     2,
			Includes: []macondopb.PuzzleTag{macondopb.PuzzleTag_POWER_TILE},
			Excludes: []macondopb.PuzzleTag{},
		},
	}...)
	genId, err = Generate(ctx, cfg, db, gs, ps, pgrjReq)
	is.NoErr(err)
	_, _, _, fulfilledOption, errorStatusOption, _, totalGames, _, err = GetJobInfo(ctx, ps, genId)
	is.NoErr(err)
	is.Equal(*fulfilledOption, false)
	is.Equal(errorStatusOption, nil)
	is.Equal(totalGames, 20)

	us.Disconnect()
	gs.Disconnect()
	ps.Disconnect()
	db.Close()
}
