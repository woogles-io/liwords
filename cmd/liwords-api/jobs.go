package main

import (
	"context"
	"fmt"
	"time"

	"github.com/woogles-io/liwords/pkg/analysis"
	"github.com/woogles-io/liwords/pkg/broadcasts"
	"github.com/woogles-io/liwords/pkg/bus"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/emailer"
	"github.com/woogles-io/liwords/pkg/jobs"
	"github.com/woogles-io/liwords/pkg/notify"
	"github.com/woogles-io/liwords/pkg/outbox"
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
	outbox     *outbox.Sender
}

// vdoPollOffset is how long after each interval boundary the VDO poll runs.
const vdoPollOffset = 17 * time.Second

// registerJobs registers the API's scheduled jobs on r. Schedules are cron
// expressions in UTC (jobs.Cron); six fields have seconds first.
func registerJobs(r *jobs.Runner, d jobDeps) error {
	schedule := func(kind, expr string, f func(context.Context) error) {
		r.Schedule(kind, jobs.Cron(expr), func(ctx context.Context, _ jobs.Job) error { return f(ctx) })
	}
	// A fallback: a player's client ends the game when the opponent's clock
	// runs out. This catches games no client is watching, and cancels games
	// never started (bus.CancelAfter).
	schedule("adjudicate_realtime_games", "*/30 * * * * *", d.bus.AdjudicateRealtimeGames)
	schedule("adjudicate_correspondence_games", "* * * * *", d.bus.AdjudicateCorrespondenceGames)
	schedule("expire_seeks", "*/10 * * * *", d.bus.ExpireSeeks)
	schedule("reclaim_analysis_jobs", "*/30 * * * * *", d.analysis.ReclaimStaleJobs)
	schedule("poll_broadcasts", "*/30 * * * * *", d.broadcasts.PollDue)
	schedule("prune_client_records", "0 5 * * *", pruneClientRecords(d.users))
	// Emails and Discord posts written to the outbox (league notices). One
	// run sends all that is waiting, 2 s apart; a run cut off by a deploy
	// leaves the rest for the next.
	schedule("send_outbox", "* * * * *", d.outbox.Send)

	// Set by VDO_POLLING_INTERVAL_SECONDS (300 in production): at :00:17,
	// :05:17 and so on, off the minute boundaries when other people's
	// scheduled jobs call VDO.Ninja too. Each poll also spreads its checks
	// over 30s.
	vdoExpr, err := jobs.CronEvery(d.vdo.PollInterval(), vdoPollOffset)
	if err != nil {
		return fmt.Errorf("vdo polling interval: %w", err)
	}
	schedule("poll_vdo_streams", vdoExpr, d.vdo.PollActiveStreams)
	return nil
}

// outboxChannels sends outbox messages: emails through SES, league posts to
// the league Discord webhook (dropped when none is configured, as before).
func outboxChannels(cfg *config.Config) map[string]outbox.SendFunc {
	return map[string]outbox.SendFunc{
		outbox.ChannelEmail: func(_ context.Context, m outbox.Message) error {
			_, err := emailer.SendSimpleMessage(cfg.EmailDebugMode, m.Recipient, m.Subject, m.Body)
			return err
		},
		outbox.ChannelLeagueDiscord: func(_ context.Context, m outbox.Message) error {
			if cfg.LeagueDiscordToken == "" {
				return nil
			}
			return notify.Send(m.Body, cfg.LeagueDiscordToken)
		},
	}
}
