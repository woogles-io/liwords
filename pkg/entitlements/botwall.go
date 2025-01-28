package entitlements

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/integrations"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

var BestBotUserID int

func init() {
	var err error
	BestBotUserID, err = strconv.Atoi(os.Getenv("BEST_BOT_USERID"))
	if err != nil {
		// could consider panicking.
		log.Err(err).Msg("best-bot-user-id-not-defined")
	}
}

func entitledBestBotGamesFor(tiername string) int {
	switch tiername {
	case "Dalmatian":
		return 50
	case "Chihuahua":
		return 4
	case "Golden Retriever":
		// Still add some limit.
		return 500
	}
	return 0
}

// For paywalling certain bots etc.
func EntitledToBestBot(ctx context.Context, queries *models.Queries, tierData *integrations.PaidTierData,
	userID uint, now time.Time) (bool, error) {
	log := log.Ctx(ctx)

	// assume the last charge date is within the last charge period. If this
	// function is called, it should be called with an active subscription
	// (at least, according to Patreon's currently_entitled_tiers)

	bbGamesPlayedThisPeriod, err := queries.GetNumberOfBotGames(ctx, models.GetNumberOfBotGamesParams{
		BotID:       pgtype.Int4{Int32: int32(BestBotUserID), Valid: true},
		UserID:      pgtype.Int4{Int32: int32(userID), Valid: true},
		CreatedDate: pgtype.Timestamptz{Valid: true, Time: tierData.LastChargeDate},
	})
	if err != nil {
		return false, err
	}
	log.Info().Time("last-charge", tierData.LastChargeDate).Uint("user-id", userID).
		Int64("bestbot-games", bbGamesPlayedThisPeriod).Msg("number-of-bot-games")

	if int(bbGamesPlayedThisPeriod) >= entitledBestBotGamesFor(tierData.TierName) {
		log.Info().Str("tierdata", tierData.TierName).Msg("not-entitled")
		return false, nil
	}

	return true, nil
}
