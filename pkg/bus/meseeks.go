package bus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/entitlements"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/integrations"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"

	"github.com/domino14/macondo/game"
)

var BootedReceiversMax = 5

func (b *Bus) seekRequest(ctx context.Context, auth, userID, connID string,
	data []byte) error {

	if auth == "anon" {
		// Require login for now (forever?)
		return errors.New("please log in to start a game")
	}

	err := b.errIfGamesDisabled(ctx)
	if err != nil {
		return err
	}

	req := &pb.SeekRequest{}
	err = proto.Unmarshal(data, req)
	if err != nil {
		return err
	}

	if len(req.BootedReceivers) > BootedReceiversMax {
		return fmt.Errorf("cannot boot more than %d players", BootedReceiversMax)
	}

	if req.ReceivingUser != nil && receiverIsBooted(req.BootedReceivers, req.ReceivingUser.UserId) {
		return fmt.Errorf("player %s has been booted", req.ReceivingUser.DisplayName)
	}

	gameRequest, lastOpp, err := b.gameRequestForSeek(ctx, req, userID)
	if err != nil {
		return err
	}

	err = actionExists(ctx, b.stores.UserStore, userID, gameRequest)
	if err != nil {
		return err
	}

	exists, err := b.stores.SoughtGameStore.ExistsForUser(ctx, gameRequest.RequestId)
	if err != nil {
		return err
	}

	if exists {
		log.Debug().Str("user", userID).Msg("updating-seek-request")
		err = b.updateSeekRequest(ctx, auth, userID, connID, req)
	} else {
		log.Debug().Str("user", userID).Msg("new-seek-request")
		err = b.newSeekRequest(ctx, auth, userID, connID, req, gameRequest, lastOpp)
	}

	return err
}

func ratingKey(gameRequest *pb.GameRequest) (entity.VariantKey, error) {
	timefmt, variant, err := entity.VariantFromGameReq(gameRequest)
	if err != nil {
		return "", err
	}
	return entity.ToVariantKey(gameRequest.Lexicon, variant, timefmt), nil
}

func (b *Bus) newSeekRequest(ctx context.Context, auth, userID, connID string,
	req *pb.SeekRequest, gameRequest *pb.GameRequest, lastOpp string) error {

	if gameRequest == nil {
		return errors.New("no game request was found")
	}
	if req.GameRequest == nil {
		req.GameRequest = gameRequest
	}

	// Note that the seek request should not come with a requesting user;
	// instead this is in the topic/subject. It is HERE in the API server that
	// we set the requesting user's display name, rating, etc.
	reqUser := &pb.MatchUser{}
	reqUser.IsAnonymous = auth == "anon" // this is never true here anymore, see check above
	reqUser.UserId = userID
	req.User = reqUser

	err := entity.ValidateGameRequest(ctx, gameRequest)
	if err != nil {
		return err
	}

	// Look up user.
	ratingKey, err := ratingKey(gameRequest)
	if err != nil {
		return err
	}
	req.RatingKey = string(ratingKey)

	u, err := b.stores.UserStore.GetByUUID(ctx, reqUser.UserId)
	if err != nil {
		return err
	}
	reqUser.RelevantRating = u.GetRelevantRating(ratingKey)
	reqUser.DisplayName = u.Username

	req.SeekerConnectionId = connID

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

	if req.ReceiverIsPermanent && req.ReceivingUser == nil {
		return errors.New("receiver is marked as permanent but is nil")
	}

	if req.ReceivingUser != nil {
		req.ReceivingUser.DisplayName = strings.TrimSpace(req.ReceivingUser.DisplayName)
		receiver, err := b.stores.UserStore.Get(ctx, req.ReceivingUser.DisplayName)
		if err != nil {
			// No such user, most likely.
			return err
		}
		requester, err := b.stores.UserStore.GetByUUID(ctx, reqUser.UserId)
		if err != nil {
			return err
		}
		block, err := checkForBlock(ctx, b, connID, requester, receiver)
		if err != nil {
			return err
		} else if block == 0 || block == 1 {
			return nil
		}
		req.ReceivingUser.UserId = receiver.UUID
	}

	sg, err := gameplay.NewSoughtGame(ctx, b.stores.SoughtGameStore, req)
	if err != nil {
		return err
	}
	log.Debug().Interface("sought-game", sg).Msg("new seek request")

	return publishSeek(ctx, b, sg, userID, connID, pb.SeekState_ABSENT)
}

func (b *Bus) updateSeekRequest(ctx context.Context, auth, userID, connID string,
	newReq *pb.SeekRequest) error {
	// If we are here the seek exists and the game request is not nil
	if newReq.GameRequest == nil {
		return errors.New("nil game request for seek to update")
	}
	reqId := newReq.GameRequest.RequestId
	sg, err := b.stores.SoughtGameStore.Get(ctx, reqId)
	if err != nil {
		return err
	}

	oldReceiverState := sg.SeekRequest.ReceiverState

	seekerUserID, err := sg.SeekerUserID()
	if err != nil {
		return err
	}

	receiverUserID, err := sg.ReceiverUserID()
	if err != nil {
		return err
	}

	receiverDisplayName, err := sg.ReceiverDisplayName()
	if err != nil {
		return err
	}

	sg.SeekRequest.ReceivingUser.DisplayName = strings.TrimSpace(receiverDisplayName)
	receiver, err := b.stores.UserStore.Get(ctx, receiverDisplayName)
	if err != nil {
		// No such user, most likely.
		return err
	}
	requester, err := b.stores.UserStore.GetByUUID(ctx, seekerUserID)
	if err != nil {
		return err
	}

	block, err := checkForBlock(ctx, b, connID, receiver, requester)
	if err != nil {
		return err
	} else if block == 0 || block == 1 {
		return nil
	}

	if userID == seekerUserID {
		sg.SeekRequest.UserState = newReq.UserState
		sg.SeekRequest.BootedReceivers = newReq.BootedReceivers
	} else if receiverUserID == "" || userID == receiverUserID {
		sg.SeekRequest.ReceiverState = newReq.ReceiverState
	} else {
		return errors.New("you are not a seeker or receiver for this seek")
	}

	return publishSeek(ctx, b, sg, userID, connID, oldReceiverState)
}

func publishSeek(ctx context.Context, b *Bus, sg *entity.SoughtGame, userID string, connID string, oldReceiverState pb.SeekState) error {
	// If both players are ready, start the game
	// If the receiver is absent, send a new seek to everyone
	// Otherwise, only send it to the two players
	if sg.SeekRequest.ReceiverState == pb.SeekState_READY &&
		sg.SeekRequest.UserState == pb.SeekState_READY {
		reqId, err := sg.ID()
		if err != nil {
			return err
		}
		return b.gameAccepted(ctx, &pb.SoughtGameProcessEvent{RequestId: reqId}, userID, connID)
	} else if sg.SeekRequest.ReceiverIsPermanent || sg.SeekRequest.ReceiverState != pb.SeekState_ABSENT {
		log.Debug().Interface("sought-game", sg).Msg("processing seek as match")
		// Update the current seek request
		seekerUserID, err := sg.SeekerUserID()
		if err != nil {
			return err
		}

		receiverUserID, err := sg.ReceiverUserID()
		if err != nil {
			return err
		}

		seekerConnID, err := sg.SeekerConnID()
		if err != nil {
			return err
		}

		receiverConnID, err := sg.ReceiverConnID()
		if err != nil {
			return err
		}

		publishSeekToPlayers(b, sg, seekerUserID, seekerConnID, receiverUserID, receiverConnID)
	} else {
		log.Debug().Interface("sought-game", sg).Msg("publishing to lobby")
		// If the receiver is absent or not permanent and this is not a match request, resend or send to everyone to let them
		// know that this seek is open. If the old receiver state was absent
		// and the new receiver state is not absent, this seek is no longer
		// open and everyone must be notified.

		// Check if this is an "only followed players" seek
		if sg.SeekRequest.OnlyFollowedPlayers {
			// Get the seeker user ID for the topic
			seekerUserID, err := sg.SeekerUserID()
			if err != nil {
				return err
			}

			// Publish to a special topic for efficient socketsrv fan-out
			log.Debug().Str("seekerUserID", seekerUserID).Msg("publishing seek to followed users only")
			err = publishSeekToFollowed(b, sg, seekerUserID)
			if err != nil {
				return err
			}
		} else {
			publishSeekToLobby(b, sg)
		}
	}
	return nil
}

func publishSeekToLobby(b *Bus, sg *entity.SoughtGame) error {
	evt := entity.WrapEvent(sg.SeekRequest, pb.MessageType_SEEK_REQUEST)
	outdata, err := evt.Serialize()
	if err != nil {
		return err
	}
	log.Debug().Interface("evt", evt).Msg("republishing seek request that was abandoned to lobby topic")
	b.natsconn.Publish("lobby.seekRequest", outdata)
	return nil
}

func publishSeekToFollowed(b *Bus, sg *entity.SoughtGame, seekerUserID string) error {
	// Publish a single NATS message to a special topic.
	// The socketsrv will handle fan-out to all users that the seeker follows.
	// This is much more efficient than sending individual messages to each followed user.
	data, err := proto.Marshal(sg.SeekRequest)
	if err != nil {
		return err
	}
	topic := "seek.followed." + seekerUserID
	log.Debug().Str("topic", topic).Msg("publishing seek to followed users topic")
	return b.natsconn.Publish(topic, data)
}

func publishSeekToPlayers(b *Bus, sg *entity.SoughtGame, seekerID, seekerConnID, receiverID, receiverConnID string) error {
	evt := entity.WrapEvent(sg.SeekRequest, pb.MessageType_SEEK_REQUEST)
	b.pubToConnectionID(seekerConnID, seekerID, evt)

	if receiverConnID != "" {
		log.Debug().Interface("evt", evt).Str("receiver-conn-id", receiverConnID).Msg("publishing to receiver on connID")
		b.pubToConnectionID(receiverConnID, receiverID, evt)
	} else {
		log.Debug().Interface("evt", evt).Str("receiver", receiverID).Msg("publishing to receiver on username")
		b.pubToUser(receiverID, evt, "")
	}
	return nil
}

func receiverIsBooted(bootedPlayers []string, receiverID string) bool {
	for _, bootedPlayer := range bootedPlayers {
		if bootedPlayer == receiverID {
			return true
		}
	}
	return false
}

func (b *Bus) gameRequestForSeek(ctx context.Context, req *pb.SeekRequest,
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
		gm, err := b.stores.GameStore.GetMetadata(ctx, gameID)
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
		// NewSoughtGame in sought_game.go
		// if it is not set here. We copy the whole game request which includes
		// the OriginalRequestId
		gameRequest = proto.Clone(gm.GameRequest).(*pb.GameRequest)

		// This will get overwritten later:
		gameRequest.RequestId = ""
	}
	return gameRequest, lastOpp, nil
}

func validateCommonWordLexicon(lexicon string, botType macondopb.BotRequest_BotCode) error {
	// validate that the lexicon is compatible with the bot type, if the user has
	// selected a common-word bot
	if strings.HasPrefix(lexicon, "NWL") ||
		strings.HasPrefix(lexicon, "CSW") ||
		strings.HasPrefix(lexicon, "RD") {

		// Ironically, the CEL lexicon itself is not compatible with Common Word bots,
		// because Common Word bots work on a word list that is a subset of the actual
		// list. Just use the regular probability bots if the user is using the
		// CEL lexicon. Similarly for the CGL lexicon (if we ever support selecting that
		// directly in liwords).
		return nil
	}
	if botType == macondopb.BotRequest_LEVEL1_COMMON_WORD_BOT ||
		botType == macondopb.BotRequest_LEVEL2_COMMON_WORD_BOT ||
		botType == macondopb.BotRequest_LEVEL3_COMMON_WORD_BOT ||
		botType == macondopb.BotRequest_LEVEL4_COMMON_WORD_BOT {

		return errors.New("Common word bots are not compatible with this lexicon")
	}

	return nil
}

func (b *Bus) newBotGame(ctx context.Context, req *pb.SeekRequest, botUserID string) error {
	// NewBotGame creates and starts a new game against a bot!
	var err error
	var accUser *entity.User

	if req.GameRequest == nil || req.GameRequest.Rules == nil {
		return errors.New("game request or rules were nil")
	}

	if req.GameRequest.Rules.VariantName == string(game.VarWordSmog) &&
		req.GameRequest.BotType != macondopb.BotRequest_HASTY_BOT {
		return errors.New("only HastyBot can play WordSmog at this time")
	}

	if botUserID == "" {
		accUser, err = b.stores.UserStore.GetBot(ctx, req.GameRequest.BotType)
	} else {
		accUser, err = b.stores.UserStore.GetByUUID(ctx, botUserID)
	}
	if err != nil {
		return err
	}

	err = validateCommonWordLexicon(req.GameRequest.Lexicon, req.GameRequest.BotType)
	if err != nil {
		return err
	}

	if req.GameRequest.BotType == macondopb.BotRequest_SIMMING_BOT {
		if req.GameRequest.Rules.VariantName == string(game.VarClassicSuper) {
			return errors.New("that variant is not supported for BestBot yet")
		}
		if req.GameRequest.InitialTimeSeconds < 180 {
			return errors.New("BestBot needs more time than that to play at its best.")
		}

		reqUser, err := b.stores.UserStore.GetByUUID(ctx, req.User.UserId)
		if err != nil {
			return err
		}
		// Some special users can just bypass the check.
		bypassSubCheck, err := rbac.HasPermission(ctx, b.stores.Queries, reqUser.ID, rbac.CanPlayEliteBot)
		if err != nil {
			return err
		}
		if !bypassSubCheck {
			// Determine user tier
			tierData, err := integrations.DetermineUserTier(ctx, req.User.UserId, b.stores.Queries)
			if err != nil {
				if errors.Is(err, integrations.ErrNotPaidTier) || errors.Is(err, integrations.ErrNotSubscribed) {
					return errors.New("You don't currently appear to be subscribed to a paid Patreon tier. Please sign up at https://woogles.io/donate to have access to BestBot.")
				}
				var papierr *integrations.PatreonAPIError
				if errors.As(err, &papierr) {
					log.Err(papierr).Msg("patreon-api-error")
					return errors.New("There was an error with your connection to the Patreon API. Please go to your Settings -> Integrations and reconnect with Patreon.")
				}

				return err
			}
			log.Info().Interface("tierData", tierData).Msg("tier-for-bestbot-game")
			if tierData == nil {
				return errors.New("You don't currently appear to have a Patreon membership. Please sign up at https://woogles.io/donate to have access to BestBot.")
			}

			entitled, err := entitlements.EntitledToBestBot(ctx, b.stores.Queries, tierData, reqUser.ID, time.Now())
			if err != nil {
				return err
			}
			if !entitled {
				return errors.New("It appears you have already played your allotment of BestBot games for this period. Please upgrade your membership or wait a few days.")
			}
		}

	}

	sg := entity.NewSoughtGame(req)

	return b.instantiateAndStartGame(ctx, accUser, req.User.UserId, req.GameRequest,
		sg, BotRequestID, "")
}

func (b *Bus) sendSoughtGameDeletion(ctx context.Context, sg *entity.SoughtGame) error {
	return b.broadcastSeekDeletion(sg)
}

func (b *Bus) gameAccepted(ctx context.Context, evt *pb.SoughtGameProcessEvent,
	userID, connID string) error {
	sg, err := b.stores.SoughtGameStore.Get(ctx, evt.RequestId)
	if err != nil {
		return err
	}

	requester := sg.SeekRequest.User.UserId
	gameReq := sg.SeekRequest.GameRequest

	err = actionExists(ctx, b.stores.UserStore, userID, gameReq)
	if err != nil {
		return err
	}

	if requester == userID {
		log.Info().Str("sender", requester).Msg("canceling seek")
		err := gameplay.CancelSoughtGame(ctx, b.stores.SoughtGameStore, evt.RequestId)
		if err != nil {
			return err
		}
		// broadcast a seek deletion.
		return b.sendSoughtGameDeletion(ctx, sg)
	}

	accUser, err := b.stores.UserStore.GetByUUID(ctx, userID)
	if err != nil {
		return err
	}
	if !sg.SeekRequest.ReceiverIsPermanent {
		// If the receiver is not permanent, then we need to check that the
		// requester's rating is within the range of the receiver's minimum and
		// maximum ratings.

		ratingKey, err := ratingKey(gameReq)
		if err != nil {
			return err
		}

		reqUser, err := b.stores.UserStore.GetByUUID(ctx, requester)
		if err != nil {
			return err
		}

		accRating, err := accUser.GetRating(ratingKey)
		if err != nil {
			return err
		}

		reqRating, err := reqUser.GetRating(ratingKey)
		if err != nil {
			return err
		}

		log.Debug().Int32("minRatRange", sg.SeekRequest.MinimumRatingRange).
			Int32("maxRatRange", sg.SeekRequest.MaximumRatingRange).
			Float64("accRating", accRating.Rating).
			Float64("reqRating", reqRating.Rating).
			Msg("ratingsinfo")
		// assume min rating range is negative
		if accRating.Rating < reqRating.Rating+float64(sg.SeekRequest.MinimumRatingRange) ||
			accRating.Rating > reqRating.Rating+float64(sg.SeekRequest.MaximumRatingRange) {
			return errors.New("your rating is not within the requested range")
		}
	}

	// If seek requires established rating, check acceptor has one
	if sg.SeekRequest.RequireEstablishedRating {
		ratingKey, err := ratingKey(gameReq)
		if err != nil {
			return err
		}

		accRating, err := accUser.GetRating(ratingKey)
		if err != nil {
			return err
		}

		if accRating.RatingDeviation > entity.RatingDeviationConfidence {
			return errors.New("this seek requires an established rating")
		}
	}

	// If seek is only for followed players, verify acceptor is followed
	if sg.SeekRequest.OnlyFollowedPlayers {
		reqUser, err := b.stores.UserStore.GetByUUID(ctx, requester)
		if err != nil {
			return err
		}

		isFollowed, err := b.stores.UserStore.IsFollowing(ctx, reqUser.ID, accUser.ID)
		if err != nil {
			return err
		}

		if !isFollowed {
			return errors.New("this seek is only available to players followed by the seeker")
		}
	}

	reqUser, err := b.stores.UserStore.GetByUUID(ctx, requester)
	if err != nil {
		return err
	}

	block, err := checkForBlock(ctx, b, connID, accUser, reqUser)
	if err != nil {
		return err
	} else if block == 0 || block == 1 {
		return nil
	}

	// Otherwise create a game
	// If the ACCEPTOR of the seek has a seek request open, we must cancel it.
	err = b.deleteSoughtForUser(ctx, userID)
	if err != nil {
		return err
	}

	return b.instantiateAndStartGame(ctx, accUser, requester, gameReq, sg, evt.RequestId, connID)
}

// Return 0 if reqUser blocks accUser, 1 if accUser blocks reqUser, and -1 if neither blocks
// the other. Note, if they both block each other it will return 0.
func checkForBlock(ctx context.Context, b *Bus, connID string, accUser *entity.User, reqUser *entity.User) (int, error) {
	block, err := b.blockExists(ctx, reqUser, accUser)
	if err != nil {
		return -1, err
	}
	if block == 0 {
		// requesting user is blocking the accepting user.
		evt := entity.WrapEvent(&pb.ErrorMessage{
			Message: "You are not able to accept " + reqUser.Username + "'s requests.",
		}, pb.MessageType_ERROR_MESSAGE)
		b.pubToConnectionID(connID, accUser.UUID, evt)
		return 0, nil
	} else if block == 1 {
		// accepting user is blocking requesting user. They should not be able to
		// see their requests but maybe they didn't refresh after blocking.
		evt := entity.WrapEvent(&pb.ErrorMessage{
			Message: reqUser.Username + " is on your block list, thus you cannot play against them.",
		}, pb.MessageType_ERROR_MESSAGE)
		b.pubToConnectionID(connID, accUser.UUID, evt)
		return 1, nil
	}
	return -1, nil
}

func (b *Bus) seekDeclined(ctx context.Context, evt *pb.DeclineSeekRequest, userID string) error {
	// the sending user declined the match request. Send this declination
	// to the matcher and delete the request.
	sg, err := b.stores.SoughtGameStore.Get(ctx, evt.RequestId)
	if err != nil {
		return err
	}

	if sg.SeekRequest.ReceivingUser.UserId != userID {
		return errors.New("request userID does not match")
	}

	err = gameplay.CancelSoughtGame(ctx, b.stores.SoughtGameStore, evt.RequestId)
	if err != nil {
		return err
	}

	requester := sg.SeekRequest.User.UserId
	decliner := userID

	wrapped := entity.WrapEvent(evt, pb.MessageType_DECLINE_SEEK_REQUEST)

	// Publish decline to requester
	err = b.pubToUser(requester, wrapped, "")
	if err != nil {
		return err
	}
	wrapped = entity.WrapEvent(&pb.SoughtGameProcessEvent{RequestId: evt.RequestId},
		pb.MessageType_SOUGHT_GAME_PROCESS_EVENT)
	return b.pubToUser(decliner, wrapped, "")
}

func (b *Bus) broadcastSeekDeletion(sg *entity.SoughtGame) error {
	id, err := sg.ID()
	if err != nil {
		return err
	}
	toSend := entity.WrapEvent(&pb.SoughtGameProcessEvent{RequestId: id},
		pb.MessageType_SOUGHT_GAME_PROCESS_EVENT)
	data, err := toSend.Serialize()
	if err != nil {
		return err
	}
	p, err := sg.ReceiverIsPermanent()
	if err != nil {
		return err
	}
	if !p {
		return b.natsconn.Publish("lobby.soughtGameProcess", data)
	}
	if sg.SeekRequest == nil {
		return errors.New("seek-request-nil")
	}
	if sg.SeekRequest.ReceivingUser == nil {
		return errors.New("seek-request-receiving-user-nil")
	}
	if sg.SeekRequest.User == nil {
		return errors.New("seek-request-user-nil")
	}
	// If it's a permanent receiver, we need to publish to both users only.
	err = b.pubToUser(sg.SeekRequest.ReceivingUser.UserId, toSend, "")
	if err != nil {
		return err
	}
	return b.pubToUser(sg.SeekRequest.User.UserId, toSend, "")

}

func (b *Bus) sendReceiverAbsent(ctx context.Context, req *entity.SoughtGame) error {
	isMatch, err := req.ReceiverIsPermanent()
	if err != nil {
		return err
	}

	if isMatch {
		seekerConnID, err := req.SeekerConnID()
		if err != nil {
			return err
		}

		seekerID, err := req.SeekerUserID()
		if err != nil {
			return err
		}
		evt := entity.WrapEvent(req.SeekRequest, pb.MessageType_SEEK_REQUEST)
		return b.pubToConnectionID(seekerConnID, seekerID, evt)
	} else {
		evt := entity.WrapEvent(req.SeekRequest, pb.MessageType_SEEK_REQUEST)
		outdata, err := evt.Serialize()
		if err != nil {
			return err
		}
		return b.natsconn.Publish("lobby.soughtGameProcess", outdata)
	}
}

func (b *Bus) broadcastGameCreation(g *entity.Game, acceptor, requester *entity.User) error {
	timefmt, variant, err := entity.VariantFromGameReq(g.GameReq.GameRequest)
	if err != nil {
		return err
	}
	ratingKey := entity.ToVariantKey(g.GameReq.Lexicon, variant, timefmt)
	players := []*pb.PlayerInfo{
		{Rating: acceptor.GetRelevantRating(ratingKey),
			UserId:   acceptor.UUID,
			Nickname: acceptor.Username},
		{Rating: requester.GetRelevantRating(ratingKey),
			UserId:   requester.UUID,
			Nickname: requester.Username},
	}

	gameInfo := &pb.GameInfoResponse{Players: players,
		GameRequest: g.GameReq.GameRequest, GameId: g.GameID()}

	if g.TournamentData != nil {
		gameInfo.TournamentDivision = g.TournamentData.Division
		gameInfo.TournamentId = g.TournamentData.Id
		gameInfo.TournamentRound = int32(g.TournamentData.Round)
		gameInfo.TournamentGameIndex = int32(g.TournamentData.GameIndex)
	}

	toSend := entity.WrapEvent(gameInfo, pb.MessageType_ONGOING_GAME_EVENT)
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
	req, err := b.stores.SoughtGameStore.DeleteForUser(ctx, userID)
	if err != nil {
		return err
	}
	if req == nil {
		return nil
	}
	log.Debug().Interface("req", req).Str("userID", userID).Msg("deleting-sought")
	err = b.sendSoughtGameDeletion(ctx, req)
	if err != nil {
		return err
	}

	req, err = b.stores.SoughtGameStore.UpdateForReceiver(ctx, userID)
	if err != nil {
		return err
	}
	if req == nil {
		return nil
	}
	log.Debug().Interface("req", req).Str("userID", userID).Msg("deleting-sought")
	return b.sendReceiverAbsent(ctx, req)
}

func (b *Bus) deleteSoughtForConnID(ctx context.Context, connID string) error {
	req, err := b.stores.SoughtGameStore.DeleteForSeekerConnID(ctx, connID)
	if err != nil {
		return err
	}
	if req == nil {
		return nil
	}
	log.Debug().Interface("req", req).Str("connID", connID).Msg("deleting-sought-for-connid")
	err = b.sendSoughtGameDeletion(ctx, req)
	if err != nil {
		return err
	}

	req, err = b.stores.SoughtGameStore.UpdateForReceiverConnID(ctx, connID)
	if err != nil {
		return err
	}
	if req == nil {
		return nil
	}
	log.Debug().Interface("req", req).Str("connID", connID).Msg("updating-receiver-for-connid")
	return b.sendReceiverAbsent(ctx, req)
}

// shouldIncludeSeek filters a seek based on blocks, rating range, established rating, and followed players requirements.
// Returns true if the seek should be shown to the receiver, false if it should be filtered out.
func (b *Bus) shouldIncludeSeek(ctx context.Context, sg *entity.SoughtGame, receiver *entity.User) bool {
	if receiver == nil {
		// No receiver to filter for, include the seek
		return true
	}

	// Get the seeker
	seeker, err := b.stores.UserStore.GetByUUID(ctx, sg.SeekRequest.User.UserId)
	if err != nil {
		return false
	}

	// Always show seekers their own seeks (don't apply filters to your own seeks)
	if seeker.UUID == receiver.UUID {
		return true
	}

	// Always show match requests where you're the receiver (don't apply filters to direct matches)
	if sg.SeekRequest.ReceivingUser != nil && sg.SeekRequest.ReceivingUser.UserId == receiver.UUID {
		return true
	}

	// Check for blocks
	block, err := b.blockExists(ctx, seeker, receiver)
	if err != nil {
		return false
	}
	if block == 0 {
		// Receiver is blocked by seeker
		return false
	}

	// Check rating range (for open seeks only, not match requests)
	if !sg.SeekRequest.ReceiverIsPermanent {
		ratingKey, err := ratingKey(sg.SeekRequest.GameRequest)
		if err != nil {
			return false
		}

		seekerRating, err := seeker.GetRating(ratingKey)
		if err != nil {
			return false
		}

		receiverRating, err := receiver.GetRating(ratingKey)
		if err != nil {
			return false
		}

		// Check if receiver's rating is within the seeker's specified range
		// MinimumRatingRange should be negative
		minRating := seekerRating.Rating + float64(sg.SeekRequest.MinimumRatingRange)
		maxRating := seekerRating.Rating + float64(sg.SeekRequest.MaximumRatingRange)
		if receiverRating.Rating < minRating || receiverRating.Rating > maxRating {
			return false
		}
	}

	// Check if seek requires established rating
	if sg.SeekRequest.RequireEstablishedRating {
		ratingKey, err := ratingKey(sg.SeekRequest.GameRequest)
		if err != nil {
			return false
		}
		rating, err := receiver.GetRating(ratingKey)
		if err != nil || rating.RatingDeviation > entity.RatingDeviationConfidence {
			return false
		}
	}

	// Check if seek is only for followed players
	if sg.SeekRequest.OnlyFollowedPlayers {
		isFollowed, err := b.stores.UserStore.IsFollowing(ctx, seeker.ID, receiver.ID)
		if err != nil {
			return false
		}
		if !isFollowed {
			return false
		}
	}

	return true
}

func (b *Bus) openSeeks(ctx context.Context, receiverID string, tourneyID string) (*entity.EventWrapper, error) {
	sgs, err := b.stores.SoughtGameStore.ListOpenSeeks(ctx, receiverID, tourneyID)
	if err != nil {
		return nil, err
	}
	if len(sgs) == 0 {
		return nil, nil
	}

	var receiver *entity.User
	if !userIsAnon(receiverID) {
		receiver, err = b.stores.UserStore.GetByUUID(ctx, receiverID)
		if err != nil {
			return nil, err
		}
	}

	log.Debug().Str("receiver", receiverID).Interface("open-matches", sgs).Msg("open-matches")
	pbobj := &pb.SeekRequests{Requests: []*pb.SeekRequest{}}
	for _, sg := range sgs {
		if b.shouldIncludeSeek(ctx, sg, receiver) {
			pbobj.Requests = append(pbobj.Requests, sg.SeekRequest)
		}
	}
	evt := entity.WrapEvent(pbobj, pb.MessageType_SEEK_REQUESTS)
	return evt, nil
}

func actionExists(ctx context.Context, us user.Store, userID string, req *pb.GameRequest) error {

	_, err := mod.ActionExists(ctx, us, userID, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT, ms.ModActionType_SUSPEND_GAMES})
	if err != nil {
		return err
	}

	if req.RatingMode == pb.RatingMode_RATED {
		_, err = mod.ActionExists(ctx, us, userID, false, []ms.ModActionType{ms.ModActionType_SUSPEND_RATED_GAMES})
		if err != nil {
			return err
		}
	}

	return nil
}
