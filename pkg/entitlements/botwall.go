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
	case "Double-Double Tier":
		return 20
	case "Bingo Tier":
		return 4
	case "Triple-Triple Tier":
		// Still add some limit.
		return 1000
	}
	return 0
}

func getChargePeriodStart(lastPaymentDate time.Time, now time.Time) (time.Time, error) {

	// Get the current year and month
	year, month := now.Year(), now.Month()

	// Adjust the charge period start to the same time on the same day of the current month
	chargePeriodStart := time.Date(year, month, lastPaymentDate.Day(),
		lastPaymentDate.Hour(), lastPaymentDate.Minute(), lastPaymentDate.Second(),
		lastPaymentDate.Nanosecond(), time.UTC)
	// Handle invalid dates (e.g., February 30th)
	if chargePeriodStart.Month() != month {
		chargePeriodStart = chargePeriodStart.AddDate(0, 0, -chargePeriodStart.Day())
	}
	return chargePeriodStart, nil
}

func paymentsUpToDate(lastPaymentDate time.Time, paymentUpToDate bool, now time.Time) (bool, string) {
	// Get the start of the current charge period
	currentChargePeriodStart, err := getChargePeriodStart(lastPaymentDate, now)
	if err != nil {
		return false, "Error calculating charge period"
	}

	// If today is before the next charge period
	if now.Before(currentChargePeriodStart) {
		// Check if the last payment date falls within the current charge period
		if !lastPaymentDate.Before(currentChargePeriodStart.AddDate(0, -1, 0)) {
			return true, "You can play games within your quota."
		}
		return false, "Your payment has expired. Please renew to continue playing."
	}

	// If today is in or after the next charge period
	if lastPaymentDate.Before(currentChargePeriodStart.AddDate(0, -1, 0)) {
		return false, "Your payment has expired. Please renew to continue playing."
	}

	// If payment is up-to-date
	if paymentUpToDate {
		return true, "You can play games within your quota."
	}

	return false, "Your payment is not up-to-date. Please renew to continue playing."
}

// For paywalling certain bots etc.
func EntitledToBestBot(ctx context.Context, queries *models.Queries, tierData *integrations.PaidTierData,
	userID uint, now time.Time) (bool, error) {
	log := log.Ctx(ctx)

	uptodate, msg := paymentsUpToDate(tierData.LastChargeDate, tierData.LastChargeStatus == integrations.ChargeStatusPaid, now)

	if !uptodate {
		log.Info().Str("uptodate-msg", msg).Uint("user-id", userID).Msg("not-up-to-date")
		return false, nil
	}

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
