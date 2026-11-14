package main

import (
	"context"
	"time"

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

// registerJobs registers the API's scheduled jobs on r.
func registerJobs(r *jobs.Runner, d jobDeps) {
	schedule := func(kind string, s jobs.Schedule, f func(context.Context) error) {
		r.Schedule(kind, s, func(ctx context.Context, _ jobs.Job) error { return f(ctx) })
	}
	schedule("adjudicate_realtime_games", jobs.Every(bus.AdjudicateInterval), d.bus.AdjudicateRealtimeGames)
	schedule("adjudicate_correspondence_games", jobs.Every(bus.AdjudicateCorrespondenceInterval), d.bus.AdjudicateCorrespondenceGames)
	schedule("expire_seeks", jobs.Every(bus.SeeksExpireInterval), d.bus.ExpireSeeks)
	schedule("reclaim_analysis_jobs", jobs.Every(30*time.Second), d.analysis.ReclaimStaleJobs)
	schedule("poll_broadcasts", jobs.Every(broadcasts.PollInterval), d.broadcasts.PollDue)
	schedule("poll_vdo_streams", jobs.Every(d.vdo.PollInterval()), d.vdo.PollActiveStreams)
	schedule("prune_client_records", jobs.Every(24*time.Hour), pruneClientRecords(d.users))
}
