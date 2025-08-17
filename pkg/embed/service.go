package embed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/woogles-io/liwords/pkg/omgwords/stores"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type EmbedService struct {
	gameDocumentStore *stores.GameDocumentStore
}

func NewEmbedService(gds *stores.GameDocumentStore) *EmbedService {
	return &EmbedService{
		gameDocumentStore: gds,
	}
}

type EmbedOptions struct {
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	ShowControls bool   `json:"showControls"`
	ShowScores   bool   `json:"showScores"`
	ShowMoveList bool   `json:"showMoveList"`
	Theme        string `json:"theme"`
}

func parseEmbedOptions(query url.Values) EmbedOptions {
	options := EmbedOptions{
		Width:        600,
		Height:       700,
		ShowControls: true,
		ShowScores:   true,
		ShowMoveList: false,
		Theme:        "light",
	}

	if width := query.Get("width"); width != "" {
		if w, err := strconv.Atoi(width); err == nil && w > 0 {
			options.Width = w
		}
	}

	if height := query.Get("height"); height != "" {
		if h, err := strconv.Atoi(height); err == nil && h > 0 {
			options.Height = h
		}
	}

	if controls := query.Get("showControls"); controls != "" {
		options.ShowControls = controls == "true"
	}

	if scores := query.Get("showScores"); scores != "" {
		options.ShowScores = scores == "true"
	}

	if moveList := query.Get("showMoveList"); moveList != "" {
		options.ShowMoveList = moveList == "true"
	}

	if theme := query.Get("theme"); theme == "dark" || theme == "light" {
		options.Theme = theme
	}

	return options
}

func (es *EmbedService) getGameDocument(ctx context.Context, gameID string) (*ipc.GameDocument, error) {
	doc, err := es.gameDocumentStore.GetDocument(ctx, gameID, false)
	if err != nil {
		if err == stores.ErrDoesNotExist {
			return nil, fmt.Errorf("game does not exist")
		}
		return nil, err
	}

	// For now, only support annotated games (like the existing GetGameDocument service)
	if doc.Type != ipc.GameType_ANNOTATED {
		return nil, fmt.Errorf("only annotated games are supported for embedding")
	}

	return doc.GameDocument, nil
}

func (es *EmbedService) generateEmbedHTML(gameDoc *ipc.GameDocument, gameID string, options EmbedOptions) (string, error) {
	// Serialize GameDocument to JSON
	gameDocJSON, err := protojson.Marshal(gameDoc)
	if err != nil {
		return "", fmt.Errorf("failed to serialize game document: %w", err)
	}

	// Serialize options to JSON
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return "", fmt.Errorf("failed to serialize options: %w", err)
	}

	// Generate unique container ID
	containerID := fmt.Sprintf("woogles-embed-%s", gameID)

	// Build the embed HTML with complete HTML document structure
	// Use relative path for the embed script so it works with any domain
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Woogles Game Embed</title>
  <style>
    body { margin: 0; padding: 20px; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
  </style>
</head>
<body>
  <div id="%s" style="width: %dpx; height: %dpx; border: 1px solid #e0e0e0; border-radius: 8px; overflow: hidden;">
    <div style="display: flex; align-items: center; justify-content: center; height: 100%%; color: #666;">
      Loading Woogles game...
    </div>
  </div>
  <script>
    window.WooglesEmbed = {
      gameId: %q,
      containerId: %q,
      gameDocument: %s,
      options: %s
    };
	console.log("testing something");
  </script>
  <script src="/static/js/embed-standalone.js"></script>
</body>
</html>`,
		containerID,
		options.Width,
		options.Height,
		gameID,
		containerID,
		string(gameDocJSON),
		string(optionsJSON),
	)

	return html, nil
}

func (es *EmbedService) generateEmbedEndpoint(w http.ResponseWriter, r *http.Request, gameID string) {
	ctx := r.Context()

	if gameID == "" {
		http.Error(w, "Game ID is required", http.StatusBadRequest)
		return
	}

	// Parse embed options from query parameters
	options := parseEmbedOptions(r.URL.Query())

	// Get the game document
	gameDoc, err := es.getGameDocument(ctx, gameID)
	if err != nil {
		log.Err(err).Str("gameID", gameID).Msg("failed to get game document for embed")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Generate the embed HTML
	embedHTML, err := es.generateEmbedHTML(gameDoc, gameID, options)
	if err != nil {
		log.Err(err).Str("gameID", gameID).Msg("failed to generate embed HTML")
		http.Error(w, "Failed to generate embed code", http.StatusInternalServerError)
		return
	}

	// Set appropriate headers
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	// Write the HTML response
	_, err = w.Write([]byte(embedHTML))
	if err != nil {
		log.Err(err).Str("gameID", gameID).Msg("failed to write embed HTML response")
	}
}

// must end with /
const EmbedServicePrefix = "/embed/"

// ServeHTTP implements http.Handler
func (es *EmbedService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, EmbedServicePrefix) {
		path := strings.TrimPrefix(r.URL.Path, EmbedServicePrefix)

		// Handle /embed/generate/:gameID
		if strings.HasPrefix(path, "generate/") {
			gameID := strings.TrimPrefix(path, "generate/")
			es.generateEmbedEndpoint(w, r, gameID)
		} else {
			http.NotFound(w, r)
		}
	} else {
		http.NotFound(w, r)
	}
}
