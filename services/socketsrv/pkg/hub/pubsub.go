package sockets

import (
	"hash/fnv"
	"strings"

	nats "github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/natsopts"
	"github.com/woogles-io/liwords/services/socketsrv/pkg/config"
)

// subChanSize is how many messages each subscription holds while the hub
// works through earlier ones. nats.go drops whatever arrives when it is full
// ("slow consumer"). A socket deploy brings every client back within a few
// seconds, and each asks the API for its realms: several connid messages per
// client, about 2,000 for 450 clients.
const subChanSize = 4096

// followerShards is how many goroutines look up followers for
// presence.changed and seek.followed messages. A lookup is a request to the
// API, so these run apart from the forwarding of everything else.
const followerShards = 8

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
		// NEW: efficient presence change notifications
		"presence.changed.>",
		// Efficient seek notifications to followed users
		"seek.followed.>",
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

		}

	}
}

// followerProcess handles presence.changed and seek.followed messages, which
// are sent to a user's followers or follows. Looking those up is a request
// to the API, one per message unless the followers cache has them. Done in
// PubsubProcess, the lookups held up every other message: when a socket
// deploy brought hundreds of users back at once, each socket server spent
// seconds on them while connid messages piled up and were dropped.
//
// Messages are spread over followerShards goroutines by user, so one user's
// changes are handled in the order they came. A slow lookup delays only the
// users in its shard.
func (h *Hub) followerProcess(presence, seek func(userID string, data []byte)) {
	shards := newShardedRunner(followerShards, subChanSize/followerShards)
	for {
		select {
		case msg := <-h.pubsub.subchans["presence.changed.>"]:
			log.Debug().Str("topic", msg.Subject).Msg("presence-changed")
			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) < 3 {
				log.Error().Msgf("presence.changed subtopics weird %v", msg.Subject)
				continue
			}
			userID, data := subtopics[2], msg.Data
			shards.run(userID, func() { presence(userID, data) })

		case msg := <-h.pubsub.subchans["seek.followed.>"]:
			log.Debug().Str("topic", msg.Subject).Msg("seek-followed")
			subtopics := strings.Split(msg.Subject, ".")
			if len(subtopics) < 3 {
				log.Error().Msgf("seek.followed subtopics weird %v", msg.Subject)
				continue
			}
			seekerUserID, data := subtopics[2], msg.Data
			shards.run(seekerUserID, func() { seek(seekerUserID, data) })
		}
	}
}

// shardedRunner runs functions on a fixed set of goroutines, each function
// on the goroutine its key hashes to, so functions with the same key run one
// at a time in the order they were given.
type shardedRunner struct {
	shards []chan func()
}

func newShardedRunner(n, queue int) *shardedRunner {
	s := &shardedRunner{shards: make([]chan func(), n)}
	for i := range s.shards {
		ch := make(chan func(), queue)
		s.shards[i] = ch
		go func() {
			for f := range ch {
				f()
			}
		}()
	}
	return s
}

func (s *shardedRunner) shardFor(key string) int {
	f := fnv.New32a()
	f.Write([]byte(key))
	return int(f.Sum32() % uint32(len(s.shards)))
}

// run queues f on key's goroutine. It blocks while that goroutine's queue
// is full.
func (s *shardedRunner) run(key string, f func()) {
	s.shards[s.shardFor(key)] <- f
}
