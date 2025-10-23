package vdowebhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/tournament"
)

// VDOWebhookService handles webhook callbacks from VDO.Ninja
type VDOWebhookService struct {
	tournamentStore tournament.TournamentStore
}

// NewVDOWebhookService creates a new VDO webhook service
func NewVDOWebhookService(ts tournament.TournamentStore) *VDOWebhookService {
	return &VDOWebhookService{
		tournamentStore: ts,
	}
}

// VDOWebhookPayload represents the JSON payload sent by VDO.Ninja
type VDOWebhookPayload struct {
	Update struct {
		StreamID string `json:"streamID"`
		Action   string `json:"action"`
		Value    bool   `json:"value"`
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
		Bool("value", payload.Update.Value).
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
		if payload.Update.Value {
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
	} else if payload.Update.Action == "hangup" && payload.Update.Value {
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
	} else {
		// Log unhandled actions for debugging
		log.Warn().
			Str("action", payload.Update.Action).
			Bool("value", payload.Update.Value).
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
