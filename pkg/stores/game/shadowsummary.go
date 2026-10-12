package game

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
)

// shadowCounts tallies what a shadow comparison concluded, so that a quiet log
// can be told apart from a shadow that never ran. Agreement is logged only at
// debug level, which production does not emit; without these counts, "no
// mismatches this week" would be indistinguishable from "no comparisons this
// week".
type shadowCounts struct {
	ok        atomic.Int64 // compared, and agreed
	mismatch  atomic.Int64 // compared, and disagreed
	torn      atomic.Int64 // terminal event in turns beside a live games row
	moreRows  atomic.Int64 // more turn rows than events
	fewerRows atomic.Int64 // fewer turn rows than events
	noTurns   atomic.Int64 // no turn rows: predates dual-write, or no moves yet
	skipped   atomic.Int64 // not compared: rules or snapshot unavailable
	errored   atomic.Int64 // query, decode or rebuild failed
}

var (
	loadShadowCounts  shadowCounts
	turnsShadowCounts shadowCounts
)

// RunShadowSummary logs, once per interval, how many comparisons each shadow
// made and what they concluded, and starts the counts again. It returns when
// ctx is done, or straight away if neither shadow is on.
func RunShadowSummary(ctx context.Context, cfg *config.Config, every time.Duration) {
	if !cfg.ShadowTurns && !cfg.ShadowTurnsLoad {
		return
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if cfg.ShadowTurnsLoad {
				loadShadowCounts.report(log.Logger, "load", every)
			}
			if cfg.ShadowTurns {
				turnsShadowCounts.report(log.Logger, "turns", every)
			}
		}
	}
}

func (c *shadowCounts) report(l zerolog.Logger, shadow string, every time.Duration) {
	l.Info().
		Str("shadow", shadow).
		Str("period", every.String()).
		Int64("ok", c.ok.Swap(0)).
		Int64("mismatch", c.mismatch.Swap(0)).
		Int64("torn", c.torn.Swap(0)).
		Int64("more_rows", c.moreRows.Swap(0)).
		Int64("fewer_rows", c.fewerRows.Swap(0)).
		Int64("no_turns", c.noTurns.Swap(0)).
		Int64("skipped", c.skipped.Swap(0)).
		Int64("errored", c.errored.Swap(0)).
		Msg("shadow-summary")
}
