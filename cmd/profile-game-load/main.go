package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	gamestore "github.com/woogles-io/liwords/pkg/stores/game"
	userstore "github.com/woogles-io/liwords/pkg/stores/user"
)

func main() {
	gameID := flag.String("game", "", "Game ID to profile")
	iterations := flag.Int("n", 10, "Number of iterations")
	flag.Parse()

	if *gameID == "" {
		fmt.Println("Usage: go run main.go -game <gameID> [-n iterations]")
		os.Exit(1)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.ErrorLevel) // Suppress debug logs for cleaner output

	ctx := context.Background()

	// Load config
	cfg := &config.Config{}
	if err := cfg.Load(nil); err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// Add config to context (some stores might expect it)
	ctx = cfg.WithContext(ctx)

	// Connect to database
	dbPool, err := pgxpool.New(ctx, cfg.DBConnDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer dbPool.Close()

	// Create stores
	userStore, err := userstore.NewDBStore(dbPool)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create user store")
	}

	gameStore, err := gamestore.NewDBStore(cfg, userStore, dbPool)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create game store")
	}

	fmt.Printf("Profiling game load for: %s\n", *gameID)
	fmt.Printf("Running %d iterations...\n\n", *iterations)

	// Get initial GC stats
	var gcStart runtime.MemStats
	runtime.ReadMemStats(&gcStart)

	var totalDuration time.Duration
	var totalAllocs uint64
	var totalBytes uint64
	var maxGCPause time.Duration
	var totalGCPauses time.Duration

	for i := 0; i < *iterations; i++ {
		var m1, m2 runtime.MemStats
		runtime.ReadMemStats(&m1)

		start := time.Now()
		game, err := gameStore.Get(ctx, *gameID)
		duration := time.Since(start)

		runtime.ReadMemStats(&m2)

		if err != nil {
			log.Fatal().Err(err).Msg("failed to get game")
		}

		allocs := m2.Mallocs - m1.Mallocs
		bytes := m2.TotalAlloc - m1.TotalAlloc

		// Track GC pauses that happened during this iteration
		gcPauseDuring := time.Duration(0)
		if m2.NumGC > m1.NumGC {
			// GC happened during this iteration
			for j := m1.NumGC; j < m2.NumGC; j++ {
				pauseNs := m2.PauseNs[(j+255)%256]
				pause := time.Duration(pauseNs)
				gcPauseDuring += pause
				if pause > maxGCPause {
					maxGCPause = pause
				}
			}
			totalGCPauses += gcPauseDuring
		}

		totalDuration += duration
		totalAllocs += allocs
		totalBytes += bytes

		gcIndicator := ""
		if m2.NumGC > m1.NumGC {
			gcIndicator = fmt.Sprintf(" [GC: %d times, %.2fms]", m2.NumGC-m1.NumGC, float64(gcPauseDuring.Microseconds())/1000.0)
		}

		fmt.Printf("Iteration %3d: %6.2fms | %8d allocs | %8d KB | %d events%s\n",
			i+1,
			float64(duration.Microseconds())/1000.0,
			allocs,
			bytes/1024,
			len(game.History().Events),
			gcIndicator)
	}

	// Get final GC stats
	var gcEnd runtime.MemStats
	runtime.ReadMemStats(&gcEnd)

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Average time:   %6.2fms\n", float64(totalDuration.Microseconds())/float64(*iterations)/1000.0)
	fmt.Printf("Average allocs: %d\n", totalAllocs/uint64(*iterations))
	fmt.Printf("Average bytes:  %d KB\n", (totalBytes/uint64(*iterations))/1024)

	fmt.Printf("\n=== GC Statistics ===\n")
	numGCs := gcEnd.NumGC - gcStart.NumGC
	if numGCs > 0 {
		avgGCPause := time.Duration(gcEnd.PauseTotalNs-gcStart.PauseTotalNs) / time.Duration(numGCs)
		fmt.Printf("GC runs:        %d times during test\n", numGCs)
		fmt.Printf("GC pause avg:   %.3fms\n", float64(avgGCPause.Microseconds())/1000.0)
		fmt.Printf("GC pause max:   %.3fms\n", float64(maxGCPause.Microseconds())/1000.0)
		fmt.Printf("GC pause total: %.2fms\n", float64(gcEnd.PauseTotalNs-gcStart.PauseTotalNs)/1000000.0)
		fmt.Printf("GC overhead:    %.2f%% of total time\n",
			float64(gcEnd.PauseTotalNs-gcStart.PauseTotalNs)/float64(totalDuration.Nanoseconds())*100.0)
	} else {
		fmt.Printf("GC runs:        0 (no GC needed during test!)\n")
	}

	fmt.Printf("Heap in use:    %d MB\n", gcEnd.HeapAlloc/1024/1024)
	fmt.Printf("Heap from OS:   %d MB\n", gcEnd.HeapSys/1024/1024)
	fmt.Printf("Total alloc:    %d MB (cumulative)\n", gcEnd.TotalAlloc/1024/1024)

	fmt.Printf("\n=== Load Projections ===\n")
	avgMs := float64(totalDuration.Microseconds()) / float64(*iterations) / 1000.0
	fmt.Printf("At 100 req/sec:\n")
	fmt.Printf("  CPU time:     %.0f%% of one core\n", avgMs*100/10.0)
	fmt.Printf("  Memory alloc: %.2f MB/sec\n", float64(totalBytes)/float64(*iterations)/1024/1024*100)

	fmt.Printf("\nAt 1000 req/sec:\n")
	fmt.Printf("  CPU time:     %.0f%% of one core\n", avgMs*100)
	fmt.Printf("  Memory alloc: %.2f MB/sec\n", float64(totalBytes)/float64(*iterations)/1024/1024*1000)
	if numGCs > 0 {
		avgGCPause := time.Duration(gcEnd.PauseTotalNs-gcStart.PauseTotalNs) / time.Duration(numGCs)
		fmt.Printf("  Est. GC freq: ~every %.0f requests\n",
			float64(*iterations)/float64(numGCs))
		fmt.Printf("  Est. GC pause: %.3fms avg\n", float64(avgGCPause.Microseconds())/1000.0)
	}
}
