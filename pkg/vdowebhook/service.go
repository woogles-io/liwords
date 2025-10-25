package vdowebhook

import (
	"context"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/tournament"
)

// VDOWebhookService handles webhook callbacks from VDO.Ninja
type VDOWebhookService struct {
	tournamentStore           tournament.TournamentStore
	pollingIntervalSeconds    int
}

// NewVDOWebhookService creates a new VDO webhook service
func NewVDOWebhookService(ts tournament.TournamentStore, pollingIntervalSeconds int) *VDOWebhookService {
	return &VDOWebhookService{
		tournamentStore:        ts,
		pollingIntervalSeconds: pollingIntervalSeconds,
	}
}

// VDOWebhookPayload represents the JSON payload sent by VDO.Ninja
type VDOWebhookPayload struct {
	Update struct {
		StreamID string      `json:"streamID"`
		Action   string      `json:"action"`
		Value    interface{} `json:"value"` // Can be bool or object depending on action type
	} `json:"update"`
}

func (s *VDOWebhookService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for VDO.Ninja webhooks
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle CORS preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("vdo-webhook-read-body-error")
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Log raw request body for debugging
	log.Info().Str("raw_body", string(body)).Msg("vdo-webhook-raw-request")

	var payload VDOWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Error().Err(err).Str("body", string(body)).Msg("vdo-webhook-parse-error")
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	log.Info().
		Str("streamID", payload.Update.StreamID).
		Str("action", payload.Update.Action).
		Interface("value", payload.Update.Value).
		Msg("vdo-webhook-received")

	// The streamID is the key we stored when starting the stream
	// We need to find which tournament/user owns this stream by searching
	// through all tournaments. The streamKey is already stored in MonitoringData.

	// Determine stream type from the streamID format
	streamType := "camera"
	if strings.HasSuffix(payload.Update.StreamID, "_ss") {
		streamType = "screenshot"
	}

	ctx := r.Context()

	// Find the tournament and user that owns this stream key
	// We'll need to add a helper function to search all tournaments
	tournamentID, userID, err := findStreamOwner(ctx, s.tournamentStore, payload.Update.StreamID, streamType)
	if err != nil {
		log.Error().Err(err).Str("streamID", payload.Update.StreamID).Msg("vdo-webhook-stream-not-found")
		// Still return 200 OK to VDO.Ninja, just don't update anything
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	}

	// Handle the action based on type and value
	// VDO.Ninja sends "seeding" action when stream starts/stops
	// VDO.Ninja sends "hangup" action when user closes the stream window
	if payload.Update.Action == "seeding" {
		// Type assert value to bool
		boolVal, ok := payload.Update.Value.(bool)
		if !ok {
			log.Warn().Str("streamID", payload.Update.StreamID).Interface("value", payload.Update.Value).Msg("vdo-webhook-seeding-value-not-bool")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}
		if boolVal {
			// Stream started - set status to ACTIVE
			log.Info().
				Str("tournamentID", tournamentID).
				Str("userID", userID).
				Str("streamType", streamType).
				Msg("vdo-webhook-activating-stream")
			err = tournament.ActivateMonitoringStream(ctx, s.tournamentStore, tournamentID, userID, streamType)
			if err != nil {
				log.Error().Err(err).
					Str("tournamentID", tournamentID).
					Str("userID", userID).
					Str("streamType", streamType).
					Msg("vdo-webhook-activate-stream-error")
				// Don't return error to VDO.Ninja, just log it
			} else {
				log.Info().
					Str("tournamentID", tournamentID).
					Str("userID", userID).
					Str("streamType", streamType).
					Msg("vdo-webhook-stream-activated")
			}
		} else {
			// Stream stopped - set status to STOPPED
			log.Info().
				Str("tournamentID", tournamentID).
				Str("userID", userID).
				Str("streamType", streamType).
				Msg("vdo-webhook-deactivating-stream")
			err = tournament.DeactivateMonitoringStream(ctx, s.tournamentStore, tournamentID, userID, streamType)
			if err != nil {
				log.Error().Err(err).
					Str("tournamentID", tournamentID).
					Str("userID", userID).
					Str("streamType", streamType).
					Msg("vdo-webhook-deactivate-stream-error")
				// Don't return error to VDO.Ninja, just log it
			} else {
				log.Info().
					Str("tournamentID", tournamentID).
					Str("userID", userID).
					Str("streamType", streamType).
					Msg("vdo-webhook-stream-deactivated")
			}
		}
	} else if payload.Update.Action == "hangup" {
		// Type assert value to bool
		boolVal, ok := payload.Update.Value.(bool)
		if !ok {
			log.Warn().Str("streamID", payload.Update.StreamID).Interface("value", payload.Update.Value).Msg("vdo-webhook-hangup-value-not-bool")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}
		if !boolVal {
			// Hangup with false means ignore (shouldn't happen but be safe)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}
		// User closed the stream window - treat as STOPPED
		log.Info().
			Str("tournamentID", tournamentID).
			Str("userID", userID).
			Str("streamType", streamType).
			Msg("vdo-webhook-hangup-deactivating-stream")
		err = tournament.DeactivateMonitoringStream(ctx, s.tournamentStore, tournamentID, userID, streamType)
		if err != nil {
			log.Error().Err(err).
				Str("tournamentID", tournamentID).
				Str("userID", userID).
				Str("streamType", streamType).
				Msg("vdo-webhook-hangup-deactivate-error")
			// Don't return error to VDO.Ninja, just log it
		} else {
			log.Info().
				Str("tournamentID", tournamentID).
				Str("userID", userID).
				Str("streamType", streamType).
				Msg("vdo-webhook-hangup-stream-deactivated")
		}
	} else if payload.Update.Action == "details" {
		// Type assert value to map[string]any
		detailsMap, ok := payload.Update.Value.(map[string]any)
		if !ok {
			log.Warn().Str("streamID", payload.Update.StreamID).Interface("value", payload.Update.Value).Msg("vdo-webhook-details-value-not-map")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}

		// Get the stream details for this specific streamID
		streamDetails, ok := detailsMap[payload.Update.StreamID].(map[string]any)
		if !ok {
			log.Warn().Str("streamID", payload.Update.StreamID).Interface("value", payload.Update.Value).Msg("vdo-webhook-details-stream-not-found")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}

		// Check the "seeding" field to determine if stream is active
		seeding, ok := streamDetails["seeding"].(bool)
		if !ok {
			log.Warn().Str("streamID", payload.Update.StreamID).Interface("streamDetails", streamDetails).Msg("vdo-webhook-details-seeding-not-bool")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}

		if seeding {
			// Stream is active - activate it
			log.Info().
				Str("tournamentID", tournamentID).
				Str("userID", userID).
				Str("streamType", streamType).
				Msg("vdo-webhook-details-activating-stream")
			err = tournament.ActivateMonitoringStream(ctx, s.tournamentStore, tournamentID, userID, streamType)
			if err != nil {
				log.Error().Err(err).
					Str("tournamentID", tournamentID).
					Str("userID", userID).
					Str("streamType", streamType).
					Msg("vdo-webhook-details-activate-error")
				// Don't return error to VDO.Ninja, just log it
			} else {
				log.Info().
					Str("tournamentID", tournamentID).
					Str("userID", userID).
					Str("streamType", streamType).
					Msg("vdo-webhook-details-stream-activated")
			}
		} else {
			// Stream is not active - deactivate it
			log.Info().
				Str("tournamentID", tournamentID).
				Str("userID", userID).
				Str("streamType", streamType).
				Msg("vdo-webhook-details-deactivating-stream")
			err = tournament.DeactivateMonitoringStream(ctx, s.tournamentStore, tournamentID, userID, streamType)
			if err != nil {
				log.Error().Err(err).
					Str("tournamentID", tournamentID).
					Str("userID", userID).
					Str("streamType", streamType).
					Msg("vdo-webhook-details-deactivate-error")
				// Don't return error to VDO.Ninja, just log it
			} else {
				log.Info().
					Str("tournamentID", tournamentID).
					Str("userID", userID).
					Str("streamType", streamType).
					Msg("vdo-webhook-details-stream-deactivated")
			}
		}
	} else {
		// Log unhandled actions for debugging
		log.Warn().
			Str("action", payload.Update.Action).
			Interface("value", payload.Update.Value).
			Str("streamID", payload.Update.StreamID).
			Msg("vdo-webhook-unhandled-action")
	}

	// Always return 200 OK to VDO.Ninja
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// findStreamOwner uses database GIN index to quickly find the owner of a stream key
// StreamKey format: woog_{8chars} or woog_{8chars}_ss
// Returns tournament UUID and full userID in "uuid:username" format
func findStreamOwner(ctx context.Context, ts tournament.TournamentStore, streamKey string, streamType string) (tournamentID string, userID string, err error) {
	// Single database query returns tournament UUID and full userID (uuid:username)
	// Uses GIN index on monitoring_data for fast lookup
	tid, uid, err := ts.FindTournamentByStreamKey(ctx, streamKey, streamType)
	if err != nil {
		return "", "", err
	}

	if tid == "" || uid == "" {
		return "", "", &StreamNotFoundError{StreamKey: streamKey}
	}

	return tid, uid, nil
}

// StreamNotFoundError represents an error when a stream key is not found
type StreamNotFoundError struct {
	StreamKey string
}

func (e *StreamNotFoundError) Error() string {
	return "stream key not found: " + e.StreamKey
}

// Start begins the polling loop that checks active stream health
// Runs continuously until context is cancelled
func (s *VDOWebhookService) Start(ctx context.Context) {
	log.Info().Msg("vdo-polling-service-started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("vdo-polling-service-stopped")
			return
		case <-time.After(s.getJitteredInterval()):
			// Poll all active streams in a separate goroutine to avoid blocking
			go s.pollActiveStreams(ctx)
		}
	}
}

// getJitteredInterval returns a random duration around the configured midpoint with ±25% jitter
// This prevents thundering herd and spreads API load on VDO.Ninja
// For example, with 60s midpoint: returns 45-75 seconds (60 ± 15)
func (s *VDOWebhookService) getJitteredInterval() time.Duration {
	midpoint := time.Duration(s.pollingIntervalSeconds) * time.Second
	// Jitter is ±25% of the midpoint
	jitterRange := s.pollingIntervalSeconds / 2 // 50% range (±25%)
	jitter := time.Duration(rand.Intn(jitterRange+1)) * time.Second
	// Subtract 25% from midpoint, then add random jitter from 0 to 50%
	base := midpoint - (midpoint / 4) // midpoint - 25%
	return base + jitter
}

// pollActiveStreams fetches all active streams and spreads API checks over 30 seconds
func (s *VDOWebhookService) pollActiveStreams(ctx context.Context) {
	streams, err := s.tournamentStore.GetActiveMonitoringStreams(ctx)
	if err != nil {
		log.Error().Err(err).Msg("vdo-poll-get-active-streams-error")
		return
	}

	if len(streams) == 0 {
		log.Debug().Msg("vdo-poll-no-active-streams")
		return
	}

	log.Info().Int("count", len(streams)).Msg("vdo-poll-checking-active-streams")

	// Spread requests evenly over 30 seconds to avoid bursts
	delayBetween := 30 * time.Second / time.Duration(len(streams))

	for i, stream := range streams {
		// Calculate staggered delay for this stream
		delay := time.Duration(i) * delayBetween
		// Add small random jitter (0-2 seconds) to each request
		delay += time.Duration(rand.Intn(2000)) * time.Millisecond

		// Launch goroutine with delay
		go func(streamData tournament.ActiveMonitoringStream, d time.Duration) {
			time.Sleep(d)
			s.checkStream(ctx, streamData)
		}(stream, delay)
	}
}

// checkStream polls VDO.Ninja API to verify stream is still active
// If stream is disconnected or API returns error, deactivates the stream
func (s *VDOWebhookService) checkStream(ctx context.Context, stream tournament.ActiveMonitoringStream) {
	apiKey := stream.StreamKey + "_api"
	url := "https://api.vdo.ninja/" + apiKey + "/getDetails"

	// Create HTTP request with 10 second timeout
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
	if err != nil {
		log.Error().Err(err).Str("streamKey", stream.StreamKey).Msg("vdo-poll-create-request-error")
		return
	}

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Warn().Err(err).
			Str("streamKey", stream.StreamKey).
			Str("tournamentID", stream.TournamentID).
			Str("userID", stream.UserID).
			Str("streamType", stream.StreamType).
			Msg("vdo-poll-http-error-skipping")
		// HTTP error could mean API is down, not that stream failed
		// Don't deactivate - wait for explicit "failed" response
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("streamKey", stream.StreamKey).Msg("vdo-poll-read-body-error")
		return
	}

	bodyStr := strings.TrimSpace(string(body))

	// Only deactivate if VDO.Ninja explicitly returns "failed", "timedout", or "timeout"
	if bodyStr == "failed" || bodyStr == "timedout" || bodyStr == "timeout" {
		log.Info().
			Str("streamKey", stream.StreamKey).
			Str("tournamentID", stream.TournamentID).
			Str("userID", stream.UserID).
			Str("streamType", stream.StreamType).
			Str("reason", bodyStr).
			Msg("vdo-poll-stream-failed-deactivating")
		s.deactivateStream(ctx, stream)
		return
	}

	// Try to parse as JSON to verify we got a valid response
	var details map[string]any
	if err := json.Unmarshal([]byte(bodyStr), &details); err != nil {
		log.Warn().
			Err(err).
			Str("streamKey", stream.StreamKey).
			Str("tournamentID", stream.TournamentID).
			Str("userID", stream.UserID).
			Str("streamType", stream.StreamType).
			Str("response", bodyStr).
			Msg("vdo-poll-invalid-json-skipping")
		// Invalid JSON could mean API issue, not stream failure
		// Don't deactivate - wait for explicit "failed" response
		return
	}

	// Stream is healthy, log success at debug level
	log.Debug().
		Str("streamKey", stream.StreamKey).
		Str("tournamentID", stream.TournamentID).
		Str("userID", stream.UserID).
		Str("streamType", stream.StreamType).
		Msg("vdo-poll-stream-healthy")
}

// deactivateStream marks a stream as disconnected
// Reuses the same DeactivateMonitoringStream logic as webhook handler
func (s *VDOWebhookService) deactivateStream(ctx context.Context, stream tournament.ActiveMonitoringStream) {
	err := tournament.DeactivateMonitoringStream(ctx, s.tournamentStore, stream.TournamentID, stream.UserID, stream.StreamType)
	if err != nil {
		log.Error().Err(err).
			Str("tournamentID", stream.TournamentID).
			Str("userID", stream.UserID).
			Str("streamType", stream.StreamType).
			Msg("vdo-poll-deactivate-stream-error")
	} else {
		log.Info().
			Str("tournamentID", stream.TournamentID).
			Str("userID", stream.UserID).
			Str("streamType", stream.StreamType).
			Msg("vdo-poll-stream-deactivated")
	}
}
