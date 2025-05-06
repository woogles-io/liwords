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
func main() {
	if len(os.Args) < 2 {
		panic("need one comma-separated list of commands")
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
		case "cancelled-game-deleter":
			err := CancelledGameDeleter()
			log.Err(err).Msg("ran cancelledGameDeleter")
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
	cfg.Load(os.Args[1:])
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
	cfg.Load(os.Args[1:])
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

func CancelledGameDeleter() error {
	log.Info().Msg("before load")
	cfg := &config.Config{}
	log.Info().Msg("after cfg")
	cfg.Load(os.Args[1:])
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

	query := `
		DELETE FROM games
		WHERE game_end_reason = 7 -- cancelled games
		AND created_at < NOW() - INTERVAL '1 day'
		RETURNING id;
	`

	rows, err := dbPool.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	deletedGameIDs := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		deletedGameIDs = append(deletedGameIDs, id)
	}

	log.Info().Int("num_deleted", len(deletedGameIDs)).Ints64("deleted_game_ids", deletedGameIDs).Msg("deleted cancelled games")
	return nil

}
