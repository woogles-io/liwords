package broadcasts

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/domino14/word-golib/tilemapping"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	omgstores "github.com/woogles-io/liwords/pkg/omgwords/stores"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// OBSHandlerPrefix is the URL prefix for broadcast-slot OBS endpoints.
const OBSHandlerPrefix = "/api/broadcasts/obs/"

// OBSGameHandlerPrefix is the URL prefix for direct per-game OBS endpoints.
const OBSGameHandlerPrefix = "/api/annotations/obs/game/"

// OBSUserHandlerPrefix is the URL prefix for user-alias OBS endpoints.
// The URL resolves dynamically to the user's most-recently-edited annotated game.
const OBSUserHandlerPrefix = "/api/annotations/obs/user/"

// natsUserAnnoSubjectPrefix mirrors omgwords.NatsUserAnnoSubjectPrefix to avoid
// a circular package import. Must stay in sync with pkg/omgwords/service.go.
const natsUserAnnoSubjectPrefix = "anno.user."

// validSuffixes lists the accepted display suffixes.
var validSuffixes = map[string]bool{
	"score":          true,
	"p1_score":       true,
	"p2_score":       true,
	"unseen_tiles":   true,
	"unseen_count":   true,
	"last_play":      true,
	"blank1":         true,
	"blank2":         true,
	"p1_name":        true,
	"p2_name":        true,
	"combined_names": true,
	"events":         true,

	// Tournament-standings fields — broadcast-slot mode only.
	"p1_record":  true,
	"p2_record":  true,
	"p1_place":   true,
	"p2_place":   true,
	"p1_spread":  true,
	"p2_spread":  true,
	"p1_rating":  true,
	"p2_rating":  true,
	"division":   true,
	"tournament": true,
	"round":      true,
	"table":      true,

	// User-alias mode only.
	"opponent_name": true,
}

// obsPlaceholder is the value shown when no game is assigned to a slot.
const obsPlaceholder = "(no game assigned)"

// obsStreamMode identifies which of the three OBS URL families a stream
// belongs to, so the live-update path knows which augmentation to apply on
// top of the base game-doc OBSData.
type obsStreamMode int

const (
	obsStreamModeGame obsStreamMode = iota
	obsStreamModeSlot
	obsStreamModeUser
)

// obsStream tracks all SSE subscribers and NATS subscriptions for one stream key.
// Key is "slug/slot" for slot streams or "game/<uuid>" for direct game streams.
type obsStream struct {
	mu            sync.Mutex
	subscribers   map[chan OBSData]struct{}
	annoSub       *nats.Subscription // per-turn game events (may be nil)
	broadcastSub  *nats.Subscription // slot reassignment events (nil for game streams)
	currentGameID string
	initialized   bool

	// Mode + context needed to augment the base OBSData on every push: slot
	// mode merges in tournament-standings fields, user mode resolves
	// opponent_name. Set once when the stream is created (slot division/
	// round/table are refreshed on slot reassignment); always read under mu.
	mode               obsStreamMode
	slug               string
	division           string
	round              int32
	table              int32
	broadcastName      string
	broadcastURL       string
	broadcastURLFormat string
	trackedUserUUID    string
	trackedUsername    string
}

// OBSHandler serves browser-source HTML pages and SSE streams for OBS.
//
// Two URL families:
//
//	/api/broadcasts/obs/<slug>/<slot>/<suffix>       → HTML
//	/api/broadcasts/obs/<slug>/<slot>/<suffix>.txt   → plain text
//	/api/broadcasts/obs/<slug>/<slot>/events         → SSE
//
//	/api/annotations/obs/game/<gameid>/<suffix>      → HTML
//	/api/annotations/obs/game/<gameid>/<suffix>.txt  → plain text
//	/api/annotations/obs/game/<gameid>/events        → SSE
type OBSHandler struct {
	queries      *models.Queries
	gameDocStore *omgstores.GameDocumentStore
	cfg          *config.Config
	natsConn     *nats.Conn
	definer      Definer // may be nil; used for symbol+definition in last_play

	// broadcastSvc gives access to the tournament feed cache (getCachedFeed)
	// for the slot-mode tournament-standings fields. Same package as
	// BroadcastService, so its unexported methods are callable directly.
	broadcastSvc *BroadcastService

	mu      sync.Mutex
	streams map[string]*obsStream
}

func NewOBSHandler(
	queries *models.Queries,
	gameDocStore *omgstores.GameDocumentStore,
	cfg *config.Config,
	natsConn *nats.Conn,
	definer Definer,
	broadcastSvc *BroadcastService,
) *OBSHandler {
	return &OBSHandler{
		queries:      queries,
		gameDocStore: gameDocStore,
		cfg:          cfg,
		natsConn:     natsConn,
		definer:      definer,
		broadcastSvc: broadcastSvc,
		streams:      make(map[string]*obsStream),
	}
}

// ServeHTTP dispatches broadcast-slot requests:
//
//	/api/broadcasts/obs/<slug>/<slot>/<suffix>[.txt]
func (h *OBSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tail := strings.TrimPrefix(r.URL.Path, OBSHandlerPrefix)
	parts := strings.SplitN(tail, "/", 3)
	if len(parts) != 3 {
		http.Error(w, "usage: /api/broadcasts/obs/<slug>/<slot>/<suffix>", http.StatusBadRequest)
		return
	}
	slug, slotName, rawSuffix := parts[0], parts[1], parts[2]
	if slug == "" || slotName == "" {
		http.Error(w, "slug and slot are required", http.StatusBadRequest)
		return
	}

	rawText := strings.HasSuffix(rawSuffix, ".txt")
	suffix := strings.TrimSuffix(rawSuffix, ".txt")
	if !validSuffixes[suffix] {
		http.Error(w, fmt.Sprintf("unknown suffix %q", suffix), http.StatusBadRequest)
		return
	}

	streamKey := slug + "/" + slotName

	if suffix == "events" {
		h.serveSlotSSE(w, r, streamKey, slug, slotName)
		return
	}

	ctx := r.Context()
	slotRow, err := h.queries.GetBroadcastSlotGame(ctx, models.GetBroadcastSlotGameParams{
		Slug:     slug,
		SlotName: slotName,
	})
	if err != nil {
		http.Error(w, "slot not found", http.StatusNotFound)
		return
	}

	var value string
	if !slotRow.GameUuid.Valid || slotRow.GameUuid.String == "" {
		value = obsPlaceholder
	} else {
		data, _, err := h.loadGameOBSData(ctx, slotRow.GameUuid.String)
		if err != nil {
			value = obsPlaceholder
		} else {
			// Broadcast-row lookup is best-effort: on failure the
			// tournament-standings fields simply fall back to placeholders.
			broadcastRow, berr := h.queries.GetBroadcastBySlug(ctx, slug)
			if berr != nil {
				log.Err(berr).Str("slug", slug).Msg("obs-slot-broadcast-lookup-error")
			}
			h.augmentSlotFields(&data, slug, slotRow.Division, slotRow.Round, slotRow.TableNumber,
				broadcastRow.Name, broadcastRow.BroadcastUrl, broadcastRow.BroadcastUrlFormat)
			value = obsFieldValue(data, suffix)
		}
	}

	eventsURL := OBSHandlerPrefix + slug + "/" + slotName + "/events"
	h.serveResponse(w, suffix, value, eventsURL, rawText)
}

// ServeGameHTTP dispatches direct per-game requests:
//
//	/api/annotations/obs/game/<gameid>/<suffix>[.txt]
func (h *OBSHandler) ServeGameHTTP(w http.ResponseWriter, r *http.Request) {
	tail := strings.TrimPrefix(r.URL.Path, OBSGameHandlerPrefix)
	parts := strings.SplitN(tail, "/", 2)
	if len(parts) != 2 {
		http.Error(w, "usage: /api/annotations/obs/game/<gameid>/<suffix>", http.StatusBadRequest)
		return
	}
	gameID, rawSuffix := parts[0], parts[1]
	if gameID == "" {
		http.Error(w, "gameid is required", http.StatusBadRequest)
		return
	}

	rawText := strings.HasSuffix(rawSuffix, ".txt")
	suffix := strings.TrimSuffix(rawSuffix, ".txt")
	if !validSuffixes[suffix] {
		http.Error(w, fmt.Sprintf("unknown suffix %q", suffix), http.StatusBadRequest)
		return
	}

	streamKey := "game/" + gameID

	if suffix == "events" {
		h.serveGameSSE(w, r, streamKey, gameID)
		return
	}

	ctx := r.Context()
	var value string
	if data, _, err := h.loadGameOBSData(ctx, gameID); err != nil {
		value = obsPlaceholder
	} else {
		value = obsFieldValue(data, suffix)
	}
	eventsURL := OBSGameHandlerPrefix + gameID + "/events"
	h.serveResponse(w, suffix, value, eventsURL, rawText)
}

// loadGameOBSData loads the game document + letter distribution for gameUUID
// and returns both the base (game-doc-only) OBSData and the raw document —
// the latter is needed by mode-specific augmentation (e.g. opponent
// resolution), which callers apply on top of the returned data.
func (h *OBSHandler) loadGameOBSData(ctx context.Context, gameUUID string) (OBSData, *ipc.GameDocument, error) {
	doc, err := h.gameDocStore.GetDocument(ctx, gameUUID)
	if err != nil {
		log.Err(err).Str("gameUUID", gameUUID).Msg("obs-load-doc-error")
		return OBSData{}, nil, err
	}
	dist, err := tilemapping.GetDistribution(h.cfg.WGLConfig(), doc.LetterDistribution)
	if err != nil {
		log.Err(err).Str("gameUUID", gameUUID).Msg("obs-load-dist-error")
		return OBSData{}, nil, err
	}
	return ComputeOBSData(doc, dist, h.definer), doc, nil
}

// augmentSlotFields fills OBSData's tournament-standings fields for a
// broadcast slot. division/round/tableNumber and tournamentName are always
// applied; the feed lookup (record/place/spread/rating) is best-effort and
// silently falls back to placeholders when the URL/format is missing or the
// feed fetch fails.
func (h *OBSHandler) augmentSlotFields(data *OBSData, slug, division string, round, table int32, tournamentName, broadcastURL, broadcastURLFormat string) {
	var fd *FeedData
	if broadcastURL != "" && broadcastURLFormat != "" {
		fd = h.broadcastSvc.getCachedFeed(slug, division, broadcastURL, broadcastURLFormat)
	}
	applyTournamentFields(data, fd, division, round, table, tournamentName, data.P1Name, data.P2Name)
}

// serveResponse writes either a text/plain or text/html response.
func (h *OBSHandler) serveResponse(w http.ResponseWriter, suffix, value, eventsURL string, rawText bool) {
	if rawText {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprint(w, value)
		return
	}
	serveOBSPage(w, suffix, value, eventsURL)
}

// obsFieldValue picks the right string from an OBSData by suffix name.
func obsFieldValue(d OBSData, suffix string) string {
	switch suffix {
	case "score":
		return d.Score
	case "p1_score":
		return d.P1Score
	case "p2_score":
		return d.P2Score
	case "unseen_tiles":
		return d.UnseenTiles
	case "unseen_count":
		return d.UnseenCount
	case "last_play":
		return d.LastPlay
	case "blank1":
		return d.Blank1
	case "blank2":
		return d.Blank2
	case "p1_name":
		return d.P1Name
	case "p2_name":
		return d.P2Name
	case "combined_names":
		return d.CombinedNames
	case "p1_record":
		return orPlaceholder(d.P1Record)
	case "p2_record":
		return orPlaceholder(d.P2Record)
	case "p1_place":
		return orPlaceholder(d.P1Place)
	case "p2_place":
		return orPlaceholder(d.P2Place)
	case "p1_spread":
		return orPlaceholder(d.P1Spread)
	case "p2_spread":
		return orPlaceholder(d.P2Spread)
	case "p1_rating":
		return orPlaceholder(d.P1Rating)
	case "p2_rating":
		return orPlaceholder(d.P2Rating)
	case "division":
		return orPlaceholder(d.Division)
	case "tournament":
		return orPlaceholder(d.Tournament)
	case "round":
		return orPlaceholder(d.Round)
	case "table":
		return orPlaceholder(d.Table)
	case "opponent_name":
		return orPlaceholder(d.OpponentName)
	}
	return ""
}

// ---------------------------------------------------------------------------
// SSE
// ---------------------------------------------------------------------------

// serveSlotSSE opens a Server-Sent Events stream for a broadcast slot.
// P0 #4: resolves the slot first; returns 404 if it doesn't exist.
func (h *OBSHandler) serveSlotSSE(w http.ResponseWriter, r *http.Request, streamKey, slug, slotName string) {
	ctx := r.Context()

	// Resolve slot before writing headers so we can still send 404.
	slotRow, err := h.queries.GetBroadcastSlotGame(ctx, models.GetBroadcastSlotGameParams{
		Slug:     slug,
		SlotName: slotName,
	})
	if err != nil {
		http.Error(w, "slot not found", http.StatusNotFound)
		return
	}
	// Best-effort: on failure, tournament-standings fields fall back to
	// placeholders but the stream still serves the game-doc fields fine.
	broadcastRow, err := h.queries.GetBroadcastBySlug(ctx, slug)
	if err != nil {
		log.Err(err).Str("slug", slug).Msg("obs-slot-broadcast-lookup-error")
	}

	flusher, ok := sseSetup(w)
	if !ok {
		return
	}

	ch := make(chan OBSData, 8)
	stream := h.addSubscriber(streamKey, ch)
	defer h.removeSubscriber(streamKey, ch)

	gameUUID := ""
	if slotRow.GameUuid.Valid {
		gameUUID = slotRow.GameUuid.String
	}

	h.ensureSlotNATSSubs(stream, streamKey, slug, slotName, slotRow, broadcastRow)

	h.serveSSELoop(w, ctx, stream, ch, flusher, gameUUID)
}

// serveGameSSE opens a Server-Sent Events stream for a direct per-game annotation.
func (h *OBSHandler) serveGameSSE(w http.ResponseWriter, r *http.Request, streamKey, gameUUID string) {
	flusher, ok := sseSetup(w)
	if !ok {
		return
	}

	ch := make(chan OBSData, 8)
	stream := h.addSubscriber(streamKey, ch)
	defer h.removeSubscriber(streamKey, ch)

	h.ensureGameNATSSub(stream, streamKey, gameUUID)

	h.serveSSELoop(w, r.Context(), stream, ch, flusher, gameUUID)
}

// sseSetup writes the standard SSE response headers and returns the flusher.
func sseSetup(w http.ResponseWriter) (http.Flusher, bool) {
	rc := http.NewResponseController(w)
	rc.SetWriteDeadline(time.Time{})
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return nil, false
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	return flusher, true
}

// serveSSELoop sends an initial snapshot (if a game is active) then pumps
// events until the client disconnects or the context is cancelled.
func (h *OBSHandler) serveSSELoop(w http.ResponseWriter, ctx context.Context, stream *obsStream, ch chan OBSData, flusher http.Flusher, initialGameUUID string) {
	if initialGameUUID != "" {
		if data, ok := h.reloadDoc(stream, initialGameUUID); ok {
			if bts, err := json.Marshal(data); err == nil {
				fmt.Fprintf(w, "data: %s\n\n", bts)
				flusher.Flush()
			}
		}
	}

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case data := <-ch:
			bts, err := json.Marshal(data)
			if err != nil {
				log.Err(err).Msg("obs-sse-marshal")
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", bts)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Subscriber fan-out
// ---------------------------------------------------------------------------

// addSubscriber registers ch for streamKey, creating the stream if needed.
// Returns the stream so the caller can pass it to ensureXNATSSubs.
func (h *OBSHandler) addSubscriber(key string, ch chan OBSData) *obsStream {
	h.mu.Lock()
	stream := h.streams[key]
	if stream == nil {
		stream = &obsStream{subscribers: make(map[chan OBSData]struct{})}
		h.streams[key] = stream
	}
	stream.mu.Lock()
	stream.subscribers[ch] = struct{}{}
	stream.mu.Unlock()
	h.mu.Unlock()
	return stream
}

// removeSubscriber deregisters ch from streamKey. When the last subscriber
// leaves it tears down both NATS subscriptions and removes the stream entry.
func (h *OBSHandler) removeSubscriber(key string, ch chan OBSData) {
	h.mu.Lock()
	stream := h.streams[key]
	if stream == nil {
		h.mu.Unlock()
		return
	}
	// Hold both locks together so ensureXNATSSubs can't race the liveness check.
	stream.mu.Lock()
	delete(stream.subscribers, ch)
	empty := len(stream.subscribers) == 0
	var annoSub, broadcastSub *nats.Subscription
	if empty {
		annoSub = stream.annoSub
		broadcastSub = stream.broadcastSub
		stream.annoSub = nil
		stream.broadcastSub = nil
		delete(h.streams, key)
	}
	stream.mu.Unlock()
	h.mu.Unlock()

	if annoSub != nil {
		annoSub.Unsubscribe()
	}
	if broadcastSub != nil {
		broadcastSub.Unsubscribe()
	}
}

func (h *OBSHandler) fanout(key string, data OBSData) {
	h.mu.Lock()
	stream := h.streams[key]
	h.mu.Unlock()
	if stream == nil {
		return
	}
	stream.mu.Lock()
	for ch := range stream.subscribers {
		select {
		case ch <- data:
		default:
		}
	}
	stream.mu.Unlock()
}

// reloadDoc rebuilds OBSData from the current game document, then applies
// whatever mode-specific augmentation stream carries (tournament-standings
// fields for slot mode, opponent resolution for user mode).
func (h *OBSHandler) reloadDoc(stream *obsStream, gameUUID string) (OBSData, bool) {
	ctx := context.Background()
	data, doc, err := h.loadGameOBSData(ctx, gameUUID)
	if err != nil {
		return OBSData{}, false
	}

	stream.mu.Lock()
	mode := stream.mode
	slug := stream.slug
	division := stream.division
	round := stream.round
	table := stream.table
	broadcastName := stream.broadcastName
	broadcastURL := stream.broadcastURL
	broadcastURLFormat := stream.broadcastURLFormat
	trackedUserUUID := stream.trackedUserUUID
	trackedUsername := stream.trackedUsername
	stream.mu.Unlock()

	switch mode {
	case obsStreamModeSlot:
		h.augmentSlotFields(&data, slug, division, round, table, broadcastName, broadcastURL, broadcastURLFormat)
	case obsStreamModeUser:
		data.OpponentName = opponentName(doc, trackedUserUUID, trackedUsername)
	}
	return data, true
}

func (h *OBSHandler) reloadAndFanout(stream *obsStream, key, gameUUID string) {
	if data, ok := h.reloadDoc(stream, gameUUID); ok {
		h.fanout(key, data)
	}
}

// ---------------------------------------------------------------------------
// NATS subscriptions
// ---------------------------------------------------------------------------

// ensureGameNATSSub subscribes once to per-turn events for a direct-game stream.
func (h *OBSHandler) ensureGameNATSSub(stream *obsStream, key, gameUUID string) {
	stream.mu.Lock()
	if stream.initialized {
		stream.mu.Unlock()
		return
	}
	stream.initialized = true
	stream.mu.Unlock()

	sub, err := h.natsConn.Subscribe("channel.anno"+gameUUID, func(_ *nats.Msg) {
		h.reloadAndFanout(stream, key, gameUUID)
	})
	if err != nil {
		log.Err(err).Str("gameUUID", gameUUID).Msg("obs-game-sub-error")
		return
	}

	// Only assign if the stream is still live (client didn't disconnect already).
	h.mu.Lock()
	if h.streams[key] != stream {
		h.mu.Unlock()
		sub.Unsubscribe()
		return
	}
	stream.mu.Lock()
	stream.annoSub = sub
	stream.mu.Unlock()
	h.mu.Unlock()
}

// ensureSlotNATSSubs subscribes once to:
//  1. channel.anno<gameUUID>      — per-turn events for the current game
//  2. broadcasts.<broadcastUUID>  — slot reassignment events
func (h *OBSHandler) ensureSlotNATSSubs(stream *obsStream, key, slug, slotName string, slotRow models.GetBroadcastSlotGameRow, broadcastRow models.GetBroadcastBySlugRow) {
	gameUUID := ""
	if slotRow.GameUuid.Valid {
		gameUUID = slotRow.GameUuid.String
	}
	broadcastUUID := slotRow.BroadcastUuid.String()

	stream.mu.Lock()
	if stream.initialized {
		stream.mu.Unlock()
		return
	}
	stream.initialized = true
	stream.currentGameID = gameUUID
	stream.mode = obsStreamModeSlot
	stream.slug = slug
	stream.division = slotRow.Division
	stream.round = slotRow.Round
	stream.table = slotRow.TableNumber
	stream.broadcastName = broadcastRow.Name
	stream.broadcastURL = broadcastRow.BroadcastUrl
	stream.broadcastURLFormat = broadcastRow.BroadcastUrlFormat
	stream.mu.Unlock()

	// Per-turn game subscription (only if a game is currently assigned).
	var annoSub *nats.Subscription
	if gameUUID != "" {
		var err error
		annoSub, err = h.natsConn.Subscribe("channel.anno"+gameUUID, func(_ *nats.Msg) {
			h.reloadAndFanout(stream, key, gameUUID)
		})
		if err != nil {
			log.Err(err).Str("gameUUID", gameUUID).Msg("obs-slot-anno-sub-error")
			annoSub = nil
		}
	}

	// Broadcast-level subscription for slot reassignment.
	broadcastSub, err := h.natsConn.Subscribe(NatsBroadcastSubjectPrefix+broadcastUUID, func(_ *nats.Msg) {
		ctx := context.Background()
		row, err := h.queries.GetBroadcastSlotGame(ctx, models.GetBroadcastSlotGameParams{
			Slug:     slug,
			SlotName: slotName,
		})
		if err != nil {
			log.Err(err).Str("streamKey", key).Msg("obs-slot-rebind-error")
			return
		}
		newGameUUID := ""
		if row.GameUuid.Valid {
			newGameUUID = row.GameUuid.String
		}

		stream.mu.Lock()
		currentGameID := stream.currentGameID
		// Slot metadata (division/round/table) can change on reassignment;
		// refresh it so subsequent pushes reflect the new target.
		stream.division = row.Division
		stream.round = row.Round
		stream.table = row.TableNumber
		stream.mu.Unlock()

		if newGameUUID != currentGameID {
			h.rebindAnnoSub(stream, key, newGameUUID)
		}
		if newGameUUID != "" {
			h.reloadAndFanout(stream, key, newGameUUID)
		}
	})
	if err != nil {
		log.Err(err).Str("broadcastUUID", broadcastUUID).Msg("obs-broadcast-sub-error")
		if annoSub != nil {
			annoSub.Unsubscribe()
		}
		return
	}

	// Atomically assign both subs, but only if the stream is still live.
	// Hold both h.mu + stream.mu together to prevent racing removeSubscriber.
	h.mu.Lock()
	if h.streams[key] != stream {
		h.mu.Unlock()
		if annoSub != nil {
			annoSub.Unsubscribe()
		}
		broadcastSub.Unsubscribe()
		return
	}
	stream.mu.Lock()
	stream.annoSub = annoSub
	stream.broadcastSub = broadcastSub
	stream.mu.Unlock()
	h.mu.Unlock()
}

// rebindAnnoSub swaps the per-game NATS subscription to a new game UUID.
// Called from the broadcast NATS callback when the slot target changes.
func (h *OBSHandler) rebindAnnoSub(stream *obsStream, key, newGameUUID string) {
	// Swap out old sub under stream.mu only.
	stream.mu.Lock()
	oldSub := stream.annoSub
	stream.annoSub = nil
	stream.currentGameID = newGameUUID
	stream.mu.Unlock()

	if oldSub != nil {
		oldSub.Unsubscribe()
	}

	if newGameUUID == "" {
		return
	}

	newSub, err := h.natsConn.Subscribe("channel.anno"+newGameUUID, func(_ *nats.Msg) {
		h.reloadAndFanout(stream, key, newGameUUID)
	})
	if err != nil {
		log.Err(err).Str("gameUUID", newGameUUID).Msg("obs-rebind-sub-error")
		return
	}

	// Only assign if stream is still live AND the game ID hasn't changed again.
	h.mu.Lock()
	if h.streams[key] != stream {
		h.mu.Unlock()
		newSub.Unsubscribe()
		return
	}
	stream.mu.Lock()
	if stream.currentGameID != newGameUUID {
		stream.mu.Unlock()
		h.mu.Unlock()
		newSub.Unsubscribe()
		return
	}
	stream.annoSub = newSub
	stream.mu.Unlock()
	h.mu.Unlock()
}

// ---------------------------------------------------------------------------
// User-alias handler
// ---------------------------------------------------------------------------

// ServeUserHTTP dispatches user-alias requests:
//
//	/api/annotations/obs/user/<username>/<suffix>[.txt]
//
// The username is resolved to the user's most-recently-edited annotated game.
// If no annotated game exists yet the placeholder text is returned, but the
// SSE stream is still connectable — it will push data once a game appears.
func (h *OBSHandler) ServeUserHTTP(w http.ResponseWriter, r *http.Request) {
	tail := strings.TrimPrefix(r.URL.Path, OBSUserHandlerPrefix)
	parts := strings.SplitN(tail, "/", 2)
	if len(parts) != 2 {
		http.Error(w, "usage: /api/annotations/obs/user/<username>/<suffix>", http.StatusBadRequest)
		return
	}
	username, rawSuffix := parts[0], parts[1]
	if username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	rawText := strings.HasSuffix(rawSuffix, ".txt")
	suffix := strings.TrimSuffix(rawSuffix, ".txt")
	if !validSuffixes[suffix] {
		http.Error(w, fmt.Sprintf("unknown suffix %q", suffix), http.StatusBadRequest)
		return
	}

	streamKey := "user/" + strings.ToLower(username)

	if suffix == "events" {
		h.serveUserSSE(w, r, streamKey, username)
		return
	}

	ctx := r.Context()
	gameUUID := h.resolveUserGame(ctx, username)

	var value string
	if gameUUID == "" {
		value = obsPlaceholder
	} else if data, doc, err := h.loadGameOBSData(ctx, gameUUID); err != nil {
		value = obsPlaceholder
	} else {
		// Best-effort: an unresolved UUID just falls back to username matching.
		userUUID, _ := h.queries.GetUserUUIDByUsername(ctx, username)
		data.OpponentName = opponentName(doc, userUUID, username)
		value = obsFieldValue(data, suffix)
	}

	eventsURL := OBSUserHandlerPrefix + username + "/events"
	h.serveResponse(w, suffix, value, eventsURL, rawText)
}

// resolveUserGame returns the game UUID of the user's most-recently-edited
// annotated game, or "" if none exists.
func (h *OBSHandler) resolveUserGame(ctx context.Context, username string) string {
	row, err := h.queries.GetLatestAnnotatedGameForUsername(ctx, username)
	if err != nil {
		return ""
	}
	return row.GameUuid
}

// serveUserSSE opens a Server-Sent Events stream for the user-alias endpoint.
func (h *OBSHandler) serveUserSSE(w http.ResponseWriter, r *http.Request, streamKey, username string) {
	ctx := r.Context()

	// Resolve username → userUUID once at connect time (needed for NATS subject).
	userUUID, err := h.queries.GetUserUUIDByUsername(ctx, username)
	if err != nil || userUUID == "" {
		// Unknown username — still serve an open SSE so the browser source doesn't
		// error out. It will just receive heartbeats until data appears.
		flusher, ok := sseSetup(w)
		if !ok {
			return
		}
		heartbeat := time.NewTicker(20 * time.Second)
		defer heartbeat.Stop()
		for {
			select {
			case <-heartbeat.C:
				fmt.Fprintf(w, ": heartbeat\n\n")
				flusher.Flush()
			case <-ctx.Done():
				return
			}
		}
	}

	gameUUID := h.resolveUserGame(ctx, username)

	flusher, ok := sseSetup(w)
	if !ok {
		return
	}

	ch := make(chan OBSData, 8)
	stream := h.addSubscriber(streamKey, ch)
	defer h.removeSubscriber(streamKey, ch)

	h.ensureUserNATSSubs(stream, streamKey, userUUID, gameUUID, username)

	h.serveSSELoop(w, ctx, stream, ch, flusher, gameUUID)
}

// ensureUserNATSSubs subscribes once to:
//  1. channel.anno<gameUUID>   — per-turn events for the current game
//  2. anno.user.<userUUID>     — activity signal; fires when the user's latest game may change
func (h *OBSHandler) ensureUserNATSSubs(stream *obsStream, key, userUUID, gameUUID, username string) {
	stream.mu.Lock()
	if stream.initialized {
		stream.mu.Unlock()
		return
	}
	stream.initialized = true
	stream.currentGameID = gameUUID
	stream.mode = obsStreamModeUser
	stream.trackedUserUUID = userUUID
	stream.trackedUsername = username
	stream.mu.Unlock()

	// Per-turn subscription for the currently-resolved game.
	var annoSub *nats.Subscription
	if gameUUID != "" {
		var err error
		annoSub, err = h.natsConn.Subscribe("channel.anno"+gameUUID, func(_ *nats.Msg) {
			h.reloadAndFanout(stream, key, gameUUID)
		})
		if err != nil {
			log.Err(err).Str("gameUUID", gameUUID).Msg("obs-user-anno-sub-error")
			annoSub = nil
		}
	}

	// User-level activity subscription — re-queries and rebinds if the latest game changed.
	userSub, err := h.natsConn.Subscribe(natsUserAnnoSubjectPrefix+userUUID, func(_ *nats.Msg) {
		ctx := context.Background()
		newGameUUID := h.resolveUserGame(ctx, username)

		stream.mu.Lock()
		currentGameID := stream.currentGameID
		stream.mu.Unlock()

		if newGameUUID != currentGameID {
			h.rebindAnnoSub(stream, key, newGameUUID)
		}
		if newGameUUID != "" {
			h.reloadAndFanout(stream, key, newGameUUID)
		}
	})
	if err != nil {
		log.Err(err).Str("userUUID", userUUID).Msg("obs-user-sub-error")
		if annoSub != nil {
			annoSub.Unsubscribe()
		}
		return
	}

	// Atomically assign both subs only if the stream is still live.
	h.mu.Lock()
	if h.streams[key] != stream {
		h.mu.Unlock()
		if annoSub != nil {
			annoSub.Unsubscribe()
		}
		userSub.Unsubscribe()
		return
	}
	stream.mu.Lock()
	stream.annoSub = annoSub
	stream.broadcastSub = userSub // reuse broadcastSub slot for the user-level sub
	stream.mu.Unlock()
	h.mu.Unlock()
}

// ---------------------------------------------------------------------------
// HTML template
// ---------------------------------------------------------------------------

// obsDefaultSize returns the default font size (px) for a given suffix.
func obsDefaultSize(suffix string) int {
	switch suffix {
	case "score", "p1_score", "p2_score":
		return 48
	case "blank1", "blank2":
		return 36
	case "last_play":
		return 24
	case "p1_name", "p2_name", "combined_names":
		return 32
	default: // unseen_tiles, unseen_count
		return 20
	}
}

// obsIsMarquee returns true for fields that should scroll horizontally.
func obsIsMarquee(suffix string) bool {
	return suffix == "last_play"
}

// obsIsBlank returns true for fields that render blank-designated letters in color.
func obsIsBlank(suffix string) bool {
	return suffix == "blank1" || suffix == "blank2"
}

// obsPageTmpl renders an auto-updating OBS browser-source page.
//
// Appearance is controlled by URL query parameters — all optional:
//
//	bg        background color  (default: white)
//	color     text color        (default: black)
//	size      font size in px   (default: per field — 48 for score, 24 for last_play, 20 otherwise)
//	font      font-family CSS   (default: 'Courier New', monospace)
//	bold      0 to disable bold (default: bold on)
//	speed     marquee loop duration in seconds  (default: 20; last_play only)
//	padding   body padding in px               (default: 8)
//	wrap      max characters per line before wrapping (default: 0 = no wrap)
var obsPageTmpl = template.Must(template.New("obs").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body {
  overflow: hidden;
  width: 100vw; height: 100vh;
  display: flex;
  align-items: center;
}
#content {
  width: 100%;
  font-family: 'Courier New', monospace;
  font-weight: bold;
  white-space: pre;
  line-height: 1.2;
  text-align: center;
}
/* marquee */
.mq-wrap {
  width: 100%;
  overflow: hidden;
}
.mq-inner {
  display: inline-flex;
  white-space: nowrap;
  /* animation set in JS below, once the text's rendered width is known */
}
.mq-seg {
  flex: 0 0 auto;
  padding-right: 2em;
}
/* One-shot lead-in: slides in the last 1em before the resting/flush
   position, then hands off to the perfectly seamless mq-scroll loop. Baking
   this offset into mq-scroll's own keyframes instead would break the loop's
   math — it only repeats seamlessly when it travels exactly one copy-width
   per cycle, so a jump would reappear at every restart. */
@keyframes mq-intro {
  from { transform: translateX(1em); }
  to   { transform: translateX(0); }
}
@keyframes mq-scroll {
  from { transform: translateX(0); }
  to   { transform: translateX(-50%); }
}
/* blank-designated letters */
.blank-letter { color: var(--blank-color, #d33300); }
</style>
</head>
<body>
<pre id="content"></pre>
<script>
(function() {
  var field      = {{.Field}};
  var isMarquee  = {{.IsMarquee}};
  var isBlank    = {{.IsBlank}};
  var defSize    = {{.DefaultSize}};
  var eventsURL  = {{.EventsURL}};
  var initVal    = {{.InitialValue}};
  /* ---- read query params ---- */
  var p = new URLSearchParams(location.search);
  var bg         = p.get('bg')      || 'white';
  var color      = p.get('color')   || 'black';
  var size       = parseInt(p.get('size') || defSize, 10);
  var font       = p.get('font')    || "'Courier New', monospace";
  var bold       = p.get('bold')    !== '0';
  var speed      = parseFloat(p.get('speed') || '80');  /* px/s */
  var padding    = parseInt(p.get('padding') || '8', 10);
  var blankColor = p.get('blank')   || '#d33300';
  var maxWrap    = parseInt(p.get('wrap')  || '0', 10);

  /* ---- apply styles ---- */
  document.documentElement.style.setProperty('--blank-color', blankColor);
  document.body.style.background = bg;
  document.body.style.padding    = padding + 'px';
  var content = document.getElementById('content');
  content.style.color      = color;
  content.style.fontSize   = size + 'px';
  content.style.fontFamily = font;
  content.style.fontWeight = bold ? 'bold' : 'normal';

  /* ---- marquee setup ---- */
  function setMarqueeSpeed(el) {
    /* el contains two identical copies of the text (for a seamless loop),
       so one copy's width is half of the element's total width. Chain a
       one-shot mq-intro (the 1em head start) into the infinite mq-scroll
       loop, handing off at the exact position/time mq-scroll expects —
       otherwise the loop restarts from a different offset than it ended
       on, and jumps every cycle. */
    var copyWidth = el.offsetWidth / 2;
    if (copyWidth <= 0) return;
    var introDur = size / speed;   /* 1em resolves to the size (px) variable */
    var loopDur  = copyWidth / speed;
    el.style.animation =
      'mq-intro ' + introDur + 's linear 1 forwards, ' +
      'mq-scroll ' + loopDur + 's linear ' + introDur + 's infinite';
  }

  /* ---- line wrapping ---- */
  function wrapText(text) {
    if (!maxWrap) return text;
    var tokens = text.split(' ');
    var lines = [];
    var cur = '';
    for (var i = 0; i < tokens.length; i++) {
      var tok = tokens[i];
      if (tok === '') continue;
      if (cur === '') {
        cur = tok;
      } else if (cur.length + 1 + tok.length <= maxWrap) {
        cur += ' ' + tok;
      } else {
        lines.push(cur);
        cur = tok;
      }
    }
    if (cur) lines.push(cur);
    return lines.join('\n');
  }

  /* ---- blank highlighting ---- */
  function htmlEscape(s) {
    return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
  }
  function applyBlanks(text) {
    return htmlEscape(text).replace(/[a-z]/g, function(ch) {
      return '<span class="blank-letter">' + ch + '</span>';
    });
  }

  var lastText = null;

  function setText(text) {
    text = wrapText(text);
    /* Skip the DOM rebuild when the value hasn't actually changed — the SSE
       stream re-sends every field on any game-state update (e.g. a rack
       edit), not just the fields that changed, and rebuilding the marquee's
       DOM restarts its CSS animation from scratch. */
    if (text === lastText) return;
    lastText = text;

    if (isMarquee) {
      var mwrap = document.createElement('div');
      mwrap.className = 'mq-wrap';
      var inner = document.createElement('div');
      inner.className = 'mq-inner';
      var seg1 = document.createElement('span');
      seg1.className = 'mq-seg';
      seg1.textContent = text;
      var seg2 = document.createElement('span');
      seg2.className = 'mq-seg';
      seg2.textContent = text;
      inner.appendChild(seg1);
      inner.appendChild(seg2);
      mwrap.appendChild(inner);
      content.innerHTML = '';
      content.appendChild(mwrap);
      setMarqueeSpeed(inner);  /* measure after in DOM so offsetWidth is live */
    } else if (isBlank) {
      content.innerHTML = applyBlanks(text);
    } else {
      content.textContent = text;
    }
  }

  setText(initVal);

  /* ---- SSE ---- */
  var retryTimer;
  function connectSSE() {
    var es = new EventSource(eventsURL);
    es.onmessage = function(e) {
      var d = JSON.parse(e.data);
      if (d[field] !== undefined) setText(d[field]);
    };
    es.onerror = function() {
      console.warn('OBS SSE disconnected — retrying in 5s');
      es.close();
      clearTimeout(retryTimer);
      retryTimer = setTimeout(connectSSE, 5000);
    };
  }
  connectSSE();
})();
</script>
</body>
</html>
`))

type obsPageData struct {
	Field        template.JS
	IsMarquee    template.JS
	IsBlank      template.JS
	DefaultSize  int
	InitialValue template.JS // JS string literal, so quotes are included
	EventsURL    template.JS
}

func serveOBSPage(w http.ResponseWriter, suffix, initialValue, eventsURL string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	obsPageTmpl.Execute(w, obsPageData{
		Field:        template.JS(fmt.Sprintf("%q", suffix)),
		IsMarquee:    template.JS(fmt.Sprintf("%v", obsIsMarquee(suffix))),
		IsBlank:      template.JS(fmt.Sprintf("%v", obsIsBlank(suffix))),
		DefaultSize:  obsDefaultSize(suffix),
		InitialValue: template.JS(fmt.Sprintf("%q", initialValue)),
		EventsURL:    template.JS(fmt.Sprintf("%q", eventsURL)),
	})
}
