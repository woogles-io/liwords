package sockets

import (
	"strings"

	nats "github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"

	"github.com/woogles-io/liwords/pkg/natsopts"
	"github.com/woogles-io/liwords/services/socketsrv/pkg/config"
)

// subChanSize is how many messages each subscription buffers. NATS drops
// messages for a subscription whose buffer is full ("slow consumer"). When a
// socket server restarts, every browser reconnects to the new one within
// seconds, and the API answers each with several messages; 512 was enough
// only while nothing slowed down forwarding them.
const subChanSize = 4096

// PubSub encapsulates the various subscriptions to the different channels.
// The `liwords` package should have a very similar structure.
type PubSub struct {
	natsconn      *nats.Conn
	topics        []string
	subscriptions []*nats.Subscription
	subchans      map[string]chan *nats.Msg
	config        *config.Config
}

func newPubSub(natsURL string, cfg *config.Config, onReconnect func()) (*PubSub, error) {
	natsconn, err := nats.Connect(natsURL, natsopts.Options("liwords-socket", onReconnect)...)

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
		// leagues
		"league.>",
		// chats
		"chat.>",
		// generic channels
		"channel.>",
		// live broadcast updates
		"broadcasts.>",
		// a user's presence changed; for the users who follow them
		"presence.changed.>",
		// a user followed or unfollowed someone
		"follows.>",
		// a seek open only to the players its seeker follows
		"seek.recipients.>",
		// a user's sessions were revoked; disconnect their sockets
		"sessionend.>",
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
		ch := make(chan *nats.Msg, subChanSize)
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

		case msg := <-h.pubsub.subchans["league.>"]:
			log.Debug().Str("topic", msg.Subject).Int("type", int(msg.Data[2])).Msg("got league message, forwarding along")

			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) < 2 {
				log.Error().Msgf("league subtopics weird %v", msg.Subject)
				continue
			}
			leagueID := subtopics[1]
			h.sendToRealm(Realm("league-"+leagueID), msg.Data)

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

		case msg := <-h.pubsub.subchans["sessionend.>"]:
			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) != 2 || subtopics[1] == "" {
				log.Error().Msgf("sessionend subtopics weird %v", msg.Subject)
				continue
			}
			h.endUserSockets <- subtopics[1]

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

		case msg := <-h.pubsub.subchans["broadcasts.>"]:
			log.Debug().Str("topic", msg.Subject).Msg("broadcast-msg")
			subtopics := strings.SplitN(msg.Subject, ".", 2)
			if len(subtopics) < 2 {
				log.Error().Msgf("broadcasts subtopics weird %v", msg.Subject)
				continue
			}
			h.sendToRealm(Realm("broadcasts-"+subtopics[1]), msg.Data)

		case msg := <-h.pubsub.subchans["presence.changed.>"]:
			log.Debug().Str("topic", msg.Subject).Msg("presence-changed")
			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) != 3 || subtopics[2] == "" {
				log.Error().Msgf("presence.changed subtopics weird %v", msg.Subject)
				continue
			}
			h.sendFollowers <- FollowersMessage{
				userID: subtopics[2],
				msg:    entity.BytesFromSerializedEvent(msg.Data, byte(pb.MessageType_PRESENCE_ENTRY)),
			}

		case msg := <-h.pubsub.subchans["follows.>"]:
			change, ok := parseFollowChange(msg.Subject)
			if !ok {
				log.Error().Msgf("follows subtopics weird %v", msg.Subject)
				continue
			}
			h.followChanges <- change

		case msg := <-h.pubsub.subchans["seek.recipients.>"]:
			log.Debug().Str("topic", msg.Subject).Msg("seek-recipients")
			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) != 3 || subtopics[2] == "" {
				log.Error().Msgf("seek.recipients subtopics weird %v", msg.Subject)
				continue
			}
			m, err := followedPlayersSeek(subtopics[2], msg.Data)
			if err != nil {
				log.Err(err).Str("topic", msg.Subject).Msg("seek-recipients")
				continue
			}
			h.sendUsers <- m
		}

	}
}

// parseFollowChange parses follows.added.<follower>.<followed> or
// follows.removed.<follower>.<followed>.
func parseFollowChange(subject string) (FollowChange, bool) {
	subtopics := strings.Split(subject, ".")
	if len(subtopics) != 4 || subtopics[2] == "" || subtopics[3] == "" {
		return FollowChange{}, false
	}
	switch subtopics[1] {
	case "added", "removed":
	default:
		return FollowChange{}, false
	}
	return FollowChange{
		follower:  subtopics[2],
		followed:  subtopics[3],
		following: subtopics[1] == "added",
	}, true
}

// followedPlayersSeek is the message for a seek open only to the players its
// seeker follows: the seek, for those players and the seeker.
func followedPlayersSeek(seekerUserID string, data []byte) (UsersMessage, error) {
	fps := &pb.FollowedPlayersSeek{}
	if err := proto.Unmarshal(data, fps); err != nil {
		return UsersMessage{}, err
	}
	seek, err := proto.Marshal(fps.SeekRequest)
	if err != nil {
		return UsersMessage{}, err
	}
	userIDs := []string{seekerUserID}
	for _, id := range fps.RecipientUserIds {
		if id != seekerUserID {
			userIDs = append(userIDs, id)
		}
	}
	return UsersMessage{
		userIDs: userIDs,
		msg:     entity.BytesFromSerializedEvent(seek, byte(pb.MessageType_SEEK_REQUEST)),
	}, nil
}
