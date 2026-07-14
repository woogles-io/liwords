package main

import (
	"io"
	"os"
	"strings"
	"time"

	macondo "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
	pb "github.com/woogles-io/liwords/rpc/api/proto/puzzle_service"
	"google.golang.org/protobuf/encoding/protojson"
)

type lexiconPuzzleConfig struct {
	letterDistribution     string
	avoidBotGames          bool
	equityLossTotalLimit   uint32
	gameConsiderationLimit int32
	buckets                []*macondo.PuzzleBucket
}

var puzzleGenConfigs = map[string]lexiconPuzzleConfig{
	// English: avoid bots, strict equity filter, CEL_ONLY buckets, high game volume.
	"CSW24": {
		letterDistribution:     "english",
		avoidBotGames:          true,
		equityLossTotalLimit:   150,
		gameConsiderationLimit: 10000,
		buckets:                englishBuckets(),
	},
	"NWL23": {
		letterDistribution:     "english",
		avoidBotGames:          true,
		equityLossTotalLimit:   150,
		gameConsiderationLimit: 10000,
		buckets:                englishBuckets(),
	},
	// French: include bot games (low PvP volume), very lenient equity filter, no CEL_ONLY.
	"FRA24": {
		letterDistribution:     "french",
		avoidBotGames:          false,
		equityLossTotalLimit:   300,
		gameConsiderationLimit: 20000,
		buckets:                frenchBuckets(),
	},
	// German: now avoid bots (enough PvP games), medium equity filter, no CEL_ONLY.
	// Third bucket uses excludes=[NON_BINGO] to capture power-tile / high-scoring plays.
	"RD29": {
		letterDistribution:     "german",
		avoidBotGames:          true,
		equityLossTotalLimit:   160,
		gameConsiderationLimit: 25000,
		buckets:                germanBuckets(),
	},
}

func englishBuckets() []*macondo.PuzzleBucket {
	// Proportions mirror historical runs (1000:1000:500:1000:500 = 2:2:1:2:1), scaled to 60/week.
	return []*macondo.PuzzleBucket{
		{
			Includes: []macondo.PuzzleTag{macondo.PuzzleTag_EQUITY, macondo.PuzzleTag_NON_BINGO, macondo.PuzzleTag_CEL_ONLY},
			Excludes: []macondo.PuzzleTag{macondo.PuzzleTag_POWER_TILE},
			Size:     15,
		},
		{
			Includes: []macondo.PuzzleTag{macondo.PuzzleTag_EQUITY, macondo.PuzzleTag_BINGO},
			Size:     15,
		},
		{
			Includes: []macondo.PuzzleTag{macondo.PuzzleTag_EQUITY, macondo.PuzzleTag_NON_BINGO},
			Size:     8,
		},
		{
			Includes: []macondo.PuzzleTag{macondo.PuzzleTag_EQUITY, macondo.PuzzleTag_BINGO, macondo.PuzzleTag_CEL_ONLY},
			Size:     15,
		},
		{
			Includes: []macondo.PuzzleTag{macondo.PuzzleTag_EQUITY, macondo.PuzzleTag_CEL_ONLY, macondo.PuzzleTag_NON_BINGO},
			Size:     7,
		},
	}
}

func frenchBuckets() []*macondo.PuzzleBucket {
	// Proportions mirror historical runs (750:1000:250 = 3:4:1), scaled to 60/week.
	return []*macondo.PuzzleBucket{
		{
			Includes: []macondo.PuzzleTag{macondo.PuzzleTag_EQUITY, macondo.PuzzleTag_NON_BINGO},
			Excludes: []macondo.PuzzleTag{macondo.PuzzleTag_POWER_TILE},
			Size:     22,
		},
		{
			Includes: []macondo.PuzzleTag{macondo.PuzzleTag_EQUITY, macondo.PuzzleTag_BINGO},
			Size:     30,
		},
		{
			Includes: []macondo.PuzzleTag{macondo.PuzzleTag_EQUITY, macondo.PuzzleTag_NON_BINGO},
			Size:     8,
		},
	}
}

func germanBuckets() []*macondo.PuzzleBucket {
	// Proportions mirror historical runs (750:1000:250 = 3:4:1), scaled to 60/week.
	// Third bucket excludes NON_BINGO (not the same as BINGO — also captures power-tile plays).
	return []*macondo.PuzzleBucket{
		{
			Includes: []macondo.PuzzleTag{macondo.PuzzleTag_EQUITY, macondo.PuzzleTag_NON_BINGO},
			Excludes: []macondo.PuzzleTag{macondo.PuzzleTag_POWER_TILE},
			Size:     22,
		},
		{
			Includes: []macondo.PuzzleTag{macondo.PuzzleTag_EQUITY, macondo.PuzzleTag_BINGO},
			Size:     30,
		},
		{
			Includes: []macondo.PuzzleTag{macondo.PuzzleTag_EQUITY},
			Excludes: []macondo.PuzzleTag{macondo.PuzzleTag_NON_BINGO},
			Size:     8,
		},
	}
}

// WeeklyPuzzleGen fires a puzzle generation job for each configured lexicon,
// sampling the previous week's games via the existing ECS task.
func WeeklyPuzzleGen() error {
	secretKey := os.Getenv("PUZZLE_GEN_SECRET_KEY")
	lexicaEnv := os.Getenv("WEEKLY_PUZZLE_LEXICA")
	if lexicaEnv == "" {
		lexicaEnv = "CSW24,NWL23,FRA24,RD29"
	}
	lexica := strings.Split(lexicaEnv, ",")

	now := time.Now().UTC()
	startDate := now.Format("2006-01-02")
	earliestStartDate := now.AddDate(0, 0, -7).Format("2006-01-02")

	var firstErr error
	for _, lexicon := range lexica {
		lexicon = strings.TrimSpace(lexicon)
		cfg, ok := puzzleGenConfigs[lexicon]
		if !ok {
			log.Error().Str("lexicon", lexicon).Msg("no puzzle gen config for lexicon, skipping")
			continue
		}

		req := &pb.APIPuzzleGenerationJobRequest{
			SecretKey: secretKey,
			Request: &pb.PuzzleGenerationJobRequest{
				BotVsBot:               false,
				Lexicon:                lexicon,
				LetterDistribution:     cfg.letterDistribution,
				AvoidBotGames:          cfg.avoidBotGames,
				StartDate:              startDate,
				DaysPerChunk:           7,
				EarliestStartDate:      earliestStartDate,
				GameConsiderationLimit: cfg.gameConsiderationLimit,
				EquityLossTotalLimit:   cfg.equityLossTotalLimit,
				Request: &macondo.PuzzleGenerationRequest{
					Buckets: cfg.buckets,
				},
			},
		}

		bts, err := protojson.Marshal(req)
		if err != nil {
			log.Err(err).Str("lexicon", lexicon).Msg("failed to marshal puzzle gen request")
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		resp, err := WooglesAPIRequest(
			"puzzle_service.PuzzleService",
			"StartPuzzleGenJob",
			bts,
		)
		if err != nil {
			log.Err(err).Str("lexicon", lexicon).Msg("failed to call StartPuzzleGenJob")
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		log.Info().
			Str("lexicon", lexicon).
			Int("status", resp.StatusCode).
			Str("body", string(body)).
			Msg("weekly puzzle gen job started")
	}

	return firstErr
}
