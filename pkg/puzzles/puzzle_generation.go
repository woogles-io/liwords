package puzzles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lithammer/shortuuid"

	"github.com/domino14/liwords/pkg/common"
	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stores/game"
	puzzlesstore "github.com/domino14/liwords/pkg/stores/puzzles"
	"github.com/domino14/macondo/alphabet"
	"github.com/domino14/macondo/cross_set"
	"github.com/domino14/macondo/gaddag"
	macondogame "github.com/domino14/macondo/game"

	"github.com/domino14/liwords/rpc/api/proto/ipc"
	pb "github.com/domino14/liwords/rpc/api/proto/puzzle_service"

	"github.com/domino14/macondo/automatic"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

func Generate(ctx context.Context, cfg *config.Config, db *pgxpool.Pool, gs *game.DBStore, ps *puzzlesstore.DBStore, req *pb.PuzzleGenerationJobRequest, printStatus bool) error {
	genId, err := CreateGenerationLog(ctx, ps, req)
	if err != nil {
		return err
	}
	fulfilled, err := processJob(ctx, cfg, db, req, genId, gs, ps)
	err = UpdateGenerationLogStatus(ctx, ps, genId, fulfilled, err)
	if err != nil {
		return err
	}
	if printStatus {
		stats, err := getJobStatsString(ctx, db, genId)
		if err != nil {
			return err
		}
		fmt.Println(stats)
	}
	return err
}

func processJob(ctx context.Context, cfg *config.Config, db *pgxpool.Pool, req *pb.PuzzleGenerationJobRequest, genId int, gs *game.DBStore, ps *puzzlesstore.DBStore) (bool, error) {
	if req == nil {
		return false, errors.New("request is nil")
	}
	if !req.BotVsBot {
		rows, err := db.Query(ctx,
			`SELECT uuid FROM games WHERE games.id NOT IN
				(SELECT game_id FROM puzzles) AND
				(stats->'d1'->'Unchallenged Phonies'->'t')::int = 0 AND
				(stats->'d2'->'Unchallenged Phonies'->'t')::int = 0 AND
				game_end_reason != 0 LIMIT $1 OFFSET $2`, req.MaxGames, req.SqlOffset)
		if err != nil {
			return false, err
		}
		defer rows.Close()

		// For non-bot-v-bot games we need to "hydrate" the game we get back
		// from the database with the right data structures in order for it
		// to generate moves properly.
		gd, err := gaddag.Get(&cfg.MacondoConfig, req.Lexicon)
		if err != nil {
			return false, err
		}
		dist, err := alphabet.Get(&cfg.MacondoConfig, req.LetterDistribution)
		if err != nil {
			return false, err
		}
		csgen := cross_set.GaddagCrossSetGenerator{Dist: dist, Gaddag: gd}

		for rows.Next() {
			var UUID string
			if err := rows.Scan(&UUID); err != nil {
				return false, err
			}
			entGame, err := gs.Get(ctx, UUID)
			if err != nil {
				return false, err
			}
			if entGame.GameReq.Lexicon != req.Lexicon {
				continue
			}
			if entGame.GameReq.Rules.LetterDistributionName != req.LetterDistribution {
				continue
			}
			_, variant, err := entity.VariantFromGameReq(entGame.GameReq)
			if err != nil {
				return false, err
			}
			if variant != macondogame.VarClassic {
				continue
			}
			// Set cross-set generator so that it can actually generate moves.
			entGame.Game.SetCrossSetGen(csgen)
			fulfilled, err := processGame(ctx, req.Request, genId, gs, ps, entGame, "", ipc.GameType_NATIVE)
			if err != nil {
				return false, err
			}
			if fulfilled {
				return true, nil
			}
		}
	} else {
		for i := 0; i < int(req.MaxGames); i++ {
			r := automatic.NewGameRunner(nil, &cfg.MacondoConfig)
			err := r.CompVsCompStatic(true)
			if err != nil {
				return false, err
			}
			g := newBotvBotPuzzleGame(r.Game(), req.Lexicon)
			fulfilled, err := processGame(ctx, req.Request, genId, gs, ps, g, "", ipc.GameType_BOT_VS_BOT)
			if err != nil {
				return false, err
			}
			if fulfilled {
				return true, nil
			}
		}
	}
	return false, nil
}

func CreateGenerationLog(ctx context.Context, ps *puzzlesstore.DBStore, req *pb.PuzzleGenerationJobRequest) (int, error) {
	return ps.CreateGenerationLog(ctx, req)
}

func UpdateGenerationLogStatus(ctx context.Context, ps *puzzlesstore.DBStore, genId int, fulfilled bool, procErr error) error {
	return ps.UpdateGenerationLogStatus(ctx, genId, fulfilled, procErr)
}

func getJobStatsString(ctx context.Context, db *pgxpool.Pool, genId int) (string, error) {
	startTime, endTime, dur, fulfilledOption, errorStatusOption, err := getJobInfo(ctx, db, genId)
	if err != nil {
		return "", err
	}
	totalPuzzles, totalGames, breakdowns, err := getPuzzleGameBreakdowns(ctx, db, genId)
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
	eso := "incomplete"
	if errorStatusOption != nil {
		if *errorStatusOption == "" {
			eso = "ok"
		} else {
			eso = *errorStatusOption
		}
	}
	var report strings.Builder
	fmt.Fprintf(&report, "Start:         %s\nEnd:           %s\nDuration:      %s\nFulfilled:     %s\nStatus:        %s\nTotal Puzzles: %d\nTotal Games:   %d\n",
		startTime.Format(time.RFC3339Nano), endTime.Format(time.RFC3339Nano), dur, fo, eso, totalPuzzles, totalGames)

	for _, bd := range breakdowns {
		fmt.Fprintf(&report, "Bucket: %4d, Puzzles: %4d, Games: %4d\n", bd[0], bd[1], bd[2])
	}
	return report.String(), nil
}

func getJobInfo(ctx context.Context, db *pgxpool.Pool, genId int) (time.Time, time.Time, time.Duration, *bool, *string, error) {
	createdAtTime := time.Time{}
	completedAtTime := time.Time{}
	fulfilled := &sql.NullBool{}
	errorStatus := &sql.NullString{}
	err := db.QueryRow(ctx, `SELECT created_at, completed_at, fulfilled, error_status FROM puzzle_generation_logs WHERE id = $1`, genId).Scan(&createdAtTime, &completedAtTime, fulfilled, errorStatus)
	if err == pgx.ErrNoRows {
		return time.Time{}, time.Time{}, 0, nil, nil, fmt.Errorf("row not found while calculating job duration: %d", genId)
	}
	if err != nil {
		return time.Time{}, time.Time{}, 0, nil, nil, err
	}

	fo := false
	fulfilledOption := &fo
	if fulfilled.Valid {
		*fulfilledOption = fulfilled.Bool
	} else {
		fulfilledOption = nil
	}
	eso := ""
	errorStatusOption := &eso
	if errorStatus.Valid {
		*errorStatusOption = errorStatus.String
	} else {
		errorStatusOption = nil
	}
	return createdAtTime, completedAtTime, createdAtTime.Sub(completedAtTime), fulfilledOption, errorStatusOption, nil
}

func getPuzzleGameBreakdowns(ctx context.Context, db *pgxpool.Pool, genId int) (int, int, [][]int, error) {
	rows, err := db.Query(ctx, `SELECT bucket_index, COUNT(*) FROM puzzles WHERE generation_id = $1 GROUP BY bucket_index ORDER BY bucket_index ASC`, genId)
	if err != nil {
		return 0, 0, nil, err
	}
	defer rows.Close()
	if err == pgx.ErrNoRows {
		return 0, 0, nil, fmt.Errorf("no rows found for generation_id: %d", genId)
	}
	if err != nil {
		return 0, 0, nil, err
	}
	numTotalPuzzles := 0
	numTotalGames := 0
	breakdowns := [][]int{}
	for rows.Next() {
		var bucketIndex int
		var numPuzzles int
		if err := rows.Scan(&bucketIndex, &numPuzzles); err != nil {
			return 0, 0, nil, err
		}
		var numGames int
		err := db.QueryRow(ctx, `SELECT COUNT(*) FROM (SELECT DISTINCT game_id FROM puzzles WHERE generation_id = $1 AND bucket_index = $2) as unique_games`, genId, bucketIndex).Scan(&numGames)
		if err != nil {
			return 0, 0, nil, err
		}
		defer rows.Close()
		if err == pgx.ErrNoRows {
			return 0, 0, nil, fmt.Errorf("no rows found for generation_id, bucket_index: %d, %d", genId, bucketIndex)
		}
		if err != nil {
			return 0, 0, nil, err
		}
		breakdowns = append(breakdowns, []int{bucketIndex, numPuzzles, numGames})
		numTotalPuzzles += numPuzzles
		numTotalGames += numGames
	}
	return numTotalPuzzles, numTotalGames, breakdowns, nil
}

func processGame(ctx context.Context, req *macondopb.PuzzleGenerationRequest, genId int, gs *game.DBStore, ps *puzzlesstore.DBStore,
	g *entity.Game, authorId string, gameType ipc.GameType) (bool, error) {

	_, err := CreatePuzzlesFromGame(ctx, req, genId, gs, ps, g, "", gameType)
	if err != nil {
		return false, err
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
	return fulfilled, nil
}

func newBotvBotPuzzleGame(mcg *macondogame.Game, lexicon string) *entity.Game {
	common.DefaultGameReq.Lexicon = lexicon
	g := entity.NewGame(mcg, common.DefaultGameReq)
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
