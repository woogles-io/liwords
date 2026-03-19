package league

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"sync"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/emailer"
	"github.com/woogles-io/liwords/pkg/notify"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
)

const (
	maxConcurrentEmails = 5               // Max simultaneous email sends
	emailSendDelay      = 2 * time.Second // Delay between email launches (~1 every 2 seconds)
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

// SendSeasonStartingSoonEmail sends reminder email 1 day before season starts
func SendSeasonStartingSoonEmail(ctx context.Context, cfg *config.Config, userStore user.Store, leagueName, leagueSlug string, seasonNumber int, startTime time.Time, registeredUserIDs []string) {

	leagueURL := fmt.Sprintf("https://woogles.io/leagues/%s", leagueSlug)
	startTimeString := startTime.Format("Monday, January 2, 2006 at 3:04 PM MST")
	startTimeZones := formatTimeInTimezones(startTime)

	// Semaphore for concurrency control and WaitGroup to wait for completion
	sem := make(chan struct{}, maxConcurrentEmails)
	var wg sync.WaitGroup

	for _, userID := range registeredUserIDs {
		// Fetch user details
		user, err := userStore.GetByUUID(ctx, userID)
		if err != nil {
			log.Err(err).Str("userID", userID).Msg("failed-to-fetch-user-for-league-email")
			continue
		}

		if user.Email == "" {
			log.Debug().Str("username", user.Username).Msg("user-has-no-email-skipping")
			continue
		}

		emailInfo := &LeagueEmailInfo{
			Username:       user.Username,
			LeagueName:     leagueName,
			SeasonNumber:   seasonNumber,
			StartTime:      startTimeString,
			StartTimeZones: startTimeZones,
			LeagueURL:      leagueURL,
			IsStartingSoon: true,
		}

		emailContent, emailSubject, err := instantiateLeagueEmail(emailInfo)
		if err != nil {
			log.Err(err).Str("username", user.Username).Msg("failed-to-instantiate-league-email")
			continue
		}

		// Acquire semaphore slot
		wg.Add(1)
		sem <- struct{}{}

		// Send email asynchronously with rate limiting
		go func(email, subject, body, username string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			_, err := emailer.SendSimpleMessage(
				cfg.EmailDebugMode,
				email,
				subject,
				body)
			if err != nil {
				log.Err(err).Str("username", username).Msg("failed-to-send-season-starting-soon-email")
			} else {
				log.Info().Str("username", username).Str("league", leagueName).Msg("sent-season-starting-soon-email")
			}
		}(user.Email, emailSubject, emailContent, user.Username)

		// Rate limit: delay before launching next goroutine
		time.Sleep(emailSendDelay)
	}

	// Wait for all emails to complete
	wg.Wait()
	log.Info().Int("count", len(registeredUserIDs)).Str("league", leagueName).Msg("completed-sending-season-starting-soon-emails")
}

// SendSeasonStartedEmail sends notification when season starts and games are created
func SendSeasonStartedEmail(ctx context.Context, cfg *config.Config, userStore user.Store, leagueName, leagueSlug string, seasonNumber int, playerAssignments map[string]*PlayerSeasonInfo) {

	leagueURL := fmt.Sprintf("https://woogles.io/leagues/%s", leagueSlug)

	// Semaphore for concurrency control and WaitGroup to wait for completion
	sem := make(chan struct{}, maxConcurrentEmails)
	var wg sync.WaitGroup

	for userID, assignment := range playerAssignments {
		// Fetch user details
		user, err := userStore.GetByUUID(ctx, userID)
		if err != nil {
			log.Err(err).Str("userID", userID).Msg("failed-to-fetch-user-for-league-email")
			continue
		}

		if user.Email == "" {
			log.Debug().Str("username", user.Username).Msg("user-has-no-email-skipping")
			continue
		}

		emailInfo := &LeagueEmailInfo{
			Username:       user.Username,
			LeagueName:     leagueName,
			SeasonNumber:   seasonNumber,
			LeagueURL:      leagueURL,
			DivisionName:   assignment.DivisionName,
			Opponents:      assignment.OpponentNames,
			IsStartingSoon: false,
		}

		emailContent, emailSubject, err := instantiateLeagueEmail(emailInfo)
		if err != nil {
			log.Err(err).Str("username", user.Username).Msg("failed-to-instantiate-league-email")
			continue
		}

		// Acquire semaphore slot
		wg.Add(1)
		sem <- struct{}{}

		// Send email asynchronously with rate limiting
		go func(email, subject, body, username string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			_, err := emailer.SendSimpleMessage(
				cfg.EmailDebugMode,
				email,
				subject,
				body)
			if err != nil {
				log.Err(err).Str("username", username).Msg("failed-to-send-season-started-email")
			} else {
				log.Info().Str("username", username).Str("league", leagueName).Msg("sent-season-started-email")
			}
		}(user.Email, emailSubject, emailContent, user.Username)

		// Rate limit: delay before launching next goroutine
		time.Sleep(emailSendDelay)
	}

	// Wait for all emails to complete
	wg.Wait()
	log.Info().Int("count", len(playerAssignments)).Str("league", leagueName).Msg("completed-sending-season-started-emails")
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

// SendRegistrationOpenEmail sends bulk email to current + previous season registrants
func SendRegistrationOpenEmail(ctx context.Context, cfg *config.Config, userStore user.Store, leagueName, leagueSlug string, seasonNumber int, currentRegistrants []models.GetSeasonRegistrationsRow, previousRegistrants []models.GetPreviousSeasonRegistrantsNotInCurrentRow) {
	leagueURL := fmt.Sprintf("https://woogles.io/leagues/%s", leagueSlug)

	// Combine all recipients (current + previous) into a map to deduplicate
	recipients := make(map[string]struct {
		username string
		email    string
	})

	// Add current registrants
	for _, reg := range currentRegistrants {
		if reg.UserUuid.Valid {
			recipients[reg.UserUuid.String] = struct {
				username string
				email    string
			}{username: reg.Username.String, email: ""} // Will fetch email from user store
		}
	}

	// Add previous registrants (already filtered to not include current)
	for _, prev := range previousRegistrants {
		if prev.Uuid.Valid && prev.Email.Valid {
			recipients[prev.Uuid.String] = struct {
				username string
				email    string
			}{username: prev.Username.String, email: prev.Email.String}
		}
	}

	log.Info().
		Int("total_recipients", len(recipients)).
		Str("league", leagueName).
		Int("season", seasonNumber).
		Msg("sending-registration-open-emails")

	// Semaphore for concurrency control and WaitGroup to wait for completion
	sem := make(chan struct{}, maxConcurrentEmails)
	var wg sync.WaitGroup

	for userID, recipient := range recipients {
		// Fetch user details if email not already available
		var email string
		var username string
		if recipient.email != "" {
			email = recipient.email
			username = recipient.username
		} else {
			user, err := userStore.GetByUUID(ctx, userID)
			if err != nil {
				log.Err(err).Str("userID", userID).Msg("failed-to-fetch-user-for-registration-email")
				continue
			}
			email = user.Email
			username = user.Username
		}

		if email == "" {
			log.Debug().Str("username", username).Msg("user-has-no-email-skipping")
			continue
		}

		emailInfo := &RegistrationEmailInfo{
			Username:           username,
			LeagueName:         leagueName,
			SeasonNumber:       seasonNumber,
			LeagueURL:          leagueURL,
			IsRegistrationOpen: true,
		}

		emailContent, emailSubject, err := instantiateRegistrationEmail(emailInfo)
		if err != nil {
			log.Err(err).Str("username", username).Msg("failed-to-instantiate-registration-email")
			continue
		}

		// Acquire semaphore slot
		wg.Add(1)
		sem <- struct{}{}

		// Send email asynchronously with rate limiting
		go func(email, subject, body, username string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			_, err := emailer.SendSimpleMessage(
				cfg.EmailDebugMode,
				email,
				subject,
				body)
			if err != nil {
				log.Err(err).Str("username", username).Msg("failed-to-send-registration-open-email")
			} else {
				log.Info().Str("username", username).Str("league", leagueName).Msg("sent-registration-open-email")
			}
		}(email, emailSubject, emailContent, username)

		// Rate limit: delay before launching next goroutine
		time.Sleep(emailSendDelay)
	}

	// Wait for all emails to complete
	wg.Wait()
	log.Info().Int("count", len(recipients)).Str("league", leagueName).Msg("completed-sending-registration-open-emails")
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

// PostLeagueDiscordNotification posts a message to the league Discord channel
func PostLeagueDiscordNotification(message string, cfg *config.Config) {
	if cfg.LeagueDiscordToken != "" {
		notify.Post(message, cfg.LeagueDiscordToken)
	}
}

// SendRegistrationOpenDiscord posts to #woogleague when registration opens
func SendRegistrationOpenDiscord(cfg *config.Config, leagueName, leagueSlug string, seasonNumber int) {
	message := fmt.Sprintf("🎮 Registration is now open for **%s Season %d**!\n\nRegister now at: https://woogles.io/leagues/%s", leagueName, seasonNumber, leagueSlug)
	PostLeagueDiscordNotification(message, cfg)
	log.Info().Str("league", leagueName).Int("season", seasonNumber).Msg("posted-registration-open-discord")
}

// SendSeasonStartedDiscord posts to #woogleague when season starts
func SendSeasonStartedDiscord(cfg *config.Config, leagueName, leagueSlug string, seasonNumber int) {
	message := fmt.Sprintf("🏁 **%s Season %d** has officially started!\n\nGames have been created. Good luck to all players!\n\nhttps://woogles.io/leagues/%s", leagueName, seasonNumber, leagueSlug)
	PostLeagueDiscordNotification(message, cfg)
	log.Info().Str("league", leagueName).Int("season", seasonNumber).Msg("posted-season-started-discord")
}

// SendUnstartedGameReminderEmail sends reminders to players who haven't started their games
func SendUnstartedGameReminderEmail(
	ctx context.Context,
	cfg *config.Config,
	userStore user.Store,
	leagueName, leagueSlug string,
	seasonNumber int,
	playersWithUnstartedGames []models.GetSeasonPlayersWithUnstartedGamesRow,
	isFirm bool,
) {
	leagueURL := fmt.Sprintf("https://woogles.io/leagues/%s", leagueSlug)
	reminderType := "gentle"
	if isFirm {
		reminderType = "firm"
	}

	log.Info().
		Int("count", len(playersWithUnstartedGames)).
		Str("league", leagueName).
		Str("type", reminderType).
		Msg("sending-unstarted-game-reminder-emails")

	// Semaphore for concurrency control and WaitGroup to wait for completion
	sem := make(chan struct{}, maxConcurrentEmails)
	var wg sync.WaitGroup

	for _, player := range playersWithUnstartedGames {
		// Fetch user details
		u, err := userStore.GetByUUID(ctx, player.UserUuid.String)
		if err != nil {
			log.Err(err).Str("userUUID", player.UserUuid.String).Msg("failed-to-fetch-user-for-unstarted-game-reminder")
			continue
		}

		if u.Email == "" {
			log.Debug().Str("username", u.Username).Msg("user-has-no-email-skipping")
			continue
		}

		// Prepare email content
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

		// Acquire semaphore slot
		wg.Add(1)
		sem <- struct{}{}

		// Send email asynchronously with rate limiting
		go func(email, subject, body, username string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			_, err := emailer.SendSimpleMessage(cfg.EmailDebugMode, email, subject, body)
			if err != nil {
				log.Err(err).Str("username", username).Msg("failed-to-send-unstarted-game-reminder-email")
			} else {
				log.Info().Str("username", username).Str("league", leagueName).Str("type", reminderType).Msg("sent-unstarted-game-reminder-email")
			}
		}(u.Email, emailSubject, emailBody, u.Username)

		// Rate limit: delay before launching next goroutine
		time.Sleep(emailSendDelay)
	}

	// Wait for all emails to complete
	wg.Wait()
	log.Info().Int("count", len(playersWithUnstartedGames)).Str("league", leagueName).Str("type", reminderType).Msg("completed-sending-unstarted-game-reminder-emails")
}
