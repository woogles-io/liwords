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

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	pb "github.com/domino14/liwords/rpc/api/proto/tournament_service"
)

type TournamentStore interface {
	Get(context.Context, string) (*entity.Tournament, error)
	GetBySlug(context.Context, string) (*entity.Tournament, error)
	Set(context.Context, *entity.Tournament) error
	Create(context.Context, *entity.Tournament) error
	GetRecentGames(ctx context.Context, tourneyID string, numGames int, offset int) (*pb.RecentGamesResponse, error)
	Unload(context.Context, string)
	SetTournamentEventChan(c chan<- *entity.EventWrapper)
	TournamentEventChan() chan<- *entity.EventWrapper

	GetRecentClubSessions(ctx context.Context, clubID string, numSessions int, offset int) (*pb.ClubSessionsResponse, error)
}

func HandleTournamentGameEnded(ctx context.Context, ts TournamentStore, us user.Store,
	g *entity.Game) error {

	Results := []realtime.TournamentGameResult{realtime.TournamentGameResult_DRAW,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS}

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
		g)
}

func NewTournament(ctx context.Context,
	tournamentStore TournamentStore,
	name string,
	description string,
	directors *entity.TournamentPersons,
	ttype entity.CompetitionType,
	parent string,
	slug string,
) (*entity.Tournament, error) {

	executiveDirector, err := getExecutiveDirector(directors)
	if err != nil {
		return nil, err
	}

	id := shortuuid.New()

	entTournament := &entity.Tournament{Name: name,
		Description:       description,
		Directors:         directors,
		ExecutiveDirector: executiveDirector,
		IsStarted:         false,
		Divisions:         map[string]*entity.TournamentDivision{},
		UUID:              id,
		Type:              ttype,
		ParentID:          parent,
		Slug:              slug,
	}

	err = tournamentStore.Create(ctx, entTournament)
	if err != nil {
		return nil, err
	}
	return entTournament, nil
}

// SendTournamentMessage sends updated tournament information on the channel.
func SendTournamentMessage(ctx context.Context, ts TournamentStore, id string) error {

	_, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	eventChannel := ts.TournamentEventChan()

	tdevt, err := TournamentDataResponse(ctx, ts, id)
	if err != nil {
		return err
	}
	log.Debug().Interface("tdevt", tdevt).Msg("sending tournament message")
	wrapped := entity.WrapEvent(tdevt, realtime.MessageType_TOURNAMENT_MESSAGE)
	wrapped.AddAudience(entity.AudTournament, id)
	if eventChannel != nil {
		eventChannel <- wrapped
	}
	log.Debug().Str("tid", id).Msg("sent tournament message")
	return nil
}

// SendTournamentDivisionMessage sends an updated division on the channel.
func SendTournamentDivisionMessage(ctx context.Context, ts TournamentStore, id string,
	division string) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	eventChannel := ts.TournamentEventChan()

	_, ok := t.Divisions[division]

	var wrapped *entity.EventWrapper
	if ok {
		tdevt, err := TournamentDivisionDataResponse(ctx, ts, id, division)
		if err != nil {
			return err
		}
		log.Debug().Interface("tdevt", tdevt).Msg("sending tournament division message")
		wrapped = entity.WrapEvent(tdevt, realtime.MessageType_TOURNAMENT_DIVISION_MESSAGE)
	} else {
		tddevt := &realtime.TournamentDivisionDeletedResponse{Id: id, Division: division}
		log.Debug().Interface("tdevt", tddevt).Msg("sending tournament division deleted message")
		wrapped = entity.WrapEvent(tddevt, realtime.MessageType_TOURNAMENT_DIVISION_DELETED_MESSAGE)
	}
	wrapped.AddAudience(entity.AudTournament, id)
	if eventChannel != nil {
		eventChannel <- wrapped
	}
	log.Debug().Str("tid:did", id+":"+division).Msg("sent tournament division changed or deleted message")
	return nil
}

func TournamentSetPairingsEvent(ctx context.Context, ts TournamentStore, id string, division string) error {
	return SendTournamentDivisionMessage(ctx, ts, id, division)
}

func SetTournamentMetadata(ctx context.Context, ts TournamentStore, id string, name string, description string) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	t.Name = name
	t.Description = description

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentMessage(ctx, ts, id)
}

func SetSingleRoundControls(ctx context.Context, ts TournamentStore, id string, division string, round int, controls *entity.RoundControls) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	if t.IsStarted {
		return errors.New("cannot change tournament controls after it has started")
	}

	divisionObject.Controls.RoundControls[round] = controls

	err = createDivisionManager(t, division)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
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
		return fmt.Errorf("division %s does not exist", division)
	}

	if t.IsStarted {
		return errors.New("cannot change tournament controls after it has started")
	}

	divisionObject.Controls = controls

	err = createDivisionManager(t, division)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
}

func AddDivision(ctx context.Context, ts TournamentStore, id string, division string) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsStarted {
		return errors.New("cannot add division after the tournament has started")
	}

	_, ok := t.Divisions[division]

	if ok {
		return fmt.Errorf("division %s already exists", division)
	}

	t.Divisions[division] = emptyDivision()

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
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
		return fmt.Errorf("division %s does not exist", division)
	}

	if t.IsStarted {
		return errors.New("cannot remove division after the tournament has started")
	}

	delete(t.Divisions, division)
	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
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
		return fmt.Errorf("division %s does not exist", division)
	}

	if divisionObject.DivisionManager == nil {
		return fmt.Errorf("division %s does not have enough players or controls to set pairings", division)
	}

	err = divisionObject.DivisionManager.SetPairing(playerOneId, playerTwoId, round, false)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return TournamentSetPairingsEvent(ctx, ts, id, division)
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
	g *entity.Game) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()
	if t.Type == entity.TypeClub {
		// This game was played in a legacy "Clubhouse".
		// This is a tournament of "club" type (note, not a club *session*). This
		// is a casual type of tournament game with no defined divisions, pairings,
		// game settings, etc, so we bypass all of this code and just send a
		// tournament game ended message.

		p1user, err := us.GetByUUID(ctx, playerOneId)
		if err != nil {
			return err
		}
		p2user, err := us.GetByUUID(ctx, playerTwoId)
		if err != nil {
			return err
		}

		players := []*realtime.TournamentGameEndedEvent_Player{
			{Username: p1user.Username, Score: int32(playerOneScore), Result: playerOneResult},
			{Username: p2user.Username, Score: int32(playerTwoScore), Result: playerTwoResult},
		}

		tevt := &realtime.TournamentGameEndedEvent{
			GameId:    g.GameID(),
			Players:   players,
			EndReason: reason,
			Time:      time.Now().Unix(),
		}
		log.Debug().Interface("tevt", tevt).Msg("sending legacy tournament game ended evt")
		wrapped := entity.WrapEvent(tevt, realtime.MessageType_TOURNAMENT_GAME_ENDED_EVENT)
		wrapped.AddAudience(entity.AudTournament, id)
		evtChan := ts.TournamentEventChan()
		if evtChan != nil {
			evtChan <- wrapped
		}
		log.Debug().Str("tid", id).Msg("sent legacy tournament game ended event")
		return nil
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	if !t.IsStarted {
		return errors.New("cannot set tournament results before the tournament has started")
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
	return SendTournamentDivisionMessage(ctx, ts, id, division)
}

func StartTournament(ctx context.Context, ts TournamentStore, id string, manual bool) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	// Do not lock, StartRound will do that

	for division, _ := range t.Divisions {
		err := StartRoundCountdown(ctx, ts, id, division, 0, manual)
		if err != nil {
			return err
		}
		t.IsStarted = true
	}
	if !t.IsStarted {
		return fmt.Errorf("cannot start tournament %s with no divisions", t.Name)
	}
	return ts.Set(ctx, t)
}

func StartRoundCountdown(ctx context.Context, ts TournamentStore, id string, division string, round int, manual bool) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	if divisionObject.DivisionManager == nil {
		return fmt.Errorf("division %s does not have enough players or controls to set pairings", division)
	}

	if manual && divisionObject.Controls.AutoStart {
		return fmt.Errorf("division %s has autostart enabled and cannot be manually started", division)
	}

	ready, err := divisionObject.DivisionManager.IsRoundReady(round)
	if err != nil {
		return err
	}

	if ready {
		// Send some stuff to the channels
	} else {
		return fmt.Errorf("division %s is not ready to be started", division)
	}
	return nil
}

func PairRound(ctx context.Context, ts TournamentStore, id string, division string, round int) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	if !t.IsStarted {
		return errors.New("cannot pair a round before the tournament has started")
	}

	err = divisionObject.DivisionManager.PairRound(round)

	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
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
		return false, errors.New("cannot check if round is complete before the tournament has started")
	}

	_, ok := t.Divisions[division]

	if !ok {
		return false, fmt.Errorf("division %s does not exist", division)
	}

	return t.Divisions[division].DivisionManager.IsRoundComplete(round)
}

func IsFinished(ctx context.Context, ts TournamentStore, id string, division string) (bool, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return false, err
	}

	if !t.IsStarted {
		return false, errors.New("cannot check if tournament is finished before the tournament has started")
	}

	_, ok := t.Divisions[division]

	if !ok {
		return false, fmt.Errorf("division %s does not exist", division)
	}

	return t.Divisions[division].DivisionManager.IsFinished()
}

func TournamentDataResponse(ctx context.Context, ts TournamentStore, id string) (*realtime.TournamentDataResponse, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	directors := []string{}
	for key, _ := range t.Directors.Persons {
		directors = append(directors, key)
	}

	return &realtime.TournamentDataResponse{Id: t.UUID,
		Name:              t.Name,
		Description:       t.Description,
		ExecutiveDirector: t.ExecutiveDirector,
		Directors:         directors,
		IsStarted:         t.IsStarted}, nil
}

func TournamentDivisionDataResponse(ctx context.Context, ts TournamentStore,
	id string, division string) (*realtime.TournamentDivisionDataResponse, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return nil, fmt.Errorf("division %s does not exist", division)
	}

	response := &realtime.TournamentDivisionDataResponse{}
	if divisionObject.DivisionManager != nil {
		response, err = divisionObject.DivisionManager.ToResponse()
		if err != nil {
			return nil, err
		}
	}
	return response, nil
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
		return fmt.Errorf("division %s does not exist", division)
	}

	var personsMap map[string]int
	if isPlayers {
		personsMap = divisionObject.Players.Persons
	} else {
		personsMap = t.Directors.Persons
		if executiveDirectorExists(persons.Persons) {
			return errors.New("cannot add another executive director")
		}
	}

	// Only perform the add operation if all persons can be added.
	for k, _ := range persons.Persons {
		_, ok := personsMap[k]
		if ok {
			return fmt.Errorf("person (%s, %d) already exists", k, personsMap[k])
		}
	}

	for k, v := range persons.Persons {
		personsMap[k] = v
	}

	if t.IsStarted {
		err := divisionObject.DivisionManager.AddPlayers(persons)
		if err != nil {
			return err
		}
	} else if isPlayers {
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
		return fmt.Errorf("division %s does not exist", division)
	}

	if t.IsStarted && isPlayers {
		return errors.New("cannot cannot remove players after the tournament has started.")
	}

	var personsMap map[string]int
	if isPlayers {
		personsMap = divisionObject.Players.Persons
	} else {
		personsMap = t.Directors.Persons
		if executiveDirectorExists(persons.Persons) {
			return errors.New("cannot remove the executive director")
		}
	}

	// Only perform the remove operation if all persons can be removed.
	for k, _ := range persons.Persons {
		_, ok := personsMap[k]
		if !ok {
			return fmt.Errorf("person (%s, %d) does not exist", k, personsMap[k])
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
		return fmt.Errorf("division %s does not exist", division)
	}

	// Create a new division if there are sufficient players
	// otherwise, destroy the old one.
	if len(divisionObject.Players.Persons) > 1 &&
		divisionObject.Controls.NumberOfRounds > 0 &&
		divisionObject.Controls.RoundControls != nil {
		rankedPlayers := rankPlayers(divisionObject.Players)
		d, err := NewClassicDivision(rankedPlayers,
			divisionObject.Controls.RoundControls)
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
	err := errors.New("tournament must have exactly one executive director")
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
