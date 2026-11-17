package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var client = &http.Client{Timeout: 15 * time.Second}

// Post sends message to a Discord webhook, logging any error.
func Post(message string, token string) {
	if err := Send(context.Background(), message, token); err != nil {
		log.Err(err).Msg("notification-post-error")
		return
	}
	log.Debug().Msg("discord-notification-sent-successfully")
}

// Send sends message to a Discord webhook (token is the webhook URL). The
// send is a span in ctx's trace that records the status but not the URL,
// which holds the webhook's secret.
func Send(ctx context.Context, message string, token string) (err error) {
	ctx, span := otel.Tracer("notify").Start(ctx, "discord.webhook")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()
	requestBody, err := json.Marshal(map[string]string{"content": message})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, token, bytes.NewBuffer(requestBody))
	if err != nil {
		// The error would quote the URL.
		return fmt.Errorf("discord webhook: bad webhook URL")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		// The error quotes the URL.
		return fmt.Errorf("discord webhook: request failed")
	}
	defer resp.Body.Close()
	span.SetAttributes(attribute.Int("http.response.status_code", resp.StatusCode))
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("discord webhook: %s", resp.Status)
	}
	return nil
}
