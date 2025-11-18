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
	"github.com/woogles-io/liwords/pkg/user"
)

const (
	maxConcurrentEmails = 5                // Max simultaneous email sends
	emailSendDelay      = 2 * time.Second  // Delay between email launches (~1 every 2 seconds)
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
	if cfg.MailgunKey == "" {
		log.Debug().Msg("mailgun-key-not-set-skipping-league-email")
		return
	}

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
				cfg.MailgunKey,
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
	if cfg.MailgunKey == "" {
		log.Debug().Msg("mailgun-key-not-set-skipping-league-email")
		return
	}

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
				cfg.MailgunKey,
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
