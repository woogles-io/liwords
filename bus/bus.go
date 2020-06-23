// Package bus is the message bus. This package listens on various NATS channels
// for requests and publishes back responses to the same, or other channels.
// Responsible for talking to the liwords-socket server.
package bus

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	nats "github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto"
)

// Bus is the struct; it should contain all the stores to verify messages, etc.
type Bus struct {
	natsconn        *nats.Conn
	userStore       user.Store
	gameStore       gameplay.GameStore
	soughtGameStore gameplay.SoughtGameStore

	subscriptions []*nats.Subscription
	subchans      map[string]chan *nats.Msg
}

func NewBus(natsURL string, userStore user.Store, gameStore gameplay.GameStore, soughtGameStore gameplay.SoughtGameStore) (*Bus, error) {
	natsconn, err := nats.Connect(natsURL)

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
			err := b.handleNatsPublish(ctx, subtopics[2], msg.Data)
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
		var topic string
		if strings.HasPrefix(path, "/game/") {
			gameID := strings.TrimPrefix(path, "/game/")
			game, err := b.gameStore.Get(ctx, gameID)
			if err != nil {
				return err
			}
			var foundPlayer bool
			for i := 0; i < 2; i++ {
				if game.History().Players[0].UserId == userID {
					foundPlayer = true
				}
			}
			if !foundPlayer {
				topic = "gametv." + gameID
			}
		} else {
			return errors.New("realm request not handled")
		}
		resp := &pb.RegisterRealmResponse{}
		resp.Realm = topic
		retdata, err := proto.Marshal(resp)
		if err != nil {
			return err
		}
		// Note: if the player is involved IN a game, there is no realm
		// that they should belong to. They will get game publishes directly
		// to their user topic! see pubsub.go in liwords-socket. That's why
		// the topic is blank for this case.
		b.natsconn.Publish(replyTopic, retdata)
		log.Debug().Str("topic", topic).Str("replyTopic", replyTopic).
			Msg("published response")
	default:
		return fmt.Errorf("unhandled request topic: %v", topic)
	}
	return nil
}

func (b *Bus) handleNatsPublish(ctx context.Context, topic string, data []byte) error {
	switch topic {
	case "seekRequest":
		req := &pb.SeekRequest{}
		err := proto.Unmarshal(data, req)
		if err != nil {
			return err
		}
		sg, err := gameplay.NewSoughtGame(ctx, b.soughtGameStore, req)
		if err != nil {
			return err
		}
		// XXX: Later look up user rating so we can attach to this request.
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

		return b.gameAccepted(ctx, evt)

	case "gameplayEvent":

	case "initRealmInfo":

	default:
		return fmt.Errorf("unhandled publish topic: %v", topic)
	}
	return nil
}

func (b *Bus) gameAccepted(ctx context.Context, evt *pb.GameAcceptedEvent) error {
	sg, err := b.soughtGameStore.Get(ctx, evt.RequestId)
	if err != nil {
		return err
	}
	requester := sg.SeekRequest.User.Username
	if requester == sender {
		log.Info().Str("sender", sender).Msg("canceling seek")
		err := gameplay.CancelSoughtGame(ctx, h.soughtGameStore, evt.RequestId)
		if err != nil {
			return err
		}
		// broadcast a seek deletion.
		err = h.DeleteSeek(evt.RequestId)
		if err != nil {
			return err
		}
		return err
	}
}
