package emailer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/mailgun/mailgun-go/v4"
	"github.com/rs/zerolog/log"
)

// Send emails using mailgun.

const domain = "mg.woogles.io"
const WooglesAdministratorAddress = "woogles@woogles.io"

func SendSimpleMessage(apiKey, recipient, subject, body string) (string, error) {
	if apiKey == "default" {
		// Print the email to stdout for debugging purposes
		fmt.Printf("=== EMAIL DEBUG ===\nTo: %s\nSubject: %s\nBody:\n%s\n=== END EMAIL ===\n", recipient, subject, body)
		return "", nil // Skip sending if using default key
	}

	mg := mailgun.NewMailgun(domain, apiKey)
	m := mailgun.NewMessage(
		"Woogles <woogles@"+domain+">",
		subject,
		body,
		recipient,
	)

	// Retry with exponential backoff for rate limit errors
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			log.Warn().
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Str("recipient", recipient).
				Msg("retrying-email-send-after-rate-limit")
			time.Sleep(backoff)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		_, id, err := mg.Send(ctx, m)
		cancel()

		if err == nil {
			return id, nil
		}

		// Check if it's a rate limit error (429 Too Many Requests)
		var respErr *mailgun.UnexpectedResponseError
		if errors.As(err, &respErr) && respErr.Actual == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limit exceeded (429): %w", err)
			log.Warn().
				Int("attempt", attempt+1).
				Str("recipient", recipient).
				Msg("mailgun-rate-limit-detected")
			continue // Retry
		}

		// Other errors, don't retry
		return "", err
	}

	// All retries exhausted
	return "", fmt.Errorf("failed after 3 attempts: %w", lastErr)
}
