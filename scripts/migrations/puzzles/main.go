package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/macondo/puzzles"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/common"
	puzzlestore "github.com/woogles-io/liwords/pkg/stores/puzzles"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Migrate puzzles from old to new lexicon.

func migrate(cfg *config.Config, pool *pgxpool.Pool, oldLex, newLex string) error {
	ctx := context.Background()
	tx, err := pool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
	SELECT puzzles.uuid, game_id, games.history, games.request, turn_number, answer FROM puzzles
	JOIN games on game_id = games.id
	WHERE lexicon = $1 AND valid = true`

	rows, err := tx.Query(ctx, query, oldLex)
	if err != nil {
		return err
	}
	defer rows.Close()
	invalidPuzzles := 0
	validPuzzles := 0

	for rows.Next() {
		var puuid string
		var gid int64
		var gh []byte
		var req []byte
		var turn int
		var answer []byte
		if err := rows.Scan(&puuid, &gid, &gh, &req, &turn, &answer); err != nil {
			return err
		}
		gevt, err := puzzlestore.AnswerBytesToGameEvent(answer)
		if err != nil {
			return err
		}
		ghist := &macondopb.GameHistory{}
		if err = proto.Unmarshal(gh, ghist); err != nil {
			return err
		}
		greq := &ipc.GameRequest{}
		if err = proto.Unmarshal(req, greq); err != nil {
			return err
		}
		rules, err := game.NewBasicGameRules(&cfg.MacondoConfig, oldLex, greq.Rules.BoardLayoutName,
			greq.Rules.LetterDistributionName, game.CrossScoreOnly, game.Variant(greq.Rules.VariantName))
		if err != nil {
			return err
		}
		mcg, err := game.NewFromHistory(ghist, rules, turn)
		if err != nil {
			return err
		}
		valid, err := puzzles.IsEquityPuzzleStillValid(&cfg.MacondoConfig, mcg, turn, gevt, newLex)
		if err != nil {
			return err
		}
		if !valid {
			fmt.Printf("Puzzle %v no longer valid\n", puuid)
			invalidPuzzles++
		} else {
			validPuzzles++
		}
	}
	fmt.Printf("Invalid puzzles: %d, valid puzzles: %d\n", invalidPuzzles, validPuzzles)

	return nil
}

func main() {
	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Msgf("Loaded config: %v", cfg)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	log.Debug().Msg("debug logging on")
	pool, err := common.OpenDB(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
	if err != nil {
		panic(err)
	}

	err = migrate(cfg, pool, "NWL20", "NWL23")
	if err != nil {
		panic(err)
	}
}
