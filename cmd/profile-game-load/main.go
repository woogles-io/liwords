// profile-game-load measures what it costs to load a game.
//
// It exists to answer one question, and the question keeps coming back: can we
// afford to load games from the database rather than hold them in memory? The
// in-memory cache is gone, so every request now pays this cost, and the next
// change to the load path -- serving a position rebuilt from game_turns
// instead of from the stored history -- has to be measured against it.
//
//	go run ./cmd/profile-game-load -games 20 -n 20
//	go run ./cmd/profile-game-load -game <id> -compare
//	go run ./cmd/profile-game-load -games 20 -cpuprofile cpu.out
//
// Two paths can be measured:
//
//   - "stored"      what DBStore.Get does today: read the history (S3 for
//     archived games, bytea otherwise) and replay it through
//     macondo.
//   - "from turns"  the candidate: read the games row and its event log in one
//     statement and rebuild the position with xwordbridge. Only
//     measured with -compare, and only for games that have turn
//     rows.
//
// Reported as percentiles rather than an average, because the average is not
// the number anyone acts on -- production alerting watches p99, and a mean over
// ten iterations hides exactly the tail that matters.
package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/common"
	gamestore "github.com/woogles-io/liwords/pkg/stores/game"
	"github.com/woogles-io/liwords/pkg/stores/models"
	userstore "github.com/woogles-io/liwords/pkg/stores/user"
	"github.com/woogles-io/liwords/pkg/xwordbridge"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// measuredRate is what production actually does, from Jaeger: ~1.5 loads/sec
// through the cache, ~12/sec once every load is a real load. Projecting at
// 100 and 1000 -- which this tool used to do -- answers a question nobody has.
const measuredRate = 12.0

func main() {
	gameID := flag.String("game", "", "profile this game; if empty, sample live games")
	nGames := flag.Int("games", 10, "how many games to sample when -game is not given")
	iterations := flag.Int("n", 20, "timed iterations per game")
	warmup := flag.Int("warmup", 2, "iterations to discard first (lexicons and pools load lazily)")
	rate := flag.Float64("rate", measuredRate, "loads/sec to project the cost at")
	compare := flag.Bool("compare", false, "also measure rebuilding the position from game_turns")
	cpuProfile := flag.String("cpuprofile", "", "write a pprof CPU profile here")
	memProfile := flag.String("memprofile", "", "write a pprof heap profile here")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)

	ctx := context.Background()
	cfg := &config.Config{}
	if err := cfg.Load(nil); err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
	ctx = cfg.WithContext(ctx)

	dbPool, err := pgxpool.New(ctx, cfg.DBConnDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer dbPool.Close()

	userStore, err := userstore.NewDBStore(dbPool)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create user store")
	}
	gameStore, err := gamestore.NewDBStore(cfg, userStore, dbPool)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create game store")
	}
	queries := models.New(dbPool)

	ids := []string{*gameID}
	if *gameID == "" {
		if ids, err = sampleLiveGames(ctx, dbPool, *nGames); err != nil {
			log.Fatal().Err(err).Msg("failed to sample games")
		}
		if len(ids) == 0 {
			fmt.Println("no live games to sample; pass -game <id>")
			os.Exit(1)
		}
	}
	fmt.Printf("%d game(s), %d iterations each (%d warmup discarded)\n\n", len(ids), *iterations, *warmup)

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal().Err(err).Msg("cpuprofile")
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal().Err(err).Msg("cpuprofile")
		}
		defer pprof.StopCPUProfile()
	}

	stored := &samples{name: "stored (history replay)"}
	fromTurns := &samples{name: "from turns (reconstruction)"}

	for _, id := range ids {
		// Warmup is discarded, not averaged in. The first load of a game pays
		// for lexicons and connections that every later one gets free, and over
		// ten iterations that one outlier moves the mean enough to matter.
		for range *warmup {
			if _, err := gameStore.Get(ctx, id); err != nil {
				log.Fatal().Err(err).Str("game", id).Msg("failed to get game")
			}
		}
		for range *iterations {
			if err := stored.measure(func() (int, error) {
				g, err := gameStore.Get(ctx, id)
				if err != nil {
					return 0, err
				}
				return len(g.History().Events), nil
			}); err != nil {
				log.Fatal().Err(err).Str("game", id).Msg("stored path")
			}
		}

		if !*compare {
			continue
		}
		for range *warmup {
			_, _ = loadFromTurns(ctx, queries, cfg, id)
		}
		for range *iterations {
			if err := fromTurns.measure(func() (int, error) {
				return loadFromTurns(ctx, queries, cfg, id)
			}); err != nil {
				// A game with no turn rows is not a failure, just not
				// comparable -- say so once and move on.
				fmt.Printf("  %s: cannot rebuild from turns (%v)\n", id, err)
				break
			}
		}
	}

	stored.report(*rate)
	if *compare && fromTurns.n > 0 {
		fromTurns.report(*rate)
		compareReport(stored, fromTurns)
	}

	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			log.Fatal().Err(err).Msg("memprofile")
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal().Err(err).Msg("memprofile")
		}
	}
}

// sampleLiveGames picks unfinished games, because those are the ones a load
// path change affects -- finished games come from S3.
func sampleLiveGames(ctx context.Context, pool *pgxpool.Pool, n int) ([]string, error) {
	rows, err := pool.Query(ctx,
		`SELECT uuid FROM games WHERE game_end_reason = 0 ORDER BY updated_at DESC LIMIT $1`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// loadFromTurns is the candidate load path end to end: one statement for the
// games row and its event log, then a rebuild. Rules are resolved inside the
// timed section deliberately -- production resolves them per load too, and the
// lexicon and distribution lookups they do are cache hits, so excluding them
// would flatter this path against the one it is being compared to.
func loadFromTurns(ctx context.Context, q *models.Queries, cfg *config.Config, id string) (int, error) {
	row, err := q.GetGameWithTurns(ctx, models.GetGameWithTurnsParams{
		WithTurns: true, Uuid: common.ToPGTypeText(id),
	})
	if err != nil {
		return 0, err
	}
	if len(row.TurnEvents) == 0 {
		return 0, fmt.Errorf("no turn rows")
	}
	events, err := xwordbridge.DecodeTurns(row.TurnEvents)
	if err != nil {
		return 0, err
	}
	spec := xwordbridge.RulesSpec{}
	if req := row.GameRequest.GameRequest; req != nil {
		spec.Lexicon = req.Lexicon
		spec.ChallengeRule = req.ChallengeRule
		if req.Rules != nil {
			spec.BoardLayout = req.Rules.BoardLayoutName
			spec.LetterDistribution = req.Rules.LetterDistributionName
			spec.Variant = req.Rules.VariantName
		}
	}
	rules, err := xwordbridge.RulesFor(cfg.MacondoConfig(), spec)
	if err != nil {
		return 0, err
	}
	_, err = xwordbridge.StateFromTurns(xwordbridge.TurnsInput{
		Events:         events,
		LastKnownRacks: row.LastKnownRacks,
		GameEnded:      pb.GameEndReason(row.GameEndReason.Int32) != pb.GameEndReason_NONE,
	}, rules, nil)
	if err != nil {
		return 0, err
	}
	return len(events), nil
}

// samples accumulates one path's measurements.
type samples struct {
	name      string
	n         int
	durations []time.Duration
	allocs    uint64
	bytes     uint64
	events    int
}

func (s *samples) measure(load func() (int, error)) error {
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)
	start := time.Now()
	events, err := load()
	d := time.Since(start)
	runtime.ReadMemStats(&m2)
	if err != nil {
		return err
	}
	s.n++
	s.durations = append(s.durations, d)
	s.allocs += m2.Mallocs - m1.Mallocs
	s.bytes += m2.TotalAlloc - m1.TotalAlloc
	s.events += events
	return nil
}

func (s *samples) percentile(p float64) time.Duration {
	if len(s.durations) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), s.durations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	i := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(sorted) {
		i = len(sorted) - 1
	}
	return sorted[i]
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000.0 }

func (s *samples) report(rate float64) {
	if s.n == 0 {
		return
	}
	fmt.Printf("=== %s ===\n", s.name)
	fmt.Printf("  loads:       %d, averaging %d events each\n", s.n, s.events/s.n)
	fmt.Printf("  p50 / p90:   %.2f ms / %.2f ms\n", ms(s.percentile(50)), ms(s.percentile(90)))
	fmt.Printf("  p99 / max:   %.2f ms / %.2f ms\n", ms(s.percentile(99)), ms(s.percentile(100)))
	fmt.Printf("  allocations: %d per load, %d KB\n", s.allocs/uint64(s.n), (s.bytes/uint64(s.n))/1024)

	// Projected from p50, and only the load. A real request also plays the
	// move, saves it, takes the game lock and publishes to NATS.
	p50 := ms(s.percentile(50))
	perLoadMB := float64(s.bytes) / float64(s.n) / 1024 / 1024
	fmt.Printf("  at %.0f loads/sec, loads alone: %.2f%% of one core, %.1f MB/sec allocated\n\n",
		rate, p50*rate/10.0, perLoadMB*rate)
}

func compareReport(a, b *samples) {
	ap, bp := ms(a.percentile(50)), ms(b.percentile(50))
	if ap == 0 {
		return
	}
	fmt.Printf("=== comparison (p50) ===\n")
	fmt.Printf("  %-28s %.2f ms\n", a.name, ap)
	fmt.Printf("  %-28s %.2f ms  (%.2fx)\n", b.name, bp, bp/ap)
	if b.n > 0 && a.n > 0 {
		aKB := (a.bytes / uint64(a.n)) / 1024
		bKB := (b.bytes / uint64(b.n)) / 1024
		fmt.Printf("  allocated per load:          %d KB vs %d KB\n", aKB, bKB)
	}
}
