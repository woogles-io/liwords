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
)

const (
	GameStartDelay = 3 * time.Second

	MaxMessageLength = 256
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
	gameStore.SetGameEventChan(bus.gameEventChan)

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

outerfor:
	for {
		select {
		case msg := <-b.subchans["ipc.pb.>"]:
			// Regular messages.
			log.Debug().Str("topic", msg.Subject).Msg("got ipc.pb message")
			subtopics := strings.Split(msg.Subject, ".")
			err := b.handleNatsPublish(ctx, subtopics[2:], msg.Data)
			if err != nil {
				log.Err(err).Msg("process-message-publish-error")
				// The user ID should have hopefully come in the topic name.
				// It would be in subtopics[4]
				if len(subtopics) > 4 {
					userID := subtopics[4]
					b.pubToUser(userID, entity.WrapEvent(&pb.ErrorMessage{Message: err.Error()},
						pb.MessageType_ERROR_MESSAGE, ""))
				}
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
		case <-ctx.Done():
			log.Info().Msg("context done, breaking")
			break outerfor
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
			return errors.New("realm-req-not-handled")
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
		return fmt.Errorf("unhandled-req-topic: %v", topic)
	}
	return nil
}

// A somewhat silly function to get around Go's lack of generics
func setMatchUser(msg proto.Message, reqUser *pb.MatchUser) {
	switch sought := msg.(type) {
	case *pb.SeekRequest:
		sought.User = reqUser
	case *pb.MatchRequest:
		// lol
		sought.User = reqUser
	}
}

func (b *Bus) handleNatsPublish(ctx context.Context, subtopics []string, data []byte) error {
	log.Debug().Interface("subtopics", subtopics).Msg("handling nats publish")
	switch subtopics[0] {
	case "seekRequest", "matchRequest":
		return b.seekRequest(ctx, subtopics[0], subtopics[1], subtopics[2], data)
	case "chat":
		// The user is subtopics[2]
		evt := &pb.ChatMessage{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		log.Debug().Str("user", subtopics[2]).Str("msg", evt.Message).Str("channel", evt.Channel).Msg("chat")
		return b.chat(ctx, subtopics[2], evt)
	case "declineMatchRequest":
		evt := &pb.DeclineMatchRequest{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		log.Debug().Str("user", subtopics[2]).Str("reqid", evt.RequestId).Msg("decline-rematch")
		return b.matchDeclined(ctx, evt, subtopics[2])

	case "soughtGameProcess":
		evt := &pb.SoughtGameProcessEvent{}
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
		return gameplay.PlayMove(ctx, b.gameStore, b.userStore, subtopics[2], evt)

	case "timedOut":
		evt := &pb.TimedOut{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return gameplay.TimedOut(ctx, b.gameStore, b.userStore, evt.UserId, evt.GameId)

	case "initRealmInfo":
		evt := &pb.InitRealmInfo{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return b.initRealmInfo(ctx, evt)

	case "leaveSite":
		// There is no event here. We have the user ID in the subject.
		// req.User.IsAnonymous = subtopics[1] == "anon"
		userID := subtopics[2]
		return b.leaveSite(ctx, userID)
	default:
		return fmt.Errorf("unhandled-publish-topic: %v", subtopics)
	}
}

func (b *Bus) seekRequest(ctx context.Context, seekOrMatch, auth, userID string, data []byte) error {
	var req proto.Message
	var gameRequest *pb.GameRequest

	if auth == "anon" {
		// Require login for now (forever?)
		return errors.New("please log in to start a game")
	}

	if seekOrMatch == "seekRequest" {
		req = &pb.SeekRequest{}
	} else {
		req = &pb.MatchRequest{}
	}
	err := proto.Unmarshal(data, req)
	if err != nil {
		return err
	}

	if seekOrMatch == "seekRequest" {
		gameRequest = req.(*pb.SeekRequest).GameRequest
	} else {
		// Get the game request from the passed in "rematchFor", if it
		// is provided. Otherwise, the game request must have been provided
		// in the request itself.
		gameID := req.(*pb.MatchRequest).RematchFor
		if gameID == "" {
			gameRequest = req.(*pb.MatchRequest).GameRequest
		} else {
			g, err := b.gameStore.Get(ctx, gameID)
			if err != nil {
				return err
			}
			gameRequest = proto.Clone(g.GameReq).(*pb.GameRequest)
			// This will get overwritten later:
			gameRequest.RequestId = ""
			req.(*pb.MatchRequest).GameRequest = gameRequest
		}
	}
	if gameRequest == nil {
		return errors.New("no game request was found")
	}
	// Note that the seek request should not come with a requesting user;
	// instead this is in the topic/subject. It is HERE in the API server that
	// we set the requesting user's display name, rating, etc.
	reqUser := &pb.MatchUser{}
	reqUser.IsAnonymous = auth == "anon" // this is never true here anymore, see check above
	reqUser.UserId = userID
	setMatchUser(req, reqUser)

	err = gameplay.ValidateSoughtGame(ctx, gameRequest)
	if err != nil {
		return err
	}

	// Look up user.
	timefmt, variant, err := entity.VariantFromGameReq(gameRequest)
	ratingKey := entity.ToVariantKey(gameRequest.Lexicon, variant, timefmt)

	u, err := b.userStore.GetByUUID(ctx, reqUser.UserId)
	if err != nil {
		return err
	}
	reqUser.RelevantRating = u.GetRelevantRating(ratingKey)
	reqUser.DisplayName = u.Username

	if seekOrMatch == "seekRequest" {
		sg, err := gameplay.NewSoughtGame(ctx, b.soughtGameStore, req.(*pb.SeekRequest))
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
	} else {
		// Check if the user being matched exists.
		receiver, err := b.userStore.Get(ctx, req.(*pb.MatchRequest).ReceivingUser.DisplayName)
		if err != nil {
			// No such user, most likely.
			return err
		}
		// Set the actual UUID of the receiving user.
		req.(*pb.MatchRequest).ReceivingUser.UserId = receiver.UUID
		mg, err := gameplay.NewMatchRequest(ctx, b.soughtGameStore, req.(*pb.MatchRequest))
		if err != nil {
			return err
		}
		evt := entity.WrapEvent(mg.MatchRequest, pb.MessageType_MATCH_REQUEST, "")
		log.Debug().Interface("evt", evt).Interface("receiver", mg.MatchRequest.ReceivingUser).
			Str("sender", reqUser.UserId).Msg("publishing match request to user")
		b.pubToUser(receiver.UUID, evt)
		// Publish it to the requester as well. This is so they can see it on
		// their own screen and cancel it if they wish.
		b.pubToUser(reqUser.UserId, evt)
	}
	return nil
}

func (b *Bus) gameAccepted(ctx context.Context, evt *pb.SoughtGameProcessEvent, userID string) error {
	sg, err := b.soughtGameStore.Get(ctx, evt.RequestId)
	if err != nil {
		return err
	}
	var requester string
	var gameReq *pb.GameRequest
	if sg.Type() == entity.TypeSeek {
		requester = sg.SeekRequest.User.UserId
		gameReq = sg.SeekRequest.GameRequest
	} else if sg.Type() == entity.TypeMatch {
		requester = sg.MatchRequest.User.UserId
		gameReq = sg.MatchRequest.GameRequest
	}
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
	// If the ACCEPTOR of the seek has a seek request open, we must cancel it.
	err = b.deleteSoughtForUser(ctx, userID)
	if err != nil {
		return err
	}

	accUser, err := b.userStore.GetByUUID(ctx, userID) // acceptor
	if err != nil {
		return err
	}
	reqUser, err := b.userStore.GetByUUID(ctx, requester) //requester
	if err != nil {
		return err
	}
	// disallow anon game acceptance for now.
	if accUser.Anonymous || reqUser.Anonymous {
		return errors.New("you must log in to play games")
	}

	if (accUser.Anonymous || reqUser.Anonymous) && gameReq.RatingMode == pb.RatingMode_RATED {
		return errors.New("anonymous-players-cant-play-rated")
	}

	log.Debug().Interface("req", sg).Msg("game-request-accepted")
	assignedFirst := -1
	if sg.Type() == entity.TypeMatch {
		if sg.MatchRequest.RematchFor != "" {
			// Assign firsts to be the the other player.
			gameID := sg.MatchRequest.RematchFor
			g, err := b.gameStore.Get(ctx, gameID)
			if err != nil {
				return err
			}
			wentFirst := 0
			players := g.History().Players
			if g.History().SecondWentFirst {
				wentFirst = 1
			}
			log.Debug().Str("went-first", players[wentFirst].Nickname).Msg("determining-first")

			// These are indices in the array passed to InstantiateNewGame
			if accUser.UUID == players[wentFirst].UserId {
				assignedFirst = 1 // reqUser should go first
			} else if reqUser.UUID == players[wentFirst].UserId {
				assignedFirst = 0 // accUser should go first
			}
		}
	}

	g, err := gameplay.InstantiateNewGame(ctx, b.gameStore, b.config,
		[2]*entity.User{accUser, reqUser}, assignedFirst, gameReq)
	if err != nil {
		return err
	}
	// Broadcast a seek delete event, and send both parties a game redirect.
	b.soughtGameStore.Delete(ctx, evt.RequestId)
	err = b.broadcastSeekDeletion(evt.RequestId)
	if err != nil {
		log.Err(err).Msg("broadcasting-seek")
	}
	err = b.broadcastGameCreation(g, accUser, reqUser)
	if err != nil {
		log.Err(err).Msg("broadcasting-game-creation")
	}
	// This event will result in a redirect.
	ngevt := entity.WrapEvent(&pb.NewGameEvent{
		GameId: g.GameID(),
	}, pb.MessageType_NEW_GAME_EVENT, "")
	b.pubToUser(accUser.UUID, ngevt)
	b.pubToUser(reqUser.UUID, ngevt)

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

func (b *Bus) matchDeclined(ctx context.Context, evt *pb.DeclineMatchRequest, userID string) error {
	// the sending user declined the match request. Send this declination
	// to the matcher and delete the request.
	sg, err := b.soughtGameStore.Get(ctx, evt.RequestId)
	if err != nil {
		return err
	}
	if sg.Type() != entity.TypeMatch {
		return errors.New("wrong-entity-type")
	}

	if sg.MatchRequest.ReceivingUser.UserId != userID {
		return errors.New("request userID does not match")
	}

	err = gameplay.CancelSoughtGame(ctx, b.soughtGameStore, evt.RequestId)
	if err != nil {
		return err
	}

	requester := sg.MatchRequest.User.UserId
	decliner := userID

	wrapped := entity.WrapEvent(evt, pb.MessageType_DECLINE_MATCH_REQUEST, "")

	// Publish decline to requester
	err = b.pubToUser(requester, wrapped)
	if err != nil {
		return err
	}
	wrapped = entity.WrapEvent(&pb.SoughtGameProcessEvent{RequestId: evt.RequestId},
		pb.MessageType_SOUGHT_GAME_PROCESS_EVENT, "")
	return b.pubToUser(decliner, wrapped)
}

func (b *Bus) chat(ctx context.Context, userID string, evt *pb.ChatMessage) error {
	if len(evt.Message) > MaxMessageLength {
		return errors.New("message-too-long")
	}
	username, err := b.userStore.Username(ctx, userID)
	if err != nil {
		return err
	}

	toSend := entity.WrapEvent(&pb.ChatMessage{
		Username: username,
		Channel:  evt.Channel,
		Message:  evt.Message,
	}, pb.MessageType_CHAT_MESSAGE, "")
	data, err := toSend.Serialize()
	if err != nil {
		return err
	}
	return b.natsconn.Publish(evt.Channel, data)
}

func (b *Bus) broadcastSeekDeletion(seekID string) error {
	toSend := entity.WrapEvent(&pb.SoughtGameProcessEvent{RequestId: seekID},
		pb.MessageType_SOUGHT_GAME_PROCESS_EVENT, "")
	data, err := toSend.Serialize()
	if err != nil {
		return err
	}
	return b.natsconn.Publish("lobby.soughtGameProcess", data)
}

func (b *Bus) broadcastGameCreation(g *entity.Game, acceptor, requester *entity.User) error {
	timefmt, variant, err := entity.VariantFromGameReq(g.GameReq)
	if err != nil {
		return err
	}
	ratingKey := entity.ToVariantKey(g.GameReq.Lexicon, variant, timefmt)
	users := []*pb.GameMeta_UserMeta{
		{RelevantRating: acceptor.GetRelevantRating(ratingKey),
			DisplayName: acceptor.Username},
		{RelevantRating: requester.GetRelevantRating(ratingKey),
			DisplayName: requester.Username},
	}

	toSend := entity.WrapEvent(&pb.GameMeta{Users: users,
		GameRequest: g.GameReq, Id: g.GameID()},
		pb.MessageType_GAME_META_EVENT, "")
	data, err := toSend.Serialize()
	if err != nil {
		return err
	}
	return b.natsconn.Publish("lobby.newLiveGame", data)
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
		// open seeks
		seeks, err := b.openSeeks(ctx)
		if err != nil {
			return err
		}
		err = b.pubToUser(evt.UserId, seeks)
		if err != nil {
			return err
		}
		// live games
		activeGames, err := b.activeGames(ctx)
		if err != nil {
			return err
		}
		err = b.pubToUser(evt.UserId, activeGames)
		if err != nil {
			return err
		}
		// open match reqs
		matches, err := b.openMatches(ctx, evt.UserId)
		if err != nil {
			return err
		}
		err = b.pubToUser(evt.UserId, matches)
		if err != nil {
			return err
		}
		// TODO: send followed online
	} else if strings.HasPrefix(evt.Realm, "game-") || strings.HasPrefix(evt.Realm, "gametv-") {
		// Get a sanitized history
		gameID := strings.Split(evt.Realm, "-")[1]
		refresher, err := b.gameRefresher(ctx, gameID)
		if err != nil {
			return err
		}
		return b.pubToUser(evt.UserId, refresher)
	} else {
		log.Debug().Interface("evt", evt).Msg("no init realm info")
	}
	return nil
}

func (b *Bus) deleteSoughtForUser(ctx context.Context, userID string) error {
	reqID, err := b.soughtGameStore.DeleteForUser(ctx, userID)
	if err != nil {
		return err
	}
	if reqID == "" {
		return nil
	}
	log.Debug().Str("reqID", reqID).Str("userID", userID).Msg("deleting-sought")
	return b.broadcastSeekDeletion(reqID)
}

func (b *Bus) leaveSite(ctx context.Context, userID string) error {
	return b.deleteSoughtForUser(ctx, userID)
}

func (b *Bus) openSeeks(ctx context.Context) (*entity.EventWrapper, error) {
	sgs, err := b.soughtGameStore.ListOpenSeeks(ctx)
	if err != nil {
		return nil, err
	}
	log.Debug().Interface("open-seeks", sgs).Msg("open-seeks")

	pbobj := &pb.SeekRequests{Requests: []*pb.SeekRequest{}}
	for _, sg := range sgs {
		pbobj.Requests = append(pbobj.Requests, sg.SeekRequest)
	}
	evt := entity.WrapEvent(pbobj, pb.MessageType_SEEK_REQUESTS, "")
	return evt, nil
}

func (b *Bus) openMatches(ctx context.Context, receiverID string) (*entity.EventWrapper, error) {
	sgs, err := b.soughtGameStore.ListOpenMatches(ctx, receiverID)
	if err != nil {
		return nil, err
	}
	log.Debug().Str("receiver", receiverID).Interface("open-matches", sgs).Msg("open-seeks")
	pbobj := &pb.MatchRequests{Requests: []*pb.MatchRequest{}}
	for _, sg := range sgs {
		pbobj.Requests = append(pbobj.Requests, sg.MatchRequest)
	}
	evt := entity.WrapEvent(pbobj, pb.MessageType_MATCH_REQUESTS, "")
	return evt, nil
}

func (b *Bus) activeGames(ctx context.Context) (*entity.EventWrapper, error) {
	gs, err := b.gameStore.ListActive(ctx)

	if err != nil {
		return nil, err
	}
	log.Debug().Interface("active-games", gs).Msg("active-games")

	pbobj := &pb.ActiveGames{Games: []*pb.GameMeta{}}
	for _, g := range gs {
		pbobj.Games = append(pbobj.Games, g)
	}
	evt := entity.WrapEvent(pbobj, pb.MessageType_ACTIVE_GAMES, "")
	return evt, nil
}

func (b *Bus) gameRefresher(ctx context.Context, gameID string) (*entity.EventWrapper, error) {
	// Get a game refresher event. If sanitize is true, opponent racks will be
	// hidden from the given userID.
	entGame, err := b.gameStore.Get(ctx, string(gameID))
	if err != nil {
		return nil, err
	}
	if !entGame.Started {
		return nil, errors.New("game-starting-soon")
	}
	evt := entity.WrapEvent(entGame.HistoryRefresherEvent(),
		pb.MessageType_GAME_HISTORY_REFRESHER, entGame.GameID())
	return evt, nil
}
