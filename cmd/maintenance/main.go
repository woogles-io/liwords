package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/integrations"
	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/config_service"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const BlogURL = "https://blog.woogles.io"

const (
	BlogRSSFeedURL   = BlogURL + "/index.xml"
	BlogSearchString = BlogURL
)

var AdminAPIKey = os.Getenv("ADMIN_API_KEY")
var WooglesAPIBasePath = os.Getenv("WOOGLES_API_BASE_PATH")

// A set of maintenance functions on Woogles that can run at some given
// cadence.

// go run . blogrss-updater,foo,bar,baz
// Use LEAGUE_NOW environment variable to override current time for testing:
//
//	LEAGUE_NOW="2025-01-15T08:00:00Z" go run . league-midnight-runner
func main() {
	if len(os.Args) < 2 {
		panic("need at least one command")
	}

	commands := strings.Split(os.Args[1], ",")

	log.Info().Interface("commands", commands).Msg("starting maintenance")

	for _, command := range commands {
		switch strings.ToLower(command) {
		case "blogrss-updater":
			err := BlogRssUpdater()
			log.Err(err).Msg("ran blogRssUpdater")
		case "integrations-refresher":
			err := IntegrationsRefresher()
			log.Err(err).Msg("ran integrationsRefresher")
		case "sub-badge-updater":
			err := SubBadgeUpdater()
			log.Err(err).Msg("ran subBadgeUpdater")
		case "cancelled-games-cleanup":
			err := CancelledGamesCleanup()
			log.Err(err).Msg("ran cancelledGamesCleanup")
		case "monitoring-streams-cleanup":
			err := MonitoringStreamsCleanup()
			log.Err(err).Msg("ran monitoringStreamsCleanup")
		case "league-hourly-runner":
			err := LeagueHourlyRunner()
			log.Err(err).Msg("ran leagueHourlyRunner")
		case "league-unstarted-game-reminder":
			err := LeagueUnstartedGameReminder()
			log.Err(err).Msg("ran leagueUnstartedGameReminder")
		case "unverified-users-cleanup":
			err := UnverifiedUsersCleanup()
			log.Err(err).Msg("ran unverifiedUsersCleanup")
		case "expired-sessions-cleanup":
			err := ExpiredSessionsCleanup()
			log.Err(err).Msg("ran expiredSessionsCleanup")
		case "resend-season-started-emails":
			err := ResendSeasonStartedEmails()
			log.Err(err).Msg("ran resendSeasonStartedEmails")
		default:
			log.Error().Str("command", command).Msg("command not recognized")
		}
	}
}

func WooglesAPIRequest(service, rpc string, bts []byte) (*http.Response, error) {
	path, err := url.JoinPath(WooglesAPIBasePath, service, rpc)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", path, bytes.NewReader(bts))
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-Api-Key", AdminAPIKey)
	req.Header.Add("Content-Type", "application/json")
	return http.DefaultClient.Do(req)

}

// BlogRssUpdater updates the announcements homepage if a new blog post is found
// It subscribes to our blog RSS feed.
func BlogRssUpdater() error {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(BlogRSSFeedURL)
	if err != nil {
		return err
	}
	if len(feed.Items) < 1 {
		return errors.New("unexpected feed length")
	}
	authors := feed.Items[0].Authors
	authorsArr := make([]string, 0, len(authors))
	for _, a := range authors {
		authorsArr = append(authorsArr, a.Name)
	}
	authorsPrint := strings.Join(authorsArr, ", ")
	emoji := ""

	switch {
	case strings.Contains(feed.Items[0].Link, "/posts/"):
		emoji = "✏️"
	case strings.Contains(feed.Items[0].Link, "/guides/"):
		emoji = "📜"
	case strings.Contains(feed.Items[0].Link, "/articles/"):
		emoji = "📰"
	}

	img := feed.Items[0].Custom["image"]

	annobody := feed.Items[0].Description + " (Click to read more)"
	if img != "" {
		annobody = "![Image](" + img + ")\n\n" + annobody
	}
	b := &pb.SetSingleAnnouncementRequest{
		Announcement: &pb.Announcement{
			Title: emoji + " " + feed.Items[0].Title + " - written by " + authorsPrint,
			Body:  annobody,
			Link:  feed.Items[0].Link},
		LinkSearchString: BlogSearchString,
	}

	bts, err := protojson.Marshal(b)
	if err != nil {
		return err
	}
	resp, err := WooglesAPIRequest(
		"config_service.ConfigService",
		"SetSingleAnnouncement",
		bts)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Info().Str("body", string(body)).Msg("received")

	return nil
}

func IntegrationsRefresher() error {
	log.Info().Msg("before load")
	cfg := &config.Config{}
	log.Info().Msg("after cfg")
	cfg.Load(nil)
	log.Info().Msg("after load")
	log.Info().Interface("config", cfg).Msg("started")

	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	log.Debug().Msg("debug log is on")

	dbCfg, err := pgxpool.ParseConfig(cfg.DBConnUri)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	dbPool, err := pgxpool.NewWithConfig(ctx, dbCfg)

	q := models.New(dbPool)

	oauthIntegrationService := integrations.NewOAuthIntegrationService(nil, q, cfg)

	refreshPatreonIntegrationTokens(ctx, q, oauthIntegrationService)

	return nil
}

func SubBadgeUpdater() error {
	log.Info().Msg("before load")
	cfg := &config.Config{}
	log.Info().Msg("after cfg")
	cfg.Load(nil)
	log.Info().Msg("after load")
	log.Info().Interface("config", cfg).Msg("started")

	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	log.Debug().Msg("debug log is on")

	dbCfg, err := pgxpool.ParseConfig(cfg.DBConnUri)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	dbPool, err := pgxpool.NewWithConfig(ctx, dbCfg)
	if err != nil {
		panic(err)
	}
	q := models.New(dbPool)

	return updateBadges(q, dbPool)
}

func refreshPatreonIntegrationTokens(ctx context.Context, q *models.Queries, svc *integrations.OAuthIntegrationService) {
	expiringIntegrations, err := q.GetExpiringPatreonIntegrations(ctx)
	if err != nil {
		panic(err)
	}
	for _, integration := range expiringIntegrations {
		refreshPatreonToken(ctx, q, integration, svc)
		time.Sleep(2 * time.Second)
	}
	bts, err := q.GetExpiringGlobalPatreonIntegration(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Info().Msg("no global patreon integration to refresh")
			return
		}
		panic(err)
	}
	var integrationData integrations.PatreonTokenResponse

	if err := json.Unmarshal(bts, &integrationData); err != nil {
		log.Err(err).Msg("failed to unmarshal integration data")
		return
	}

	tokres, err := svc.RefreshPatreonToken(integrationData.RefreshToken)
	if err != nil {
		log.Err(err).Msg("failed to refresh global patreon token")
		return
	}
	tokresjson, err := json.Marshal(tokres)
	if err != nil {
		panic(err)
	}

	err = q.AddOrUpdateGlobalIntegration(ctx, models.AddOrUpdateGlobalIntegrationParams{
		IntegrationName: integrations.PatreonIntegrationName,
		Data:            tokresjson,
	})
	if err != nil {
		log.Err(err).Msg("failed to update global patreon integration data")
		return
	} else {
		log.Info().Msg("refreshed and saved global integration token")
	}
}

func refreshPatreonToken(ctx context.Context, q *models.Queries, integration models.GetExpiringPatreonIntegrationsRow, svc *integrations.OAuthIntegrationService) {
	var integrationData integrations.PatreonTokenResponse

	if err := json.Unmarshal(integration.Data, &integrationData); err != nil {
		log.Err(err).Msg("failed to unmarshal integration data")
		return
	}

	tokres, err := svc.RefreshPatreonToken(integrationData.RefreshToken)
	if err != nil {
		log.Err(err).Str("integration-uuid", integration.Uuid.String()).Msg("failed to refresh patreon token")
		return
	}
	tokresjson, err := json.Marshal(tokres)
	if err != nil {
		panic(err)
	}

	err = q.UpdateIntegrationData(ctx, models.UpdateIntegrationDataParams{
		Uuid: integration.Uuid,
		Data: tokresjson,
	})
	if err != nil {
		log.Err(err).Str("integration-uuid", integration.Uuid.String()).Msg("failed to update integration data")
		return
	} else {
		log.Info().Str("integration-uuid", integration.Uuid.String()).Msg("refreshed and saved token")
	}
}

type PatreonBadge struct {
	PatreonUserID string `json:"patreon_user_id"`
	BadgeCode     string `json:"badge_code"`
}

// CancelledGamesCleanup deletes cancelled games older than 2 days
func CancelledGamesCleanup() error {
	log.Info().Msg("starting cancelled games cleanup")
	cfg := &config.Config{}
	cfg.Load(nil)

	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	dbCfg, err := pgxpool.ParseConfig(cfg.DBConnUri)
	if err != nil {
		return err
	}

	ctx := context.Background()
	dbPool, err := pgxpool.NewWithConfig(ctx, dbCfg)
	if err != nil {
		return err
	}
	defer dbPool.Close()

	// Delete cancelled games older than 2 days
	// game_end_reason = 7 means CANCELLED
	query := `
		DELETE FROM games
		WHERE game_end_reason = 7
		  AND created_at < NOW() - INTERVAL '2 days'
	`

	result, err := dbPool.Exec(ctx, query)
	if err != nil {
		return err
	}

	rowsDeleted := result.RowsAffected()
	log.Info().Int64("rowsDeleted", rowsDeleted).Msg("deleted old cancelled games")

	return nil
}

func updateBadges(q *models.Queries, pool *pgxpool.Pool) error {
	ctx := context.Background()

	data, err := q.GetGlobalIntegrationData(ctx, integrations.PatreonIntegrationName)
	if err != nil {
		return err
	}

	// Get all currently entitled users according to Patreon.
	var integrationData integrations.PatreonTokenResponse
	if err := json.Unmarshal(data, &integrationData); err != nil {
		return err
	}

	subscribers, err := integrations.GetCampaignSubscribers(ctx, integrationData.AccessToken)
	if err != nil {
		return err
	}

	subsWithTier := map[string]integrations.Tier{}
	for _, sub := range subscribers.Data {
		if len(sub.Relationships.CurrentlyEntitledTiers.Data) > 0 {
			tier := integrations.HighestTier(&sub)
			if tier != integrations.TierNone && tier != integrations.TierFree {
				subsWithTier[sub.Relationships.User.Data.ID] = tier
			}
		}
	}

	log.Debug().Int("num-paid-subscriptions", len(subsWithTier)).
		Interface("subs-with-tier", subsWithTier).Msg("subscribers")

	badges := make([]PatreonBadge, 0, len(subsWithTier))

	for patreonUserID, tierID := range subsWithTier {
		badgeCode := integrations.Tier(tierID).String()
		badges = append(badges, PatreonBadge{
			PatreonUserID: patreonUserID,
			BadgeCode:     badgeCode,
		})
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}

	badgesBts, err := json.Marshal(badges)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)
	qtx := q.WithTx(tx)

	err = qtx.BulkRemoveBadges(ctx, []string{"Chihuahua", "Dalmatian", "Golden Retriever"})
	if err != nil {
		return err
	}

	rowsAffected, err := qtx.UpsertPatreonBadges(ctx, badgesBts)
	if err != nil {
		return err
	}
	log.Info().Int64("rowsAffected", rowsAffected).Msg("affected-rows")

	return tx.Commit(ctx)
}

// MonitoringStreamsCleanup deletes old monitoring stream keys
// Removes streams from finished tournaments and streams older than 1 month
func MonitoringStreamsCleanup() error {
	log.Info().Msg("starting monitoring streams cleanup")
	cfg := &config.Config{}
	cfg.Load(nil)

	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	dbCfg, err := pgxpool.ParseConfig(cfg.DBConnUri)
	if err != nil {
		return err
	}

	ctx := context.Background()
	dbPool, err := pgxpool.NewWithConfig(ctx, dbCfg)
	if err != nil {
		return err
	}
	defer dbPool.Close()

	// Delete monitoring streams for finished tournaments
	queryFinished := `
		DELETE FROM monitoring_streams
		WHERE tournament_id IN (
			SELECT uuid FROM tournaments WHERE is_finished = true
		)
	`
	resultFinished, err := dbPool.Exec(ctx, queryFinished)
	if err != nil {
		return err
	}
	rowsDeletedFinished := resultFinished.RowsAffected()
	log.Info().Int64("rowsDeleted", rowsDeletedFinished).Msg("deleted monitoring streams for finished tournaments")

	// Delete monitoring streams older than 1 month
	queryOld := `
		DELETE FROM monitoring_streams
		WHERE created_at < NOW() - INTERVAL '1 month'
	`
	resultOld, err := dbPool.Exec(ctx, queryOld)
	if err != nil {
		return err
	}
	rowsDeletedOld := resultOld.RowsAffected()
	log.Info().Int64("rowsDeleted", rowsDeletedOld).Msg("deleted monitoring streams older than 1 month")

	totalDeleted := rowsDeletedFinished + rowsDeletedOld
	log.Info().Int64("totalDeleted", totalDeleted).Msg("monitoring streams cleanup complete")

	return nil
}

// UnverifiedUsersCleanup deletes users who haven't verified their email after 48 hours
func UnverifiedUsersCleanup() error {
	log.Info().Msg("starting unverified users cleanup")
	cfg := &config.Config{}
	cfg.Load(nil)

	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	dbCfg, err := pgxpool.ParseConfig(cfg.DBConnUri)
	if err != nil {
		return err
	}

	ctx := context.Background()
	dbPool, err := pgxpool.NewWithConfig(ctx, dbCfg)
	if err != nil {
		return err
	}
	defer dbPool.Close()

	// Begin transaction
	tx, err := dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete followings first (foreign key constraint)
	// Remove both where unverified users are being followed AND where they're following others
	queryFollowings := `
		DELETE FROM followings
		WHERE user_id IN (
			SELECT id FROM users
			WHERE verified = false AND created_at < NOW() - INTERVAL '48 hours'
		)
		OR follower_id IN (
			SELECT id FROM users
			WHERE verified = false AND created_at < NOW() - INTERVAL '48 hours'
		)
	`
	resultFollowings, err := tx.Exec(ctx, queryFollowings)
	if err != nil {
		return err
	}
	followingsDeleted := resultFollowings.RowsAffected()
	log.Info().Int64("followingsDeleted", followingsDeleted).Msg("deleted unverified user followings")

	// Delete profiles (foreign key constraint)
	queryProfiles := `
		DELETE FROM profiles
		WHERE user_id IN (
			SELECT id FROM users
			WHERE verified = false AND created_at < NOW() - INTERVAL '48 hours'
		)
	`
	resultProfiles, err := tx.Exec(ctx, queryProfiles)
	if err != nil {
		return err
	}
	profilesDeleted := resultProfiles.RowsAffected()
	log.Info().Int64("profilesDeleted", profilesDeleted).Msg("deleted unverified user profiles")

	// Delete users
	queryUsers := `
		DELETE FROM users
		WHERE verified = false AND created_at < NOW() - INTERVAL '48 hours'
	`
	resultUsers, err := tx.Exec(ctx, queryUsers)
	if err != nil {
		return err
	}
	usersDeleted := resultUsers.RowsAffected()
	log.Info().Int64("usersDeleted", usersDeleted).Msg("deleted unverified users")

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	log.Info().
		Int64("followingsDeleted", followingsDeleted).
		Int64("profilesDeleted", profilesDeleted).
		Int64("usersDeleted", usersDeleted).
		Msg("unverified users cleanup complete")

	return nil
}

// ExpiredSessionsCleanup deletes sessions that have already expired
func ExpiredSessionsCleanup() error {
	log.Info().Msg("starting expired sessions cleanup")
	cfg := &config.Config{}
	cfg.Load(nil)

	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	dbCfg, err := pgxpool.ParseConfig(cfg.DBConnUri)
	if err != nil {
		return err
	}

	ctx := context.Background()
	dbPool, err := pgxpool.NewWithConfig(ctx, dbCfg)
	if err != nil {
		return err
	}
	defer dbPool.Close()

	// Delete sessions that have expired
	query := `
		DELETE FROM db_sessions
		WHERE expires_at < NOW()
	`
	result, err := dbPool.Exec(ctx, query)
	if err != nil {
		return err
	}

	rowsDeleted := result.RowsAffected()
	log.Info().Int64("rowsDeleted", rowsDeleted).Msg("deleted expired sessions")

	return nil
}
