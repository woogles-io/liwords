package bus

import (
	"context"
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	gs "github.com/domino14/liwords/rpc/api/proto/game_service"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
)

func (b *Bus) seekRequest(ctx context.Context, auth, userID, connID string,
	data []byte) error {

	var gameRequest *pb.GameRequest

	if auth == "anon" {
		// Require login for now (forever?)
		return errors.New("please log in to start a game")
	}

	req := &pb.SeekRequest{}
	err := proto.Unmarshal(data, req)
	if err != nil {
		return err
	}

	gameRequest = req.GameRequest
	if gameRequest == nil {
		return errors.New("no game request was found")
	}
	// Note that the seek request should not come with a requesting user;
	// instead this is in the topic/subject. It is HERE in the API server that
	// we set the requesting user's display name, rating, etc.
	reqUser := &pb.MatchUser{}
	reqUser.IsAnonymous = auth == "anon" // this is never true here anymore, see check above
	reqUser.UserId = userID
	req.User = reqUser

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

	req.ConnectionId = connID
	sg, err := gameplay.NewSoughtGame(ctx, b.soughtGameStore, req)
	if err != nil {
		return err
	}
	evt := entity.WrapEvent(sg.SeekRequest, pb.MessageType_SEEK_REQUEST)
	outdata, err := evt.Serialize()
	if err != nil {
		return err
	}

	log.Debug().Interface("evt", evt).Msg("publishing seek request to lobby topic")

	b.natsconn.Publish("lobby.seekRequest", outdata)

	return nil
}

func (b *Bus) matchRequest(ctx context.Context, auth, userID, connID string,
	data []byte) error {

	if auth == "anon" {
		// Require login for now (forever?)
		return errors.New("please log in to start a game")
	}

	req := &pb.MatchRequest{}
	err := proto.Unmarshal(data, req)
	if err != nil {
		return err
	}

	gameRequest, lastOpp, err := b.gameRequestForMatch(ctx, req, userID)
	if err != nil {
		return err
	}
	if gameRequest == nil {
		return errors.New("no game request was found")
	}
	if req.GameRequest == nil {
		req.GameRequest = gameRequest
	}

	err = gameplay.ValidateSoughtGame(ctx, gameRequest)
	if err != nil {
		return err
	}

	// Look up user.
	// Note that the seek request should not come with a requesting user;
	// instead this is in the topic/subject. It is HERE in the API server that
	// we set the requesting user's display name, rating, etc.
	reqUser := &pb.MatchUser{}
	reqUser.IsAnonymous = auth == "anon" // this is never true here anymore, see check above
	reqUser.UserId = userID
	req.User = reqUser

	timefmt, variant, err := entity.VariantFromGameReq(gameRequest)
	ratingKey := entity.ToVariantKey(gameRequest.Lexicon, variant, timefmt)

	u, err := b.userStore.GetByUUID(ctx, reqUser.UserId)
	if err != nil {
		return err
	}
	reqUser.RelevantRating = u.GetRelevantRating(ratingKey)
	reqUser.DisplayName = u.Username

	log.Debug().Bool("vsBot", gameRequest.PlayerVsBot).Msg("seeking-bot?")

	req.ConnectionId = connID
	// It's a direct match request.
	if gameRequest.PlayerVsBot {
		// There is no user being matched. Find a bot to play instead.
		// No need to create a match request in the store.
		botToPlay := ""
		if req.RematchFor != "" {
			botToPlay = lastOpp
			log.Debug().Str("bot", botToPlay).Msg("forcing-bot")
		}
		return b.newBotGame(ctx, req, botToPlay)
	}
	// Check if the user being matched exists.
	req.ReceivingUser.DisplayName = strings.TrimSpace(req.ReceivingUser.DisplayName)
	receiver, err := b.userStore.Get(ctx, req.ReceivingUser.DisplayName)
	if err != nil {
		// No such user, most likely.
		return err
	}
	// Set the actual UUID of the receiving user.
	req.ReceivingUser.UserId = receiver.UUID

	// Check if receiving user is blocking the reqUser.
	blockedUsers, err := b.userStore.GetBlocks(ctx, receiver.ID)
	if err != nil {
		return err
	}
	for _, bu := range blockedUsers {
		if bu.UUID == reqUser.UserId {
			evt := entity.WrapEvent(&pb.ErrorMessage{
				Message: receiver.Username + " is not available for match requests",
			}, pb.MessageType_ERROR_MESSAGE)
			b.pubToUser(reqUser.UserId, evt, "")
			return nil
		}
	}

	mg, err := gameplay.NewMatchRequest(ctx, b.soughtGameStore, req)
	if err != nil {
		return err
	}
	evt := entity.WrapEvent(mg.MatchRequest, pb.MessageType_MATCH_REQUEST)
	log.Debug().Interface("evt", evt).Interface("req", req).Interface("receiver", mg.MatchRequest.ReceivingUser).
		Str("sender", reqUser.UserId).Msg("publishing match request to user")
	b.pubToUser(receiver.UUID, evt, "")
	// Publish it to the requester as well. This is so they can see it on
	// their own screen and cancel it if they wish.
	b.pubToUser(reqUser.UserId, evt, "")

	return nil
}

func (b *Bus) gameRequestForMatch(ctx context.Context, req *pb.MatchRequest,
	userID string) (*pb.GameRequest, string, error) {
	// Get the game request from the passed in "rematchFor", if it
	// is provided. Otherwise, the game request must have been provided
	// in the request itself.

	var gameRequest *pb.GameRequest
	gameID := req.RematchFor
	lastOpp := ""

	if gameID == "" {
		gameRequest = req.GameRequest
	} else { // It's a rematch.
		gm, err := b.gameStore.GetMetadata(ctx, gameID)
		if err != nil {
			return nil, "", err
		}
		// Figure out who we played against.
		for _, u := range gm.Players {
			if u.UserId == userID {
				continue
			}
			lastOpp = u.UserId
		}
		// If this game is a rematch, set the OriginalRequestId
		// to the previous game's OriginalRequestId. In this way,
		// we maintain a constant OriginalRequestId value across
		// rematch streaks. The OriginalRequestId is set in
		// NewSoughtGame and NewMatchRequest in sought_game.go
		// if it is not set here. We copy the whole game request which includes
		// the OriginalRequestId
		gameRequest = proto.Clone(gm.GameRequest).(*pb.GameRequest)

		// This will get overwritten later:
		gameRequest.RequestId = ""
	}
	return gameRequest, lastOpp, nil
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
	sg := entity.NewMatchRequest(req)
	return b.instantiateAndStartGame(ctx, accUser, req.User.UserId, req.GameRequest,
		sg, BotRequestID, "")
}

func (b *Bus) sendSoughtGameDeletion(ctx context.Context, sg *entity.SoughtGame) error {
	if sg.Type() == entity.TypeSeek {
		return b.broadcastSeekDeletion(sg.ID())
	} else if sg.Type() == entity.TypeMatch {
		return b.sendMatchCancellation(sg.MatchRequest.ReceivingUser.UserId, sg.Seeker(), sg.ID())
	}
	return errors.New("no-sg-type-match")
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
		return b.sendSoughtGameDeletion(ctx, sg)
	}

	accUser, err := b.userStore.GetByUUID(ctx, userID)
	if err != nil {
		return err
	}

	reqUser, err := b.userStore.GetByUUID(ctx, requester)
	if err != nil {
		return err
	}

	// Check if requesting user is blocking the accepting user.
	blockedUsers, err := b.userStore.GetBlocks(ctx, reqUser.ID)
	if err != nil {
		return err
	}
	for _, bu := range blockedUsers {
		if bu.UUID == accUser.UUID {
			evt := entity.WrapEvent(&pb.ErrorMessage{
				Message: reqUser.Username + " is not available for seek requests",
			}, pb.MessageType_ERROR_MESSAGE)
			b.pubToUser(accUser.UUID, evt, "")
			return nil
		}
	}

	// Otherwise create a game
	// If the ACCEPTOR of the seek has a seek request open, we must cancel it.
	err = b.deleteSoughtForUser(ctx, userID)
	if err != nil {
		return err
	}

	return b.instantiateAndStartGame(ctx, accUser, requester, gameReq, sg, evt.RequestId, connID)
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

func (b *Bus) sendMatchCancellation(userID, seekerID, requestID string) error {
	toSend := entity.WrapEvent(&pb.MatchRequestCancellation{RequestId: requestID},
		pb.MessageType_MATCH_REQUEST_CANCELLATION)
	err := b.pubToUser(userID, toSend, "")
	if err != nil {
		return err
	}
	return b.pubToUser(seekerID, toSend, "")
}

func (b *Bus) broadcastGameCreation(g *entity.Game, acceptor, requester *entity.User) error {
	timefmt, variant, err := entity.VariantFromGameReq(g.GameReq)
	if err != nil {
		return err
	}
	ratingKey := entity.ToVariantKey(g.GameReq.Lexicon, variant, timefmt)
	players := []*gs.PlayerInfo{
		{Rating: acceptor.GetRelevantRating(ratingKey),
			Nickname: acceptor.Username},
		{Rating: requester.GetRelevantRating(ratingKey),
			Nickname: requester.Username},
	}

	toSend := entity.WrapEvent(&gs.GameInfoResponse{Players: players,
		GameRequest: g.GameReq, GameId: g.GameID()},
		pb.MessageType_ONGOING_GAME_EVENT)
	data, err := toSend.Serialize()
	if err != nil {
		return err
	}

	err = b.natsconn.Publish("lobby.newLiveGame", data)
	if err != nil {
		return err
	}

	// Also publish to tournament channel if this is a tournament game.
	if g.TournamentData != nil && g.TournamentData.Id != "" {
		channelName := "tournament." + g.TournamentData.Id + ".newLiveGame"
		err = b.natsconn.Publish(channelName, data)
		if err != nil {
			return err
		}
	}
	return nil

}

func (b *Bus) deleteSoughtForUser(ctx context.Context, userID string) error {
	req, err := b.soughtGameStore.DeleteForUser(ctx, userID)
	if err != nil {
		return err
	}
	if req == nil {
		return nil
	}
	log.Debug().Interface("req", req).Str("userID", userID).Msg("deleting-sought")
	return b.sendSoughtGameDeletion(ctx, req)
}

func (b *Bus) deleteSoughtForConnID(ctx context.Context, connID string) error {
	req, err := b.soughtGameStore.DeleteForConnID(ctx, connID)
	if err != nil {
		return err
	}
	if req == nil {
		return nil
	}
	log.Debug().Interface("req", req).Str("connID", connID).Msg("deleting-sought-for-connid")
	return b.sendSoughtGameDeletion(ctx, req)
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

func (b *Bus) openMatches(ctx context.Context, receiverID string, tourneyID string) (*entity.EventWrapper, error) {
	sgs, err := b.soughtGameStore.ListOpenMatches(ctx, receiverID, tourneyID)
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
