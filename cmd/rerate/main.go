// rerate replays every rated game under a rating system, from the first game
// on. See docs/ratings_plan.md.
//
//	rerate extract -out data.gob.gz [-from older.gob.gz]
//	    Reads finished rated games, users, moderator rating resets and saved
//	    ratings from the database (read-only). With -from, reads only what
//	    ended since that file was written. Uses the same DB_* settings as
//	    liwords-api.
//
//	rerate replay -in data.gob.gz [-system 2] [-report report.txt]
//	    Replays the games and writes a dry-run report comparing the result
//	    with the saved ratings. -since/-until (YYYY-MM-DD) limit the games;
//	    -games-tsv and -ratings-tsv write the games and the result for
//	    checking against another implementation.
//
// Nothing here writes to the database.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/rerate"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: rerate extract|replay [flags]")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var err error
	switch os.Args[1] {
	case "extract":
		err = extract(ctx, os.Args[2:])
	case "replay":
		err = replay(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		log.Fatal().Err(err).Msg("rerate")
	}
}

func extract(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("extract", flag.ExitOnError)
	out := fs.String("out", "", "file to write")
	from := fs.String("from", "", "earlier file to bring up to date")
	batch := fs.Int64("batch", 20000, "game ids per query")
	pause := fs.Duration("pause", 100*time.Millisecond, "pause between queries")
	fs.Parse(args)
	if *out == "" {
		return fmt.Errorf("-out is required")
	}
	var prev *rerate.Data
	if *from != "" {
		var err error
		if prev, err = rerate.Load(*from); err != nil {
			return err
		}
	}
	cfg := &config.Config{}
	cfg.Load(nil)
	poolCfg, err := pgxpool.ParseConfig(cfg.DBConnUri)
	if err != nil {
		return err
	}
	poolCfg.MaxConns = 2
	poolCfg.ConnConfig.RuntimeParams["statement_timeout"] = "300000"
	poolCfg.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	d, err := rerate.Extract(ctx, pool, prev, rerate.ExtractOptions{BatchSize: *batch, Pause: *pause})
	if err != nil {
		return err
	}
	log.Info().Int("games", len(d.Games)).Int("users", len(d.Users)).Int("unfinished", len(d.Unfinished)).
		Interface("skipped", d.Skipped).Msg("extracted")
	return d.Save(*out)
}

func replay(args []string) error {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)
	in := fs.String("in", "", "file written by extract")
	system := fs.Int("system", 2, "rating system: 1 or 2")
	since := fs.String("since", "", "replay games that ended on or after this date (YYYY-MM-DD), starting everyone from scratch")
	until := fs.String("until", "", "replay games that ended before this date (YYYY-MM-DD)")
	reportPath := fs.String("report", "", "dry-run report to write")
	gamesTSV := fs.String("games-tsv", "", "write the replayed games here")
	ratingsTSV := fs.String("ratings-tsv", "", "write the resulting ratings here")
	fs.Parse(args)
	if *in == "" {
		return fmt.Errorf("-in is required")
	}
	d, err := rerate.Load(*in)
	if err != nil {
		return err
	}
	sys := gameplay.RatingSystemV2
	if *system == 1 {
		sys = gameplay.RatingSystemV1
	}
	s, err := parseDate(*since)
	if err != nil {
		return err
	}
	u, err := parseDate(*until)
	if err != nil {
		return err
	}
	start := time.Now()
	res := rerate.Replay(sys, d, rerate.ReplayOptions{Since: s, Until: u})
	log.Info().Int("rated", res.Rated).Interface("skipped", res.Skipped).Dur("took", time.Since(start)).Msg("replayed")

	write := func(path string, f func(*os.File) error) error {
		if path == "" {
			return nil
		}
		fh, err := os.Create(path)
		if err != nil {
			return err
		}
		defer fh.Close()
		return f(fh)
	}
	if err := write(*reportPath, func(f *os.File) error {
		return rerate.WriteReport(f, sys, d, nil, res, rerate.ReportOptions{TopPools: []entity.VariantKey{
			"CSW19.classic.regular", "NWL18.classic.regular", "CSW19.classic.corres", "NWL18.classic.corres",
			"CSW19.classic.blitz", "NWL18.classic.blitz"}})
	}); err != nil {
		return err
	}
	if err := write(*gamesTSV, func(f *os.File) error { return rerate.WriteGamesTSV(f, d, s, u) }); err != nil {
		return err
	}
	return write(*ratingsTSV, func(f *os.File) error { return rerate.WriteRatingsTSV(f, res) })
}

func parseDate(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}
