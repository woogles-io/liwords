package puzzles

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/common"
	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	puzzlesstore "github.com/domino14/liwords/pkg/stores/puzzles"
	"github.com/domino14/macondo/cross_set"
	macondogame "github.com/domino14/macondo/game"
	macondopuzzles "github.com/domino14/macondo/puzzles"
	"github.com/domino14/word-golib/kwg"
	"github.com/domino14/word-golib/tilemapping"

	"github.com/domino14/liwords/rpc/api/proto/ipc"
	pb "github.com/domino14/liwords/rpc/api/proto/puzzle_service"

	"github.com/domino14/macondo/automatic"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

var mockBotGameReq *ipc.GameRequest

func init() {
	mockBotGameReq = &ipc.GameRequest{
		Rules: &ipc.GameRules{
			BoardLayoutName: entity.CrosswordGame,
			VariantName:     "classic",
		},
		InitialTimeSeconds: 25 * 60,
		IncrementSeconds:   0,
		ChallengeRule:      macondopb.ChallengeRule_FIVE_POINT,
		GameMode:           ipc.GameMode_REAL_TIME,
		RatingMode:         ipc.RatingMode_RATED,
		RequestId:          "puzzlereq",
		OriginalRequestId:  "puzzlereq",
	}
}

func Generate(ctx context.Context, cfg *config.Config, gs gameplay.GameStore, ps PuzzleStore, req *pb.PuzzleGenerationJobRequest) (int, error) {
	genId, err := ps.CreateGenerationLog(ctx, req)
	if err != nil {
		return -1, err
	}
	fulfilled, err := processJob(ctx, cfg, req, genId, gs, ps)
	return genId, ps.UpdateGenerationLogStatus(ctx, genId, fulfilled, err)
}

func GetJobInfoString(ctx context.Context, ps *puzzlesstore.DBStore, genId int) (string, error) {
	startTime, endTime, dur, fulfilledOption, errorStatusOption, totalPuzzles, totalGames, breakdowns, err := GetJobInfo(ctx, ps, genId)
	if err != nil {
		return "", err
	}

	fo := "incomplete"
	if fulfilledOption != nil {
		if *fulfilledOption {
			fo = "fulfilled"
		} else {
			fo = "unfulfilled"
		}
	}
	eso := "OK"
	if errorStatusOption != nil {
		eso = *errorStatusOption
	}
	var report strings.Builder
	fmt.Fprintf(&report, "Start:         %s\nEnd:           %s\nDuration:      %s\nFulfilled:     %s\nStatus:        %s\nTotal Puzzles: %d\nTotal Games:   %d\n",
		startTime.Format(time.RFC3339Nano), endTime.Format(time.RFC3339Nano), dur, fo, eso, totalPuzzles, totalGames)

	for _, bd := range breakdowns {
		fmt.Fprintf(&report, "Bucket: %4d, Puzzles: %4d, Games: %4d\n", bd[0], bd[1], bd[2])
	}
	return report.String(), nil
}

func processWithRealGames(ctx context.Context, cfg *config.Config, req *pb.PuzzleGenerationJobRequest, ps PuzzleStore, gs gameplay.GameStore, genId int) (bool, error) {
	numProcessedGames := 0
	if req.DaysPerChunk < 1 {
		return false, errors.New("must have more than 1 day per chunk")
	}
	startTime, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return false, err
	}

	// For non-bot-v-bot games we need to "hydrate" the game we get back
	// from the database with the right data structures in order for it
	// to generate moves properly.
	gd, err := kwg.Get(cfg.MacondoConfigMap, req.Lexicon)
	if err != nil {
		return false, err
	}
	dist, err := tilemapping.GetDistribution(cfg.MacondoConfigMap, req.LetterDistribution)
	if err != nil {
		return false, err
	}
	csgen := &cross_set.GaddagCrossSetGenerator{Dist: dist, Gaddag: gd}

	minimumStartTime, err := time.Parse("2006-01-02", "2021-01-01")
	if err != nil {
		return false, err
	}

	for startTime.After(minimumStartTime) {
		createdBeginning := startTime.Add(-time.Hour * 24 * time.Duration(req.DaysPerChunk))
		createdEnd := startTime
		log.Info().Time("start", createdBeginning).Time("end", createdEnd).Msg("searching...")
		gameIDs, err := ps.GetPotentialPuzzleGames(
			ctx, createdBeginning, createdEnd, int(req.GameConsiderationLimit),
			req.Lexicon, req.AvoidBotGames)

		if err != nil {
			return false, err
		}

		log.Info().Int("ct", len(gameIDs)).Msg("potential-games")

		for _, gid := range gameIDs {
			uuid := gid.String

			entGame, err := gs.Get(ctx, uuid)
			if err != nil {
				return false, err
			}
			entGame.Game.SetCrossSetGen(csgen)
			_, fulfilled, err := processGame(ctx, req.EquityLossTotalLimit, req.Request, genId, gs, ps, entGame, "", ipc.GameType_NATIVE)
			if err != nil {
				return false, err
			}
			numProcessedGames += 1
			if numProcessedGames%1000 == 0 {
				log.Info().Msgf("processed %d games...", numProcessedGames)
			}
			gs.Unload(ctx, uuid)
			if fulfilled {
				return true, nil
			}
		}
		startTime = createdBeginning
	}
	return false, errors.New("ran out of games")

}

func processJob(ctx context.Context, cfg *config.Config, req *pb.PuzzleGenerationJobRequest,
	genId int, gs gameplay.GameStore, ps PuzzleStore) (bool, error) {

	if req == nil {
		return false, errors.New("request is nil")
	}
	err := macondopuzzles.InitializePuzzleGenerationRequest(req.Request)
	if err != nil {
		return false, err
	}
	if !req.BotVsBot {
		return processWithRealGames(ctx, cfg, req, ps, gs, genId)
	} else {
		gamesCreated := 0
		for i := 0; i < int(req.GameConsiderationLimit); i++ {
			r := automatic.NewGameRunner(nil, &cfg.MacondoConfig)
			err := r.CompVsCompStatic(true)
			if err != nil {
				return false, err
			}
			g := newBotvBotPuzzleGame(r.Game(), req.Lexicon, req.LetterDistribution)
			// equity loss total limit is some big number for this, don't worry about it.
			gameCreated, fulfilled, err := processGame(ctx, 1000, req.Request, genId, gs, ps, g, "", ipc.GameType_BOT_VS_BOT)
			if err != nil {
				return false, err
			}
			if gameCreated {
				gamesCreated++
			}
			if fulfilled {
				return true, nil
			}
			if gamesCreated >= int(req.GameCreationLimit) {
				return false, nil
			}
		}
	}
	return false, nil
}

func processGame(ctx context.Context, eqLossLimit uint32, req *macondopb.PuzzleGenerationRequest, genId int, gs gameplay.GameStore, ps PuzzleStore,
	g *entity.Game, authorId string, gameType ipc.GameType) (bool, bool, error) {

	pzls, err := CreatePuzzlesFromGame(ctx, eqLossLimit, req, genId, gs, ps, g, "", gameType, false)
	if err != nil {
		return false, false, err
	}
	if len(pzls) == 0 {
		return false, false, nil
	}
	lastBucketIndex := len(req.Buckets) - 1
	for i := len(req.Buckets) - 1; i >= 0; i-- {
		if req.Buckets[i].Size == 0 {
			req.Buckets[i], req.Buckets[lastBucketIndex] = req.Buckets[lastBucketIndex], req.Buckets[i]
			lastBucketIndex--
		}
	}
	req.Buckets = req.Buckets[:lastBucketIndex+1]
	fulfilled := false
	if len(req.Buckets) == 0 {
		fulfilled = true
	}
	return true, fulfilled, nil
}

func newBotvBotPuzzleGame(mcg *macondogame.Game, lexicon, letterdistribution string) *entity.Game {
	mockBotGameReq.Lexicon = lexicon
	mockBotGameReq.Rules.LetterDistributionName = letterdistribution
	g := entity.NewGame(mcg, mockBotGameReq)
	g.Started = true
	uuid := shortuuid.New()
	g.GameEndReason = ipc.GameEndReason_STANDARD
	g.Quickdata.FinalScores = []int32{int32(g.Game.PointsFor(0)), int32(g.Game.PointsFor(1))}
	g.Quickdata.PlayerInfo = []*ipc.PlayerInfo{&common.DefaultPlayerOneInfo, &common.DefaultPlayerTwoInfo}
	// add a fake uuid for each user
	g.Game.History().Players[0].UserId = common.DefaultPlayerOneInfo.UserId
	g.Game.History().Players[1].UserId = common.DefaultPlayerTwoInfo.UserId
	g.Game.History().Uid = uuid
	g.Game.History().PlayState = macondopb.PlayState_GAME_OVER
	g.Timers = entity.Timers{
		TimeRemaining: []int{0, 0},
		MaxOvertime:   0,
	}

	return g
}
