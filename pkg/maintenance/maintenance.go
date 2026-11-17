// Package maintenance holds the periodic site upkeep that used to run in its
// own container (cmd/maintenance), started by EventBridge rules. It now runs
// as jobs in the API servers (cmd/liwords-api/jobs.go), with the API's own
// stores and event channel.
//
// Every task is safe to run again from the start after being cut off by a
// deploy: each is a single statement or transaction, or picks only the work
// not yet done (see docs/mikado/zdt_plan.md, section 7).
package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/integrations"
	"github.com/woogles-io/liwords/pkg/integrations/organizations"
	"github.com/woogles-io/liwords/pkg/league"
	"github.com/woogles-io/liwords/pkg/organization"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/config_service"
)

// Tasks runs the maintenance tasks with the API's stores.
type Tasks struct {
	cfg         *config.Config
	stores      *stores.Stores
	gameCreator league.GameCreator
	oauth       *integrations.OAuthIntegrationService
}

func New(cfg *config.Config, s *stores.Stores, gameCreator league.GameCreator, oauth *integrations.OAuthIntegrationService) *Tasks {
	return &Tasks{cfg: cfg, stores: s, gameCreator: gameCreator, oauth: oauth}
}

const (
	blogURL           = "https://blog.woogles.io"
	blogRSSFeedURL    = blogURL + "/index.xml"
	blogSearchString  = blogURL
	archiveSweepGrace = 10 * time.Minute
	// Keeps one run short; whatever is left over waits for the next hour.
	archiveSweepMaxGames = 200
)

// BlogRSS shows the newest blog post as the homepage announcement that links
// to the blog, replacing the one before.
func (t *Tasks) BlogRSS(ctx context.Context) error {
	feed, err := gofeed.NewParser().ParseURLWithContext(blogRSSFeedURL, ctx)
	if err != nil {
		return err
	}
	if len(feed.Items) < 1 {
		return errors.New("unexpected feed length")
	}
	item := feed.Items[0]
	authors := make([]string, 0, len(item.Authors))
	for _, a := range item.Authors {
		authors = append(authors, a.Name)
	}
	emoji := ""
	switch {
	case strings.Contains(item.Link, "/posts/"):
		emoji = "✏️"
	case strings.Contains(item.Link, "/guides/"):
		emoji = "📜"
	case strings.Contains(item.Link, "/articles/"):
		emoji = "📰"
	}
	body := item.Description + " (Click to read more)"
	if img := item.Custom["image"]; img != "" {
		body = "![Image](" + img + ")\n\n" + body
	}
	return t.stores.ConfigStore.SetAnnouncement(ctx, blogSearchString, &pb.Announcement{
		Title: emoji + " " + item.Title + " - written by " + strings.Join(authors, ", "),
		Body:  body,
		Link:  item.Link,
	})
}

type patreonBadge struct {
	PatreonUserID string `json:"patreon_user_id"`
	BadgeCode     string `json:"badge_code"`
}

// SubBadges gives every paying Patreon member their tier's badge, and takes
// it from everyone else, in one transaction.
func (t *Tasks) SubBadges(ctx context.Context) error {
	q := t.stores.Queries
	data, err := q.GetGlobalIntegrationData(ctx, integrations.PatreonIntegrationName)
	if err != nil {
		return err
	}
	var integrationData integrations.PatreonTokenResponse
	if err := json.Unmarshal(data, &integrationData); err != nil {
		return err
	}
	subscribers, err := integrations.GetCampaignSubscribers(ctx, integrationData.AccessToken)
	if err != nil {
		return err
	}
	var badges []patreonBadge
	for _, sub := range subscribers.Data {
		if len(sub.Relationships.CurrentlyEntitledTiers.Data) == 0 {
			continue
		}
		tier := integrations.HighestTier(&sub)
		if tier != integrations.TierNone && tier != integrations.TierFree {
			badges = append(badges, patreonBadge{
				PatreonUserID: sub.Relationships.User.Data.ID,
				BadgeCode:     tier.String(),
			})
		}
	}
	badgesBts, err := json.Marshal(badges)
	if err != nil {
		return err
	}

	tx, err := t.stores.DBPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := q.WithTx(tx)
	if err := qtx.BulkRemoveBadges(ctx, []string{"Chihuahua", "Dalmatian", "Golden Retriever"}); err != nil {
		return err
	}
	n, err := qtx.UpsertPatreonBadges(ctx, badgesBts)
	if err != nil {
		return err
	}
	log.Ctx(ctx).Info().Int("paid", len(badges)).Int64("rows", n).Msg("sub-badges-updated")
	return tx.Commit(ctx)
}

// LeagueHourly runs each active league's lifecycle: closing, preparing,
// opening registration, notices and starting seasons
// (league.RunLeagueLifecycleTasks). Each step checks what is done already,
// so a run cut off partway carries on in the next.
func (t *Tasks) LeagueHourly(ctx context.Context) error {
	leagues, err := t.stores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}
	clock := league.NewClockFromEnv()
	var errs []error
	for _, l := range leagues {
		if err := ctx.Err(); err != nil {
			return err
		}
		res, err := league.RunLeagueLifecycleTasks(ctx, t.cfg, t.stores, t.gameCreator, l.Uuid, clock)
		if err != nil {
			errs = append(errs, fmt.Errorf("league %s: %w", l.Slug, err))
			continue
		}
		log.Ctx(ctx).Info().
			Str("league", l.Slug).
			Int("tasksRun", res.TasksRun).
			Bool("registrationClosed", res.RegistrationClosed).
			Bool("divisionsPrepared", res.DivisionsPrepared).
			Bool("registrationOpened", res.RegistrationOpened).
			Bool("startingSoonNotification", res.StartingSoonNotification).
			Bool("seasonStarted", res.SeasonStarted).
			Int("gamesCreated", res.GamesCreated).
			Msg("league-lifecycle-done")
	}
	return errors.Join(errs...)
}

// LeagueUnstartedReminders writes reminders to players who haven't started
// their games: gentle 16-24 hours into a season, firm 40-48 hours in. Each
// player gets each reminder once, however often this runs.
func (t *Tasks) LeagueUnstartedReminders(ctx context.Context) error {
	leagues, err := t.stores.LeagueStore.GetAllLeagues(ctx, true)
	if err != nil {
		return err
	}
	clock := league.NewClockFromEnv()
	lifecycle := league.NewSeasonLifecycleManager(t.stores, clock)
	now := clock.Now()
	var errs []error
	for _, l := range leagues {
		season, err := t.stores.LeagueStore.GetCurrentSeason(ctx, l.Uuid)
		if err != nil {
			continue // no current season
		}
		if season.Status != int32(pbActive) {
			continue
		}
		hours := now.Sub(season.StartDate.Time).Hours()
		var firm bool
		switch {
		case hours >= 16 && hours < 24:
			firm = false
		case hours >= 40 && hours < 48:
			firm = true
		default:
			continue
		}
		if _, err := lifecycle.QueueUnstartedGameReminders(ctx, l.Uuid, season.Uuid, firm); err != nil {
			errs = append(errs, fmt.Errorf("league %s: %w", l.Slug, err))
		}
	}
	return errors.Join(errs...)
}

// ArchiveSweep archives finished games that still have turn rows and no S3
// archive: an upload that failed, or that never ran because the server
// stopped first. A game archived is no longer picked.
func (t *Tasks) ArchiveSweep(ctx context.Context) error {
	if t.stores.GameHistoryArchiver == nil {
		return errors.New("game history archiver is not configured")
	}
	ids, err := t.stores.Queries.ListStalePendingArchival(ctx, models.ListStalePendingArchivalParams{
		EndedBefore: timestamptz(time.Now().Add(-archiveSweepGrace)),
		MaxGames:    archiveSweepMaxGames,
	})
	if err != nil {
		return err
	}
	archived := 0
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return err
		}
		g, err := t.stores.GameStore.Get(ctx, id.String)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Str("gameID", id.String).Msg("archive-sweep-load-failed")
			continue
		}
		if err := t.stores.GameHistoryArchiver.ArchiveAndCleanup(ctx, g); err != nil {
			log.Ctx(ctx).Error().Err(err).Str("gameID", id.String).Msg("archive-sweep-failed")
			continue
		}
		archived++
	}
	log.Ctx(ctx).Info().Int("pending", len(ids)).Int("archived", archived).Msg("archive-sweep-done")
	return nil
}

// IntegrationsRefresh refreshes the Patreon creator token, and the titles of
// players linked to SG and ABSP.
func (t *Tasks) IntegrationsRefresh(ctx context.Context) error {
	q := t.stores.Queries
	patreonErr := t.refreshPatreonCreatorToken(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	// SG's titles list and ABSP's ratings list need no login, so their
	// linked players' titles are refreshed here, daily: a title gained or
	// lost shows without anyone pressing refresh.
	sgErr := organization.RefreshOrganizationTitles(ctx, q, organizations.OrgSG)
	abspErr := organization.RefreshOrganizationTitles(ctx, q, organizations.OrgABSP)
	// Every linked player's title by their username is picked again from the
	// stored titles, with nothing fetched, so that a change to the ranks or
	// the tie order reaches them without anyone pressing refresh.
	return errors.Join(patreonErr, sgErr, abspErr, organization.RepickProfileTitles(ctx, q))
}

// refreshPatreonCreatorToken keeps the creator token alive. Users' own
// tokens aren't stored: tier checks find their patreon_user_id in the
// campaign member list, which the creator token fetches.
func (t *Tasks) refreshPatreonCreatorToken(ctx context.Context) error {
	q := t.stores.Queries
	bts, err := q.GetExpiringGlobalPatreonIntegration(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var data integrations.PatreonTokenResponse
	if err := json.Unmarshal(bts, &data); err != nil {
		return err
	}
	tokres, err := t.oauth.RefreshPatreonToken(data.RefreshToken)
	if err != nil {
		return fmt.Errorf("global patreon token: %w", err)
	}
	tokresjson, err := json.Marshal(tokres)
	if err != nil {
		return err
	}
	if err := q.AddOrUpdateGlobalIntegration(ctx, models.AddOrUpdateGlobalIntegrationParams{
		IntegrationName: integrations.PatreonIntegrationName,
		Data:            tokresjson,
	}); err != nil {
		return err
	}
	log.Ctx(ctx).Info().Msg("patreon-creator-token-refreshed")
	return nil
}
