// Package bus is the message bus. This package listens on various NATS channels
// for requests and publishes back responses to the same, or other channels.
// Responsible for talking to the liwords-socket server.
package bus

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"

	nats "github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/stats"
	"github.com/domino14/liwords/pkg/tournament"
	"github.com/domino14/liwords/pkg/user"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

const (
	MaxMessageLength = 500

	AdjudicateInterval   = 10 * time.Second
	GamesCounterInterval = 60 * time.Minute
	// Cancel a game if it hasn't started after this much time.
	CancelAfter = 60 * time.Second
)

const (
	BotRequestID = "bot-request"
)

// Bus is the struct; it should contain all the stores to verify messages, etc.
type Bus struct {
	natsconn        *nats.Conn
	config          *config.Config
	userStore       user.Store
	gameStore       gameplay.GameStore
	soughtGameStore gameplay.SoughtGameStore
	presenceStore   user.PresenceStore
	listStatStore   stats.ListStatStore
	tournamentStore tournament.TournamentStore
	configStore     config.ConfigStore

	redisPool *redis.Pool

	subscriptions []*nats.Subscription
	subchans      map[string]chan *nats.Msg

	gameEventChan       chan *entity.EventWrapper
	tournamentEventChan chan *entity.EventWrapper
}

func NewBus(cfg *config.Config, userStore user.Store, gameStore gameplay.GameStore,
	soughtGameStore gameplay.SoughtGameStore, presenceStore user.PresenceStore,
	listStatStore stats.ListStatStore, tournamentStore tournament.TournamentStore,
	configStore config.ConfigStore, redisPool *redis.Pool) (*Bus, error) {

	natsconn, err := nats.Connect(cfg.NatsURL)

	if err != nil {
		return nil, err
	}
	bus := &Bus{
		natsconn:            natsconn,
		userStore:           userStore,
		gameStore:           gameStore,
		soughtGameStore:     soughtGameStore,
		presenceStore:       presenceStore,
		listStatStore:       listStatStore,
		tournamentStore:     tournamentStore,
		configStore:         configStore,
		subscriptions:       []*nats.Subscription{},
		subchans:            map[string]chan *nats.Msg{},
		config:              cfg,
		gameEventChan:       make(chan *entity.EventWrapper, 64),
		tournamentEventChan: make(chan *entity.EventWrapper, 64),
		redisPool:           redisPool,
	}
	gameStore.SetGameEventChan(bus.gameEventChan)
	tournamentStore.SetTournamentEventChan(bus.tournamentEventChan)

	topics := []string{
		// ipc.pb are generic publishes
		"ipc.pb.>",
		// ipc.request are NATS requests. also uses protobuf
		"ipc.request.>",
	}

	for _, topic := range topics {
		ch := make(chan *nats.Msg, 64)
		var err error
		var sub *nats.Subscription
		if strings.Contains(topic, ".request.") {
			sub, err = natsconn.ChanQueueSubscribe(topic, "requestworkers", ch)
			if err != nil {
				return nil, err
			}
		} else {
			sub, err = natsconn.ChanQueueSubscribe(topic, "pbworkers", ch)
			if err != nil {
				return nil, err
			}
		}
		bus.subscriptions = append(bus.subscriptions, sub)
		bus.subchans[topic] = ch
	}
	return bus, nil
}

// ProcessMessages is very similar to the PubsubProcess in liwords-socket,
// but that's because they do similar things.
func (b *Bus) ProcessMessages(ctx context.Context) {

	ctx = context.WithValue(ctx, gameplay.ConfigCtxKey("config"), &b.config.MacondoConfig)

	// Adjudicate unfinished games every few seconds.
	adjudicator := time.NewTicker(AdjudicateInterval)
	defer adjudicator.Stop()

	gameCounter := time.NewTicker(GamesCounterInterval)
	defer gameCounter.Stop()

outerfor:
	for {
		select {
		case msg := <-b.subchans["ipc.pb.>"]:
			// Regular messages.
			log.Debug().Str("topic", msg.Subject).Msg("got ipc.pb message")
			subtopics := strings.Split(msg.Subject, ".")

			go func() {
				err := b.handleNatsPublish(ctx, subtopics[2:], msg.Data)
				if err != nil {
					log.Err(err).Msg("process-message-publish-error")
					// The user ID should have hopefully come in the topic name.
					// It would be in subtopics[4]
					if len(subtopics) > 4 {
						userID := subtopics[4]
						// XXX might have to make realm specific
						b.pubToUser(userID, entity.WrapEvent(&pb.ErrorMessage{Message: err.Error()},
							pb.MessageType_ERROR_MESSAGE), "")
					}
				}
			}()

		case msg := <-b.subchans["ipc.request.>"]:
			log.Debug().Str("topic", msg.Subject).Msg("got ipc.request")
			// Requests. We must respond on a specific topic.
			subtopics := strings.Split(msg.Subject, ".")

			go func() {
				err := b.handleNatsRequest(ctx, subtopics[2], msg.Reply, msg.Data)
				if err != nil {
					log.Err(err).Msg("process-message-request-error")
					// just send a blank response so there isn't a timeout on
					// the other side.
					// XXX: this is a very specific response to a handleNatsRequest func
					rrResp := &pb.RegisterRealmResponse{
						Realms: []string{""},
					}
					data, err := proto.Marshal(rrResp)
					if err != nil {
						log.Err(err).Msg("marshalling-error")
					} else {
						b.natsconn.Publish(msg.Reply, data)
					}
				}
			}()

		case msg := <-b.gameEventChan:
			// A game event. Publish directly to the right realm.
			topics := msg.Audience()
			data, err := msg.Serialize()
			if err != nil {
				log.Err(err).Msg("serialize-error")
				break
			}
			for _, topic := range topics {
				if strings.HasPrefix(topic, "user.") {

					components := strings.SplitN(topic, ".", 3)
					user := components[1]
					suffix := ""
					if len(components) > 2 {
						suffix = components[2]
					}

					b.pubToUser(user, msg, suffix)
				} else {
					b.natsconn.Publish(topic, data)
				}
			}

		case msg := <-b.tournamentEventChan:
			// A tournament event. Publish to the right realm.
			log.Debug().Interface("msg", msg).Msg("tournament event chan")
			topics := msg.Audience()
			data, err := msg.Serialize()
			if err != nil {
				log.Err(err).Msg("serialize-error")
				break
			}
			for _, topic := range topics {
				b.natsconn.Publish(topic, data)
			}

		case <-ctx.Done():
			log.Info().Msg("pubsub context done, breaking")
			break outerfor

		case <-adjudicator.C:
			err := b.adjudicateGames(ctx)
			if err != nil {
				log.Err(err).Msg("adjudicate-error")
				break
			}

		case <-gameCounter.C:
			n, err := b.gameStore.Count(ctx)
			if err != nil {
				log.Err(err).Msg("count-error")
				break
			}
			log.Info().Int64("game-count", n).Msg("game-stats")
		}

	}

	log.Info().Msg("exiting processMessages loop")
}

func (b *Bus) handleNatsRequest(ctx context.Context, topic string,
	replyTopic string, data []byte) error {

	switch topic {
	case "registerRealm":
		msg := &pb.RegisterRealmRequest{}
		err := proto.Unmarshal(data, msg)
		if err != nil {
			return err
		}
		// The socket server needs to know what realm to subscribe the user to,
		// given they went to the given path. Don't handle the lobby or tournaments,
		// the socket already handles that. Other pages like /about, etc
		// will get a blank realm back. (Eventually we'll create a "global" realm
		// so we can track presence / deliver notifications even on non-game pages)
		path := msg.Path
		userID := msg.UserId
		var realm, tourneyID string
		if strings.HasPrefix(path, "/game/") {
			gameID := strings.TrimPrefix(path, "/game/")
			game, err := b.gameStore.Get(ctx, gameID)
			if err != nil {
				return err
			}
			var foundPlayer bool
			log.Debug().Str("gameID", gameID).Interface("gameHistory", game.History()).Str("userID", userID).
				Msg("register-game-path")
			for i := 0; i < 2; i++ {
				if game.History().Players[i].UserId == userID {
					foundPlayer = true
				}
			}
			if !foundPlayer {
				realm = "gametv-" + gameID
			} else {
				realm = "game-" + gameID
			}
			log.Debug().Str("computed-realm", realm)
			if game.TournamentData != nil {
				tourneyID = strings.ToLower(game.TournamentData.Id)
				log.Debug().Str("tourney-realm-for", tourneyID)
			}
		} else {
			log.Info().Str("path", path).Msg("realm-req-not-handled-sending-blank-realm")
		}
		resp := &pb.RegisterRealmResponse{}
		resp.Realms = []string{realm}
		if realm != "" {
			if tourneyID != "" {
				// Add to the tourney realm, but only the chat component.
				// Players in game don't need to see all the other tourney events
				// (pairings, standings, etc?)
				resp.Realms = append(resp.Realms, "chat-tournament-"+tourneyID)
			}
			// Also add the chat- realm (whether chat-game- or chat-gametv-)
			resp.Realms = append(resp.Realms, "chat-"+realm)
		}
		retdata, err := proto.Marshal(resp)
		if err != nil {
			return err
		}
		b.natsconn.Publish(replyTopic, retdata)
		log.Debug().Str("topic", topic).Str("replyTopic", replyTopic).
			Msg("published response")
	default:
		return fmt.Errorf("unhandled-req-topic: %v", topic)
	}
	return nil
}

// handleNatsPublish runs in a separate goroutine
func (b *Bus) handleNatsPublish(ctx context.Context, subtopics []string, data []byte) error {
	log.Debug().Interface("subtopics", subtopics).Msg("handling nats publish")

	msgType := subtopics[0]
	auth := ""
	userID := ""
	if len(subtopics) > 2 {
		auth = subtopics[1]
		userID = subtopics[2]
	}
	wsConnID := ""
	if len(subtopics) > 3 {
		wsConnID = subtopics[3]
	}

	switch msgType {
	case "seekRequest":
		return b.seekRequest(ctx, auth, userID, wsConnID, data)
	case "matchRequest":
		return b.matchRequest(ctx, auth, userID, wsConnID, data)

	case "chat":
		// The user is subtopics[2]
		evt := &pb.ChatMessage{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		log.Debug().Str("user", userID).Str("msg", evt.Message).Str("channel", evt.Channel).Msg("chat")
		return b.chat(ctx, userID, evt)
	case "declineMatchRequest":
		evt := &pb.DeclineMatchRequest{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		log.Debug().Str("user", userID).Str("reqid", evt.RequestId).Msg("decline-rematch")
		return b.matchDeclined(ctx, evt, userID)

	case "soughtGameProcess":
		evt := &pb.SoughtGameProcessEvent{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}

		return b.gameAccepted(ctx, evt, userID, wsConnID)

	case "gameplayEvent":
		evt := &pb.ClientGameplayEvent{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		entGame, err := gameplay.HandleEvent(ctx, b.gameStore, b.userStore, b.listStatStore,
			b.tournamentStore, userID, evt)
		if err != nil {
			return err
		}
		entGame.RLock()
		// Determine if one of our players is a bot (no bot-vs-bot supported yet?)
		// and if it is the bot's turn.
		if entGame.GameReq != nil &&
			entGame.GameReq.PlayerVsBot &&
			entGame.Game.Playing() != macondopb.PlayState_GAME_OVER &&
			entGame.PlayerIDOnTurn() != userID {

			entGame.RUnlock()
			// Do this in a separate goroutine as it blocks while waiting for bot move.
			go b.handleBotMove(ctx, entGame)
		} else {
			entGame.RUnlock()
		}
		return nil

	case "timedOut":
		evt := &pb.TimedOut{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return gameplay.TimedOut(ctx, b.gameStore, b.userStore, b.listStatStore, b.tournamentStore, evt.UserId, evt.GameId)

	case "initRealmInfo":
		evt := &pb.InitRealmInfo{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return b.initRealmInfo(ctx, evt, wsConnID)
	case "readyForGame":
		evt := &pb.ReadyForGame{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return b.readyForGame(ctx, evt, userID)
	case "leaveSite":
		// There is no event here. We have the user ID in the subject.
		return b.leaveSite(ctx, userID)
	case "leaveTab":
		return b.leaveTab(ctx, userID, wsConnID)
	case "pongReceived":
		return b.pongReceived(ctx, userID, wsConnID)
	default:
		return fmt.Errorf("unhandled-publish-topic: %v", subtopics)
	}
}

func (b *Bus) TournamentEventChannel() chan *entity.EventWrapper {
	return b.tournamentEventChan
}

func (b *Bus) broadcastPresence(username, userID string, anon bool,
	presenceChannels []string, deleting bool) error {

	// broadcast username's presence to the channels.
	log.Debug().Str("username", username).Str("userID", userID).
		Bool("anon", anon).
		Interface("presenceChannels", presenceChannels).
		Bool("deleting", deleting).
		Msg("broadcast-presence")

	for _, c := range presenceChannels {
		toSend := entity.WrapEvent(&pb.UserPresence{
			Username:    username,
			UserId:      userID,
			Channel:     c,
			IsAnonymous: anon,
			Deleting:    deleting,
		}, pb.MessageType_USER_PRESENCE)
		data, err := toSend.Serialize()
		if err != nil {
			return err
		}
		err = b.natsconn.Publish(c, data)
		if err != nil {
			return err
		}
	}
	return nil
}

// XXX: this function needs to be rewritten to be something like pubToConnectionID
// for the most part.
func (b *Bus) pubToUser(userID string, evt *entity.EventWrapper,
	channel string) error {
	// Publish to a user, but pass in a specific channel. Only publish to those
	// user sockets that are in this channel/realm/what-have-you.
	sanitized, err := sanitize(evt, userID)
	bts, err := sanitized.Serialize()
	if err != nil {
		return err
	}
	var fullChannel string
	if channel == "" {
		fullChannel = "user." + userID
	} else {
		fullChannel = "user." + userID + "." + channel
	}

	return b.natsconn.Publish(fullChannel, bts)
}

func (b *Bus) initRealmInfo(ctx context.Context, evt *pb.InitRealmInfo, connID string) error {
	// For consistency sake, use the `dotted` channels for presence
	// i.e. game.<gameID>, gametv.<gameID>
	// The reasoning is that realms should only be cared about by the socket
	// server. The channels are NATS pubsub channels and we use these for chat
	// too.
	username, anon, err := b.userStore.Username(ctx, evt.UserId)
	if err != nil {
		return err
	}

	// The channels with presence should be:
	// chat.lobby
	// chat.tournament.foo
	// chat.game.bar
	// chat.gametv.baz
	// global.presence (when it comes, we edit this later)

	presenceChan := strings.ReplaceAll(evt.Realm, "-", ".")
	if !strings.HasPrefix(presenceChan, "chat.") {
		presenceChan = ""
	}

	if presenceChan != "" {
		log.Debug().Str("presence-chan", presenceChan).Str("username", username).Msg("SetPresence")
		b.presenceStore.SetPresence(ctx, evt.UserId, username, anon, presenceChan, connID)
	}

	if evt.Realm == "lobby" {
		// open seeks
		seeks, err := b.openSeeks(ctx)
		if err != nil {
			return err
		}
		err = b.pubToUser(evt.UserId, seeks, "lobby")
		if err != nil {
			return err
		}
		// live games
		activeGames, err := b.activeGames(ctx, "")
		if err != nil {
			return err
		}
		err = b.pubToUser(evt.UserId, activeGames, "lobby")
		if err != nil {
			return err
		}
		// open match reqs
		matches, err := b.openMatches(ctx, evt.UserId, "")
		if err != nil {
			return err
		}
		err = b.pubToUser(evt.UserId, matches, "lobby")
		if err != nil {
			return err
		}
		// TODO: send followed online

	} else if strings.HasPrefix(evt.Realm, "game-") || strings.HasPrefix(evt.Realm, "gametv-") {
		components := strings.Split(evt.Realm, "-")
		prefix := components[0]
		// Get a sanitized history
		gameID := components[1]
		refresher, err := b.gameRefresher(ctx, gameID)
		if err != nil {
			return err
		}
		err = b.pubToUser(evt.UserId, refresher, prefix+"."+gameID)
		if err != nil {
			return err
		}
	} else if strings.HasPrefix(evt.Realm, "tournament-") {
		components := strings.Split(evt.Realm, "-")
		prefix := components[0]
		// Get a sanitized history
		tourneyID := components[1]
		// live games
		activeGames, err := b.activeGames(ctx, tourneyID)
		if err != nil {
			return err
		}
		err = b.pubToUser(evt.UserId, activeGames, prefix+"."+tourneyID)
		if err != nil {
			return err
		}
		// open match reqs
		matches, err := b.openMatches(ctx, evt.UserId, tourneyID)
		if err != nil {
			return err
		}
		err = b.pubToUser(evt.UserId, matches, prefix+"."+tourneyID)
		if err != nil {
			return err
		}

	} else if strings.HasPrefix(evt.Realm, "chat-") {
		chatChan := strings.ReplaceAll(evt.Realm, "-", ".")
		err = b.sendOldChats(evt.UserId, chatChan)
		if err != nil {
			return err
		}
	} else {
		log.Debug().Interface("evt", evt).Msg("no init realm info")
	}

	// Get presence
	if presenceChan != "" {
		pres, err := b.getPresence(ctx, presenceChan)
		if err != nil {
			return err
		}
		err = b.pubToUser(evt.UserId, pres, presenceChan)
		if err != nil {
			return err
		}
		// Also send OUR presence to users in this channel.
		err = b.broadcastPresence(username, evt.UserId, anon, []string{presenceChan}, false)
		if err != nil {
			return err
		}
	}
	return nil
	// send chat info

}

func (b *Bus) getPresence(ctx context.Context, presenceChan string) (*entity.EventWrapper, error) {
	users, err := b.presenceStore.GetInChannel(ctx, presenceChan)
	if err != nil {
		return nil, err
	}
	pbobj := &pb.UserPresences{Presences: []*pb.UserPresence{}}
	for _, u := range users {
		pbobj.Presences = append(pbobj.Presences, &pb.UserPresence{
			Username:    u.Username,
			UserId:      u.UUID,
			Channel:     presenceChan,
			IsAnonymous: u.Anonymous,
		})
	}

	log.Debug().Interface("presences", pbobj.Presences).Msg("get-presences")

	evt := entity.WrapEvent(pbobj, pb.MessageType_USER_PRESENCES)
	return evt, nil
}

func (b *Bus) leaveTab(ctx context.Context, userID, connID string) error {
	username, anon, err := b.userStore.Username(ctx, userID)
	if err != nil {
		return err
	}
	channels, err := b.presenceStore.ClearPresence(ctx, userID, username, anon, connID)
	if err != nil {
		return err
	}
	log.Debug().Interface("channels", channels).Str("connID", connID).Str("username", username).
		Msg("clear presence")

	err = b.broadcastPresence(username, userID, anon, channels, true)
	if err != nil {
		return err
	}

	return b.deleteSoughtForConnID(ctx, connID)
}

func (b *Bus) leaveSite(ctx context.Context, userID string) error {
	log.Debug().Str("userid", userID).Msg("left-site")
	return nil
}

func (b *Bus) pongReceived(ctx context.Context, userID, connID string) error {
	username, anon, err := b.userStore.Username(ctx, userID)
	if err != nil {
		return err
	}
	return b.presenceStore.RenewPresence(ctx, userID, username, anon, connID)
}

func (b *Bus) activeGames(ctx context.Context, tourneyID string) (*entity.EventWrapper, error) {
	gs, err := b.gameStore.ListActive(ctx, tourneyID)

	if err != nil {
		return nil, err
	}
	log.Debug().Interface("active-games", gs).Msg("active-games")

	pbobj := &pb.ActiveGames{Games: []*pb.GameMeta{}}
	for _, g := range gs {
		pbobj.Games = append(pbobj.Games, g)
	}
	evt := entity.WrapEvent(pbobj, pb.MessageType_ACTIVE_GAMES)
	return evt, nil
}
