package gameplay

import (
	"context"
	"errors"
	"math"
	"strconv"

	"github.com/domino14/liwords/pkg/entity"

	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/game_service"
	rtpb "github.com/domino14/liwords/rpc/api/proto/realtime"
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

func getRelevantRating(ratingKey entity.VariantKey, u *entity.User) string {
	if u.Profile == nil {
		return "UnratedAnon"
	}
	if u.Profile.Ratings.Data == nil {
		return "Unrated"
	}
	ratdict, ok := u.Profile.Ratings.Data[ratingKey]
	if ok {
		return strconv.Itoa(int(math.Round(ratdict.Rating)))
	}
	return "Unrated"
}

// GetMetadata gets metadata for the given game.
func (gs *GameService) GetMetadata(ctx context.Context, req *pb.GameInfoRequest) (*pb.GameInfoResponse, error) {

	entGame, err := gs.gameStore.Get(ctx, req.GameId)
	if err != nil {
		return nil, err
	}
	gamereq := entGame.CreationRequest()
	timefmt, variant, err := variantFromGamereq(gamereq)
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
			Rating:   getRelevantRating(ratingKey, u),
		}
		if u.Profile != nil {
			if u.Profile.FirstName != "" {
				pinfo.FullName = u.Profile.FirstName + " " + u.Profile.LastName
			} else {
				pinfo.FullName = u.Profile.LastName
			}
			pinfo.CountryCode = u.Profile.CountryCode
			pinfo.Title = u.Profile.Title
		}
		playerInfo = append(playerInfo, pinfo)
	}
	timeCtrl := strconv.Itoa(int(gamereq.InitialTimeSeconds)/60) + " " + strconv.Itoa(int(gamereq.IncrementSeconds))
	resp := &pb.GameInfoResponse{
		Players:         playerInfo,
		Lexicon:         gamereq.Lexicon,
		Variant:         string(variant),
		TimeControlName: string(timefmt),
		TimeControl:     timeCtrl,
		ChallengeRule:   gamereq.ChallengeRule,
		RatingMode:      gamereq.RatingMode,
		Done:            done,
	}

	return resp, nil

}

func variantFromGamereq(gamereq *rtpb.GameRequest) (entity.TimeControl, entity.Variant, error) {
	// hardcoded values here; fix sometime
	var timefmt entity.TimeControl
	if gamereq.InitialTimeSeconds <= entity.CutoffUltraBlitz {
		timefmt = entity.TCUltraBlitz
	} else if gamereq.InitialTimeSeconds <= entity.CutoffBlitz {
		timefmt = entity.TCBlitz
	} else if gamereq.InitialTimeSeconds <= entity.CutoffRapid {
		timefmt = entity.TCRapid
	} else {
		timefmt = entity.TCRegular
	}
	var variant entity.Variant
	if gamereq.Rules.BoardLayoutName == CrosswordGame {
		variant = entity.VarClassic
	} else {
		return "", "", errors.New("unsupported game type")
	}

	return timefmt, variant, nil

}
