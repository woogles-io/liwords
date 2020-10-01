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

	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"

	nats "github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/stats"
	"github.com/domino14/liwords/pkg/user"
	macondogame "github.com/domino14/macondo/game"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

const (
	GameStartDelay = 3 * time.Second

	MaxMessageLength = 500

	AdjudicateInterval = 300 * time.Second
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

	redisPool *redis.Pool

	subscriptions []*nats.Subscription
	subchans      map[string]chan *nats.Msg

	gameEventChan chan *entity.EventWrapper
}

func NewBus(cfg *config.Config, userStore user.Store, gameStore gameplay.GameStore,
	soughtGameStore gameplay.SoughtGameStore, presenceStore user.PresenceStore,
	listStatStore stats.ListStatStore, redisPool *redis.Pool) (*Bus, error) {

	natsconn, err := nats.Connect(cfg.NatsURL)

	if err != nil {
		return nil, err
	}
	bus := &Bus{
		natsconn:        natsconn,
		userStore:       userStore,
		gameStore:       gameStore,
		soughtGameStore: soughtGameStore,
		presenceStore:   presenceStore,
		listStatStore:   listStatStore,
		subscriptions:   []*nats.Subscription{},
		subchans:        map[string]chan *nats.Msg{},
		config:          cfg,
		gameEventChan:   make(chan *entity.EventWrapper, 64),
		redisPool:       redisPool,
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

	// Adjudicate unfinished games every few minutes.
	// adjudicator := time.NewTicker(AdjudicateInterval)
	// defer adjudicator.Stop()
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
						Realm: "",
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
			log.Debug().Interface("msg", msg).Msg("game event chan")
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
		case <-ctx.Done():
			log.Info().Msg("context done, breaking")
			break outerfor

			// case <-adjudicator.C:
			// 	err := b.adjudicateGames(ctx)
			// 	if err != nil {
			// 		log.Err(err).Msg("adjudicate-error")
			// 		break
			// 	}

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
			log.Info().Str("path", path).Msg("realm-req-not-handled-sending-blank-realm")
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

func (b *Bus) handleBotMove(ctx context.Context, g *entity.Game) {
	// This function should only be called if it's the bot's turn.
	onTurn := g.Game.PlayerOnTurn()
	userID := g.Game.PlayerIDOnTurn()
	g.Lock()
	defer g.Unlock()
	// We check if that game is not over because a triple challenge
	// could have ended it
	for g.PlayerOnTurn() == onTurn && g.Game.Playing() != macondopb.PlayState_GAME_OVER {
		hist := g.History()
		req := macondopb.BotRequest{GameHistory: hist}
		data, err := proto.Marshal(&req)
		if err != nil {
			log.Err(err).Msg("bot-cant-move")
			return
		}
		res, err := b.natsconn.Request("macondo.bot", data, 10*time.Second)

		if err != nil {
			if b.natsconn.LastError() != nil {
				log.Error().Msgf("bot-cant-move %v for request", b.natsconn.LastError())
			}
			log.Error().Msgf("bot-cant-move %v for request", err)
			return
		}
		log.Debug().Msgf("res: %v", string(res.Data))

		resp := macondopb.BotResponse{}
		err = proto.Unmarshal(res.Data, &resp)
		if err != nil {
			log.Err(err).Msg("bot-cant-move-unmarshal-error")
			return
		}
		switch r := resp.Response.(type) {
		case *macondopb.BotResponse_Move:
			timeRemaining := g.TimeRemaining(onTurn)

			m := macondogame.MoveFromEvent(r.Move, g.Alphabet(), g.Board())
			err = gameplay.PlayMove(ctx, g, b.userStore, b.listStatStore, userID, onTurn, timeRemaining, m)
			if err != nil {
				log.Err(err).Msg("bot-cant-move-play-error")
				return
			}
		case *macondopb.BotResponse_Error:
			log.Error().Str("error", r.Error).Msg("bot-error")
			return
		default:
			log.Err(errors.New("should never happen")).Msg("bot-cant-move")
		}
	}

	err := b.gameStore.Set(ctx, g)
	if err != nil {
		log.Err(err).Msg("setting-game-after-bot-move")
	}

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
	case "seekRequest", "matchRequest":
		return b.seekRequest(ctx, msgType, auth, userID, wsConnID, data)
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
			userID, evt)
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
		return gameplay.TimedOut(ctx, b.gameStore, b.userStore, b.listStatStore, evt.UserId, evt.GameId)

	case "initRealmInfo":
		evt := &pb.InitRealmInfo{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}
		return b.initRealmInfo(ctx, evt)
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
	default:
		return fmt.Errorf("unhandled-publish-topic: %v", subtopics)
	}
}

func (b *Bus) seekRequest(ctx context.Context, seekOrMatch, auth, userID, connID string,
	data []byte) error {

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
	isRematch := false
	lastOpp := ""

	if seekOrMatch == "seekRequest" {
		gameRequest = req.(*pb.SeekRequest).GameRequest
	} else {
		// Get the game request from the passed in "rematchFor", if it
		// is provided. Otherwise, the game request must have been provided
		// in the request itself.
		gameID := req.(*pb.MatchRequest).RematchFor
		if gameID == "" {
			gameRequest = req.(*pb.MatchRequest).GameRequest
		} else { // It's a rematch.
			isRematch = true
			g, err := b.gameStore.Get(ctx, gameID)
			if err != nil {
				return err
			}
			// Figure out who we played against.
			for _, u := range g.History().Players {
				if u.UserId == userID {
					continue
				}
				lastOpp = u.UserId
			}
			gameRequest = proto.Clone(g.GameReq).(*pb.GameRequest)
			// If this game is a rematch, set the OriginalRequestId
			// to the previous game's OriginalRequestId. In this way,
			// we maintain a constant OriginalRequestId value across
			// rematch streaks. The OriginalRequestId is set in
			// NewSoughtGame and NewMatchRequest in sought_game.go
			// if it is not set here.
			gameRequest.OriginalRequestId = g.GameReq.OriginalRequestId
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

	log.Debug().Bool("vsBot", gameRequest.PlayerVsBot).Msg("seeking-bot?")

	if seekOrMatch == "seekRequest" {
		sr := req.(*pb.SeekRequest)
		sr.ConnectionId = connID
		sg, err := gameplay.NewSoughtGame(ctx, b.soughtGameStore, sr)
		if err != nil {
			return err
		}
		evt := entity.WrapEvent(sg.SeekRequest, pb.MessageType_SEEK_REQUEST)
		data, err := evt.Serialize()
		if err != nil {
			return err
		}

		log.Debug().Interface("evt", evt).Msg("publishing seek request to lobby topic")
		b.natsconn.Publish("lobby.seekRequest", data)
	} else {
		mr := req.(*pb.MatchRequest)
		mr.ConnectionId = connID
		// It's a direct match request.
		if gameRequest.PlayerVsBot {
			// There is no user being matched. Find a bot to play instead.
			// No need to create a match request in the store.
			botToPlay := ""
			if isRematch {
				botToPlay = lastOpp
				log.Debug().Str("bot", botToPlay).Msg("forcing-bot")
			}
			return b.newBotGame(ctx, mr, botToPlay)
		}
		// Check if the user being matched exists.
		receiver, err := b.userStore.Get(ctx, mr.ReceivingUser.DisplayName)
		if err != nil {
			// No such user, most likely.
			return err
		}
		// Set the actual UUID of the receiving user.
		mr.ReceivingUser.UserId = receiver.UUID
		mg, err := gameplay.NewMatchRequest(ctx, b.soughtGameStore, mr)
		if err != nil {
			return err
		}
		evt := entity.WrapEvent(mg.MatchRequest, pb.MessageType_MATCH_REQUEST)
		log.Debug().Interface("evt", evt).Interface("req", mr).Interface("receiver", mg.MatchRequest.ReceivingUser).
			Str("sender", reqUser.UserId).Msg("publishing match request to user")
		b.pubToUser(receiver.UUID, evt, "")
		// Publish it to the requester as well. This is so they can see it on
		// their own screen and cancel it if they wish.
		b.pubToUser(reqUser.UserId, evt, "")
	}
	return nil
}

func (b *Bus) newBotGame(ctx context.Context, req *pb.MatchRequest, botUserID string) error {
	// NewBotGame creates and starts a new game against a bot!
	var err error
	var accUser *entity.User
	if botUserID == "" {
		accUser, err = b.userStore.GetRandomBot(ctx)
	} else {
		accUser, err = b.userStore.GetByUUID(ctx, botUserID)
	}
	if err != nil {
		return err
	}
	sg := &entity.SoughtGame{MatchRequest: req}
	return b.instantiateAndStartGame(ctx, accUser, req.User.UserId, req.GameRequest,
		sg, BotRequestID, "")
}

func (b *Bus) gameAccepted(ctx context.Context, evt *pb.SoughtGameProcessEvent,
	userID, connID string) error {
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

	accUser, err := b.userStore.GetByUUID(ctx, userID)
	if err != nil {
		return err
	}

	return b.instantiateAndStartGame(ctx, accUser, requester, gameReq, sg, evt.RequestId, connID)
}

func (b *Bus) instantiateAndStartGame(ctx context.Context, accUser *entity.User, requester string,
	gameReq *pb.GameRequest, sg *entity.SoughtGame, reqID, acceptingConnID string) error {

	reqUser, err := b.userStore.GetByUUID(ctx, requester)
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

	log.Debug().Interface("req", sg).
		Str("seek-conn", sg.ConnID()).
		Str("accepting-conn", acceptingConnID).Msg("game-request-accepted")
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
	if reqID != BotRequestID {
		b.soughtGameStore.Delete(ctx, reqID)
		err = b.broadcastSeekDeletion(reqID)
		if err != nil {
			log.Err(err).Msg("broadcasting-seek")
		}
	}

	err = b.broadcastGameCreation(g, accUser, reqUser)
	if err != nil {
		log.Err(err).Msg("broadcasting-game-creation")
	}
	// This event will result in a redirect.
	ngevt := entity.WrapEvent(&pb.NewGameEvent{
		GameId:       g.GameID(),
		AccepterCid:  acceptingConnID,
		RequesterCid: sg.ConnID(),
	}, pb.MessageType_NEW_GAME_EVENT)
	// The front end keeps track of which tabs seek/accept games etc
	// so we don't attach any extra channel info here.
	b.pubToUser(accUser.UUID, ngevt, "")
	b.pubToUser(reqUser.UUID, ngevt, "")

	log.Info().Str("newgameid", g.History().Uid).
		Str("sender", accUser.UUID).
		Str("requester", requester).
		Str("reqID", reqID).
		Interface("starting-in", GameStartDelay).
		Str("onturn", g.NickOnTurn()).Msg("game-accepted")

	return nil
}

func (b *Bus) readyForGame(ctx context.Context, evt *pb.ReadyForGame, userID string) error {
	g, err := b.gameStore.Get(ctx, evt.GameId)
	if err != nil {
		return err
	}
	g.Lock()
	defer g.Unlock()
	log.Debug().Str("userID", userID).Interface("playing", g.Playing()).Msg("ready-for-game")
	if g.Playing() != macondopb.PlayState_PLAYING {
		return errors.New("game is over")
	}

	if g.History().Players[0].UserId == userID {
		g.PlayersReady[0] = true
	} else if g.History().Players[1].UserId == userID {
		g.PlayersReady[1] = true
	} else {
		log.Error().Str("userID", userID).Str("gameID", evt.GameId).Msg("not-in-game")
		return errors.New("ready for game but not in game")
	}

	// Start the game if both players are ready (or if it's a bot game).
	if g.PlayersReady[0] && g.PlayersReady[1] || g.GameReq.PlayerVsBot {
		err = gameplay.StartGame(ctx, b.gameStore, b.gameEventChan, g.GameID())
		if err != nil {
			log.Err(err).Msg("starting-game")
		}

		if g.GameReq.PlayerVsBot && g.PlayerIDOnTurn() != userID {
			// Make a bot move if it's the bot's turn at the beginning.
			go b.handleBotMove(ctx, g)
		}
	}
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

	wrapped := entity.WrapEvent(evt, pb.MessageType_DECLINE_MATCH_REQUEST)

	// Publish decline to requester
	err = b.pubToUser(requester, wrapped, "")
	if err != nil {
		return err
	}
	wrapped = entity.WrapEvent(&pb.SoughtGameProcessEvent{RequestId: evt.RequestId},
		pb.MessageType_SOUGHT_GAME_PROCESS_EVENT)
	return b.pubToUser(decliner, wrapped, "")
}

func (b *Bus) broadcastSeekDeletion(seekID string) error {
	toSend := entity.WrapEvent(&pb.SoughtGameProcessEvent{RequestId: seekID},
		pb.MessageType_SOUGHT_GAME_PROCESS_EVENT)
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
		pb.MessageType_GAME_META_EVENT)
	data, err := toSend.Serialize()
	if err != nil {
		return err
	}
	return b.natsconn.Publish("lobby.newLiveGame", data)
}

func (b *Bus) broadcastPresence(username, userID string, anon bool, presenceChan string, deleting bool) error {
	// broadcast username's presence to the channel.
	log.Debug().Str("username", username).Str("userID", userID).
		Bool("anon", anon).
		Str("presenceChan", presenceChan).
		Bool("deleting", deleting).
		Msg("broadcast-presence")

	evtChannel := presenceChan

	if deleting {
		evtChannel = ""
	}

	toSend := entity.WrapEvent(&pb.UserPresence{
		Username:    username,
		UserId:      userID,
		Channel:     evtChannel,
		IsAnonymous: anon,
	},
		pb.MessageType_USER_PRESENCE)
	data, err := toSend.Serialize()
	if err != nil {
		return err
	}
	if presenceChan != "" {
		return b.natsconn.Publish(presenceChan, data)
	}
	// If the presence channel is empty we are in some other page, like the
	// about page or something. We need to clean this up a bit, but we don't
	// want to log errors here.
	return nil
}

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

func (b *Bus) initRealmInfo(ctx context.Context, evt *pb.InitRealmInfo) error {
	// For consistency sake, use the `dotted` channels for presence
	// i.e. game.<gameID>, gametv.<gameID>
	// The reasoning is that realms should only be cared about by the socket
	// server. The channels are NATS pubsub channels and we use these for chat
	// too.
	username, anon, err := b.userStore.Username(ctx, evt.UserId)
	if err != nil {
		return err
	}
	presenceChan := strings.ReplaceAll(evt.Realm, "-", ".")
	chatChan := presenceChan
	if presenceChan == "lobby" {
		presenceChan = "lobby.presence"
		chatChan = "lobby.chat"
	}
	b.presenceStore.SetPresence(ctx, evt.UserId, username, anon, presenceChan)

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
		activeGames, err := b.activeGames(ctx)
		if err != nil {
			return err
		}
		err = b.pubToUser(evt.UserId, activeGames, "lobby")
		if err != nil {
			return err
		}
		// open match reqs
		matches, err := b.openMatches(ctx, evt.UserId)
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
	} else {
		log.Debug().Interface("evt", evt).Msg("no init realm info")
	}

	// Get presence
	pres, err := b.getPresence(ctx, presenceChan)
	if err != nil {
		return err
	}
	err = b.pubToUser(evt.UserId, pres, presenceChan)
	if err != nil {
		return err
	}
	// Also send OUR presence to users in this channel.
	err = b.broadcastPresence(username, evt.UserId, anon, presenceChan, false)
	if err != nil {
		return err
	}
	// send chat info
	return b.sendOldChats(evt.UserId, chatChan)
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

func (b *Bus) deleteSoughtForConnID(ctx context.Context, connID string) error {
	reqID, err := b.soughtGameStore.DeleteForConnID(ctx, connID)
	if err != nil {
		return err
	}
	if reqID == "" {
		return nil
	}
	log.Debug().Str("reqID", reqID).Str("connID", connID).Msg("deleting-sought")
	return b.broadcastSeekDeletion(reqID)
}

func (b *Bus) leaveTab(ctx context.Context, userID, connID string) error {
	return b.deleteSoughtForConnID(ctx, connID)
}

func (b *Bus) leaveSite(ctx context.Context, userID string) error {
	username, anon, err := b.userStore.Username(ctx, userID)
	if err != nil {
		return err
	}
	oldchannel, err := b.presenceStore.ClearPresence(ctx, userID, username, anon)
	if err != nil {
		return err
	}
	log.Debug().Str("oldchannel", oldchannel).Str("userid", userID).Msg("left-site")

	err = b.broadcastPresence(username, userID, anon, oldchannel, true)
	if err != nil {
		return err
	}

	return nil
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
	evt := entity.WrapEvent(pbobj, pb.MessageType_SEEK_REQUESTS)
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
	evt := entity.WrapEvent(pbobj, pb.MessageType_MATCH_REQUESTS)
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
	evt := entity.WrapEvent(pbobj, pb.MessageType_ACTIVE_GAMES)
	return evt, nil
}

func (b *Bus) adjudicateGames(ctx context.Context) error {
	gs, err := b.gameStore.ListActive(ctx)

	if err != nil {
		return err
	}
	log.Debug().Interface("active-games", gs).Msg("adjudicating...")
	for _, g := range gs {
		// These will likely be in the cache.
		entGame, err := b.gameStore.Get(ctx, g.Id)
		if err != nil {
			return err
		}
		onTurn := entGame.Game.PlayerOnTurn()
		if entGame.TimeRanOut(onTurn) {
			log.Debug().Str("gid", g.Id).Msg("time-ran-out")
			err = gameplay.TimedOut(ctx, b.gameStore, b.userStore,
				b.listStatStore, entGame.Game.PlayerIDOnTurn(), g.Id)
			log.Err(err).Msg("gameplay-timed-out")
		}
	}
	return nil
}

func (b *Bus) gameRefresher(ctx context.Context, gameID string) (*entity.EventWrapper, error) {
	// Get a game refresher event.
	entGame, err := b.gameStore.Get(ctx, string(gameID))
	if err != nil {
		return nil, err
	}
	entGame.RLock()
	defer entGame.RUnlock()
	if !entGame.Started {
		return entity.WrapEvent(&pb.ServerMessage{Message: "Game is starting soon!"},
			pb.MessageType_SERVER_MESSAGE), nil
	}
	evt := entity.WrapEvent(entGame.HistoryRefresherEvent(),
		pb.MessageType_GAME_HISTORY_REFRESHER)
	return evt, nil
}
