package tournament

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
)

type TournamentStore interface {
	Get(context.Context, string) (*entity.Tournament, error)
	Set(context.Context, *entity.Tournament) error
	Create(context.Context, *entity.Tournament) error
	Unload(context.Context, string)
}

func HandleTournamentGameEnded(ctx context.Context, ts TournamentStore, g *entity.Game) error {

	Results := []pb.TournamentGameResult{pb.TournamentGameResult_DRAW,
		pb.TournamentGameResult_WIN,
		pb.TournamentGameResult_LOSS}

	err := SetResult(ctx,
		ts,
		g.Tournamentdata.Id,
		g.Tournamentdata.Division,
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

	return TournamentGameEndedEvent(ctx, ts, g.Tournamentdata.Id, g.Tournamentdata.Division, g.Tournamentdata.Round)
}

func NewTournament(ctx context.Context,
	tournamentStore TournamentStore,
	cfg *config.Config,
	name string,
	description string,
	directors *entity.TournamentPersons,
	ttype *entity.TournamentType) (*entity.Tournament, error) {

	entTournament := &entity.Tournament{Name: name,
		Description: description,
		Directors:   directors}

	// Save the tournament to the store.
	if err := tournamentStore.Create(ctx, entTournament); err != nil {
		return nil, err
	}
	return entTournament, nil
}

func TournamentGameEndedEvent(ctx context.Context, ts TournamentStore, id string, division string, round int) error {

	// Send new results to some socket or something
	isFinished, err := IsFinished(ctx, ts, id, division)
	if err != nil {
		return err
	}
	if isFinished {
		// Send stuff to sockets and whatnot
	} else {
		isRoundComplete, err := IsRoundComplete(ctx, ts, id, division, round)
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

func SetTournamentControls(ctx context.Context, ts TournamentStore, id string, division string, name string, description string, controls *entity.TournamentControls) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return errors.New(fmt.Sprintf("Division %s does not exist.", division))
	}

	if t.IsStarted {
		return errors.New("Cannot change tournament controls after it has started.")
	}

	t.Lock()
	defer t.Unlock()

	t.Name = name
	t.Description = description
	divisionObject.Controls = controls

	return ts.Set(ctx, t)
}

func AddDivision(ctx context.Context, ts TournamentStore, id string, division string) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	if !t.IsStarted {
		return errors.New("Cannot add division after the tournament has started.")
	}

	t.Lock()
	defer t.Unlock()

	t.Divisions[division] = emptyDivision()

	return ts.Set(ctx, t)
}

func RemoveDivision(ctx context.Context, ts TournamentStore, id string, division string) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	_, ok := t.Divisions[division]

	if !ok {
		return errors.New(fmt.Sprintf("Division %s does not exist.", division))
	}

	if !t.IsStarted {
		return errors.New("Cannot remove division after the tournament has started.")
	}

	t.Lock()
	defer t.Unlock()

	delete(t.Divisions, division)
	return ts.Set(ctx, t)
}

func AddDirectors(ctx context.Context, ts TournamentStore, id string, directors *entity.TournamentPersons) error {
	return addTournamentPersons(ctx, ts, id, "", directors, false)
}

func RemoveDirectors(ctx context.Context, ts TournamentStore, id string, directors *entity.TournamentPersons) error {
	return removeTournamentPersons(ctx, ts, id, "", directors, false)
}

func AddPlayers(ctx context.Context, ts TournamentStore, id string, division string, players *entity.TournamentPersons) error {
	return addTournamentPersons(ctx, ts, id, division, players, true)
}

func RemovePlayers(ctx context.Context, ts TournamentStore, id string, division string, players *entity.TournamentPersons) error {
	return removeTournamentPersons(ctx, ts, id, division, players, true)
}

func SetPairing(ctx context.Context, ts TournamentStore, id string, division string, playerOneId string, playerTwoId string, round int) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return errors.New(fmt.Sprintf("Division %s does not exist.", division))
	}

	t.Lock()

	err = divisionObject.DivisionManager.SetPairing(playerOneId, playerTwoId, round)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	t.Unlock()
	return TournamentSetPairingsEvent(ctx, ts)
}

func SetResult(ctx context.Context,
	ts TournamentStore,
	id string,
	division string,
	playerOneId string,
	playerTwoId string,
	playerOneScore int,
	playerTwoScore int,
	playerOneResult realtime.TournamentGameResult,
	playerTwoResult realtime.TournamentGameResult,
	reason realtime.GameEndReason,
	round int,
	gameIndex int,
	amendment bool) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return errors.New(fmt.Sprintf("Division %s does not exist.", division))
	}

	if !t.IsStarted {
		return errors.New("Cannot set tournament results before the tournament has started.")
	}

	t.Lock()

	err = divisionObject.DivisionManager.SubmitResult(round,
		playerOneId,
		playerTwoId,
		playerOneScore,
		playerTwoScore,
		playerOneResult,
		playerTwoResult,
		reason,
		amendment,
		gameIndex)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	t.Unlock()
	return TournamentGameEndedEvent(ctx, ts, id, division, round)
}

func StartRound(ctx context.Context, ts TournamentStore, id string, division string, round int) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return errors.New(fmt.Sprintf("Division %s does not exist.", division))
	}

	t.Lock()
	defer t.Unlock()

	if round == 0 {
		if divisionObject.DivisionManager != nil {
			return errors.New("This division has already been started.")
		}

		rankedPlayers := rankPlayers(t.Divisions[division].Players)

		if divisionObject.Controls.Type == entity.ClassicTournamentType {
			d, err := NewClassicDivision(rankedPlayers,
				divisionObject.Controls.NumberOfRounds,
				divisionObject.Controls.GamesPerRound,
				divisionObject.Controls.PairingMethods,
				divisionObject.Controls.FirstMethods)
			if err != nil {
				return err
			}
			divisionObject.DivisionManager = d
		} else {
			return errors.New("Only Classic Tournaments have been implemented")
		}
		t.IsStarted = true
	}

	err = divisionObject.DivisionManager.StartRound(round)
	if err != nil {
		return err
	}

	return ts.Set(ctx, t)
}

func IsStarted(ctx context.Context, ts TournamentStore, id string) (bool, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return false, err
	}
	return t.IsStarted, nil
}

func IsRoundComplete(ctx context.Context, ts TournamentStore, id string, division string, round int) (bool, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return false, err
	}

	if !t.IsStarted {
		return false, errors.New("Cannot check if round is complete before the tournament has started.")
	}

	return t.Divisions[division].DivisionManager.IsRoundComplete(round)
}

func IsFinished(ctx context.Context, ts TournamentStore, id string, division string) (bool, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return false, err
	}

	if !t.IsStarted {
		return false, errors.New("Cannot check if tournament is finished before the tournament has started.")
	}

	return t.Divisions[division].DivisionManager.IsFinished()
}

func emptyDivision() *entity.TournamentDivision {
	return &entity.TournamentDivision{Players: &entity.TournamentPersons{Persons: map[string]int{}},
		Controls: &entity.TournamentControls{}, DivisionManager: nil}
}

func addTournamentPersons(ctx context.Context,
	ts TournamentStore,
	id string,
	division string,
	persons *entity.TournamentPersons,
	isPlayers bool) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return errors.New(fmt.Sprintf("Division %s does not exist.", division))
	}

	t.Lock()
	defer t.Unlock()

	var personsMap map[string]int
	if isPlayers {
		personsMap = divisionObject.Players.Persons
	} else {
		personsMap = t.Directors.Persons
	}

	// Only perform the add operation if all persons can be added.
	for k, _ := range persons.Persons {
		_, ok := personsMap[k]
		if ok {
			return errors.New(fmt.Sprintf("Person (%s, %d) already exists.", k, personsMap[k]))
		}
	}

	for k, v := range persons.Persons {
		personsMap[k] = v
	}

	return ts.Set(ctx, t)
}

func removeTournamentPersons(ctx context.Context,
	ts TournamentStore,
	id string,
	division string,
	persons *entity.TournamentPersons,
	isPlayers bool) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return errors.New(fmt.Sprintf("Division %s does not exist.", division))
	}

	t.Lock()
	defer t.Unlock()

	var personsMap map[string]int
	if isPlayers {
		personsMap = divisionObject.Players.Persons
	} else {
		personsMap = t.Directors.Persons
	}

	// Only perform the remove operation if all persons can be removed.
	for k, _ := range persons.Persons {
		_, ok := personsMap[k]
		if !ok {
			return errors.New(fmt.Sprintf("Person (%s, %d) does not exist.", k, personsMap[k]))
		}
	}

	for k, _ := range persons.Persons {
		delete(personsMap, k)
	}

	return ts.Set(ctx, t)
}

func rankPlayers(players *entity.TournamentPersons) []string {
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
	return rankedPlayers
}

func resultsToScores(results []int) (int, int, int) {
	wins := results[int(realtime.TournamentGameResult_WIN)] +
		results[int(realtime.TournamentGameResult_BYE)] +
		results[int(realtime.TournamentGameResult_FORFEIT_WIN)]
	losses := results[int(realtime.TournamentGameResult_LOSS)] +
		results[int(realtime.TournamentGameResult_FORFEIT_LOSS)]
	draws := results[int(realtime.TournamentGameResult_DRAW)]

	return wins, losses, draws
}

func reverseMap(m map[string]int) map[int]string {
	n := make(map[int]string)
	for k, v := range m {
		n[v] = k
	}
	return n
}
