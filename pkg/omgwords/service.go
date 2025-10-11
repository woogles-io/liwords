package omgwords

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/macondo/gcgio"
	"github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/word-golib/tilemapping"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/cwgame"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/entity/utilities"
	"github.com/woogles-io/liwords/pkg/omgwords/stores"
	"github.com/woogles-io/liwords/pkg/user"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
	pb "github.com/woogles-io/liwords/rpc/api/proto/omgwords_service"
)

type OMGWordsService struct {
	userStore     user.Store
	cfg           *config.Config
	gameStore     *stores.GameDocumentStore
	metadataStore *stores.DBStore
	gameEventChan chan *entity.EventWrapper
}

func NewOMGWordsService(u user.Store, cfg *config.Config, gs *stores.GameDocumentStore,
	ms *stores.DBStore) *OMGWordsService {
	return &OMGWordsService{
		userStore:     u,
		cfg:           cfg,
		gameStore:     gs,
		metadataStore: ms}
}

func AnnotatedChannelName(gameID string) string {
	return "anno" + gameID
}

const GamesLimit = 50

func (gs *OMGWordsService) failIfSessionDoesntOwn(ctx context.Context, gameID string) error {
	if gameID == "" {
		return apiserver.InvalidArg("game ID must be provided")
	}
	u, err := apiserver.AuthUser(ctx, gs.userStore)
	if err != nil {
		return err
	}
	owns, err := gs.metadataStore.GameOwnedBy(ctx, gameID, u.UUID)
	if err != nil {
		return apiserver.InvalidArg(err.Error())
	}
	if !owns {
		return apiserver.InvalidArg("user does not own this game")
	}
	return nil
}

func (gs *OMGWordsService) SetEventChannel(c chan *entity.EventWrapper) {
	gs.gameEventChan = c
}

func (gs *OMGWordsService) createGDoc(ctx context.Context, u *entity.User, req *pb.CreateBroadcastGameRequest) (*ipc.GameDocument, error) {
	players := req.PlayersInfo
	if len(players) != 2 {
		return nil, apiserver.InvalidArg("need two players")
	}
	if players[0].Nickname == players[1].Nickname {
		return nil, apiserver.InvalidArg("player nicknames must be unique")
	}
	if players[0].Nickname == "" || players[1].Nickname == "" {
		return nil, apiserver.InvalidArg("player nicknames must not be blank")
	}
	if !players[0].First || players[1].First {
		return nil, apiserver.InvalidArg("only first player must be marked as first")
	}

	if req.Rules == nil {
		return nil, apiserver.InvalidArg("no rules")
	}
	if req.Lexicon == "" {
		return nil, apiserver.InvalidArg("lexicon is empty")
	}
	unfinished, err := gs.metadataStore.OutstandingGames(ctx, u.UUID)
	if err != nil {
		return nil, err
	}
	log.Debug().Int("unfinished", len(unfinished)).Str("userID", u.UUID).Msg("unfinished-anno-games")
	if len(unfinished) > 0 {
		return nil, apiserver.InvalidArg("please finish or delete your unfinished games before starting a new one")
	}

	// We can just make the user ID the same as the nickname, as it
	// doesn't matter in this case
	mcplayers := []*ipc.GameDocument_MinimalPlayerInfo{
		{Nickname: players[0].Nickname, RealName: players[0].FullName, UserId: "internal-" + players[0].Nickname},
		{Nickname: players[1].Nickname, RealName: players[1].FullName, UserId: "internal-" + players[1].Nickname},
	}

	// Create an untimed game:
	cwgameRules := cwgame.NewBasicGameRules(
		req.Lexicon, req.Rules.BoardLayoutName, req.Rules.LetterDistributionName,
		req.ChallengeRule, cwgame.Variant(req.Rules.VariantName), []int{0, 0}, 0, 0, true,
	)

	g, err := cwgame.NewGame(gs.cfg.WGLConfig(), cwgameRules, mcplayers)
	if err != nil {
		return nil, err
	}
	// Overwrite the type (NewGame assumes this is a native game)
	g.Type = ipc.GameType_ANNOTATED

	qd := &entity.Quickdata{PlayerInfo: req.PlayersInfo}
	g.CreatedAt = timestamppb.Now()

	// Create a legacy game request. Sadly, we need this for now in order to
	// get the old game paths to work properly. In the future we should use
	// the GameDocument as the single source of truth for as many things as possible.
	greq := &ipc.GameRequest{
		Lexicon:            req.Lexicon,
		Rules:              req.Rules,
		InitialTimeSeconds: 0,
		IncrementSeconds:   0,
		ChallengeRule:      macondo.ChallengeRule(req.ChallengeRule),
		GameMode:           ipc.GameMode_REAL_TIME,
		RatingMode:         ipc.RatingMode_CASUAL,
		RequestId:          "dummy",
		MaxOvertimeMinutes: 0,
		OriginalRequestId:  "dummy",
	}

	if err = gs.metadataStore.CreateAnnotatedGame(ctx, u.UUID, g.Uid, true, qd, greq); err != nil {
		return nil, err
	}
	if err = gs.gameStore.SetDocument(ctx, g); err != nil {
		// If we can't add the document to the game store, delete
		// the annotated game we just created it. Not the best pattern, but
		// we have different data stores.
		if derr := gs.metadataStore.DeleteAnnotatedGame(ctx, g.Uid); derr != nil {
			log.Err(derr).Msg("deleting-annotated-game")
		}
		return nil, err
	}

	// We should also send a new game event on the channel.
	err = announceGameCreation(g, req.PlayersInfo, gs.gameEventChan)
	if err != nil {
		log.Err(err).Msg("broadcasting-game-creation")
	}
	return g, nil
}

func (gs *OMGWordsService) CreateBroadcastGame(ctx context.Context, req *connect.Request[pb.CreateBroadcastGameRequest],
) (*connect.Response[pb.CreateBroadcastGameResponse], error) {

	u, err := apiserver.AuthUser(ctx, gs.userStore)
	if err != nil {
		return nil, err
	}
	gdoc, err := gs.createGDoc(ctx, u, req.Msg)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.CreateBroadcastGameResponse{
		GameId: gdoc.Uid,
	}), nil
}

func (gs *OMGWordsService) SendGameEvent(ctx context.Context, req *connect.Request[pb.AnnotatedGameEvent]) (
	*connect.Response[pb.GameEventResponse], error) {
	if err := gs.failIfSessionDoesntOwn(ctx, req.Msg.Event.GameId); err != nil {
		return nil, err
	}
	if req.Msg.Event == nil {
		return nil, apiserver.InvalidArg("event is required")
	}

	justEnded, err := handleEvent(ctx, gs.cfg.WGLConfig(), req.Msg.UserId, req.Msg.Event, req.Msg.Amendment, req.Msg.EventNumber, gs.gameStore, gs.gameEventChan)
	if err != nil {
		return nil, err
	}
	// justEnded indicates if the handled event resulted in the game ending.
	// Since this is an annotated game, we must mark it as done.
	if justEnded {
		if err = gs.metadataStore.MarkAnnotatedGameDone(ctx, req.Msg.Event.GameId); err != nil {
			return nil, err
		}
	}

	return connect.NewResponse(&pb.GameEventResponse{}), nil
}

// UpdateGameDocument updates a game document for an annotated game. It doesn't
// really have meaning outside annotated games, as players should instead use an
// individual event update call.
func (gs *OMGWordsService) ReplaceGameDocument(ctx context.Context, req *connect.Request[pb.ReplaceDocumentRequest],
) (*connect.Response[pb.GameEventResponse], error) {

	if req.Msg.Document == nil {
		return nil, errors.New("nil game document")
	}
	gid := req.Msg.Document.Uid

	err := gs.failIfSessionDoesntOwn(ctx, gid)
	if err != nil {
		return nil, err
	}

	// Just willy-nilly update the thing. Kind of scary.
	err = gs.gameStore.UpdateDocument(ctx, &stores.MaybeLockedDocument{GameDocument: req.Msg.Document})
	if err != nil {
		return nil, err
	}
	// And send an event.
	evt := &ipc.GameDocumentEvent{
		Doc: proto.Clone(req.Msg.Document).(*ipc.GameDocument),
	}
	wrapped := entity.WrapEvent(evt, ipc.MessageType_OMGWORDS_GAMEDOCUMENT)
	wrapped.AddAudience(entity.AudChannel, AnnotatedChannelName(gid))
	gs.gameEventChan <- wrapped

	return connect.NewResponse(&pb.GameEventResponse{}), nil
}

// PatchGameDocument merges the requested game document into the existing one.
// For now, we just use this to update various metadata (like description, player names, etc).
// Disallow updating game structures directly until the front end can implement
// GameDocument on its own.
func (gs *OMGWordsService) PatchGameDocument(ctx context.Context, req *connect.Request[pb.PatchDocumentRequest],
) (*connect.Response[pb.GameEventResponse], error) {
	if req.Msg.Document == nil {
		return nil, apiserver.InvalidArg("nil game document")
	}
	gid := req.Msg.Document.Uid

	err := gs.failIfSessionDoesntOwn(ctx, gid)
	if err != nil {
		return nil, err
	}

	// For now don't allow direct patches to these fields. Maybe we can allow this
	// kind of stuff later.
	if len(req.Msg.Document.Events) > 0 || req.Msg.Document.Board != nil || req.Msg.Document.Bag != nil || req.Msg.Document.Racks != nil {
		return nil, errors.New("patch operation not supported at this time for these fields")
	}

	g, err := gs.gameStore.GetDocument(ctx, gid, true)
	if err != nil {
		return nil, err
	}

	err = MergeGameDocuments(g.GameDocument, req.Msg.Document)
	if err != nil {
		// Since we acquired a lock in the GetDocument call above,
		// we must unlock explicitly in any error case.
		uerr := gs.gameStore.UnlockDocument(ctx, g)
		if uerr != nil {
			log.Err(err).Msg("error-unlocking")
		}
		return nil, err
	}

	err = gs.gameStore.UpdateDocument(ctx, g)
	if err != nil {
		return nil, err
	}

	err = gs.metadataStore.UpdateAnnotatedGameQuickdata(
		ctx, gid, &entity.Quickdata{
			PlayerInfo: lo.Map(req.Msg.Document.Players,
				func(p *ipc.GameDocument_MinimalPlayerInfo, idx int) *ipc.PlayerInfo {
					return &ipc.PlayerInfo{
						UserId:   p.UserId,
						Nickname: p.Nickname,
						FullName: p.RealName,
					}
				}),
		})
	if err != nil {
		return nil, err
	}
	// And send an event.
	evt := &ipc.GameDocumentEvent{
		Doc: proto.Clone(g.GameDocument).(*ipc.GameDocument),
	}
	wrapped := entity.WrapEvent(evt, ipc.MessageType_OMGWORDS_GAMEDOCUMENT)
	wrapped.AddAudience(entity.AudChannel, AnnotatedChannelName(gid))
	gs.gameEventChan <- wrapped

	return connect.NewResponse(&pb.GameEventResponse{}), nil
}

func (gs *OMGWordsService) SetBroadcastGamePrivacy(ctx context.Context, req *connect.Request[pb.BroadcastGamePrivacy]) (
	*connect.Response[pb.GameEventResponse], error) {

	return nil, nil
}

func (gs *OMGWordsService) GetGamesForEditor(ctx context.Context, req *connect.Request[pb.GetGamesForEditorRequest]) (
	*connect.Response[pb.BroadcastGamesResponse], error) {

	if req.Msg.Limit > GamesLimit {
		return nil, apiserver.InvalidArg("too many games")
	}

	games, err := gs.metadataStore.GamesForEditor(ctx, req.Msg.UserId, req.Msg.Unfinished, int(req.Msg.Limit), int(req.Msg.Offset))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.BroadcastGamesResponse{
		Games: lo.Map(games, func(bg *stores.BroadcastGame, i int) *pb.BroadcastGamesResponse_BroadcastGame {
			return &pb.BroadcastGamesResponse_BroadcastGame{
				GameId:      bg.GameUUID,
				CreatorId:   bg.CreatorUUID,
				Private:     bg.Private,
				Finished:    bg.Finished,
				PlayersInfo: bg.Players,
				Lexicon:     bg.Lexicon,
				CreatedAt:   timestamppb.New(bg.Created),
			}
		}),
	}), nil
}

func (gs *OMGWordsService) GetRecentAnnotatedGames(ctx context.Context, req *connect.Request[pb.GetRecentAnnotatedGamesRequest]) (
	*connect.Response[pb.BroadcastGamesResponse], error) {

	if req.Msg.Limit > GamesLimit {
		return nil, apiserver.InvalidArg("too many games")
	}

	games, err := gs.metadataStore.GamesForEditor(ctx, "", req.Msg.Unfinished, int(req.Msg.Limit), int(req.Msg.Offset))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.BroadcastGamesResponse{
		Games: lo.Map(games, func(bg *stores.BroadcastGame, i int) *pb.BroadcastGamesResponse_BroadcastGame {
			return &pb.BroadcastGamesResponse_BroadcastGame{
				GameId:          bg.GameUUID,
				CreatorId:       bg.CreatorUUID,
				Private:         bg.Private,
				Finished:        bg.Finished,
				PlayersInfo:     bg.Players,
				Lexicon:         bg.Lexicon,
				CreatedAt:       timestamppb.New(bg.Created),
				CreatorUsername: bg.CreatorUsername,
			}
		}),
	}), nil
}

func (gs *OMGWordsService) GetMyUnfinishedGames(ctx context.Context, req *connect.Request[pb.GetMyUnfinishedGamesRequest]) (
	*connect.Response[pb.BroadcastGamesResponse], error) {

	u, err := apiserver.AuthUser(ctx, gs.userStore)
	if err != nil {
		return nil, err
	}

	unfinished, err := gs.metadataStore.OutstandingGames(ctx, u.UUID)
	if err != nil {
		return nil, err
	}
	log.Debug().Interface("unfinished", unfinished).Str("userID", u.UUID).Msg("unfinished-anno-games")

	games := lo.Map(unfinished, func(item *stores.BroadcastGame, idx int) *pb.BroadcastGamesResponse_BroadcastGame {
		return &pb.BroadcastGamesResponse_BroadcastGame{
			GameId:    item.GameUUID,
			CreatorId: item.CreatorUUID,
			Private:   item.Private,
			Finished:  item.Finished,
		}
	})

	return connect.NewResponse(&pb.BroadcastGamesResponse{
		Games: games,
	}), nil
}

func (gs *OMGWordsService) GetGameDocument(ctx context.Context, req *connect.Request[pb.GetGameDocumentRequest],
) (*connect.Response[ipc.GameDocument], error) {
	doc, err := gs.gameStore.GetDocument(ctx, req.Msg.GameId, false)
	if err != nil {
		if err == stores.ErrDoesNotExist {
			// Clean up the game if it is still in a store.
			derr := gs.metadataStore.DeleteAnnotatedGame(ctx, req.Msg.GameId)
			if derr != nil {
				return nil, derr
			}
			return nil, apiserver.InvalidArg("game does not exist")
		}
		return nil, err
	}
	if doc.Type == ipc.GameType_ANNOTATED {
		return connect.NewResponse(doc.GameDocument), nil
	}
	// Otherwise, we need to "censor" the game document by deleting information
	// this user should not have, if they're a player in this game.
	return nil, errors.New("not implemented for non-annotated games")
}

// SetRacks sets the player racks per user. It checks to make sure that the
// rack can be set before actually setting it.
func (gs *OMGWordsService) SetRacks(ctx context.Context, req *connect.Request[pb.SetRacksEvent],
) (*connect.Response[pb.GameEventResponse], error) {
	err := gs.failIfSessionDoesntOwn(ctx, req.Msg.GameId)
	if err != nil {
		return nil, err
	}

	g, err := gs.gameStore.GetDocument(ctx, req.Msg.GameId, true)
	if err != nil {
		return nil, err
	}
	// if g.PlayState == ipc.PlayState_GAME_OVER {
	// 	gs.gameStore.UnlockDocument(ctx, g)
	// 	return nil, twirp.NewError(twirp.InvalidArgument, "game is over")
	// }
	if len(req.Msg.Racks) != len(g.Players) {
		gs.gameStore.UnlockDocument(ctx, g)
		return nil, apiserver.InvalidArg("number of racks must match number of players")
	}
	if req.Msg.Amendment && len(g.Events)-1 < int(req.Msg.EventNumber) {
		gs.gameStore.UnlockDocument(ctx, g)
		return nil, apiserver.InvalidArg("tried to amend a rack for a non-existing event")
	}
	// Put back the current racks, if any.
	// Note that tiles.PutBack assumes the player racks are not adulterated in any way.
	// This should be the case, because only cwgame is responsible for dealing
	// racks.
	if req.Msg.Amendment {
		evt := g.Events[req.Msg.EventNumber]
		err = cwgame.EditOldRack(ctx, gs.cfg.WGLConfig(), g.GameDocument, req.Msg.EventNumber, req.Msg.Racks[evt.PlayerIndex])
		if err != nil {
			gs.gameStore.UnlockDocument(ctx, g)
			return nil, err
		}
	} else {

		err = cwgame.AssignRacks(g.GameDocument, req.Msg.Racks, cwgame.AssignEmptyIfUnambiguous)
		if err != nil {
			gs.gameStore.UnlockDocument(ctx, g)
			return nil, apiserver.InvalidArg(err.Error())
		}
	}

	err = gs.gameStore.UpdateDocument(ctx, g)
	if err != nil {
		return nil, err
	}

	// And send an event.
	evt := &ipc.GameDocumentEvent{
		Doc: proto.Clone(g.GameDocument).(*ipc.GameDocument),
	}
	wrapped := entity.WrapEvent(evt, ipc.MessageType_OMGWORDS_GAMEDOCUMENT)
	wrapped.AddAudience(entity.AudChannel, AnnotatedChannelName(g.Uid))
	gs.gameEventChan <- wrapped

	return connect.NewResponse(&pb.GameEventResponse{}), nil
}

func (gs *OMGWordsService) GetCGP(ctx context.Context, req *connect.Request[pb.GetCGPRequest],
) (*connect.Response[pb.CGPResponse], error) {
	gid := req.Msg.GameId
	g, err := gs.gameStore.GetDocument(ctx, gid, false)
	if err != nil {
		return nil, err
	}

	cgp, err := cwgame.ToCGP(gs.cfg.WGLConfig(), g.GameDocument)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.CGPResponse{Cgp: cgp}), nil
}

func (gs *OMGWordsService) DeleteBroadcastGame(ctx context.Context, req *connect.Request[pb.DeleteBroadcastGameRequest],
) (*connect.Response[pb.DeleteBroadcastGameResponse], error) {
	gid := req.Msg.GameId
	err := gs.failIfSessionDoesntOwn(ctx, gid)
	if err != nil {
		return nil, err
	}

	done, err := gs.metadataStore.GameIsDone(ctx, gid)
	if err != nil {
		return nil, err
	}
	if done {
		return nil, apiserver.InvalidArg("you cannot delete a game that is already done")
	}
	err = gs.metadataStore.DeleteAnnotatedGame(ctx, gid)
	if err != nil {
		return nil, err
	}
	// Clean up collection_games entries for this game
	err = gs.metadataStore.RemoveGameFromAllCollections(ctx, gid)
	if err != nil {
		return nil, err
	}
	err = gs.gameStore.DeleteDocument(ctx, gid)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.DeleteBroadcastGameResponse{}), nil
}

func (gs *OMGWordsService) ImportGCG(ctx context.Context, req *connect.Request[pb.ImportGCGRequest],
) (*connect.Response[pb.ImportGCGResponse], error) {
	if len(req.Msg.Gcg) > 1.28e5 {
		return nil, apiserver.InvalidArg("gcg string is too long")
	}
	u, err := apiserver.AuthUser(ctx, gs.userStore)
	if err != nil {
		return nil, err
	}

	// XXX: eventually we need our own GCG parsing library here (in liwords) that
	// just deals with GameDocuments or whatever their successor might be, to
	// avoid this config ugliness:
	cfgCopy := macondoconfig.DefaultConfig()
	cfgCopy.SetDefault(macondoconfig.ConfigDefaultLexicon, req.Msg.Lexicon)
	cfgCopy.SetDefault(macondoconfig.ConfigDefaultLetterDistribution, req.Msg.Rules.LetterDistributionName)

	r := strings.NewReader(req.Msg.Gcg)

	gh, err := gcgio.ParseGCGFromReader(cfgCopy, r)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	letterdist, err := tilemapping.GetDistribution(gs.cfg.WGLConfig(), req.Msg.Rules.LetterDistributionName)
	if err != nil {
		return nil, err
	}

	cbr := &pb.CreateBroadcastGameRequest{
		PlayersInfo: []*ipc.PlayerInfo{
			{UserId: gh.Players[0].Nickname,
				FullName: gh.Players[0].RealName,
				Nickname: gh.Players[0].Nickname,
				First:    true},
			{UserId: gh.Players[1].Nickname,
				FullName: gh.Players[1].RealName,
				Nickname: gh.Players[1].Nickname},
		},
		Lexicon:       req.Msg.Lexicon,
		Rules:         req.Msg.Rules,
		ChallengeRule: req.Msg.ChallengeRule,
	}

	gdoc, err := gs.createGDoc(ctx, u, cbr)
	if err != nil {
		return nil, err
	}

	// Add a dummy pass event at the end. GameHistory does not have this.
	foundEndRack := -1
	eventOwner := -1
	lastRack := ""
	for i := len(gh.Events) - 1; i >= 0; i-- {
		if gh.Events[i].Type == macondo.GameEvent_END_RACK_PTS {
			// insert right before
			foundEndRack = i
			eventOwner = int(gh.Events[i].PlayerIndex)
			lastRack = gh.Events[i].Rack
			break
		}
	}

	if foundEndRack > 0 && gh.Events[foundEndRack-1].Type != macondo.GameEvent_CHALLENGE_BONUS {
		gh.Events = append(gh.Events, &macondo.GameEvent{
			Type:        macondo.GameEvent_PASS,
			PlayerIndex: uint32(1 - eventOwner),
			Rack:        lastRack,
			Cumulative:  gh.FinalScores[1-eventOwner],
		})
		nevt := len(gh.Events)
		gh.Events[nevt-1], gh.Events[foundEndRack] = gh.Events[foundEndRack], gh.Events[nevt-1]
	}

	// Then replay events.
	err = cwgame.ReplayEvents(ctx, cfgCopy.WGLConfig(), gdoc, lo.Map(gh.Events, func(evt *macondo.GameEvent, index int) *ipc.GameEvent {
		return utilities.MacondoEvtToOMGEvt(evt, index, letterdist)
	}), false)
	if err != nil {
		return nil, err
	}

	if gdoc.PlayState == ipc.PlayState_GAME_OVER {
		if err = gs.metadataStore.MarkAnnotatedGameDone(ctx, gdoc.Uid); err != nil {
			return nil, err
		}
	}

	err = gs.gameStore.UpdateDocument(ctx, &stores.MaybeLockedDocument{GameDocument: gdoc})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.ImportGCGResponse{GameId: gdoc.Uid}), nil
}

func (gs *OMGWordsService) GetGameOwner(ctx context.Context, req *connect.Request[pb.GetGameOwnerRequest]) (
	*connect.Response[pb.GetGameOwnerResponse], error) {

	if req.Msg.GameId == "" {
		return nil, apiserver.InvalidArg("game ID is required")
	}

	owner, err := gs.metadataStore.GetGameOwner(ctx, req.Msg.GameId)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return connect.NewResponse(&pb.GetGameOwnerResponse{
				Found: false,
			}), nil
		}
		return nil, err
	}

	return connect.NewResponse(&pb.GetGameOwnerResponse{
		CreatorId:       owner.CreatorUuid,
		CreatorUsername: owner.Username.String,
		Found:           true,
	}), nil
}
