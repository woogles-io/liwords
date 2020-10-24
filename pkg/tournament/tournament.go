package tournament

import (
	"context"
	"errors"
	"sort"
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
	AddDirectors(context.Context, string, *entity.TournamentPersons) error
	RemoveDirectors(context.Context, string, *entity.TournamentPersons) error
	AddPlayers(context.Context, string, *entity.TournamentPersons) error
	RemovePlayers(context.Context, string, *entity.TournamentPersons) error
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
	players *entity.TournamentPersons,
	directors *entity.TournamentPersons,
	req *pb.GameRequest,
	pairingMethods []entity.PairingMethod,
	numberOfRounds int,
	gamesPerRound int,
	ttype entity.TournamentType) (*entity.Tournament, error) {

	var entTournament *entity.Tournament

	// Sort players by descending int (which is probably rating)
	var values []int
	for _, v := range players.Persons {
		values = append(values, v)
	}
	sort.Ints(values)
	reversedPlayersMap := reverseMap(players.Persons)
	rankedPlayers := []string{}
	for i := len(values) - 1; i >= 0; i-- {
		rankedPlayers = append(rankedPlayers, reversedPlayersMap[values[i]])
	}

	if ttype == entity.ClassicTournamentType {
		tm, err := entity.NewTournamentClassic(rankedPlayers, numberOfRounds, pairingMethods, gamesPerRound)
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

func reverseMap(m map[string]int) map[int]string {
	n := make(map[int]string)
	for k, v := range m {
		n[v] = k
	}
	return n
}
