package notify

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

func Post(message string, token string) {
	requestBody, err := json.Marshal(map[string]string{"content": message})
	// Errors should not be fatal, just log them
	if err != nil {
		log.Err(err).Str("error", err.Error()).Msg("mod-action-discord-notification-marshal")
	} else {
		go func() {
			resp, err := http.Post(token, "application/json", bytes.NewBuffer(requestBody))
			// Errors should not be fatal, just log them
			if err != nil {
				log.Err(err).Str("error", err.Error()).Msg("notification-post-error")
			} else if resp.StatusCode != 204 { // No Content
				// We do not expect any other response
				log.Err(err).Str("status", resp.Status).Msg("notification-post-bad-response")
			}
		}()
	}
}
