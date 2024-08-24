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

	channelsForReqId     map[requestID]chan []byte
	reqIDsForChannelName map[string]map[requestID]struct{} // req ID to channel name
}

func NewEventApiServer(ustore user.Store, gstore gameplay.GameStore) *EventAPIServer {
	return &EventAPIServer{
		channelsForReqId:     make(map[requestID]chan []byte),
		reqIDsForChannelName: make(map[string]map[requestID]struct{}),
		uStore:               ustore,
		gamestore:            gstore,
	}
}

func (s *EventAPIServer) processEvent(subject string, data []byte) error {
	if sub, ok := s.reqIDsForChannelName[subject]; !ok {
		return nil // no one is listening on this channel.
	} else {
		go func() {
			for rid := range sub {
				c := s.channelsForReqId[rid]
				if c != nil {
					c <- data
				}
			}
		}()
	}
	return nil
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
	reqChan := make(chan []byte)

	// subscribe
	s.Lock()
	userGameChannel := "user." + user.UUID + ".game." + gid
	gameChannel := "game." + gid
	if s.reqIDsForChannelName == nil {
		s.reqIDsForChannelName = make(map[string]map[requestID]struct{})
	}

	for _, chname := range [2]string{userGameChannel, gameChannel} {
		reqs := s.reqIDsForChannelName[chname]
		if reqs == nil {
			s.reqIDsForChannelName[chname] = make(map[requestID]struct{})
		}
		s.reqIDsForChannelName[chname][reqID] = struct{}{}
	}
	if s.channelsForReqId == nil {
		s.channelsForReqId = make(map[requestID]chan []byte)
	}
	s.channelsForReqId[reqID] = reqChan

	log.Debug().Str("reqID", string(reqID)).
		Int("rifcn-map-len", len(s.reqIDsForChannelName)).
		Int("cnfri-map-len", len(s.channelsForReqId)).
		Msg("event-api-new-subscription")

	s.Unlock()

	// clean up at the end
	defer func() {
		s.Lock()
		log.Debug().Str("reqID", string(reqID)).Msg("event-api-cleaning-up")
		delete(s.channelsForReqId, reqID)
		for _, chname := range [2]string{userGameChannel, gameChannel} {
			delete(s.reqIDsForChannelName[chname], reqID)
			if len(s.reqIDsForChannelName[chname]) == 0 {
				delete(s.reqIDsForChannelName, chname)
			}
		}
		log.Debug().Int("rifcn-map-len", len(s.reqIDsForChannelName)).Msg("event-api-cleaned-up")
		log.Debug().Int("cnfri-map-len", len(s.channelsForReqId)).Msg("event-api-cleaned-up")
		s.Unlock()
	}()

infloop:
	for {
		select {
		case msg := <-reqChan:
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
