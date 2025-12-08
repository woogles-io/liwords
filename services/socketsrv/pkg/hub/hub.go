package sockets

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"

	"github.com/woogles-io/liwords/services/socketsrv/pkg/config"
)

// A Realm is basically a set of clients. It can be thought of as a game room,
// or perhaps a lobby.
type Realm string

const NullRealm Realm = ""
const LobbyRealm Realm = "lobby"

const ConnPollPeriod = 60 * time.Second

// Metrics tracking for getFollowers API requests
var (
	getFollowersRequestCount int64 // Total API requests made
	getFollowersDataBytes    int64 // Total bytes received from API responses
	lastReportedRequestCount int64 // For calculating per-minute rate
	lastReportedDataBytes    int64 // For calculating per-minute data volume
)

// A RealmMessage is a message that should be sent to a socket Realm.
type RealmMessage struct {
	realm Realm
	msg   []byte
}

// A UserMessage is a message that should be sent to a user (across all
// of the sockets that they are connected to, unless the channel says otherwise).
type UserMessage struct {
	userID  string
	channel string
	msg     []byte
}

// A ConnMessage is a message that just gets sent to a single socket connection.
type ConnMessage struct {
	connID string
	msg    []byte
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients         map[*Client][]Realm
	clientsByUserID map[string]map[*Client]bool
	clientsByConnID map[string]*Client
	// Inbound messages from the clients.
	// broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	pubsub *PubSub

	realmMutex sync.Mutex
	// Each realm has a list of clients in it.
	realms map[Realm]map[*Client]bool

	broadcastRealm  chan RealmMessage
	broadcastUser   chan UserMessage
	sendConnMessage chan ConnMessage
}

func NewHub(cfg *config.Config) (*Hub, error) {
	pubsub, err := newPubSub(cfg.NatsURL, cfg)
	if err != nil {
		return nil, err
	}

	// Initialize followers cache
	InitFollowersCache(cfg.FollowersCacheSize, cfg.FollowersCacheTTL)

	// Initialize follows cache (uses same config as followers cache)
	InitFollowsCache(cfg.FollowersCacheSize, cfg.FollowersCacheTTL)

	return &Hub{
		// broadcast:         make(chan []byte),
		broadcastRealm:  make(chan RealmMessage),
		broadcastUser:   make(chan UserMessage),
		sendConnMessage: make(chan ConnMessage),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		clients:         make(map[*Client][]Realm),
		clientsByUserID: make(map[string]map[*Client]bool),
		clientsByConnID: make(map[string]*Client),
		realms:          make(map[Realm]map[*Client]bool),
		pubsub:          pubsub,
	}, nil
}

func (h *Hub) addClient(client *Client) error {

	// Add client to appropriate maps
	byUser := h.clientsByUserID[client.userID]
	if byUser == nil {
		h.clientsByUserID[client.userID] = make(map[*Client]bool)
	}
	// Add the new user ID to the map.
	h.clientsByUserID[client.userID][client] = true
	h.clientsByConnID[client.connID] = client
	// add to the realm map.
	h.addToRealm(client.tempRealms, client)
	client.tempRealms = []string{}

	// Meow, depending on the realm, request that the API publish
	// initial information pertaining to this realm. For example,
	// lobby visitors will want to see a list of sought games,
	// or newcomers to a game realm will want to see the history
	// of the game so far.
	return h.sendRealmInitInfo(client)
	// The API will publish the initial realm information to this user's channel.
	// (user.userID - see pubsub.go)

}

func (h *Hub) removeClient(c *Client) error {
	// no need to protect with mutex, only called from
	// single-threaded Run
	log.Debug().Str("client", c.username).Str("connid", c.connID).Str("userid", c.userID).Msg("removing client")
	close(c.send)

	realms := h.clients[c]

	for _, realm := range realms {
		delete(h.realms[realm], c)
		log.Debug().Msgf("deleted client %v from realm %v. New length %v", c.connID, realm, len(
			h.realms[realm]))

		if len(h.realms[realm]) == 0 {
			delete(h.realms, realm)
		}
	}

	delete(h.clients, c)
	log.Debug().Msgf("deleted client %v from clients. New length %v", c.connID, len(
		h.clients))
	delete(h.clientsByConnID, c.connID)

	// xxx: trigger leaveSite even if this isn't the last tab. We would
	// pass in a conn ID of some sort. We would associate outgoing
	// seek / match requests with a conn ID.

	h.pubsub.natsconn.Publish(extendTopic(c, "ipc.pb.leaveTab"), []byte{})

	if (len(h.clientsByUserID[c.userID])) == 1 {
		delete(h.clientsByUserID, c.userID)
		log.Debug().Msgf("deleted client from clientsbyuserid. New length %v", len(
			h.clientsByUserID))

		// Tell the backend that this user has left the site. The backend
		// can then do things (cancel seek requests, inform players their
		// opponent has left, etc).
		h.pubsub.natsconn.Publish(extendTopic(c, "ipc.pb.leaveSite"), []byte{})
		return nil
	}
	// Otherwise, delete just the right socket (this one: c)
	log.Debug().Interface("userid", c.userID).Int("numconn", len(h.clientsByUserID[c.userID])).
		Msg("non-one-num-conns")
	delete(h.clientsByUserID[c.userID], c)

	return nil
}

func (h *Hub) sendToRealm(realm Realm, msg []byte) error {
	h.broadcastRealm <- RealmMessage{realm: realm, msg: msg}
	return nil
}

func (h *Hub) sendToConnID(connID string, msg []byte) error {
	h.sendConnMessage <- ConnMessage{connID: connID, msg: msg}
	return nil
}

func (h *Hub) sendToUser(userID string, msg []byte) error {
	h.broadcastUser <- UserMessage{userID: userID, msg: msg}
	return nil
}

func (h *Hub) sendToUserChannel(userID string, msg []byte, channel string) error {
	h.broadcastUser <- UserMessage{userID: userID, msg: msg, channel: channel}
	return nil
}

func realmToChannel(realm Realm) string {
	return strings.ReplaceAll(string(realm), "-", ".")
}

func channelToRealm(channel string) Realm {
	return Realm(strings.ReplaceAll(channel, ".", "-"))
}

func (h *Hub) Run() {
	go h.PubsubProcess()
	ticker := time.NewTicker(ConnPollPeriod)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case client := <-h.register:
			err := h.addClient(client)
			if err != nil {
				log.Err(err).Msg("error-adding-client")
			}
			log.Info().Str("username", client.username).Msg("registered-client")

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				err := h.removeClient(client)
				if err != nil {
					log.Err(err).Msg("error-removing-client")
				}
				log.Info().Str("username", client.username).Msg("unregistered-client")
			} else {
				log.Error().Msg("unregistered-but-not-in-map")
			}

		case message := <-h.broadcastRealm:
			// {"level":"debug","realm":"lobby","clients":2,"time":"2020-08-22T20:40:40Z","message":"sending broadcast message to realm"}
			log.Debug().Str("realm", string(message.realm)).
				Int("clients", len(h.realms[message.realm])).
				Msg("sending broadcast message to realm")
			for client := range h.realms[message.realm] {
				select {
				case client.send <- message.msg:
				default:
					log.Debug().Str("username", client.username).Msg("in broadcastRealm, removeClient")
					h.removeClient(client)
				}
			}

		case message := <-h.broadcastUser:
			log.Debug().Str("user", string(message.userID)).
				Msg("sending to all user sockets")
			// Send the message to every socket belonging to this user.
			for client := range h.clientsByUserID[message.userID] {
				canSend := true
				if message.channel != "" {
					canSend = false
					// Determine if we can send this message to this client.
					for _, realm := range client.realms {
						if strings.HasPrefix(message.channel, realmToChannel(realm)) {
							// if the message has a channel attached to it, it needs to be
							// a prefix of the realm in order to be delivered.
							canSend = true
							break
						}
					}
				}
				if !canSend {
					continue
				}
				select {
				case client.send <- message.msg:
				default:
					log.Debug().Str("username", client.username).Msg("in broadcastUser, removeClient")
					h.removeClient(client)
				}
			}

		case message := <-h.sendConnMessage:
			c, ok := h.clientsByConnID[message.connID]
			if !ok {
				// This client does not exist in this node.
				log.Debug().Str("connID", message.connID).Msg("connID-not-found")
			} else {
				select {
				case c.send <- message.msg:
				default:
					log.Debug().Str("connID", message.connID).Msg("in sendToConnID, removeClient")
					h.removeClient(c)
				}
			}

		case <-ticker.C:
			// Calculate per-minute rates for getFollowers API
			currentRequests := atomic.LoadInt64(&getFollowersRequestCount)
			currentDataBytes := atomic.LoadInt64(&getFollowersDataBytes)
			requestsPerMin := currentRequests - atomic.LoadInt64(&lastReportedRequestCount)
			dataBytesPerMin := currentDataBytes - atomic.LoadInt64(&lastReportedDataBytes)

			// Update last reported values
			atomic.StoreInt64(&lastReportedRequestCount, currentRequests)
			atomic.StoreInt64(&lastReportedDataBytes, currentDataBytes)

			followersCacheLen := GetCacheStats()
			followsCacheLen := GetFollowsCacheStats()
			log.Info().Int("num-conns", len(h.clients)).
				Int("num-users", len(h.clientsByUserID)).
				Int("num-realms", len(h.realms)).
				Int("followers-cache-entries", followersCacheLen).
				Int("follows-cache-entries", followsCacheLen).
				Int64("followers-api-requests-per-min", requestsPerMin).
				Int64("followers-api-bytes-per-min", dataBytesPerMin).
				Msg("conn-stats")
		}
	}
}

func (h *Hub) addToRealm(realms []string, client *Client) {
	// a client can be in a set of realms, but these realms are basically
	// immutable (for now). If the client wants to change realms, we have
	// to create new client connection.
	// NOTE: For protocol v2, use JoinRealms/LeaveRealms instead for dynamic realm management.

	h.clients[client] = []Realm{}
	for _, realm := range realms {
		realm := Realm(realm)
		if h.realms[realm] == nil {
			h.realms[realm] = make(map[*Client]bool)
		}
		client.realms = append(client.realms, realm)
		h.realms[realm][client] = true
		h.clients[client] = append(h.clients[client], realm)
	}

}

// JoinRealms adds client to specified realms dynamically (for protocol v2).
// This is thread-safe and can be called during the connection lifetime.
func (h *Hub) JoinRealms(client *Client, realmStrs []string) {
	h.realmMutex.Lock()
	defer h.realmMutex.Unlock()

	for _, realmStr := range realmStrs {
		realm := Realm(realmStr)

		// Check if already in realm
		alreadyIn := false
		for _, r := range client.realms {
			if r == realm {
				alreadyIn = true
				break
			}
		}
		if alreadyIn {
			continue
		}

		// Add to realm
		if h.realms[realm] == nil {
			h.realms[realm] = make(map[*Client]bool)
		}
		h.realms[realm][client] = true
		client.realms = append(client.realms, realm)
		h.clients[client] = append(h.clients[client], realm)

		log.Debug().Str("connID", client.connID).Str("realm", realmStr).Msg("joined-realm")
	}
}

// LeaveRealms removes client from specified realms dynamically (for protocol v2).
// This is thread-safe and can be called during the connection lifetime.
func (h *Hub) LeaveRealms(client *Client, realmStrs []string) {
	h.realmMutex.Lock()
	defer h.realmMutex.Unlock()

	for _, realmStr := range realmStrs {
		realm := Realm(realmStr)

		// Remove from realm map
		delete(h.realms[realm], client)

		// Clean up empty realms
		if len(h.realms[realm]) == 0 {
			delete(h.realms, realm)
		}

		// Remove from client's realm list
		newRealms := make([]Realm, 0, len(client.realms))
		for _, r := range client.realms {
			if r != realm {
				newRealms = append(newRealms, r)
			}
		}
		client.realms = newRealms

		// Update h.clients as well
		newClientRealms := make([]Realm, 0)
		for _, r := range h.clients[client] {
			if r != realm {
				newClientRealms = append(newClientRealms, r)
			}
		}
		h.clients[client] = newClientRealms

		log.Debug().Str("connID", client.connID).Str("realm", realmStr).Msg("left-realm")
	}
}

// LeaveAllRealms removes client from all realms (for protocol v2 unsubscribe all).
func (h *Hub) LeaveAllRealms(client *Client) {
	realmStrs := make([]string, len(client.realms))
	for i, r := range client.realms {
		realmStrs[i] = string(r)
	}
	h.LeaveRealms(client, realmStrs)
}

func (h *Hub) socketLogin(c *Client) error {

	token, err := jwt.Parse(c.connToken, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(os.Getenv("SECRET_KEY")), nil
	})
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {

		c.authenticated, ok = claims["a"].(bool)
		if !ok {
			return errors.New("malformed token - a")
		}
		c.username, ok = claims["unn"].(string)
		if !ok {
			return errors.New("malformed token - unn")
		}

		c.userID = claims["uid"].(string)
		log.Debug().Str("username", c.username).Str("userID", c.userID).
			Bool("auth", c.authenticated).Msg("socket connection")
	}
	if err != nil {
		log.Err(err).Str("token", c.connToken).Msg("socket-login-failure")
	}
	if !token.Valid {
		return errors.New("invalid token")
	}
	return err
}

// Note: This is a BLOCKING call -- see natsconn.Request below.
func registerRealm(c *Client, path string, h *Hub) error {
	// There are a variety of possible realms that a person joining a game
	// can be in. We should not trust the user to send the right realm
	// (for example they can send a TV mode realm if they're a player
	// in the game or vice versa). The backend should determine the right realm
	// and assign it accordingly.
	log.Debug().Str("connid", c.connID).Str("path", path).Msg("register-realm-path")
	var realms []string

	if path == "/" {
		// This is the lobby; no need to request a realm.
		realms = []string{string(LobbyRealm), "chat-" + string(LobbyRealm)}
	} else {
		// First, create a request and send to the IPC api:
		rrr := &pb.RegisterRealmRequest{}
		rrr.Path = path
		rrr.UserId = c.userID
		data, err := proto.Marshal(rrr)
		if err != nil {
			return err
		}
		resp, err := h.pubsub.natsconn.Request("ipc.request.registerRealm", data, ipcTimeout)
		if err != nil {
			log.Err(err).Msg("timeout registering realm")
			return err
		}
		log.Debug().Msg("got response from registerRealmReq")
		// The response contains the correct realm for the user.
		rrResp := &pb.RegisterRealmResponse{}
		err = proto.Unmarshal(resp.Data, rrResp)
		if err != nil {
			return err
		}
		realms = rrResp.Realms
	}
	log.Debug().Interface("realms", realms).Msg("setting-realms")

	c.tempRealms = realms
	return nil
}

func (h *Hub) sendRealmInitInfo(c *Client) error {

	realms := []string{}
	for _, r := range c.realms {
		if r != "" {
			realms = append(realms, string(r))
		}
	}

	req := &pb.InitRealmInfo{
		Realms: realms,
		UserId: c.userID,
	}

	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	log.Debug().Interface("initRealmInfo", req).Msg("req-init-realm-info")

	return h.pubsub.natsconn.Publish(extendTopic(c, "ipc.pb.initRealmInfo"), data)

}

// handlePresenceChanged processes new-style efficient presence notifications
func (h *Hub) handlePresenceChanged(userID string, data []byte) {
	log.Debug().Str("userID", userID).Msg("handling-presence-changed")

	// Try to get followers from cache first
	followers, found := GetFollowers(userID)
	var cacheHit bool

	if !found {
		// Cache miss - request followers from main service
		req := &pb.GetFollowersRequest{
			UserId: userID,
		}
		reqData, err := proto.Marshal(req)
		if err != nil {
			log.Err(err).Msg("marshal-get-followers-request")
			return
		}

		resp, err := h.pubsub.natsconn.Request("ipc.request.getFollowers", reqData, ipcTimeout)
		if err != nil {
			log.Err(err).Str("userID", userID).Msg("get-followers-request-failed")
			return
		}

		// Track successful API request
		atomic.AddInt64(&getFollowersRequestCount, 1)
		atomic.AddInt64(&getFollowersDataBytes, int64(len(resp.Data)))

		followersResp := &pb.GetFollowersResponse{}
		err = proto.Unmarshal(resp.Data, followersResp)
		if err != nil {
			log.Err(err).Msg("unmarshal-get-followers-response")
			return
		}

		followers = followersResp.FollowerUserIds
		// Cache the result for future requests
		CacheFollowers(userID, followers)
		cacheHit = false
	} else {
		cacheHit = true
	}

	btsToSend := entity.BytesFromSerializedEvent(data, byte(pb.MessageType_PRESENCE_ENTRY))
	// Send presence notification to all followers (Run() will handle if not connected)
	for _, followerID := range followers {
		h.sendToUser(followerID, btsToSend)
	}

	log.Debug().Str("userID", userID).
		Int("total_followers", len(followers)).
		Bool("cache_hit", cacheHit).
		Msg("presence-changed-processed")
}

func (h *Hub) handleSeekFollowed(seekerUserID string, data []byte) {
	log.Debug().Str("seekerUserID", seekerUserID).Msg("handling-seek-followed")

	// Try to get follows from cache first
	follows, found := GetFollows(seekerUserID)
	var cacheHit bool

	if !found {
		// Cache miss - request follows from main service
		req := &pb.GetFollowsRequest{
			UserId: seekerUserID,
		}
		reqData, err := proto.Marshal(req)
		if err != nil {
			log.Err(err).Msg("marshal-get-follows-request")
			return
		}

		resp, err := h.pubsub.natsconn.Request("ipc.request.getFollows", reqData, ipcTimeout)
		if err != nil {
			log.Err(err).Str("seekerUserID", seekerUserID).Msg("get-follows-request-failed")
			return
		}

		followsResp := &pb.GetFollowsResponse{}
		err = proto.Unmarshal(resp.Data, followsResp)
		if err != nil {
			log.Err(err).Msg("unmarshal-get-follows-response")
			return
		}

		follows = followsResp.FollowUserIds
		// Cache the result for future requests
		CacheFollows(seekerUserID, follows)
		cacheHit = false
	} else {
		cacheHit = true
	}

	btsToSend := entity.BytesFromSerializedEvent(data, byte(pb.MessageType_SEEK_REQUEST))

	// Send seek notification to the seeker themselves (so they see their own seek)
	h.sendToUser(seekerUserID, btsToSend)

	// Send seek notification to all users that the seeker follows
	for _, followedUserID := range follows {
		h.sendToUser(followedUserID, btsToSend)
	}

	log.Debug().Str("seekerUserID", seekerUserID).
		Int("total_follows", len(follows)).
		Bool("cache_hit", cacheHit).
		Msg("seek-followed-processed")
}
