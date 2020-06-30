// Package bus is the message bus. This package listens on various NATS channels
// for requests and publishes back responses to the same, or other channels.
// Responsible for talking to the liwords-socket server.
package bus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	nats "github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

const (
	GameStartDelay = 3 * time.Second
)

// Bus is the struct; it should contain all the stores to verify messages, etc.
type Bus struct {
	natsconn        *nats.Conn
	config          *config.Config
	userStore       user.Store
	gameStore       gameplay.GameStore
	soughtGameStore gameplay.SoughtGameStore

	subscriptions []*nats.Subscription
	subchans      map[string]chan *nats.Msg

	gameEventChan chan *entity.EventWrapper
}

func NewBus(cfg *config.Config, userStore user.Store, gameStore gameplay.GameStore,
	soughtGameStore gameplay.SoughtGameStore) (*Bus, error) {

	natsconn, err := nats.Connect(cfg.NatsURL)

	if err != nil {
		return nil, err
	}
	bus := &Bus{
		natsconn:        natsconn,
		userStore:       userStore,
		gameStore:       gameStore,
		soughtGameStore: soughtGameStore,
		subscriptions:   []*nats.Subscription{},
		subchans:        map[string]chan *nats.Msg{},
		config:          cfg,
		gameEventChan:   make(chan *entity.EventWrapper, 64),
	}

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
	for {
		select {
		case msg := <-b.subchans["ipc.pb.>"]:
			// Regular messages.
			log.Debug().Str("topic", msg.Subject).Msg("got ipc.pb message")
			subtopics := strings.Split(msg.Subject, ".")
			err := b.handleNatsPublish(ctx, subtopics[2:], msg.Data)
			if err != nil {
				log.Err(err).Msg("process-message-publish-error")
			}

		case msg := <-b.subchans["ipc.request.>"]:
			log.Debug().Str("topic", msg.Subject).Msg("got ipc.request")
			// Requests. We must respond on a specific topic.
			subtopics := strings.Split(msg.Subject, ".")
			err := b.handleNatsRequest(ctx, subtopics[2], msg.Reply, msg.Data)
			if err != nil {
				log.Err(err).Msg("process-message-request-error")
				// just send a blank response so there isn't a timeout on
				// the other side.
				rrResp := &pb.RegisterRealmResponse{
					Realm: "",
				}
				data, err := proto.Marshal(rrResp)
				if err != nil {
					log.Err(err).Msg("marshalling-error")
					break
				}
				b.natsconn.Publish(msg.Reply, data)
			}

		case msg := <-b.gameEventChan:
			// A game event. Publish directly to the right realm.
			log.Debug().Interface("msg", msg).Msg("game event chan")
			topics := msg.Audience()
			data, err := msg.Serialize()
			if err != nil {
				log.Err(err).Msg("serialize-error")
				break
			}
			for _, topic := range topics {
				if strings.HasPrefix(topic, "user.") {
					b.pubToUser(strings.TrimPrefix(topic, "user."), msg)
				} else {
					b.natsconn.Publish(topic, data)
				}
			}
		}
	}
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
		// given they went to the given path. Don't handle the lobby, the socket
		// already handles that.
		path := msg.Realm
		userID := msg.UserId
		var realm string
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
		} else {
			return errors.New("realm request not handled")
		}
		resp := &pb.RegisterRealmResponse{}
		resp.Realm = realm
		retdata, err := proto.Marshal(resp)
		if err != nil {
			return err
		}
		b.natsconn.Publish(replyTopic, retdata)
		log.Debug().Str("topic", topic).Str("replyTopic", replyTopic).
			Msg("published response")
	default:
		return fmt.Errorf("unhandled request topic: %v", topic)
	}
	return nil
}

func (b *Bus) handleNatsPublish(ctx context.Context, subtopics []string, data []byte) error {
	log.Debug().Interface("subtopics", subtopics).Msg("handling nats publish")
	switch subtopics[0] {
	case "seekRequest":
		req := &pb.SeekRequest{}
		err := proto.Unmarshal(data, req)
		if err != nil {
			return err
		}
		// Note that the seek request should not come with a requesting user;
		// instead this is in the topic/subject. It is HERE in the API server that
		// we set the requesting user's display name, rating, etc.
		req.User = &pb.RequestingUser{}
		req.User.IsAnonymous = subtopics[1] == "anon"
		req.User.UserId = subtopics[2]

		if req.User.IsAnonymous {
			req.User.DisplayName = entity.DeterministicUsername(req.User.UserId)
			req.User.RelevantRating = "Unrated"
		} else {
			// Look up user.
			// XXX: Later look up user rating so we can attach to this request.
			timefmt, variant, err := gameplay.VariantFromGamereq(req)
			ratingKey := entity.ToVariantKey(req.GameRequest.Lexicon, variant, timefmt)

			u, err := b.userStore.GetByUUID(ctx, req.User.UserId)
			if err != nil {
				return err
			}
			req.User.RelevantRating = gameplay.GetRelevantRating(ratingKey, u)
			req.User.DisplayName = u.Username
		}

		sg, err := gameplay.NewSoughtGame(ctx, b.soughtGameStore, req)
		if err != nil {
			return err
		}
		evt := entity.WrapEvent(sg.SeekRequest, pb.MessageType_SEEK_REQUEST, "")
		data, err := evt.Serialize()
		if err != nil {
			return err
		}
		log.Debug().Interface("evt", evt).Msg("publishing seek request to lobby topic")
		b.natsconn.Publish("lobby.seekRequest", data)
	case "gameAccepted":
		evt := &pb.GameAcceptedEvent{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		// subtopics[2] is the user ID of the requester.
		return b.gameAccepted(ctx, evt, subtopics[2])

	case "gameplayEvent":
		// evt :=
		evt := &pb.ClientGameplayEvent{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		// subtopics[2] is the user ID of the requester.
		return gameplay.PlayMove(ctx, b.gameStore, subtopics[2], evt)

	case "timedOut":
		evt := &pb.TimedOut{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return gameplay.TimedOut(ctx, b.gameStore, evt.UserId, evt.GameId)

	case "initRealmInfo":
		evt := &pb.InitRealmInfo{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return b.initRealmInfo(ctx, evt)
	default:
		return fmt.Errorf("unhandled publish topic: %v", subtopics)
	}
	return nil
}

func (b *Bus) gameAccepted(ctx context.Context, evt *pb.GameAcceptedEvent, userID string) error {

	sg, err := b.soughtGameStore.Get(ctx, evt.RequestId)
	if err != nil {
		return err
	}
	requester := sg.SeekRequest.User.UserId
	if requester == userID {
		log.Info().Str("sender", requester).Msg("canceling seek")
		err := gameplay.CancelSoughtGame(ctx, b.soughtGameStore, evt.RequestId)
		if err != nil {
			return err
		}
		// broadcast a seek deletion.
		return b.broadcastSeekDeletion(evt.RequestId)

	}
	// Otherwise create a game
	p1, err := getPlayerInfo(ctx, b.userStore, userID) // the acceptor
	if err != nil {
		return err
	}
	p2, err := getPlayerInfo(ctx, b.userStore, requester) // the original requester
	if err != nil {
		return err
	}
	players := []*macondopb.PlayerInfo{p1, p2}
	log.Debug().Interface("seekreq", sg.SeekRequest).Msg("seek-request-accepted")

	g, err := gameplay.InstantiateNewGame(ctx, b.gameStore, b.config,
		players, sg.SeekRequest.GameRequest)
	if err != nil {
		return err
	}
	// Broadcast a seek delete event, and send both parties a game redirect.
	b.soughtGameStore.Delete(ctx, evt.RequestId)
	err = b.broadcastSeekDeletion(evt.RequestId)
	if err != nil {
		return err
	}
	// This event will result in a redirect.
	ngevt := entity.WrapEvent(&pb.NewGameEvent{
		GameId: g.GameID(),
	}, pb.MessageType_NEW_GAME_EVENT, "")
	b.pubToUser(p1.UserId, ngevt)
	b.pubToUser(p2.UserId, ngevt)

	log.Info().Str("newgameid", g.History().Uid).
		Str("sender", userID).
		Str("requester", requester).
		Interface("starting-in", GameStartDelay).
		Str("onturn", g.NickOnTurn()).Msg("game-accepted")

	// Now, reset the timer and register the event change hook.
	time.AfterFunc(GameStartDelay, func() {
		err = gameplay.StartGame(ctx, b.gameStore, b.gameEventChan, g.GameID())
		if err != nil {
			log.Err(err).Msg("starting-game")
		}
	})

	return nil
}

func (b *Bus) broadcastSeekDeletion(seekID string) error {
	toSend := entity.WrapEvent(&pb.GameAcceptedEvent{RequestId: seekID},
		pb.MessageType_GAME_ACCEPTED_EVENT, "")
	data, err := toSend.Serialize()
	if err != nil {
		return err
	}
	return b.natsconn.Publish("lobby.gameAccepted", data)
}

func (b *Bus) pubToUser(userID string, evt *entity.EventWrapper) error {
	t := time.Now()
	sanitized, err := sanitize(evt, userID)
	bts, err := sanitized.Serialize()
	if err != nil {
		return err
	}
	log.Debug().Interface("time-taken", time.Now().Sub(t)).Msg("pubToUser-serialization")
	return b.natsconn.Publish("user."+userID, bts)
}

func (b *Bus) initRealmInfo(ctx context.Context, evt *pb.InitRealmInfo) error {
	if evt.Realm == "lobby" {
		seeks, err := b.openSeeks(ctx)
		if err != nil {
			return err
		}
		return b.pubToUser(evt.UserId, seeks)
	} else if strings.HasPrefix(evt.Realm, "game-") || strings.HasPrefix(evt.Realm, "gametv-") {
		// Get a sanitized history
		gameID := strings.Split(evt.Realm, "-")[1]
		refresher, err := b.gameRefresher(ctx, gameID)
		if err != nil {
			return err
		}
		return b.pubToUser(evt.UserId, refresher)
	}
	log.Debug().Interface("evt", evt).Msg("no init realm info")
	return nil
}

func (b *Bus) openSeeks(ctx context.Context) (*entity.EventWrapper, error) {
	sgs, err := b.soughtGameStore.ListOpen(ctx)
	if err != nil {
		return nil, err
	}

	pbobj := &pb.SeekRequests{Requests: []*pb.SeekRequest{}}
	for _, sg := range sgs {
		pbobj.Requests = append(pbobj.Requests, sg.SeekRequest)
	}
	evt := entity.WrapEvent(pbobj, pb.MessageType_SEEK_REQUESTS, "")
	return evt, nil
}

func (b *Bus) gameRefresher(ctx context.Context, gameID string) (*entity.EventWrapper, error) {
	// Get a game refresher event. If sanitize is true, opponent racks will be
	// hidden from the given userID.
	entGame, err := b.gameStore.Get(ctx, string(gameID))
	if err != nil {
		return nil, err
	}
	if !entGame.Started() {
		return nil, errors.New("game-starting-soon")
	}
	evt := entity.WrapEvent(entGame.HistoryRefresherEvent(),
		pb.MessageType_GAME_HISTORY_REFRESHER, entGame.GameID())
	return evt, nil
}

func getPlayerInfo(ctx context.Context, u user.Store, id string) (*macondopb.PlayerInfo, error) {

	user, err := u.GetByUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Put in real name, etc here later.
	return &macondopb.PlayerInfo{
		Nickname: user.Username, UserId: user.UUID,
	}, nil
}
