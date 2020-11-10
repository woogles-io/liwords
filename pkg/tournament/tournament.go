package tournament

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
)

type TournamentStore interface {
	Get(context.Context, string) (*entity.Tournament, error)
	Set(context.Context, *entity.Tournament) error
	Create(context.Context, *entity.Tournament) error
	Unload(context.Context, string)
	SetTournamentEventChan(c chan<- *entity.EventWrapper)
	TournamentEventChan() chan<- *entity.EventWrapper
}

func HandleTournamentGameEnded(ctx context.Context, ts TournamentStore, us user.Store,
	g *entity.Game) error {

	Results := []pb.TournamentGameResult{pb.TournamentGameResult_DRAW,
		pb.TournamentGameResult_WIN,
		pb.TournamentGameResult_LOSS}

	p1idx, p2idx := 0, 1
	p1result, p2result := Results[g.WinnerIdx+1], Results[g.LoserIdx+1]
	if g.History().SecondWentFirst {
		p1idx, p2idx = p2idx, p1idx
		p1result, p2result = p2result, p1result
	}

	return SetResult(ctx,
		ts,
		us,
		g.TournamentData.Id,
		g.TournamentData.Division,
		g.History().Players[p1idx].UserId,
		g.History().Players[p2idx].UserId,
		int(g.History().FinalScores[p1idx]),
		int(g.History().FinalScores[p2idx]),
		p1result,
		p2result,
		g.GameEndReason,
		g.TournamentData.Round,
		g.TournamentData.GameIndex,
		false,
		g,
		ts.TournamentEventChan())
}

func NewTournament(ctx context.Context,
	tournamentStore TournamentStore,
	id string,
	name string,
	description string,
	directors *entity.TournamentPersons) (*entity.Tournament, error) {

	executiveDirector, err := getExecutiveDirector(directors)
	if err != nil {
		return nil, err
	}

	if id == "" {
		id = shortuuid.New()[2:8]
	}

	entTournament := &entity.Tournament{Name: name,
		Description:       description,
		Directors:         directors,
		ExecutiveDirector: executiveDirector,
		IsStarted:         false,
		Divisions:         map[string]*entity.TournamentDivision{},
		UUID:              id,
	}
	for {
		// Save the tournament to the store.
		err = tournamentStore.Create(ctx, entTournament)
		if err == nil {
			break
		}
		log.Err(err).Msg("tournament-create-error")
		entTournament.UUID = shortuuid.New()[2:8]

	}
	return entTournament, nil
}

// SendTournamentGameEndedEvent sends end results to a channel.
func SendTournamentGameEndedEvent(ctx context.Context, ts TournamentStore, id string,
	division string, round int, evtChan chan<- *entity.EventWrapper, gameID string,
	players []*pb.TournamentGameEndedEvent_Player, gameEndReason realtime.GameEndReason) error {

	// Send new results to some socket or something.
	// Check if this division is just finished.
	isFinished, err := IsFinished(ctx, ts, id, division)
	if err != nil {
		return err
	}
	if isFinished {
		// Send stuff to sockets and whatnot
	} else {
		// Is the entire round complete?
		isRoundComplete, err := IsRoundComplete(ctx, ts, id, division, round)
		if err != nil {
			return err
		}
		if isRoundComplete {
			// Send some other stuff, yeah?
		}
	}

	tevt := &realtime.TournamentGameEndedEvent{
		GameId:    gameID,
		Players:   players,
		EndReason: gameEndReason,
		Time:      time.Now().Unix(),
	}
	log.Debug().Interface("tevt", tevt).Msg("sending tournament game ended evt")
	wrapped := entity.WrapEvent(tevt, pb.MessageType_TOURNAMENT_GAME_ENDED_EVENT)
	wrapped.AddAudience(entity.AudTournament, id)
	evtChan <- wrapped
	log.Debug().Str("tid", id).Msg("sent tournament game ended event")
	return nil
}

func TournamentSetPairingsEvent(ctx context.Context, tournamentStore TournamentStore) error {
	// Do something probably
	return nil
}

func SetTournamentMetadata(ctx context.Context, ts TournamentStore, id string, name string, description string) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	t.Name = name
	t.Description = name

	return ts.Set(ctx, t)
}

func SetTournamentControls(ctx context.Context, ts TournamentStore, id string, division string, controls *entity.TournamentControls) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return errors.New(fmt.Sprintf("Division %s does not exist.", division))
	}

	if t.IsStarted {
		return errors.New("Cannot change tournament controls after it has started.")
	}

	divisionObject.Controls = controls

	err = createDivisionManager(t, division)
	if err != nil {
		return err
	}

	return ts.Set(ctx, t)
}

func AddDivision(ctx context.Context, ts TournamentStore, id string, division string) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsStarted {
		return errors.New("Cannot add division after the tournament has started.")
	}

	_, ok := t.Divisions[division]

	if ok {
		return errors.New(fmt.Sprintf("Division %s already exists.", division))
	}

	t.Divisions[division] = emptyDivision()

	return ts.Set(ctx, t)
}

func RemoveDivision(ctx context.Context, ts TournamentStore, id string, division string) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	_, ok := t.Divisions[division]

	if !ok {
		return errors.New(fmt.Sprintf("Division %s does not exist.", division))
	}

	if t.IsStarted {
		return errors.New("Cannot remove division after the tournament has started.")
	}

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

	t.Lock()
	defer t.Unlock()

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return errors.New(fmt.Sprintf("Division %s does not exist.", division))
	}

	if divisionObject.DivisionManager == nil {
		return errors.New(fmt.Sprintf("Division %s does not have enough players or controls to set pairings.", division))
	}

	err = divisionObject.DivisionManager.SetPairing(playerOneId, playerTwoId, round)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return TournamentSetPairingsEvent(ctx, ts)
}

// SetResult sets the result for the game. Note: playerOne and playerTwo
// went first and second, respectively.
func SetResult(ctx context.Context,
	ts TournamentStore,
	us user.Store,
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
	amendment bool,
	g *entity.Game,
	eventChan chan<- *entity.EventWrapper) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return errors.New(fmt.Sprintf("Division %s does not exist.", division))
	}

	if !t.IsStarted {
		return errors.New("Cannot set tournament results before the tournament has started.")
	}

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
	p1user, err := us.GetByUUID(ctx, playerOneId)
	if err != nil {
		return err
	}
	p2user, err := us.GetByUUID(ctx, playerTwoId)
	if err != nil {
		return err
	}

	// Here just send info about the actual completed game.

	players := []*pb.TournamentGameEndedEvent_Player{
		{Username: p1user.Username, Score: int32(playerOneScore), Result: playerOneResult},
		{Username: p2user.Username, Score: int32(playerTwoScore), Result: playerTwoResult},
	}

	gid := ""
	if g != nil {
		gid = g.GameID()
	}

	return SendTournamentGameEndedEvent(ctx, ts, id, division, round, eventChan,
		gid, players, reason)
}

func StartTournament(ctx context.Context, ts TournamentStore, id string) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	// Do not lock, StartRound will do that

	for division, _ := range t.Divisions {
		err := StartRound(ctx, ts, id, division, 0)
		if err != nil {
			return err
		}
	}
	t.IsStarted = true
	return nil
}

func StartRound(ctx context.Context, ts TournamentStore, id string, division string, round int) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return errors.New(fmt.Sprintf("Division %s does not exist.", division))
	}

	if divisionObject.DivisionManager == nil {
		return errors.New(fmt.Sprintf("Division %s does not have enough players or controls to set pairings.", division))
	}

	ready, err := divisionObject.DivisionManager.IsRoundReady(round)
	if err != nil {
		return err
	}

	if ready {
		// Send some stuff to the channels
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

	_, ok := t.Divisions[division]

	if !ok {
		return false, errors.New(fmt.Sprintf("Division %s does not exist.", division))
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

	_, ok := t.Divisions[division]

	if !ok {
		return false, errors.New(fmt.Sprintf("Division %s does not exist.", division))
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

	t.Lock()
	defer t.Unlock()

	divisionObject, ok := t.Divisions[division]

	if isPlayers && !ok {
		return errors.New(fmt.Sprintf("Division %s does not exist.", division))
	}

	if t.IsStarted {
		return errors.New("Cannot cannot change players or directors after the tournament has started.")
	}

	var personsMap map[string]int
	if isPlayers {
		personsMap = divisionObject.Players.Persons
	} else {
		personsMap = t.Directors.Persons
		if executiveDirectorExists(persons.Persons) {
			return errors.New("Cannot add another executive director.")
		}
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

	if isPlayers {
		err = createDivisionManager(t, division)
		if err != nil {
			return err
		}
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

	t.Lock()
	defer t.Unlock()

	divisionObject, ok := t.Divisions[division]

	if isPlayers && !ok {
		return errors.New(fmt.Sprintf("Division %s does not exist.", division))
	}

	if t.IsStarted {
		return errors.New("Cannot cannot change players or directors after the tournament has started.")
	}

	var personsMap map[string]int
	if isPlayers {
		personsMap = divisionObject.Players.Persons
	} else {
		personsMap = t.Directors.Persons
		if executiveDirectorExists(persons.Persons) {
			return errors.New("Cannot remove the executive director.")
		}
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

	if isPlayers {
		err = createDivisionManager(t, division)
		if err != nil {
			return err
		}
	}

	return ts.Set(ctx, t)
}

func createDivisionManager(t *entity.Tournament, division string) error {

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return errors.New(fmt.Sprintf("Division %s does not exist.", division))
	}

	// Create a new division if there are sufficient players
	// otherwise, destroy the old one.
	if len(divisionObject.Players.Persons) > 1 &&
		divisionObject.Controls.NumberOfRounds > 0 &&
		divisionObject.Controls.GamesPerRound != nil &&
		divisionObject.Controls.PairingMethods != nil &&
		divisionObject.Controls.FirstMethods != nil {
		rankedPlayers := rankPlayers(divisionObject.Players)
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
		divisionObject.DivisionManager = nil
	}
	return nil
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

func getExecutiveDirector(directors *entity.TournamentPersons) (string, error) {
	err := errors.New("Tournament must have exactly one executive director.")
	ed := ""
	for k, v := range directors.Persons {
		if v == 0 {
			if ed != "" {
				return "", err
			} else {
				ed = k
			}
		}
	}
	if ed == "" {
		return "", err
	}
	return ed, nil
}

func executiveDirectorExists(directors map[string]int) bool {
	for _, v := range directors {
		if v == 0 {
			return true
		}
	}
	return false
}

func reverseMap(m map[string]int) map[int]string {
	n := make(map[int]string)
	for k, v := range m {
		n[v] = k
	}
	return n
}
