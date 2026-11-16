package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

// Post sends message to a Discord webhook, logging any error.
func Post(message string, token string) {
	if err := Send(message, token); err != nil {
		log.Err(err).Msg("notification-post-error")
		return
	}
	log.Debug().Msg("discord-notification-sent-successfully")
}

// Send sends message to a Discord webhook (token is the webhook URL).
func Send(message string, token string) error {
	requestBody, err := json.Marshal(map[string]string{"content": message})
	if err != nil {
		return err
	}
	resp, err := http.Post(token, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("discord webhook: %s", resp.Status)
	}
	return nil
}
