// Package bus is the message bus. This package listens on various NATS channels
// for requests and publishes back responses to the same, or other channels.
// Responsible for talking to the liwords-socket server.
package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	nats "github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/sessions"
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
	SeeksExpireInterval  = 10 * time.Minute
	// Cancel a game if it hasn't started after this much time.
	CancelAfter = 60 * time.Second
)

const (
	BotRequestID = "bot-request"
)

type Stores struct {
	UserStore       user.Store
	GameStore       gameplay.GameStore
	SoughtGameStore gameplay.SoughtGameStore
	PresenceStore   user.PresenceStore
	ChatStore       user.ChatStore
	ListStatStore   stats.ListStatStore
	NotorietyStore  mod.NotorietyStore
	TournamentStore tournament.TournamentStore
	ConfigStore     config.ConfigStore
	SessionStore    sessions.SessionStore
}

// Bus is the struct; it should contain all the stores to verify messages, etc.
type Bus struct {
	natsconn        *nats.Conn
	config          *config.Config
	userStore       user.Store
	gameStore       gameplay.GameStore
	soughtGameStore gameplay.SoughtGameStore
	presenceStore   user.PresenceStore
	listStatStore   stats.ListStatStore
	notorietyStore  mod.NotorietyStore
	tournamentStore tournament.TournamentStore
	configStore     config.ConfigStore
	chatStore       user.ChatStore

	redisPool *redis.Pool

	subscriptions []*nats.Subscription
	subchans      map[string]chan *nats.Msg

	gameEventChan       chan *entity.EventWrapper
	tournamentEventChan chan *entity.EventWrapper

	genericEventChan chan *entity.EventWrapper
}

func NewBus(cfg *config.Config, stores Stores, redisPool *redis.Pool) (*Bus, error) {

	natsconn, err := nats.Connect(cfg.NatsURL)

	if err != nil {
		return nil, err
	}
	bus := &Bus{
		natsconn:            natsconn,
		userStore:           stores.UserStore,
		gameStore:           stores.GameStore,
		soughtGameStore:     stores.SoughtGameStore,
		presenceStore:       stores.PresenceStore,
		listStatStore:       stores.ListStatStore,
		tournamentStore:     stores.TournamentStore,
		configStore:         stores.ConfigStore,
		chatStore:           stores.ChatStore,
		subscriptions:       []*nats.Subscription{},
		subchans:            map[string]chan *nats.Msg{},
		config:              cfg,
		gameEventChan:       make(chan *entity.EventWrapper, 64),
		tournamentEventChan: make(chan *entity.EventWrapper, 64),
		genericEventChan:    make(chan *entity.EventWrapper, 64),
		redisPool:           redisPool,
	}
	bus.gameStore.SetGameEventChan(bus.gameEventChan)
	bus.tournamentStore.SetTournamentEventChan(bus.tournamentEventChan)
	bus.chatStore.SetEventChan(bus.genericEventChan)
	bus.presenceStore.SetEventChan(bus.genericEventChan)

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
	ctx = log.Logger.WithContext(ctx)
	log := zerolog.Ctx(ctx)
	// Adjudicate unfinished games every few seconds.
	adjudicator := time.NewTicker(AdjudicateInterval)
	defer adjudicator.Stop()

	gameCounter := time.NewTicker(GamesCounterInterval)
	defer gameCounter.Stop()

	seekExpirer := time.NewTicker(SeeksExpireInterval)
	defer seekExpirer.Stop()

outerfor:
	for {
		select {
		case msg := <-b.subchans["ipc.pb.>"]:
			// Regular messages.
			log := log.With().Interface("msg-subject", msg.Subject).Logger()
			log.Debug().Msg("got ipc.pb message")
			subtopics := strings.Split(msg.Subject, ".")

			go func(subtopics []string, data []byte) {
				err := b.handleNatsPublish(log.WithContext(ctx), subtopics[2:], data)
				if err != nil {
					log.Err(err).Msg("process-message-publish-error")
					// The user ID should have hopefully come in the topic name.
					// It would be in subtopics[4]
					if len(subtopics) > 5 {
						userID := subtopics[4]
						connID := subtopics[5]
						b.pubToConnectionID(connID, userID, entity.WrapEvent(&pb.ErrorMessage{Message: err.Error()},
							pb.MessageType_ERROR_MESSAGE))
					}
				}
			}(subtopics, msg.Data)

		case msg := <-b.subchans["ipc.request.>"]:
			log := log.With().Interface("msg-subject", msg.Subject).Logger()
			log.Debug().Msg("got ipc.request")
			// Requests. We must respond on a specific topic.
			subtopics := strings.Split(msg.Subject, ".")

			go func() {
				err := b.handleNatsRequest(log.WithContext(ctx), subtopics[2], msg.Reply, msg.Data)
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
			if msg.Type == pb.MessageType_ACTIVE_GAME_ENTRY {
				// This message usually has no audience.
				if evt, ok := msg.Event.(*pb.ActiveGameEntry); ok {
					log.Debug().Interface("event", evt).Msg("active-game-entry")
					ret, err := b.presenceStore.UpdateActiveGame(ctx, evt)
					if err != nil {
						log.Err(err).Msg("update-active-game-error")
						// but continue anyway
					} else {
						for idx, chans := range ret {
							oldChannels := chans[0]
							newChannels := chans[1]
							userId := evt.Player[idx].UserId
							username := evt.Player[idx].Username
							if err = b.broadcastChannelChanges(ctx, oldChannels, newChannels, userId, username); err != nil {
								log.Err(err).Msg("broadcast-active-game-error")
								// but continue anyway
							}
						}
					}
				} else {
					log.Error().Interface("event", msg.Event).Msg("bad-active-game-entry")
				}
			}

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

		case msg := <-b.genericEventChan:
			// a Generic event to be published via NATS.
			// Publish to the right realm.
			// XXX: This is identical to tournamentEventChan. Should possibly merge.
			log.Debug().Interface("msg", msg).Msg("generic event chan")
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
			}

		case <-gameCounter.C:
			n, err := b.gameStore.Count(ctx)
			if err != nil {
				log.Err(err).Msg("count-error")
				break
			}
			log.Info().Int64("game-count", n).Msg("game-stats")

		case <-seekExpirer.C:
			err := b.soughtGameStore.ExpireOld(ctx)
			if err != nil {
				log.Err(err).Msg("expiration-error")
			}
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

		resp := &pb.RegisterRealmResponse{}
		currentTournamentID := ""
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
			var realm string
			if !foundPlayer {
				realm = "gametv-" + gameID
			} else {
				realm = "game-" + gameID
			}
			log.Debug().Str("computed-realm", realm)
			resp.Realms = append(resp.Realms, realm, "chat-"+realm)

			if game.TournamentData != nil && game.TournamentData.Id != "" {
				currentTournamentID = game.TournamentData.Id
				tournamentRealm := "tournament-" + currentTournamentID
				resp.Realms = append(resp.Realms, tournamentRealm, "chat-"+tournamentRealm)
			}

		} else if strings.HasPrefix(path, "/tournament/") || strings.HasPrefix(path, "/club/") {
			t, err := b.tournamentStore.GetBySlug(ctx, path)
			if err != nil {
				return err
			}
			currentTournamentID = t.UUID
			tournamentRealm := "tournament-" + currentTournamentID
			resp.Realms = append(resp.Realms, tournamentRealm, "chat-"+tournamentRealm)
		} else {
			log.Info().Str("path", path).Msg("realm-req-not-handled")
		}

		activeTourneys, err := b.tournamentStore.ActiveTournamentsFor(ctx, userID)
		if err != nil {
			return err
		}
		for _, tourney := range activeTourneys {
			// If we are already physically IN the current tournament realm, do not
			// subscribe to this extra channel. This channel is used for messages sitewide.
			if tourney[0] != currentTournamentID {
				channel := tournament.DivisionChannelName(tourney[0], tourney[1])
				resp.Realms = append(resp.Realms, "channel-"+channel)
			}
		}

		retdata, err := proto.Marshal(resp)
		if err != nil {
			return err
		}
		b.natsconn.Publish(replyTopic, retdata)
		log.Debug().Str("topic", topic).Str("replyTopic", replyTopic).Interface("realms", resp.Realms).
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

	pnum, err := strconv.Atoi(msgType)
	if err == nil {
		msgType = pb.MessageType(pnum).String()
	}
	// XXX: Otherwise, ignore error for now

	switch msgType {
	// XXX: remove the camelCased version of these soon, after deploying new socket server
	case "seekRequest", pb.MessageType_SEEK_REQUEST.String():
		return b.seekRequest(ctx, auth, userID, wsConnID, data)
	case "matchRequest", pb.MessageType_MATCH_REQUEST.String():
		return b.matchRequest(ctx, auth, userID, wsConnID, data)
	case "chat", pb.MessageType_CHAT_MESSAGE.String():
		// The user is subtopics[2]
		evt := &pb.ChatMessage{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		log.Debug().Str("user", userID).Str("msg", evt.Message).Str("channel", evt.Channel).Msg("chat")
		return b.chat(ctx, userID, evt)
	case "declineMatchRequest", pb.MessageType_DECLINE_MATCH_REQUEST.String():
		evt := &pb.DeclineMatchRequest{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		log.Debug().Str("user", userID).Str("reqid", evt.RequestId).Msg("decline-rematch")
		return b.matchDeclined(ctx, evt, userID)
	case "gameMetaEvent", pb.MessageType_GAME_META_EVENT.String():
		evt := &pb.GameMetaEvent{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		log.Debug().Str("user", userID).Interface("evt", evt).Msg("game-meta-event")
		return b.gameMetaEvent(ctx, evt, userID)

	case "soughtGameProcess", pb.MessageType_SOUGHT_GAME_PROCESS_EVENT.String():
		evt := &pb.SoughtGameProcessEvent{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}

		return b.gameAccepted(ctx, evt, userID, wsConnID)

	case "gameplayEvent", pb.MessageType_CLIENT_GAMEPLAY_EVENT.String():
		evt := &pb.ClientGameplayEvent{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		entGame, err := gameplay.HandleEvent(ctx, b.gameStore, b.userStore, b.notorietyStore, b.listStatStore,
			b.tournamentStore, userID, evt)
		if err != nil {
			return err
		}
		entGame.RLock()
		defer entGame.RUnlock()
		// Determine if one of our players is a bot (no bot-vs-bot supported yet?)
		// and if it is the bot's turn.
		if entGame.GameReq != nil &&
			entGame.GameReq.PlayerVsBot &&
			entGame.Game.Playing() != macondopb.PlayState_GAME_OVER &&
			entGame.PlayerIDOnTurn() != userID {

			b.goHandleBotMove(ctx, entGame)
		}
		return nil

	case "timedOut", pb.MessageType_TIMED_OUT.String():
		evt := &pb.TimedOut{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return gameplay.TimedOut(ctx, b.gameStore, b.userStore, b.notorietyStore, b.listStatStore, b.tournamentStore, evt.UserId, evt.GameId)

	case "readyForGame", pb.MessageType_READY_FOR_GAME.String():
		evt := &pb.ReadyForGame{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return b.readyForGame(ctx, evt, userID)
	case "readyForTournamentGame", pb.MessageType_READY_FOR_TOURNAMENT_GAME.String():
		evt := &pb.ReadyForTournamentGame{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return b.readyForTournamentGame(ctx, evt, userID, wsConnID)

		// The messages after this are messages sent only from liwords-socket to liwords,
		// so there are no MessageType enums for these. It's ok:
	case "initRealmInfo":
		evt := &pb.InitRealmInfo{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return b.initRealmInfo(ctx, evt, wsConnID)

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

func (b *Bus) pubToUser(userID string, evt *entity.EventWrapper,
	channel string) error {
	// Publish to a user, but pass in a specific channel. Only publish to those
	// user sockets that are in this channel/realm/what-have-you.
	sanitized, err := sanitize(b.userStore, evt, userID)
	if err != nil {
		return err
	}
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

func (b *Bus) pubToConnectionID(connID, userID string, evt *entity.EventWrapper) error {
	// Publish to a specific connection ID.
	sanitized, err := sanitize(b.userStore, evt, userID)
	if err != nil {
		return err
	}
	bts, err := sanitized.Serialize()
	if err != nil {
		return err
	}
	return b.natsconn.Publish("connid."+connID, bts)
}

func didChannelsChange(oldChannels, newChannels []string) bool {
	// just compare the arrays because they're already sort/dedup'ed
	if len(newChannels) != len(oldChannels) {
		return true
	}
	for ri, r := range newChannels {
		if r != oldChannels[ri] {
			return true
		}
	}
	return false
}

func (b *Bus) broadcastChannelChanges(ctx context.Context, oldChannels, newChannels []string, userID, username string) error {
	if !didChannelsChange(oldChannels, newChannels) {
		return nil
	}

	// Courtesy note: followee* is not acceptable in csw19.
	followee, err := b.userStore.GetByUUID(ctx, userID)
	if err != nil {
		return err
	}

	followerUsers, err := b.userStore.GetFollowedBy(ctx, followee.ID)
	if err != nil {
		return err
	}

	if len(followerUsers) > 0 && b.genericEventChan != nil {
		wrapped := entity.WrapEvent(&pb.PresenceEntry{
			Username: username,
			UserId:   userID,
			Channel:  newChannels,
		}, pb.MessageType_PRESENCE_ENTRY)

		for _, fu := range followerUsers {
			wrapped.AddAudience(entity.AudUser, fu.UUID)
		}

		b.genericEventChan <- wrapped
	}

	return nil
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

	for _, realm := range evt.Realms {

		presenceChan := strings.ReplaceAll(realm, "-", ".")
		if !strings.HasPrefix(presenceChan, "chat.") {
			// presenceChan / presenceStore is only used for chat purposes for now.
			presenceChan = ""
		}

		if presenceChan != "" {
			log.Debug().Str("presence-chan", presenceChan).Str("username", username).Msg("SetPresence")
			oldChannels, newChannels, err := b.presenceStore.SetPresence(ctx, evt.UserId, username, anon, presenceChan, connID)
			if err != nil {
				// this was not checked?
				return err
			}
			if err = b.broadcastChannelChanges(ctx, oldChannels, newChannels, evt.UserId, username); err != nil {
				return err
			}
		}

		if realm == "lobby" {
			err := b.sendLobbyContext(ctx, evt.UserId, connID)
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(realm, "game-") || strings.HasPrefix(realm, "gametv-") {
			components := strings.Split(realm, "-")
			// Get a sanitized history
			gameID := components[1]
			refresher, err := b.gameRefresher(ctx, gameID)
			if err != nil {
				return err
			}
			err = b.pubToConnectionID(connID, evt.UserId, refresher)
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(realm, "tournament-") {
			err := b.sendTournamentContext(ctx, realm, evt.UserId, connID)
			if err != nil {
				return err
			}
		} else {
			log.Debug().Interface("evt", evt).Msg("no init realm info")
		}
		// XXX: Need initRealmInfo for `channel-` realm.
		// Get presence
		if presenceChan != "" {
			err := b.sendPresenceContext(ctx, evt.UserId, username, anon,
				presenceChan, connID)
			if err != nil {
				return err
			}
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
	oldChannels, newChannels, channels, err := b.presenceStore.ClearPresence(ctx, userID, username, anon, connID)
	if err != nil {
		return err
	}
	log.Debug().Interface("channels", channels).Str("connID", connID).Str("username", username).
		Msg("clear presence")

	err = b.broadcastPresence(username, userID, anon, channels, true)
	if err != nil {
		return err
	}

	err = b.deleteSoughtForConnID(ctx, connID)
	if err != nil {
		return err
	}
	// Delete any tournament ready messages
	err = b.deleteTournamentReadyMsgs(ctx, userID, connID)
	if err != nil {
		return err
	}
	if err = b.broadcastChannelChanges(ctx, oldChannels, newChannels, userID, username); err != nil {
		return err
	}
	return nil
}

func (b *Bus) deleteTournamentReadyMsgs(ctx context.Context, userID, connID string) error {
	// When a user leaves the site, we want to make sure to clear any of their
	// "tournament ready" messages in the actual tournament.
	conn := b.redisPool.Get()
	defer conn.Close()
	bts, err := redis.Bytes(conn.Do("GET", "tready:"+connID))
	if err != nil {
		// There are probably no such messages for this connection.
		return nil
	}
	readyEvt := pb.ReadyForTournamentGame{}
	err = json.Unmarshal(bts, &readyEvt)
	if err != nil {
		return err
	}
	readyEvt.Unready = true
	err = b.readyForTournamentGame(ctx, &readyEvt, userID, connID)
	if err != nil {
		return err
	}
	// and delete the ready event from redis
	_, err = conn.Do("DEL", "tready:"+connID)
	return err
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
	oldChannels, newChannels, err := b.presenceStore.RenewPresence(ctx, userID, username, anon, connID)
	if err != nil {
		return err
	}
	if err = b.broadcastChannelChanges(ctx, oldChannels, newChannels, userID, username); err != nil {
		return err
	}
	return nil
}

func (b *Bus) activeGames(ctx context.Context, tourneyID string) (*entity.EventWrapper, error) {
	games, err := b.gameStore.ListActive(ctx, tourneyID, false)

	if err != nil {
		return nil, err
	}
	log.Debug().Int("num-active-games", len(games.GameInfo)).Msg("active-games")

	evt := entity.WrapEvent(games, pb.MessageType_ONGOING_GAMES)
	return evt, nil
}

// Return 0 if uid1 blocks uid2, 1 if uid2 blocks uid1, and -1 if neither blocks
// the other. Note, if they both block each other it will return 0.
func (b *Bus) blockExists(ctx context.Context, u1, u2 *entity.User) (int, error) {
	blockedUsers, err := b.userStore.GetBlocks(ctx, u1.ID)
	if err != nil {
		return 0, err
	}
	for _, bu := range blockedUsers {
		if bu.UUID == u2.UUID {
			// u1 is blocking u2
			return 0, nil
		}
	}
	// Check in the other direction
	blockedUsers, err = b.userStore.GetBlockedBy(ctx, u1.ID)
	if err != nil {
		return 0, err
	}
	for _, bu := range blockedUsers {
		if bu.UUID == u2.UUID {
			// u2 is blocking u1
			return 1, nil
		}
	}
	return -1, nil
}

func (b *Bus) sendLobbyContext(ctx context.Context, userID, connID string) error {
	// open seeks
	seeks, err := b.openSeeks(ctx)
	if err != nil {
		return err
	}
	err = b.pubToConnectionID(connID, userID, seeks)
	if err != nil {
		return err
	}
	// live games
	activeGames, err := b.activeGames(ctx, "")
	if err != nil {
		return err
	}
	err = b.pubToConnectionID(connID, userID, activeGames)
	if err != nil {
		return err
	}
	// open match reqs
	matches, err := b.openMatches(ctx, userID, "")
	if err != nil {
		return err
	}
	err = b.pubToConnectionID(connID, userID, matches)
	if err != nil {
		return err
	}

	// TODO: send followed online
	return nil
}

func (b *Bus) sendTournamentContext(ctx context.Context, realm, userID, connID string) error {
	components := strings.Split(realm, "-")
	tourneyID := components[1]
	// live games
	activeGames, err := b.activeGames(ctx, tourneyID)
	if err != nil {
		return err
	}
	err = b.pubToConnectionID(connID, userID, activeGames)
	if err != nil {
		return err
	}
	// open match reqs
	matches, err := b.openMatches(ctx, userID, tourneyID)
	if err != nil {
		return err
	}
	err = b.pubToConnectionID(connID, userID, matches)
	if err != nil {
		return err
	}

	return err
}

func (b *Bus) sendPresenceContext(ctx context.Context, userID, username string, anon bool,
	presenceChan, connID string) error {

	pres, err := b.getPresence(ctx, presenceChan)
	if err != nil {
		return err
	}
	err = b.pubToConnectionID(connID, userID, pres)
	if err != nil {
		return err
	}
	// Also send OUR presence to users in this channel.
	return b.broadcastPresence(username, userID, anon, []string{presenceChan}, false)
}
