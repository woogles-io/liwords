package tournament

import (
	"context"
	"errors"
	"time"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

type TournamentStore interface {
	Get(context.Context, string) (*entity.Tournament, error)
	Set(context.Context, *entity.Tournament) error
	Create(context.Context, *entity.Tournament) error
	SetTournamentControls(context.Context, string, string, string, string, string, int32,
		macondopb.ChallengeRule, pb.RatingMode, int32, int32, time.Time) error
	AddDirectors(context.Context, string, []string) error
	RemoveDirectors(context.Context, string, []string) error
	AddPlayers(context.Context, string, []string) error
	RemovePlayers(context.Context, string, []string) error
	SetPairing(context.Context, string, string, string, int) error
	SetResult(context.Context, string, string, string, int, int,
		pb.TournamentGameResult, pb.TournamentGameResult, pb.GameEndReason, int, int, bool) error
	StartRound(context.Context, string, int) error
	IsRoundComplete(context.Context, string, int) (bool, error)
	IsFinished(context.Context, string) (bool, error)
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
	ttype entity.TournamentType) (*entity.Tournament, error) {

	var entTournament *entity.Tournament
	if ttype == entity.ClassicTournamentType {
		tm, err := entity.NewTournamentClassic(players, numberOfRounds, pairingMethods, gamesPerRound)
		if err != nil {
			return nil, err
		}
		entTournament = &entity.Tournament{Directors: directors,
			Name:              tournamentName,
			Type:              ttype,
			TournamentManager: tm}
	} else {
		return nil, errors.New("Only Classic Tournaments have been implemented")
	}

	// Save the tournament to the store.
	if err := tournamentStore.Create(ctx, entTournament); err != nil {
		return nil, err
	}
	return entTournament, nil
}

func HandleTournamentGameEnded(ctx context.Context, tournamentStore TournamentStore, g *entity.Game) error {

	Results := []pb.TournamentGameResult{pb.TournamentGameResult_DRAW,
		pb.TournamentGameResult_WIN,
		pb.TournamentGameResult_LOSS}

	err := tournamentStore.SetResult(ctx,
		g.Tournamentdata.TournamentId,
		g.History().Players[0].UserId,
		g.History().Players[1].UserId,
		int(g.History().FinalScores[0]),
		int(g.History().FinalScores[1]),
		Results[g.WinnerIdx+1],
		Results[g.LoserIdx+1],
		g.GameEndReason,
		g.Tournamentdata.Round,
		g.Tournamentdata.GameIndex,
		false)

	if err != nil {
		return err
	}

	return TournamentGameEndedEvent(ctx, tournamentStore, g.Tournamentdata.TournamentId, g.Tournamentdata.Round)
}

func TournamentGameEndedEvent(ctx context.Context, tournamentStore TournamentStore, id string, round int) error {

	// Send new results to some socket or something
	isFinished, err := tournamentStore.IsFinished(ctx, id)
	if err != nil {
		return err
	}
	if isFinished {
		// Send stuff to sockets and whatnot
	} else {
		isRoundComplete, err := tournamentStore.IsRoundComplete(ctx, id, round)
		if err != nil {
			return err
		}
		if isRoundComplete {
			// Send some other stuff, yeah?
		}
	}

	return nil
}

func TournamentSetPairingsEvent(ctx context.Context, tournamentStore TournamentStore) error {
	// Do something probably
	return nil
}
