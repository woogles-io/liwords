package gameplay

import (
	"context"
	"errors"

	"github.com/domino14/liwords/pkg/entity"

	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto"
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
	ratingKey := variantFromGamereq(gamereq)
	players := entGame.History().Players
	done := entGame.History().PlayState == macondopb.PlayState_GAME_OVER

	playerInfo := []*pb.PlayerInfo{}
	for _, p := range players {
		u, err := gs.userStore.GetByUUID(ctx, p.UserId)
		if err != nil {
			return nil, err
		}
		pinfo := &pb.PlayerInfo{
			UserId:      p.UserId,
			Nickname:    p.Nickname,
			FullName:    u.Profile.FirstName + " " + u.Profile.LastName,
			CountryCode: u.Profile.CountryCode,
			Title:       u.Profile.Title,
		}

	}

	return nil, nil

}

func variantFromGamereq(gamereq *pb.GameRequest) (entity.VariantKey, error) {
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
		return "", errors.New("unsupported game type")
	}

	return entity.ToVariantKey(gamereq.Lexicon, variant, timefmt), nil

}
