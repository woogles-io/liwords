package league

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/emailer"
	"github.com/woogles-io/liwords/pkg/outbox"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
)

type LeagueEmailInfo struct {
	Username       string
	LeagueName     string
	SeasonNumber   int
	StartTime      string
	StartTimeZones string // Formatted string with multiple timezones
	LeagueURL      string
	DivisionName   string
	Opponents      []string
	IsStartingSoon bool
}

const LeagueEmailTemplateName = "league_email"

//go:embed league_email_templates
var LeagueEmailTemplate string

const RegistrationEmailTemplateName = "registration_email"

//go:embed league_registration_email_templates
var RegistrationEmailTemplate string

type RegistrationEmailInfo struct {
	Username                   string
	LeagueName                 string
	SeasonNumber               int
	LeagueURL                  string
	IsRegistrationOpen         bool
	IsRegistrationConfirmation bool
}

// formatTimeInTimezones converts a UTC time to multiple timezones and returns a formatted string
func formatTimeInTimezones(utcTime time.Time) string {
	timezones := []struct {
		name     string
		location string
	}{
		{"US Eastern", "America/New_York"},
		{"US Pacific", "America/Los_Angeles"},
		{"UK", "Europe/London"},
		{"Singapore", "Asia/Singapore"},
		{"India", "Asia/Kolkata"},
		{"Australia (Sydney)", "Australia/Sydney"},
	}

	var result string
	for _, tz := range timezones {
		loc, err := time.LoadLocation(tz.location)
		if err != nil {
			log.Warn().Err(err).Str("location", tz.location).Msg("failed to load timezone")
			continue
		}
		localTime := utcTime.In(loc)
		formatted := localTime.Format("Monday, January 2, 2006 at 3:04 PM MST")
		result += fmt.Sprintf("  • %s: %s\n", tz.name, formatted)
	}
	return result
}

// instantiateLeagueEmail creates the email content from the template
func instantiateLeagueEmail(info *LeagueEmailInfo) (string, string, error) {
	emailTemplate, err := template.New(LeagueEmailTemplateName).Parse(LeagueEmailTemplate)
	if err != nil {
		return "", "", err
	}

	emailContentBuffer := &bytes.Buffer{}
	err = emailTemplate.Execute(emailContentBuffer, info)
	if err != nil {
		return "", "", err
	}

	var emailSubject string
	if info.IsStartingSoon {
		emailSubject = fmt.Sprintf("%s Season %d Starts Tomorrow!", info.LeagueName, info.SeasonNumber)
	} else {
		emailSubject = fmt.Sprintf("%s Season %d Has Started!", info.LeagueName, info.SeasonNumber)
	}

	return emailContentBuffer.String(), emailSubject, nil
}

// noticeKey is the outbox key of a league notice: what it is, for which
// season, and for whom ("discord" for the league channel post).
func noticeKey(seasonID uuid.UUID, notice, who string) string {
	return fmt.Sprintf("league:%s:%s:%s", seasonID, notice, who)
}

// seasonStartingSoonMessages are the emails sent 1 day before a season
// starts. Players who can't be looked up, or have no email, are skipped.
func seasonStartingSoonMessages(ctx context.Context, userStore user.Store, seasonID uuid.UUID, leagueName, leagueSlug string, seasonNumber int, startTime time.Time, registeredUserIDs []string) ([]outbox.Message, error) {
	leagueURL := fmt.Sprintf("https://woogles.io/leagues/%s", leagueSlug)
	startTimeString := startTime.Format("Monday, January 2, 2006 at 3:04 PM MST")
	startTimeZones := formatTimeInTimezones(startTime)

	var msgs []outbox.Message
	for _, userID := range registeredUserIDs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		user, err := userStore.GetByUUID(ctx, userID)
		if err != nil {
			log.Err(err).Str("userID", userID).Msg("failed-to-fetch-user-for-league-email")
			continue
		}
		if user.Email == "" {
			log.Debug().Str("username", user.Username).Msg("user-has-no-email-skipping")
			continue
		}
		body, subject, err := instantiateLeagueEmail(&LeagueEmailInfo{
			Username:       user.Username,
			LeagueName:     leagueName,
			SeasonNumber:   seasonNumber,
			StartTime:      startTimeString,
			StartTimeZones: startTimeZones,
			LeagueURL:      leagueURL,
			IsStartingSoon: true,
		})
		if err != nil {
			log.Err(err).Str("username", user.Username).Msg("failed-to-instantiate-league-email")
			continue
		}
		msgs = append(msgs, outbox.Message{
			Key:       noticeKey(seasonID, "starting-soon", userID),
			Channel:   outbox.ChannelEmail,
			Recipient: user.Email,
			Subject:   subject,
			Body:      body,
		})
	}
	return msgs, nil
}

// seasonStartedMessages are the emails sent when a season starts, with each
// player's division and opponents, and the league Discord post.
func seasonStartedMessages(ctx context.Context, userStore user.Store, seasonID uuid.UUID, leagueName, leagueSlug string, seasonNumber int, playerAssignments map[string]*PlayerSeasonInfo) ([]outbox.Message, error) {
	leagueURL := fmt.Sprintf("https://woogles.io/leagues/%s", leagueSlug)

	var msgs []outbox.Message
	for userID, assignment := range playerAssignments {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		user, err := userStore.GetByUUID(ctx, userID)
		if err != nil {
			log.Err(err).Str("userID", userID).Msg("failed-to-fetch-user-for-league-email")
			continue
		}
		if user.Email == "" {
			log.Debug().Str("username", user.Username).Msg("user-has-no-email-skipping")
			continue
		}
		body, subject, err := instantiateLeagueEmail(&LeagueEmailInfo{
			Username:       user.Username,
			LeagueName:     leagueName,
			SeasonNumber:   seasonNumber,
			LeagueURL:      leagueURL,
			DivisionName:   assignment.DivisionName,
			Opponents:      assignment.OpponentNames,
			IsStartingSoon: false,
		})
		if err != nil {
			log.Err(err).Str("username", user.Username).Msg("failed-to-instantiate-league-email")
			continue
		}
		msgs = append(msgs, outbox.Message{
			Key:       noticeKey(seasonID, "started", userID),
			Channel:   outbox.ChannelEmail,
			Recipient: user.Email,
			Subject:   subject,
			Body:      body,
		})
	}
	msgs = append(msgs, outbox.Message{
		Key:     noticeKey(seasonID, "started", "discord"),
		Channel: outbox.ChannelLeagueDiscord,
		Body:    fmt.Sprintf("🏁 **%s Season %d** has officially started!\n\nGames have been created. Good luck to all players!\n\nhttps://woogles.io/leagues/%s", leagueName, seasonNumber, leagueSlug),
	})
	return msgs, nil
}

// PlayerSeasonInfo holds information about a player's season assignment
type PlayerSeasonInfo struct {
	DivisionName  string
	OpponentNames []string
}

// instantiateRegistrationEmail creates the registration email content from the template
func instantiateRegistrationEmail(info *RegistrationEmailInfo) (string, string, error) {
	emailTemplate, err := template.New(RegistrationEmailTemplateName).Parse(RegistrationEmailTemplate)
	if err != nil {
		return "", "", err
	}

	emailContentBuffer := &bytes.Buffer{}
	err = emailTemplate.Execute(emailContentBuffer, info)
	if err != nil {
		return "", "", err
	}

	var emailSubject string
	if info.IsRegistrationOpen {
		emailSubject = fmt.Sprintf("%s Season %d Registration Now Open!", info.LeagueName, info.SeasonNumber)
	} else if info.IsRegistrationConfirmation {
		emailSubject = fmt.Sprintf("Registration Confirmed: %s Season %d", info.LeagueName, info.SeasonNumber)
	} else {
		emailSubject = fmt.Sprintf("Unregistered from %s Season %d", info.LeagueName, info.SeasonNumber)
	}

	return emailContentBuffer.String(), emailSubject, nil
}

// registrationOpenMessages are the emails to current and previous season
// registrants when registration opens, and the league Discord post.
func registrationOpenMessages(ctx context.Context, userStore user.Store, seasonID uuid.UUID, leagueName, leagueSlug string, seasonNumber int, currentRegistrants []models.GetSeasonRegistrationsRow, previousRegistrants []models.GetPreviousSeasonRegistrantsNotInCurrentRow) ([]outbox.Message, error) {
	leagueURL := fmt.Sprintf("https://woogles.io/leagues/%s", leagueSlug)

	// Current registrants' emails are looked up; previous registrants come
	// with theirs.
	type recipient struct{ username, email string }
	recipients := map[string]recipient{}
	for _, reg := range currentRegistrants {
		if reg.UserUuid != "" {
			recipients[reg.UserUuid] = recipient{username: reg.Username}
		}
	}
	for _, prev := range previousRegistrants {
		if prev.Uuid != "" && prev.Email != "" {
			recipients[prev.Uuid] = recipient{username: prev.Username, email: prev.Email}
		}
	}

	var msgs []outbox.Message
	for userID, r := range recipients {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if r.email == "" {
			user, err := userStore.GetByUUID(ctx, userID)
			if err != nil {
				log.Err(err).Str("userID", userID).Msg("failed-to-fetch-user-for-registration-email")
				continue
			}
			r = recipient{username: user.Username, email: user.Email}
		}
		if r.email == "" {
			log.Debug().Str("username", r.username).Msg("user-has-no-email-skipping")
			continue
		}
		body, subject, err := instantiateRegistrationEmail(&RegistrationEmailInfo{
			Username:           r.username,
			LeagueName:         leagueName,
			SeasonNumber:       seasonNumber,
			LeagueURL:          leagueURL,
			IsRegistrationOpen: true,
		})
		if err != nil {
			log.Err(err).Str("username", r.username).Msg("failed-to-instantiate-registration-email")
			continue
		}
		msgs = append(msgs, outbox.Message{
			Key:       noticeKey(seasonID, "registration-open", userID),
			Channel:   outbox.ChannelEmail,
			Recipient: r.email,
			Subject:   subject,
			Body:      body,
		})
	}
	msgs = append(msgs, outbox.Message{
		Key:     noticeKey(seasonID, "registration-open", "discord"),
		Channel: outbox.ChannelLeagueDiscord,
		Body:    fmt.Sprintf("🎮 Registration is now open for **%s Season %d**!\n\nRegister now at: https://woogles.io/leagues/%s", leagueName, seasonNumber, leagueSlug),
	})
	return msgs, nil
}

// SendRegistrationConfirmationEmail sends individual email when user registers
func SendRegistrationConfirmationEmail(ctx context.Context, cfg *config.Config, email, username, leagueName, leagueSlug string, seasonNumber int) {
	leagueURL := fmt.Sprintf("https://woogles.io/leagues/%s", leagueSlug)

	emailInfo := &RegistrationEmailInfo{
		Username:                   username,
		LeagueName:                 leagueName,
		SeasonNumber:               seasonNumber,
		LeagueURL:                  leagueURL,
		IsRegistrationConfirmation: true,
	}

	emailContent, emailSubject, err := instantiateRegistrationEmail(emailInfo)
	if err != nil {
		log.Err(err).Str("username", username).Msg("failed-to-instantiate-registration-confirmation-email")
		return
	}

	go func() {
		_, err := emailer.SendSimpleMessage(
			cfg.EmailDebugMode,
			email,
			emailSubject,
			emailContent)
		if err != nil {
			log.Err(err).Str("username", username).Msg("failed-to-send-registration-confirmation-email")
		} else {
			log.Info().Str("username", username).Str("league", leagueName).Msg("sent-registration-confirmation-email")
		}
	}()
}

// SendUnregistrationConfirmationEmail sends individual email when user unregisters
func SendUnregistrationConfirmationEmail(ctx context.Context, cfg *config.Config, email, username, leagueName, leagueSlug string, seasonNumber int) {
	leagueURL := fmt.Sprintf("https://woogles.io/leagues/%s", leagueSlug)

	emailInfo := &RegistrationEmailInfo{
		Username:                   username,
		LeagueName:                 leagueName,
		SeasonNumber:               seasonNumber,
		LeagueURL:                  leagueURL,
		IsRegistrationOpen:         false,
		IsRegistrationConfirmation: false,
	}

	emailContent, emailSubject, err := instantiateRegistrationEmail(emailInfo)
	if err != nil {
		log.Err(err).Str("username", username).Msg("failed-to-instantiate-unregistration-email")
		return
	}

	go func() {
		_, err := emailer.SendSimpleMessage(
			cfg.EmailDebugMode,
			email,
			emailSubject,
			emailContent)
		if err != nil {
			log.Err(err).Str("username", username).Msg("failed-to-send-unregistration-email")
		} else {
			log.Info().Str("username", username).Str("league", leagueName).Msg("sent-unregistration-email")
		}
	}()
}

// unstartedGameReminderMessages are the reminders to players who haven't
// started their games: gentle on day 1, firm on day 2.
func unstartedGameReminderMessages(
	ctx context.Context,
	userStore user.Store,
	seasonID uuid.UUID,
	leagueName, leagueSlug string,
	seasonNumber int,
	playersWithUnstartedGames []models.GetSeasonPlayersWithUnstartedGamesRow,
	isFirm bool,
) ([]outbox.Message, error) {
	leagueURL := fmt.Sprintf("https://woogles.io/leagues/%s", leagueSlug)
	notice := "unstarted-gentle"
	if isFirm {
		notice = "unstarted-firm"
	}

	var msgs []outbox.Message
	for _, player := range playersWithUnstartedGames {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		u, err := userStore.GetByUUID(ctx, player.UserUuid)
		if err != nil {
			log.Err(err).Str("userUUID", player.UserUuid).Msg("failed-to-fetch-user-for-unstarted-game-reminder")
			continue
		}
		if u.Email == "" {
			log.Debug().Str("username", u.Username).Msg("user-has-no-email-skipping")
			continue
		}

		var emailBody, emailSubject string
		gameCount := player.UnstartedGameCount

		if isFirm {
			// Day 2: Firm warning
			emailSubject = fmt.Sprintf("⏰ %s Season %d - Please Start Your Games", leagueName, seasonNumber)
			emailBody = fmt.Sprintf(`Dear %s,

You still have %d unstarted game(s) in %s Season %d.

We understand that life gets busy, but please remember to make your first moves soon! The time bank system gives you plenty of time to finish all your games, but you need to start them to keep the league running smoothly for everyone.

Important: Players who time out in multiple games may be temporarily suspended from future league seasons to ensure a good experience for all participants.

Visit the league page to see your games and make your moves:
%s

Thank you for being part of our community!

Sincerely,
The Woogles Team
`, u.Username, gameCount, leagueName, seasonNumber, leagueURL)
		} else {
			// Day 1: Gentle reminder
			emailSubject = fmt.Sprintf("%s Season %d - Time to Start Your Games!", leagueName, seasonNumber)
			emailBody = fmt.Sprintf(`Dear %s,

We're excited that you're playing in %s Season %d!

You currently have %d game(s) waiting for your first move. Remember, with the time bank system, you have plenty of time to complete all your games even if you can't play every day. Making your first moves helps keep the league active and fun for everyone!

Visit the league page to see your games and get started:
%s

Good luck and have fun!

Sincerely,
The Woogles Team
`, u.Username, leagueName, seasonNumber, gameCount, leagueURL)
		}

		msgs = append(msgs, outbox.Message{
			Key:       noticeKey(seasonID, notice, player.UserUuid),
			Channel:   outbox.ChannelEmail,
			Recipient: u.Email,
			Subject:   emailSubject,
			Body:      emailBody,
		})
	}
	return msgs, nil
}
