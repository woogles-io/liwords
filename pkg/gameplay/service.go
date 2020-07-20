package gameplay

import (
	"context"
	"strconv"

	"github.com/domino14/liwords/pkg/entity"

	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/game_service"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

type GameService struct {
	userStore user.Store
	gameStore GameStore
}

// NewGameService creates a Twirp GameService
func NewGameService(u user.Store, gs GameStore) *GameService {
	return &GameService{u, gs}
}

// GetMetadata gets metadata for the given game.
func (gs *GameService) GetMetadata(ctx context.Context, req *pb.GameInfoRequest) (*pb.GameInfoResponse, error) {

	entGame, err := gs.gameStore.Get(ctx, req.GameId)
	if err != nil {
		return nil, err
	}
	gamereq := entGame.CreationRequest()
	timefmt, variant, err := entity.VariantFromGameReq(gamereq)
	if err != nil {
		return nil, err
	}
	players := entGame.History().Players
	done := entGame.History().PlayState == macondopb.PlayState_GAME_OVER
	ratingKey := entity.ToVariantKey(gamereq.Lexicon, variant, timefmt)
	playerInfo := []*pb.PlayerInfo{}
	for _, p := range players {
		u, err := gs.userStore.GetByUUID(ctx, p.UserId)
		if err != nil {
			return nil, err
		}

		pinfo := &pb.PlayerInfo{
			UserId:   p.UserId,
			Nickname: p.Nickname,
			Rating:   u.GetRelevantRating(ratingKey),
		}
		if u.Profile != nil {
			pinfo.FullName = u.RealName()
			pinfo.CountryCode = u.Profile.CountryCode
			pinfo.Title = u.Profile.Title
		}
		playerInfo = append(playerInfo, pinfo)
	}
	timeCtrl := strconv.Itoa(int(gamereq.InitialTimeSeconds)/60) + " " + strconv.Itoa(int(gamereq.IncrementSeconds))
	resp := &pb.GameInfoResponse{
		Players:            playerInfo,
		Lexicon:            gamereq.Lexicon,
		Variant:            string(variant),
		TimeControlName:    string(timefmt),
		TimeControl:        timeCtrl,
		MaxOvertimeMinutes: gamereq.MaxOvertimeMinutes,
		ChallengeRule:      gamereq.ChallengeRule,
		RatingMode:         gamereq.RatingMode,
		Done:               done,
	}

	return resp, nil

}
