package tournament

import (
	"context"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
)

type TournamentStore interface {
	Get(context.Context, string) (*entity.Tournament, error)
	Set(context.Context, *entity.Tournament) error
	Create(context.Context, *entity.Tournament) error
	SetTournamentControls(context.Context, string, string, string, *entity.TournamentControls) error
	AddDirectors(context.Context, string, *entity.TournamentPersons) error
	RemoveDirectors(context.Context, string, *entity.TournamentPersons) error
	AddPlayers(context.Context, string, *entity.TournamentPersons) error
	RemovePlayers(context.Context, string, *entity.TournamentPersons) error
	SetPairing(context.Context, string, string, string, int) error
	SetResult(context.Context, string, string, string, int, int,
		pb.TournamentGameResult, pb.TournamentGameResult, pb.GameEndReason, int, int, bool) error
	StartRound(context.Context, string, int) error
	IsStarted(context.Context, string) (bool, error)
	IsRoundComplete(context.Context, string, int) (bool, error)
	IsFinished(context.Context, string) (bool, error)
	Unload(context.Context, string)
}

// InstantiateNewTournament instantiates a tournament and returns it.
func InstantiateNewTournament(ctx context.Context,
	tournamentStore TournamentStore,
	cfg *config.Config,
	name string,
	description string,
	players *entity.TournamentPersons,
	directors *entity.TournamentPersons,
	controls *entity.TournamentControls) (*entity.Tournament, error) {

	entTournament := &entity.Tournament{Name: name,
		Description: description,
		Directors:   directors,
		Players:     players,
		Controls:    controls}

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
