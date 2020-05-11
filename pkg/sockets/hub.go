package sockets

import (
	"errors"
	"sync"

	"github.com/domino14/crosswords/pkg/config"
	"github.com/domino14/crosswords/pkg/entity"
	"github.com/domino14/crosswords/pkg/game"
	"github.com/lithammer/shortuuid"
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

	/////

	broadcastRealm chan RealmMessage
	// evtMap is a map of eventwrapper channels. Each of these channels
	// is associated with a different realm.
	evtMap map[chan *entity.EventWrapper]Realm
}

func NewHub(gameStore game.GameStore) *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		gameStore:  gameStore,
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
				delete(h.clients, client)
				delete(h.clientsByUsername, client.username)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
					delete(h.clientsByUsername, client.username)
				}
			}

		case message := <-h.broadcastRealm:
			log.Debug().Str("realm", string(message.realm)).
				Msg("sending broadcast message to realm")
			for client := range h.realms[message.realm] {
				select {
				case client.send <- message.msg:
				default:
					close(client.send)
					delete(h.clients, client)
					delete(h.clientsByUsername, client.username)
				}
			}

		}
	}
}

// since addNewRealm can be called from different goroutines, we must protect
// the realm map.
// XXX: Consider treating this inside the main hub Run function above, this might
// get rid of the need for mutices.
func (h *Hub) addNewRealm() Realm {
	realmID := shortuuid.New()
	h.realmMutex.Lock()
	defer h.realmMutex.Unlock()
	realm := Realm(realmID)
	h.realms[realm] = make(map[*Client]bool)
	return realm
}

func (h *Hub) addToRealm(realm Realm, clientUsername string) {
	client := h.clientsByUsername[clientUsername]

	h.realmMutex.Lock()
	defer h.realmMutex.Unlock()
	h.realms[realm][client] = true
}

func (h *Hub) deleteRealm(realm Realm) {
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
