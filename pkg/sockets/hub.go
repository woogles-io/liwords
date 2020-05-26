package sockets

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/domino14/crosswords/pkg/config"
	"github.com/domino14/crosswords/pkg/entity"
	"github.com/domino14/crosswords/pkg/game"
	"github.com/rs/zerolog/log"
)

// A Realm is basically a set of clients. It can be thought of as a game room,
// or perhaps a lobby.
type Realm string

type RealmMessage struct {
	realm Realm
	msg   []byte
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients           map[*Client]bool
	clientsByUsername map[string]*Client

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	gameStore game.GameStore
	config    *config.Config

	realmMutex sync.Mutex
	// Each realm has a list of clients in it.
	realms map[Realm]map[*Client]bool

	broadcastRealm chan RealmMessage

	eventChan chan *entity.EventWrapper
}

func NewHub(gameStore game.GameStore, cfg *config.Config) *Hub {
	return &Hub{
		broadcast:         make(chan []byte),
		register:          make(chan *Client),
		unregister:        make(chan *Client),
		clients:           make(map[*Client]bool),
		clientsByUsername: make(map[string]*Client),
		realms:            make(map[Realm]map[*Client]bool),
		// eventChan should be buffered to keep the game logic itself
		// as fast as possible.
		eventChan: make(chan *entity.EventWrapper, 50),
		gameStore: gameStore,
		config:    cfg,
	}
}

func (h *Hub) removeClient(c *Client) {
	log.Debug().Str("client", c.username).Msg("removing client")
	close(c.send)
	delete(h.clients, c)
	delete(h.clientsByUsername, c.username)
	for realm := range c.realms {
		delete(h.realms[realm], c)
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.clientsByUsername[client.username] = client
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				h.removeClient(client)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					h.removeClient(client)
				}
			}

		case message := <-h.broadcastRealm:
			log.Debug().Str("realm", string(message.realm)).
				Msg("sending broadcast message to realm")
			for client := range h.realms[message.realm] {
				select {
				case client.send <- message.msg:
				default:
					h.removeClient(client)
				}
			}

		}
	}
}

// RunGameEventHandler runs a separate loop that just handles game events,
// and forwards them to the appropriate sockets. All other events, like chat,
// should be handled in the Run function (I think, subject to change).
func (h *Hub) RunGameEventHandler() {
	for {
		select {
		case w := <-h.eventChan:
			realm := Realm(w.GameID())
			err := h.sendToRealm(realm, w)
			if err != nil {
				log.Err(err).Str("realm", string(realm)).Msg("sending to realm")
			}
			n := len(h.eventChan)
			// Also send any backed up events to the appropriate realms.
			if n > 0 {
				log.Info().Int("backpressure", n).Msg("game event channel")
			}
			for i := 0; i < n; i++ {
				w := <-h.eventChan
				err := h.sendToRealm(realm, w)
				if err != nil {
					log.Err(err).Str("realm", string(realm)).Msg("sending to realm")
				}
			}

		}
	}
}

// since addNewRealm can be called from different goroutines, we must protect
// the realm map.
func (h *Hub) addNewRealm(id string) Realm {
	h.realmMutex.Lock()
	defer h.realmMutex.Unlock()
	realm := Realm(id)
	h.realms[realm] = make(map[*Client]bool)
	return realm
}

func (h *Hub) addToRealm(realm Realm, clientUsername string) {
	client := h.clientsByUsername[clientUsername]

	h.realmMutex.Lock()
	defer h.realmMutex.Unlock()
	h.realms[realm][client] = true
	client.realms[realm] = true
}

func (h *Hub) deleteRealm(realm Realm) {
	// XXX: maybe disallow if users in realm.
	h.realmMutex.Lock()
	defer h.realmMutex.Unlock()
	delete(h.realms, realm)
}

func (h *Hub) sendToRealm(realm Realm, w *entity.EventWrapper) error {
	bytes, err := w.Serialize()
	if err != nil {
		return err
	}
	if len(h.realms[realm]) == 0 {
		return errors.New("realm is empty")
	}
	h.broadcastRealm <- RealmMessage{realm: realm, msg: bytes}
	return nil
}

type ClientSeek struct {
	Seeker        string `json:"seeker"`
	Lexicon       string `json:"lexicon"`
	TimeControl   string `json:"timeControl"`
	ChallengeRule string `json:"challengeRule"`
}

func (h *Hub) NewSeekRequest(cs *ClientSeek) error {
	bts, err := json.Marshal(cs)
	if err != nil {
		return err
	}

	h.broadcast <- bts
	return nil
}
