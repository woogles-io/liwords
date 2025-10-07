package sockets

import (
	"strings"

	nats "github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/services/socketsrv/pkg/config"
)

// PubSub encapsulates the various subscriptions to the different channels.
// The `liwords` package should have a very similar structure.
type PubSub struct {
	natsconn      *nats.Conn
	topics        []string
	subscriptions []*nats.Subscription
	subchans      map[string]chan *nats.Msg
	config        *config.Config
}

func newPubSub(natsURL string, cfg *config.Config) (*PubSub, error) {
	natsconn, err := nats.Connect(natsURL)

	if err != nil {
		return nil, err
	}

	topics := []string{
		// lobby messages:
		"lobby.>",
		// for specific connections
		"connid.>",
		// user messages
		"user.>",
		// usertv messages; for when someone is watching a user's games
		"usertv.>",
		// gametv messages: for observer mode in a single game.
		"gametv.>",
		// private game messages: only for the players of a game.
		"game.>",
		// tourneys
		"tournament.>",
		// chats
		"chat.>",
		// generic channels
		"channel.>",
		// NEW: efficient presence change notifications
		"presence.changed.>",
		// Efficient seek notifications to followed users
		"seek.followed.>",
	}
	pubSub := &PubSub{
		natsconn:      natsconn,
		topics:        topics,
		subscriptions: []*nats.Subscription{},
		subchans:      map[string]chan *nats.Msg{},
		config:        cfg,
	}
	// Subscribe to the above topics.
	for _, topic := range topics {
		ch := make(chan *nats.Msg, 512)
		sub, err := natsconn.ChanSubscribe(topic, ch)
		if err != nil {
			return nil, err
		}
		pubSub.subscriptions = append(pubSub.subscriptions, sub)
		pubSub.subchans[topic] = ch

	}
	return pubSub, nil
}

// PubsubProcess processes pubsub messages.
func (h *Hub) PubsubProcess() {
	for {
		select {
		case msg := <-h.pubsub.subchans["lobby.>"]:
			// Handle lobby message. If something is published to the lobby,
			// let's just send it along to the correct sockets, we should not
			// need to parse it.
			log.Debug().Str("topic", msg.Subject).Msg("got lobby message, forwarding along")
			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) < 2 {
				log.Error().Msgf("subtopics weird %v", msg.Subject)
				continue
			}
			h.sendToRealm(LobbyRealm, msg.Data)

		case msg := <-h.pubsub.subchans["tournament.>"]:
			log.Debug().Str("topic", msg.Subject).Int("type", int(msg.Data[2])).Msg("got tournament message, forwarding along")

			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) < 2 {
				log.Error().Msgf("tournament subtopics weird %v", msg.Subject)
				continue
			}
			tournamentID := subtopics[1]
			h.sendToRealm(Realm("tournament-"+tournamentID), msg.Data)

		case msg := <-h.pubsub.subchans["user.>"]:
			// If we get a user message, we should send it along to the given
			// user.
			log.Debug().Str("topic", msg.Subject).Int("type", int(msg.Data[2])).Msg("got user message, forwarding along")
			subtopics := strings.SplitN(msg.Subject, ".", 3)
			if len(subtopics) < 2 {
				log.Error().Msgf("user subtopics weird %v", msg.Subject)
				continue
			}
			userID := subtopics[1]
			if len(subtopics) < 3 {
				h.sendToUser(userID, msg.Data)
			} else {
				h.sendToUserChannel(userID, msg.Data, subtopics[2])
			}

		case msg := <-h.pubsub.subchans["connid.>"]:
			// Forward to the given connection ID only.
			log.Debug().Str("topic", msg.Subject).Int("type", int(msg.Data[2])).Msg("got connID message, forwarding along")
			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) < 2 {
				log.Error().Msgf("connid subtopics weird %v", msg.Subject)
				continue
			}
			connID := subtopics[1]
			h.sendToConnID(connID, msg.Data)

		case msg := <-h.pubsub.subchans["usertv.>"]:
			// XXX: This might not really work. We should only send to gametv
			// and have something else follow the user across games.
			// A usertv message is meant for people who are watching a user's games.
			// Find the appropriate Realm.
			log.Debug().Str("topic", msg.Subject).Msg("got usertv message, forwarding along")
			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) < 2 {
				log.Error().Msgf("usertv subtopics weird %v", msg.Subject)
				continue
			}
			userID := subtopics[1]
			h.sendToRealm(Realm("usertv-"+userID), msg.Data)

		case msg := <-h.pubsub.subchans["gametv.>"]:
			// A gametv message is meant for people who are observing a user's games.
			log.Debug().Str("topic", msg.Subject).Msg("got gametv message, forwarding along")
			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) < 2 {
				log.Error().Msgf("gametv subtopics weird %v", msg.Subject)
				continue
			}
			gameID := subtopics[1]
			h.sendToRealm(Realm("gametv-"+gameID), msg.Data)

		case msg := <-h.pubsub.subchans["game.>"]:
			// A game message is meant for people who are playing a game.
			log.Debug().Str("topic", msg.Subject).Msg("got game message, forwarding along")
			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) < 2 {
				log.Error().Msgf("gametv subtopics weird %v", msg.Subject)
				continue
			}
			gameID := subtopics[1]
			h.sendToRealm(Realm("game-"+gameID), msg.Data)

		case msg := <-h.pubsub.subchans["chat.>"]:
			log.Debug().Str("topic", msg.Subject).Msg("chat-msg")
			if strings.HasPrefix(msg.Subject, "chat.pm.") {
				// This is a private message. Send to each recipient.
				recipients := strings.Split(strings.TrimPrefix(msg.Subject, "chat.pm."), "_")
				log.Debug().Interface("recipients", recipients).Msg("private-message")
				for _, r := range recipients {
					h.sendToUser(r, msg.Data)
				}
			} else {
				h.sendToRealm(channelToRealm(msg.Subject), msg.Data)
			}

		case msg := <-h.pubsub.subchans["channel.>"]:
			log.Debug().Str("topic", msg.Subject).Msg("channel-msg")
			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) < 2 {
				log.Error().Msgf("channel subtopics weird %v", msg.Subject)
				continue
			}
			channelID := subtopics[1]
			h.sendToRealm(Realm("channel-"+channelID), msg.Data)

		case msg := <-h.pubsub.subchans["presence.changed.>"]:
			log.Debug().Str("topic", msg.Subject).Msg("presence-changed")
			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) < 3 {
				log.Error().Msgf("presence.changed subtopics weird %v", msg.Subject)
				continue
			}
			userID := subtopics[2]
			h.handlePresenceChanged(userID, msg.Data)

		case msg := <-h.pubsub.subchans["seek.followed.>"]:
			log.Debug().Str("topic", msg.Subject).Msg("seek-followed")
			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) < 3 {
				log.Error().Msgf("seek.followed subtopics weird %v", msg.Subject)
				continue
			}
			seekerUserID := subtopics[2]
			h.handleSeekFollowed(seekerUserID, msg.Data)
		}

	}
}
