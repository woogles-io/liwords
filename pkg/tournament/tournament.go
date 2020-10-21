package tournament

import (
	"context"
	"errors"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
)

type TournamentStore interface {
	Get(ctx context.Context, id string) (*entity.TournamentManager, error)
	Set(context.Context, *entity.TournamentManager) error
	Create(context.Context, *entity.TournamentManager) error
	Unload(context.Context, string)
}

// InstantiateNewTournament instantiates a tournament and returns it.
func InstantiateNewTournament(ctx context.Context,
	tournamentStore TournamentStore,
	cfg *config.Config,
	tournamentName string,
	players []string,
	directors []string,
	req *pb.GameRequest,
	pairingMethods []entity.PairingMethod,
	numberOfRounds int,
	gamesPerRound int,
	ttype entity.TournamentType) (*entity.TournamentManager, error) {

	var entTournamentManager *entity.TournamentManager
	if ttype == entity.ClassicTournamentType {
		entTournament, err := entity.NewTournamentClassic(players, numberOfRounds, pairingMethods, gamesPerRound)
		if err != nil {
			return nil, err
		}
		entTournamentManager = &entity.TournamentManager{Directors: directors,
			Name:       tournamentName,
			Type:       ttype,
			Tournament: entTournament}
	} else {
		return nil, errors.New("Only Classic Tournaments have been implemented")
	}

	// Save the tournament to the store.
	if err := tournamentStore.Create(ctx, entTournamentManager); err != nil {
		return nil, err
	}
	return entTournamentManager, nil
}

func HandleTournamentGameEnded(ctx context.Context, tournamentStore TournamentStore) error {
	return nil
}
