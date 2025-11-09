// Package bus is the message bus. This package listens on various NATS channels
// for requests and publishes back responses to the same, or other channels.
// Responsible for talking to the services/socketsrv server.
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

	"github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/omgwords"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/tournament"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	MaxMessageLength = 500

	AdjudicateInterval               = 10 * time.Second
	AdjudicateCorrespondenceInterval = 60 * time.Second
	GamesCounterInterval             = 60 * time.Minute
	SeeksExpireInterval              = 10 * time.Minute
	ChannelMonitorInterval           = 5 * time.Second
	// Cancel a game if it hasn't started after this much time.
	CancelAfter = 60 * time.Second
)

const (
	BotRequestID = "bot-request"
)

// Bus is the struct; it should contain all the stores to verify messages, etc.
type Bus struct {
	natsconn *nats.Conn
	config   *config.Config
	stores   *stores.Stores

	redisPool *redis.Pool

	subscriptions []*nats.Subscription
	subchans      map[string]chan *nats.Msg

	gameEventChan       chan *entity.EventWrapper
	tournamentEventChan chan *entity.EventWrapper

	genericEventChan   chan *entity.EventWrapper
	gameEventAPIServer *EventAPIServer
}

func NewBus(cfg *config.Config, natsconn *nats.Conn, stores *stores.Stores, redisPool *redis.Pool) (*Bus, error) {
	bus := &Bus{
		natsconn:            natsconn,
		stores:              stores,
		subscriptions:       []*nats.Subscription{},
		subchans:            map[string]chan *nats.Msg{},
		config:              cfg,
		gameEventChan:       make(chan *entity.EventWrapper, 64),
		tournamentEventChan: make(chan *entity.EventWrapper, 64),
		// genericEventChan needs to be made a bit bigger for now because it handles
		// follower messages. XXX: Need to fix follower msg architecture ASAP!
		// See https://github.com/woogles-io/liwords/issues/1136
		genericEventChan:   make(chan *entity.EventWrapper, 512),
		redisPool:          redisPool,
		gameEventAPIServer: NewEventApiServer(stores.UserStore, stores.GameStore),
	}
	bus.stores.GameStore.SetGameEventChan(bus.gameEventChan)
	bus.stores.TournamentStore.SetTournamentEventChan(bus.tournamentEventChan)
	bus.stores.ChatStore.SetEventChan(bus.genericEventChan)
	bus.stores.PresenceStore.SetEventChan(bus.genericEventChan)

	topics := []string{
		// ipc.pb are generic publishes
		"ipc.pb.>",
		// ipc.request are NATS requests. also uses protobuf
		"ipc.request.>",

		// The socket server should handle these events, but we also want to handle them
		// here for the purposes of providing an event stream to users of our event api.
		"user.>",
		"game.>",

		// A NATS-compatible bot will publish events on this channel.
		"bot.publish_event.>",
	}

	for _, topic := range topics {
		ch := make(chan *nats.Msg, 512)
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

func (b *Bus) EventAPIServerInstance() *EventAPIServer {
	return b.gameEventAPIServer
}

// ProcessMessages is very similar to the PubsubProcess in services/socketsrv,
// but that's because they do similar things.
func (b *Bus) ProcessMessages(ctx context.Context) {
	ctx = b.config.WithContext(ctx)
	ctx = log.Logger.WithContext(ctx)
	log := zerolog.Ctx(ctx)
	// Adjudicate unfinished real-time games every few seconds.
	adjudicator := time.NewTicker(AdjudicateInterval)
	defer adjudicator.Stop()

	// Adjudicate correspondence games once a minute
	correspondenceAdjudicator := time.NewTicker(AdjudicateCorrespondenceInterval)
	defer correspondenceAdjudicator.Stop()

	seekExpirer := time.NewTicker(SeeksExpireInterval)
	defer seekExpirer.Stop()

	channelMonitor := time.NewTicker(ChannelMonitorInterval)
	defer channelMonitor.Stop()

outerfor:
	for {
		select {
		// NATS message usually from socket service:
		case msg := <-b.subchans["ipc.pb.>"]:
			// Regular messages.
			log := log.With().Str("msg-subject", msg.Subject).Logger()
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

		// NATS message usually from socket service:
		case msg := <-b.subchans["ipc.request.>"]:
			log := log.With().Str("msg-subject", msg.Subject).Logger()
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

		// NATS message from macondo bot
		case msg := <-b.subchans["bot.publish_event.>"]:
			log := log.With().Interface("msg-subject", msg.Subject).Logger()
			log.Debug().Msg("got-bot-publish")
			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) != 3 {
				log.Error().Msg("no-game-id")
				break
			}
			gid := subtopics[2]
			resp := &macondo.BotResponse{}
			err := proto.Unmarshal(msg.Data, resp)
			if err != nil {
				log.Err(err).Msg("unmarshal-bot-response-error")
				break
			}
			b.goHandleBotMove(ctx, resp, gid, msg.Reply)

		// NATS message from internal sources (within liwords)
		case msg := <-b.subchans["user.>"]:
			log := log.With().Interface("msg-user.>", msg.Subject).Logger()
			log.Debug().Msg("got-user-event")
			err := b.gameEventAPIServer.processEvent(msg.Subject, msg.Data)
			if err != nil {
				log.Err(err).Msg("user-event-process-error")
			}

		// NATS message from internal sources (within liwords)
		case msg := <-b.subchans["game.>"]:
			log := log.With().Interface("msg-game.>", msg.Subject).Logger()
			log.Debug().Msg("got-game-event")
			err := b.gameEventAPIServer.processEvent(msg.Subject, msg.Data)
			if err != nil {
				log.Err(err).Msg("game-event-process-error")
			}

		// Regular Go chan message from within liwords:
		case msg := <-b.gameEventChan:
			if msg.Type == pb.MessageType_ACTIVE_GAME_ENTRY {
				// This message usually has no audience.
				if evt, ok := msg.Event.(*pb.ActiveGameEntry); ok {
					log.Debug().Interface("event", evt).Msg("active-game-entry")
					ret, err := b.stores.PresenceStore.UpdateActiveGame(ctx, evt)
					if err != nil {
						log.Err(err).Msg("update-active-game-error")
						// but continue anyway
					} else {
						// this should be async because b.broadcastChannelChanges sends to b.genericEventChan and while we are here <-b.genericEventChan is not being drained
						go func() {
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
						}()
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
					log.Debug().Str("user", user).Str("suffix", suffix).Msg("pub-to-user")
					err := b.pubToUser(user, msg, suffix)
					if err != nil {
						log.Err(err).Str("topic", topic).Msg("pub-user-error")
					}

				} else {
					err := b.natsconn.Publish(topic, data)
					if err != nil {
						log.Err(err).Str("topic", topic).Msg("pub-error")
					}
				}
			}

		// Regular Go chan message from within liwords:
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

		// Regular Go chan message from within liwords:
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
			go func() {
				err := b.adjudicateGames(ctx, false) // false = real-time games only
				if err != nil {
					log.Err(err).Msg("adjudicate-error")
				}
			}()

		case <-correspondenceAdjudicator.C:
			go func() {
				err := b.adjudicateGames(ctx, true) // true = correspondence games only
				if err != nil {
					log.Err(err).Msg("adjudicate-correspondence-error")
				}
			}()

		case <-seekExpirer.C:
			go func() {
				err := b.stores.SoughtGameStore.ExpireOld(ctx)
				if err != nil {
					log.Err(err).Msg("expiration-error")
				}
			}()

		case <-channelMonitor.C:
			go func() {
				log.Info().
					Int("generic-events", len(b.genericEventChan)).
					Int("game-events", len(b.gameEventChan)).
					Int("tourney-events", len(b.tournamentEventChan)).
					Int("ipc.pb-events", len(b.subchans["ipc.pb.>"])).
					Int("ipc.request-events", len(b.subchans["ipc.request.>"])).
					// These next two events go out to the socket, but we are also
					// receiving them here for our someday bot API.
					Int("outgoing-socket-user-events", len(b.subchans["user.>"])).
					Int("outgoing-socket-game-events", len(b.subchans["game.>"])).
					Int("bot-publish-events", len(b.subchans["bot.publish_event.>"])).
					Msg("channel-buffer-lengths-monitor")
			}()

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
		log.Err(err).Interface("msg", msg).Msg("received-register-realm-request")
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
			game, err := b.stores.GameStore.Get(ctx, gameID)
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
			t, err := b.stores.TournamentStore.GetBySlug(ctx, path)
			if err != nil {
				return err
			}
			currentTournamentID = t.UUID
			tournamentRealm := "tournament-" + currentTournamentID
			resp.Realms = append(resp.Realms, tournamentRealm, "chat-"+tournamentRealm)
		} else if strings.HasPrefix(path, "/leagues/") {
			slug := strings.TrimPrefix(path, "/leagues/")
			league, err := b.stores.LeagueStore.GetLeagueBySlug(ctx, slug)
			if err != nil {
				return err
			}
			currentLeagueID := strings.ReplaceAll(league.Uuid.String(), "-", "")
			leagueRealm := "league-" + currentLeagueID
			resp.Realms = append(resp.Realms, leagueRealm, "chat-"+leagueRealm)
		} else if strings.HasPrefix(path, "/puzzle/") {
			// We are appending a chat realm for two reasons:
			// 1. In the future we could probably have a puzzle lobby chat
			// 2. Chat realms are the only ones that are compatible with
			// presence at this moment.
			resp.Realms = append(resp.Realms, "chat-puzzlelobby")
		} else if strings.HasPrefix(path, "/editor/") {
			gameID := strings.TrimPrefix(path, "/editor/")
			// Use the `channel-` generic realm for now.
			realm := "channel-" + omgwords.AnnotatedChannelName(gameID)
			// Appending chat-gametv-gameID for presence
			resp.Realms = append(resp.Realms,
				realm, "chat-gametv-editor-"+gameID) /// this is so ugly, clean up all this subscription code.

		} else if strings.HasPrefix(path, "/anno/") {
			// annotated games are always in TV mode for viewers
			gameID := strings.TrimPrefix(path, "/anno/")
			realm := "channel-" + omgwords.AnnotatedChannelName(gameID)
			resp.Realms = append(resp.Realms, realm, "chat-gametv-anno-"+gameID)
		} else {
			log.Debug().Str("path", path).Msg("realm-req-not-handled")
		}

		activeTourneys, err := b.stores.TournamentStore.ActiveTournamentsFor(ctx, userID)
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
		log.Err(err).Interface("resp", resp).Msg("responding-to-realm-request")

		if err != nil {
			return err
		}
		b.natsconn.Publish(replyTopic, retdata)
		log.Debug().Str("topic", topic).Str("replyTopic", replyTopic).Interface("realms", resp.Realms).
			Msg("published response")

	case "getFollowers":
		msg := &pb.GetFollowersRequest{}
		err := proto.Unmarshal(data, msg)
		if err != nil {
			return err
		}

		log.Debug().Str("userID", msg.UserId).Msg("received-get-followers-request")

		// Get user by UUID
		user, err := b.stores.UserStore.GetByUUID(ctx, msg.UserId)
		if err != nil {
			return err
		}

		// Get all followers
		followerUsers, err := b.stores.UserStore.GetFollowedBy(ctx, user.ID)
		if err != nil {
			return err
		}

		// Convert to user IDs
		followerIDs := make([]string, len(followerUsers))
		for i, fu := range followerUsers {
			followerIDs[i] = fu.UUID
		}

		resp := &pb.GetFollowersResponse{
			FollowerUserIds: followerIDs,
		}

		retdata, err := proto.Marshal(resp)
		if err != nil {
			return err
		}

		b.natsconn.Publish(replyTopic, retdata)
		log.Debug().Str("topic", topic).Str("userID", msg.UserId).Int("follower_count", len(followerIDs)).
			Msg("published get-followers response")

	case "getFollows":
		msg := &pb.GetFollowsRequest{}
		err := proto.Unmarshal(data, msg)
		if err != nil {
			return err
		}

		log.Debug().Str("userID", msg.UserId).Msg("received-get-follows-request")

		// Get user by UUID
		user, err := b.stores.UserStore.GetByUUID(ctx, msg.UserId)
		if err != nil {
			return err
		}

		// Get all users that this user follows
		followUsers, err := b.stores.UserStore.GetFollows(ctx, user.ID)
		if err != nil {
			return err
		}

		// Convert to user IDs
		followIDs := make([]string, len(followUsers))
		for i, fu := range followUsers {
			followIDs[i] = fu.UUID
		}

		resp := &pb.GetFollowsResponse{
			FollowUserIds: followIDs,
		}

		retdata, err := proto.Marshal(resp)
		if err != nil {
			return err
		}

		b.natsconn.Publish(replyTopic, retdata)
		log.Debug().Str("topic", topic).Str("userID", msg.UserId).Int("follows_count", len(followIDs)).
			Msg("published get-follows response")

	default:
		return fmt.Errorf("unhandled-req-topic: %v", topic)
	}
	return nil
}

// handleNatsPublish runs in a separate goroutine
func (b *Bus) handleNatsPublish(ctx context.Context, subtopics []string, data []byte) error {
	log := log.Ctx(ctx)

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
	// XXX: Otherwise, ignore error for now.

	switch msgType {
	case pb.MessageType_SEEK_REQUEST.String():
		log.Debug().Str("user", userID).Msg("seek-request")
		return b.seekRequest(ctx, auth, userID, wsConnID, data)
	case pb.MessageType_CHAT_MESSAGE.String():
		// The user is subtopics[2]
		evt := &pb.ChatMessage{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		log.Debug().Str("user", userID).Str("msg", evt.Message).Str("channel", evt.Channel).Msg("chat")
		return b.chat(ctx, userID, evt)

	case pb.MessageType_DECLINE_SEEK_REQUEST.String():
		evt := &pb.DeclineSeekRequest{}

		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		log.Debug().Str("user", userID).Str("reqid", evt.RequestId).Msg("decline-rematch")
		return b.seekDeclined(ctx, evt, userID)
	case pb.MessageType_GAME_META_EVENT.String():
		evt := &pb.GameMetaEvent{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		log.Debug().Str("user", userID).Interface("evt", evt).Msg("game-meta-event")
		return b.gameMetaEvent(ctx, evt, userID)

	case pb.MessageType_SOUGHT_GAME_PROCESS_EVENT.String():
		evt := &pb.SoughtGameProcessEvent{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}

		return b.gameAccepted(ctx, evt, userID, wsConnID)

	case pb.MessageType_CLIENT_GAMEPLAY_EVENT.String():
		evt := &pb.ClientGameplayEvent{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		_, err = gameplay.HandleEvent(ctx, b.stores, userID, evt)
		return err

	case pb.MessageType_TIMED_OUT.String():
		evt := &pb.TimedOut{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return gameplay.TimedOut(ctx, b.stores, evt.UserId, evt.GameId)

	case pb.MessageType_READY_FOR_GAME.String():
		evt := &pb.ReadyForGame{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return b.readyForGame(ctx, evt, userID)
	case pb.MessageType_READY_FOR_TOURNAMENT_GAME.String():
		evt := &pb.ReadyForTournamentGame{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return b.readyForTournamentGame(ctx, evt, userID, wsConnID)

	// The messages after this are internal messages sent only from services/socketsrv
	// to liwords, so there are no MessageType enums for these. It's ok:
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
		evt := &pb.Pong{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return b.pongReceived(ctx, userID, wsConnID, evt.Ips)

	default:
		return fmt.Errorf("unhandled-publish-topic: %v", subtopics)
	}
}

func (b *Bus) TournamentEventChannel() chan *entity.EventWrapper {
	return b.tournamentEventChan
}

func (b *Bus) GameEventChannel() chan *entity.EventWrapper {
	return b.gameEventChan
}

// AdjudicateGames is an exported wrapper for testing the adjudication logic.
func (b *Bus) AdjudicateGames(ctx context.Context, correspondenceOnly bool) error {
	return b.adjudicateGames(ctx, correspondenceOnly)
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
	sanitized, err := sanitize(b.stores.UserStore, b.stores.GameStore, evt, userID)
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
	sanitized, err := sanitize(b.stores.UserStore, b.stores.GameStore, evt, userID)
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

	if userIsAnon(userID) {
		return nil
	}

	log.Debug().Str("userID", userID).Str("username", username).
		Interface("channels", newChannels).
		Msg("using-new-presence-system")
	return b.broadcastPresenceChanged(userID, username, newChannels)
}

// NEW SYSTEM: Single efficient presence notification
func (b *Bus) broadcastPresenceChanged(userID, username string, channels []string) error {
	presenceEntry := &pb.PresenceEntry{
		Username: username,
		UserId:   userID,
		Channel:  channels,
	}

	data, err := proto.Marshal(presenceEntry)
	if err != nil {
		return err
	}

	// Single publish regardless of follower count - this is the key efficiency gain
	topic := "presence.changed." + userID
	err = b.natsconn.Publish(topic, data)
	if err == nil {
		log.Debug().Str("userID", userID).Str("topic", topic).
			Interface("channels", channels).
			Msg("published-new-presence-notification")
	}
	return err
}

func userIsAnon(userID string) bool {
	return strings.HasPrefix(userID, "anon-")
}

func (b *Bus) initRealmInfo(ctx context.Context, evt *pb.InitRealmInfo, connID string) error {
	// For consistency sake, use the `dotted` channels for presence
	// i.e. game.<gameID>, gametv.<gameID>
	// The reasoning is that realms should only be cared about by the socket
	// server. The channels are NATS pubsub channels and we use these for chat
	// too.
	var anon bool
	var username string
	var err error
	if userIsAnon(evt.UserId) {
		anon = true
		username = evt.UserId
	} else {
		username, err = b.stores.UserStore.Username(ctx, evt.UserId)
		if err != nil {
			return err
		}
	}

	// The channels with presence should be:
	// chat.lobby
	// chat.tournament.foo
	// chat.game.bar
	// chat.gametv.baz
	// chat.puzzlelobby
	// global.presence (when it comes, we edit this later)
	// maybe chat.puzzle.abcdef in the future.

	for _, realm := range evt.Realms {

		presenceChan := strings.ReplaceAll(realm, "-", ".")
		if !strings.HasPrefix(presenceChan, "chat.") {
			// presenceChan / presenceStore is only used for chat purposes for now.
			presenceChan = ""
		}

		if presenceChan != "" {
			log.Debug().Str("presence-chan", presenceChan).Str("username", username).Msg("SetPresence")
			oldChannels, newChannels, err := b.stores.PresenceStore.SetPresence(ctx, evt.UserId, username, anon, presenceChan, connID)
			if err != nil {
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
			gameID := components[len(components)-1]
			err := b.sendGameRefresher(ctx, gameID, connID, evt.UserId)
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(realm, "tournament-") {
			err := b.sendTournamentContext(ctx, realm, evt.UserId, connID)
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(realm, "league-") {
			err := b.sendLeagueContext(ctx, realm, evt.UserId, connID)
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
	users, err := b.stores.PresenceStore.GetInChannel(ctx, presenceChan)
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
	var anon bool
	var username string
	var err error
	if userIsAnon(userID) {
		anon = true
		username = userID
	} else {
		username, err = b.stores.UserStore.Username(ctx, userID)
		if err != nil {
			return err
		}
	}
	oldChannels, newChannels, channels, err := b.stores.PresenceStore.ClearPresence(ctx, userID, username, anon, connID)
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
	// Clean up any sought games we may have missed. This only happens if
	// there was an error in deleting sought games in the leaveTab flow.
	err := b.deleteSoughtForUser(ctx, userID)
	if err != nil {
		return err
	}
	return nil
}

func (b *Bus) pongReceived(ctx context.Context, userID, connID, ips string) error {
	var anon bool
	var username string
	var err error
	if userIsAnon(userID) {
		anon = true
		username = userID
	} else {
		username, err = b.stores.UserStore.Username(ctx, userID)
		if err != nil {
			return err
		}
	}
	log.Debug().Str("username", username).Str("connID", connID).Str("ips", ips).Msg("pong-received")

	oldChannels, newChannels, err := b.stores.PresenceStore.RenewPresence(ctx, userID, username, anon, connID)
	if err != nil {
		return err
	}
	if err = b.broadcastChannelChanges(ctx, oldChannels, newChannels, userID, username); err != nil {
		return err
	}
	return nil
}

func (b *Bus) activeGames(ctx context.Context, tourneyID string) (*entity.EventWrapper, error) {
	games, err := b.stores.GameStore.ListActive(ctx, tourneyID, false)

	if err != nil {
		return nil, err
	}
	log.Debug().Int("num-active-games", len(games.GameInfo)).Msg("active-games")

	evt := entity.WrapEvent(games, pb.MessageType_ONGOING_GAMES)
	return evt, nil
}

// correspondenceGamesForUser returns all correspondence games for a specific user
func (b *Bus) correspondenceGamesForUser(ctx context.Context, userID string) (*entity.EventWrapper, error) {
	games, err := b.stores.GameStore.ListActiveCorrespondenceForUser(ctx, userID)

	if err != nil {
		return nil, err
	}

	log.Debug().Int("num-correspondence-games", len(games.GameInfo)).Str("userID", userID).Msg("correspondence-games-for-user")

	evt := entity.WrapEvent(games, pb.MessageType_OUR_CORRESPONDENCE_GAMES)
	return evt, nil
}

// correspondenceSeeksForUser returns all correspondence match requests and open seeks for a specific user
func (b *Bus) correspondenceSeeksForUser(ctx context.Context, userID string) (*entity.EventWrapper, error) {
	seeks, err := b.stores.SoughtGameStore.ListCorrespondenceSeeksForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	var receiver *entity.User
	if !userIsAnon(userID) {
		receiver, err = b.stores.UserStore.GetByUUID(ctx, userID)
		if err != nil {
			return nil, err
		}
	}

	seekRequests := make([]*pb.SeekRequest, 0, len(seeks))
	for _, seek := range seeks {
		if seek.SeekRequest == nil {
			continue
		}

		// Use shared filtering logic (blocks, established rating, followed players)
		if b.shouldIncludeSeek(ctx, seek, receiver) {
			seekRequests = append(seekRequests, seek.SeekRequest)
		}
	}

	log.Debug().Int("num-correspondence-seeks", len(seekRequests)).Str("userID", userID).Msg("correspondence-seeks-for-user")

	evt := entity.WrapEvent(&pb.SeekRequests{Requests: seekRequests}, pb.MessageType_OUR_CORRESPONDENCE_SEEKS)
	return evt, nil
}

// leagueCorrespondenceGamesForUser returns correspondence games for a specific league and user
func (b *Bus) leagueCorrespondenceGamesForUser(ctx context.Context, userID, leagueID string) (*entity.EventWrapper, error) {
	games, err := b.stores.GameStore.ListActiveCorrespondenceForUser(ctx, userID)

	if err != nil {
		return nil, err
	}

	log.Debug().Int("total-correspondence-games", len(games.GameInfo)).Str("userID", userID).Str("leagueID", leagueID).Msg("filtering-league-games")

	// Filter games to only include those from this league
	filteredGames := make([]*pb.GameInfoResponse, 0)
	for _, game := range games.GameInfo {
		log.Debug().Str("gameID", game.GameId).Str("gameLeagueId", game.LeagueId).Str("targetLeagueId", leagueID).Bool("matches", game.LeagueId == leagueID).Msg("checking-game")
		if game.LeagueId == leagueID {
			filteredGames = append(filteredGames, game)
		}
	}

	log.Debug().Int("num-league-correspondence-games", len(filteredGames)).Str("userID", userID).Str("leagueID", leagueID).Msg("league-correspondence-games-for-user")

	evt := entity.WrapEvent(&pb.GameInfoResponses{GameInfo: filteredGames}, pb.MessageType_OUR_LEAGUE_CORRESPONDENCE_GAMES)
	return evt, nil
}

// Return 0 if uid1 blocks uid2, 1 if uid2 blocks uid1, and -1 if neither blocks
// the other. Note, if they both block each other it will return 0.
func (b *Bus) blockExists(ctx context.Context, u1, u2 *entity.User) (int, error) {
	blockedUsers, err := b.stores.UserStore.GetBlocks(ctx, u1.ID)
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
	blockedUsers, err = b.stores.UserStore.GetBlockedBy(ctx, u1.ID)
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
	if !userIsAnon(userID) {
		u, err := b.stores.UserStore.GetByUUID(ctx, userID)
		if err != nil {
			return err
		}
		// send ratings first.
		ratingProto, err := u.GetProtoRatings()
		if err != nil {
			return err
		}
		profileUpdate := &pb.ProfileUpdate{
			UserId:  userID,
			Ratings: ratingProto,
		}
		evt := entity.WrapEvent(profileUpdate, pb.MessageType_PROFILE_UPDATE_EVENT)
		err = b.pubToConnectionID(connID, userID, evt)
		if err != nil {
			return err
		}
	}
	// open seeks
	seeks, err := b.openSeeks(ctx, userID, "")
	if err != nil {
		return err
	}
	if seeks != nil {
		err = b.pubToConnectionID(connID, userID, seeks)
		if err != nil {
			return err
		}
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

	// correspondence games (only for logged-in users)
	if !userIsAnon(userID) {
		correspondenceGames, err := b.correspondenceGamesForUser(ctx, userID)
		if err != nil {
			return err
		}
		err = b.pubToConnectionID(connID, userID, correspondenceGames)
		if err != nil {
			return err
		}

		// correspondence seeks (pending match requests)
		correspondenceSeeks, err := b.correspondenceSeeksForUser(ctx, userID)
		if err != nil {
			return err
		}
		err = b.pubToConnectionID(connID, userID, correspondenceSeeks)
		if err != nil {
			return err
		}
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
	matches, err := b.openSeeks(ctx, userID, tourneyID)
	if err != nil {
		return err
	}
	if matches != nil {
		err = b.pubToConnectionID(connID, userID, matches)
		if err != nil {
			return err
		}
	}

	return err
}

func (b *Bus) sendLeagueContext(ctx context.Context, realm, userID, connID string) error {
	// Realm format is "league-{uuid}", extract the UUID part
	leagueID := strings.TrimPrefix(realm, "league-")
	// Send league correspondence games for this user
	leagueGames, err := b.leagueCorrespondenceGamesForUser(ctx, userID, leagueID)
	if err != nil {
		return err
	}
	err = b.pubToConnectionID(connID, userID, leagueGames)
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
