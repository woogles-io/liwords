package main

import (
	"context"
	"fmt"

	"github.com/woogles-io/liwords/pkg/analysis"
	"github.com/woogles-io/liwords/pkg/broadcasts"
	"github.com/woogles-io/liwords/pkg/bus"
	"github.com/woogles-io/liwords/pkg/jobs"
	"github.com/woogles-io/liwords/pkg/user"
	"github.com/woogles-io/liwords/pkg/vdowebhook"
)

// Background work that runs once across all API servers (pkg/jobs). Each
// server runs a jobs.Runner unless started with -run-jobs=false.
//
// The channel monitor (pkg/bus) and the shadow summary (pkg/stores/game)
// stay per server: they report on the process they run in.

// jobDeps is what the jobs below use.
type jobDeps struct {
	bus        *bus.Bus
	analysis   *analysis.AnalysisService
	broadcasts *broadcasts.BroadcastService
	vdo        *vdowebhook.VDOWebhookService
	users      user.Store
}

// registerJobs registers the API's scheduled jobs on r. Schedules are cron
// expressions in UTC (jobs.Cron); six fields have seconds first.
func registerJobs(r *jobs.Runner, d jobDeps) error {
	schedule := func(kind, expr string, f func(context.Context) error) {
		r.Schedule(kind, jobs.Cron(expr), func(ctx context.Context, _ jobs.Job) error { return f(ctx) })
	}
	schedule("adjudicate_realtime_games", "*/10 * * * * *", d.bus.AdjudicateRealtimeGames)
	schedule("adjudicate_correspondence_games", "* * * * *", d.bus.AdjudicateCorrespondenceGames)
	schedule("expire_seeks", "*/10 * * * *", d.bus.ExpireSeeks)
	schedule("reclaim_analysis_jobs", "*/30 * * * * *", d.analysis.ReclaimStaleJobs)
	schedule("poll_broadcasts", "*/30 * * * * *", d.broadcasts.PollDue)
	schedule("prune_client_records", "0 5 * * *", pruneClientRecords(d.users))

	// Set by VDO_POLLING_INTERVAL_SECONDS (300 in production).
	vdoExpr, err := jobs.CronEvery(d.vdo.PollInterval())
	if err != nil {
		return fmt.Errorf("vdo polling interval: %w", err)
	}
	schedule("poll_vdo_streams", vdoExpr, d.vdo.PollActiveStreams)
	return nil
}
