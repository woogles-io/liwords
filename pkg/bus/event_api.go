package bus

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lithammer/shortuuid/v4"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/user"
	"google.golang.org/protobuf/encoding/protojson"
)

// message GameEventStreamRequest { string game_id = 1; }

// message GameEventStreamResponse {

// }

// service GameEventAPIService {
//   rpc GetEventStream(GameEventStreamRequest) returns (GameEventStreamResponse);
// }

// ServeHTTP should only be used for the streaming event API.
// This can be mounted at /api/eventstream

const UserEventStreamPrefix = "/api/eventstream/"
const GameEventStreamPrefix = "/api/game/eventstream/"

type requestID string

type EventAPIServer struct {
	sync.Mutex
	uStore    user.Store
	gamestore gameplay.GameStore

	subs                 map[requestID]*eventSub
	reqIDsForChannelName map[string]map[requestID]struct{} // req ID to channel name
}

// eventSubBuffer is how many events a stream client may fall behind by
// before it is cut off. Cut off, it reconnects and gets the game's history
// again.
const eventSubBuffer = 64

// eventSub is one stream client's subscription. processEvent, on the bus
// loop, puts events on ch in the order they arrive and never waits for the
// client: if ch is full, it closes tooSlow instead, and the stream ends.
type eventSub struct {
	ch      chan []byte
	tooSlow chan struct{}
	cut     bool
}

func NewEventApiServer(ustore user.Store, gstore gameplay.GameStore) *EventAPIServer {
	return &EventAPIServer{
		subs:                 make(map[requestID]*eventSub),
		reqIDsForChannelName: make(map[string]map[requestID]struct{}),
		uStore:               ustore,
		gamestore:            gstore,
	}
}

// processEvent hands an event to the stream clients subscribed to subject.
// It runs on the bus loop, so it doesn't wait on any of them.
func (s *EventAPIServer) processEvent(subject string, data []byte) error {
	s.Lock()
	defer s.Unlock()
	for rid := range s.reqIDsForChannelName[subject] {
		sub := s.subs[rid]
		if sub == nil || sub.cut {
			continue
		}
		select {
		case sub.ch <- data:
		default:
			sub.cut = true
			close(sub.tooSlow)
		}
	}
	return nil
}

// subscribe registers a stream client for channels; the returned function
// unregisters it.
func (s *EventAPIServer) subscribe(reqID requestID, channels []string) (*eventSub, func()) {
	sub := &eventSub{ch: make(chan []byte, eventSubBuffer), tooSlow: make(chan struct{})}
	s.Lock()
	defer s.Unlock()
	for _, chname := range channels {
		if s.reqIDsForChannelName[chname] == nil {
			s.reqIDsForChannelName[chname] = make(map[requestID]struct{})
		}
		s.reqIDsForChannelName[chname][reqID] = struct{}{}
	}
	s.subs[reqID] = sub
	log.Debug().Str("reqID", string(reqID)).
		Int("rifcn-map-len", len(s.reqIDsForChannelName)).
		Int("subs-len", len(s.subs)).
		Msg("event-api-new-subscription")

	return sub, func() {
		s.Lock()
		defer s.Unlock()
		log.Debug().Str("reqID", string(reqID)).Msg("event-api-cleaning-up")
		delete(s.subs, reqID)
		for _, chname := range channels {
			delete(s.reqIDsForChannelName[chname], reqID)
			if len(s.reqIDsForChannelName[chname]) == 0 {
				delete(s.reqIDsForChannelName, chname)
			}
		}
	}
}

func (s *EventAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	apikey, err := apiserver.GetAPIKey(ctx)
	if err != nil {
		http.Error(w, "could not get api key from context", 500)
		return
	}
	user, err := s.uStore.GetByAPIKey(ctx, apikey)
	if err != nil {
		log.Err(err).Msg("getting-user-by-apikey")
		http.Error(w, "False. Black bear.", http.StatusUnauthorized)
		return
	}

	var flusher http.Flusher
	var ok bool

	if flusher, ok = w.(http.Flusher); !ok {
		http.Error(w, "no flusher", 500)
		return
	}

	var gid string
	if strings.HasPrefix(r.URL.Path, GameEventStreamPrefix) {
		gid = strings.TrimSpace(strings.TrimPrefix(r.URL.Path, GameEventStreamPrefix))
	} else {
		http.NotFound(w, r)
		return
	}
	if gid == "" {
		http.NotFound(w, r)
		return
	}
	hist, err := s.gamestore.GetHistory(ctx, gid)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	histjson, err := protojson.Marshal(hist)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(append(histjson, '\n'))
	flusher.Flush()
	reqID := requestID(shortuuid.New())
	sub, unsubscribe := s.subscribe(reqID, []string{"user." + user.UUID + ".game." + gid, "game." + gid})
	defer unsubscribe()

infloop:
	for {
		select {
		case msg := <-sub.ch:
			log.Debug().Interface("msg", msg).Msg("got-event")
			e, err := entity.EventFromByteArray(msg)
			if err != nil {
				log.Err(err).Msg("event-parse-error")
				break
			}
			bts, err := protojson.Marshal(e.Event)
			if err != nil {
				log.Err(err).Msg("event-marshal-error")
				break
			}
			log.Debug().Str("writing", string(bts)).Msg("writing-to-client")
			_, err = w.Write(append(bts, '\n'))
			if err != nil {
				log.Err(err).Msg("error-writing-to-client")
			}
			flusher.Flush()
		case <-sub.tooSlow:
			log.Info().Str("reqID", string(reqID)).Msg("event-api-client-too-slow")
			break infloop
		case <-ctx.Done():
			log.Err(ctx.Err()).Str("reqID", string(reqID)).Msg("client-disconnected")
			break infloop
		case <-time.After(12 * time.Hour):
			// this probably shouldn't happen, but just in case we have a bad client.
			log.Info().Str("reqID", string(reqID)).Msg("client-timeout")
			break infloop
		}
	}

}
