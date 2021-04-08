package mod

import (
	"context"
	"fmt"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	"github.com/domino14/liwords/rpc/api/proto/mod_service"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/golang/protobuf/ptypes"
)

func Automod(ctx context.Context, us user.Store, u0 *entity.User, u1 *entity.User, g *entity.Game) error {
	// The behavior we currently check for is fairly primitive
	// This will be expanded in the future
	if g.GameEndReason == pb.GameEndReason_TIME {
		// Someone lost on time, determine if the loser made no plays at all
		history := g.History()
		loserNickname := history.Players[1-history.Winner].Nickname
		// This should even be possible but might as well check
		if u0.Username != loserNickname && u1.Username != loserNickname {
			return fmt.Errorf("loser (%s) not found in players (%s, %s)", loserNickname, u0.Username, u1.Username)
		}
		loserMadePlay := false
		for _, evt := range history.Events {
			if evt.Nickname == loserNickname {
				loserMadePlay = true
				break
			}
		}
		if !loserMadePlay {
			loserUser := u0
			if u1.Username == loserNickname {
				loserUser = u1
			}
		}

		// IMPORTANT
		// Do not call ApplyActions in this case, that will
		// use the current context to get the applier id, which
		// will always be incorrect since this is an automod action
		applyAction(ctx, us, cs, &mod_service.ModAction{UserId: loserUser.UUID})

	}
	return nil
}

func updateNotoriousGames(user *entity.User) (bool, error) {

	instantiateActions(user)

	now := time.Now()
	updated := false
	for i := 0; i < len(user.Notoriety.Games); i++ {
		ng := user.Notoriety.Games[i]
		convertedCreationTime, err := ptypes.Timestamp(ng.CreatedAt)
		if err != nil {
			return false, err
		}
		if err == nil && now.After(convertedCreationTime) {
			removeCurrentAction(user, action.Type, "")
			updated = true
		}
	}
	for _, ng := range user.Notoriety.Games {

	}

	return updated, nil
}
