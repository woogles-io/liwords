package tournament

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/pair"
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/rs/zerolog/log"
)

type PlayerSorter []*realtime.TournamentPerson

func (a PlayerSorter) Len() int           { return len(a) }
func (a PlayerSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a PlayerSorter) Less(i, j int) bool { return a[i].Rating > a[j].Rating }

type ClassicDivision struct {
	Matrix     [][]string                   `json:"matrix"`
	PairingMap map[string]*realtime.Pairing `json:"pairingMap"`
	// By convention, players should look like userUUID:username
	Players          *realtime.TournamentPersons `json:"players"`
	PlayerIndexMap   map[string]int32            `json:"pidxMap"`
	RoundControls    []*realtime.RoundControl    `json:"roundControls"`
	DivisionControls *realtime.DivisionControls  `json:"divisionControls"`
	CurrentRound     int32                       `json:"currentRound"`
	PairingKeyInt    int                         `json:"pairingKeyInt"`
}

func NewClassicDivision() *ClassicDivision {
	return &ClassicDivision{Matrix: [][]string{},
		PairingMap:       make(map[string]*realtime.Pairing),
		Players:          &realtime.TournamentPersons{},
		PlayerIndexMap:   make(map[string]int32),
		RoundControls:    []*realtime.RoundControl{},
		DivisionControls: &realtime.DivisionControls{},
		CurrentRound:     -1}
}

func (t *ClassicDivision) SetDivisionControls(divisionControls *realtime.DivisionControls) (*realtime.DivisionControls, error) {
	t.DivisionControls = divisionControls
	return t.DivisionControls, nil
}

func (t *ClassicDivision) SetRoundControls(roundControls []*realtime.RoundControl) (map[int][]*realtime.Pairing, []*realtime.RoundControl, error) {

	numberOfRounds := len(roundControls)
	numberOfPlayers := len(t.Players.Persons)

	if numberOfRounds <= 0 {
		return nil, nil, fmt.Errorf("cannot set round controls with empty array")
	}

	if t.CurrentRound >= 0 {
		return nil, nil, fmt.Errorf("cannot set all round controls after the tournament has started")
	}

	isElimination := false

	for i := 0; i < numberOfRounds; i++ {
		control := roundControls[i]
		if control.PairingMethod == realtime.PairingMethod_ELIMINATION {
			isElimination = true
			break
		}
	}

	var initialFontes int32 = 0
	for i := 0; i < numberOfRounds; i++ {
		control := roundControls[i]
		if isElimination && control.PairingMethod != realtime.PairingMethod_ELIMINATION {
			return nil, nil, errors.New("cannot mix Elimination pairings with any other pairing method")
		} else if i != 0 {
			if control.PairingMethod == realtime.PairingMethod_INITIAL_FONTES &&
				roundControls[i-1].PairingMethod != realtime.PairingMethod_INITIAL_FONTES {
				return nil, nil, errors.New("cannot use Initial Fontes pairing when an earlier round used a different pairing method")
			} else if control.PairingMethod != realtime.PairingMethod_INITIAL_FONTES &&
				roundControls[i-1].PairingMethod == realtime.PairingMethod_INITIAL_FONTES {
				initialFontes = int32(i)
			}
		}
	}

	if initialFontes > 0 && initialFontes%2 == 0 {
		return nil, nil, fmt.Errorf("number of rounds paired with Initial Fontes must be odd, have %d instead", initialFontes)
	}

	// For now, assume we require exactly n rounds and 2 ^ n players for an elimination tournament

	if roundControls[0].PairingMethod == realtime.PairingMethod_ELIMINATION {
		expectedNumberOfPlayers := 1 << numberOfRounds
		if expectedNumberOfPlayers != numberOfPlayers {
			return nil, nil, fmt.Errorf("invalid number of players based on the number of rounds: "+
				" have %d players, expected %d players based on the number of rounds which is %d",
				numberOfPlayers, expectedNumberOfPlayers, numberOfRounds)
		}
	}

	for i := 0; i < numberOfRounds; i++ {
		roundControls[i].InitialFontes = initialFontes
		roundControls[i].Round = int32(i)
	}
	t.RoundControls = roundControls
	t.Matrix = newPairingMatrix(len(t.RoundControls), len(t.Players.Persons))
	pairingsMap, err := t.prepair()
	return pairingsMap, t.RoundControls, err
}

func (t *ClassicDivision) prepair() (map[int][]*realtime.Pairing, error) {
	pairings := make(map[int][]*realtime.Pairing)
	if t.IsStartable() {
		if t.RoundControls[0].PairingMethod != realtime.PairingMethod_MANUAL {
			firstRoundPairings, err := t.PairRound(0)
			if err != nil {
				return nil, err
			}
			pairings = combinePairingsResponses(pairings, firstRoundPairings)
		}

		// We can make all standings independent pairings right now
		for i := 1; i < len(t.RoundControls); i++ {
			pm := t.RoundControls[i].PairingMethod
			if pair.IsStandingsIndependent(pm) && pm != realtime.PairingMethod_MANUAL {
				roundIPairings, err := t.PairRound(i)
				if err != nil {
					return nil, err
				}
				pairings = combinePairingsResponses(pairings, roundIPairings)
			}
		}
	}
	return pairings, nil
}

func (t *ClassicDivision) SetSingleRoundControls(round int, controls *realtime.RoundControl) (*realtime.RoundControl, error) {
	if round >= len(t.Matrix) || round < 0 {
		return nil, fmt.Errorf("round number out of range (SetSingleRoundControls): %d", round)
	}
	controls.Round = t.RoundControls[round].Round
	controls.InitialFontes = t.RoundControls[round].InitialFontes
	t.RoundControls[round] = controls
	return controls, nil
}

func (t *ClassicDivision) SetPairing(playerOne string, playerTwo string, round int) (map[int][]*realtime.Pairing, error) {

	playerOneIndex, ok := t.PlayerIndexMap[playerOne]
	if !ok {
		return nil, fmt.Errorf("playerOne does not exist in the division: >%s<", playerOne)
	}
	playerTwoIndex, ok := t.PlayerIndexMap[playerTwo]
	if !ok {
		return nil, fmt.Errorf("playerTwo does not exist in the division: >%s<", playerTwo)
	}

	playerOneOpponent, err := t.opponentOf(playerOne, round)
	if err != nil {
		return nil, err
	}

	playerTwoOpponent, err := t.opponentOf(playerTwo, round)
	if err != nil {
		return nil, err
	}

	playerOneOpponentIndex, ok := t.PlayerIndexMap[playerOneOpponent]
	if playerOneOpponent != "" && !ok {
		return nil, fmt.Errorf("playerOneOpponent does not exist in the division: >%s<", playerOneOpponent)
	}
	playerTwoOpponentIndex, ok := t.PlayerIndexMap[playerTwoOpponent]
	if playerTwoOpponent != "" && !ok {
		return nil, fmt.Errorf("playerTwoOpponent does not exist in the division: >%s<", playerTwoOpponent)
	}

	if playerOneOpponent != "" {
		err = t.clearPairingKey(playerOneOpponentIndex, round)
		if err != nil {
			return nil, err
		}
	}

	if playerTwoOpponent != "" {
		err = t.clearPairingKey(playerTwoOpponentIndex, round)
		if err != nil {
			return nil, err
		}
	}

	newPairing := newClassicPairing(t, playerOneIndex, playerTwoIndex, round)
	pairingKey := t.makePairingKey()
	t.PairingMap[pairingKey] = newPairing

	err = t.setPairingKey(playerOne, round, pairingKey)
	if err != nil {
		return nil, err
	}

	err = t.setPairingKey(playerTwo, round, pairingKey)
	if err != nil {
		return nil, err
	}

	pairingResponse := map[int][]*realtime.Pairing{round: []*realtime.Pairing{newPairing}}

	// This pairing is a bye or forfeit, the result
	// can be submitted immediately
	if playerOne == playerTwo ||
		t.Players.Persons[playerOneIndex].Suspended ||
		t.Players.Persons[playerTwoIndex].Suspended {

		// Cases are:
		// p1 bye
		// p1 suspension loss
		// p1 forfeit loss, p2 forfeit win
		// p1 forfeit win, p2 forfeit loss
		// p1 forfeit loss, p2 forfeit loss

		var p1score int
		var p2score int
		var p1tgr realtime.TournamentGameResult
		var p2tgr realtime.TournamentGameResult

		if playerOne == playerTwo {
			if t.Players.Persons[playerOneIndex].Suspended {
				p1score = int(t.DivisionControls.SuspendedSpread)
				p2score = 0
				p1tgr = t.DivisionControls.SuspendedResult
				p2tgr = t.DivisionControls.SuspendedResult
			} else {
				p1score = entity.ByeScore
				p2score = 0
				p1tgr = realtime.TournamentGameResult_BYE
				p2tgr = realtime.TournamentGameResult_BYE
			}
		} else {
			p1score = 0
			p2score = 0
			p1tgr = realtime.TournamentGameResult_FORFEIT_WIN
			p2tgr = realtime.TournamentGameResult_FORFEIT_WIN
			if t.Players.Persons[playerOneIndex].Suspended {
				p1score = int(t.DivisionControls.SuspendedSpread)
				p1tgr = t.DivisionControls.SuspendedResult
			}
			if t.Players.Persons[playerTwoIndex].Suspended {
				p2score = int(t.DivisionControls.SuspendedSpread)
				p2tgr = t.DivisionControls.SuspendedResult
			}
		}

		// Use round < t.CurrentRound to satisfy
		// amendment checking. These results always need
		// to be submitted.
		pairing, err := t.SubmitResult(round,
			playerOne,
			playerTwo,
			p1score,
			p2score,
			p1tgr,
			p2tgr,
			realtime.GameEndReason_NONE,
			round < int(t.CurrentRound),
			0,
			"")
		if err != nil {
			return nil, err
		}
		pairingResponse = pairing
	}
	return pairingResponse, nil
}

func (t *ClassicDivision) SubmitResult(round int,
	p1 string,
	p2 string,
	p1Score int,
	p2Score int,
	p1Result realtime.TournamentGameResult,
	p2Result realtime.TournamentGameResult,
	reason realtime.GameEndReason,
	amend bool,
	gameIndex int,
	id string) (map[int][]*realtime.Pairing, error) {

	// Fetch the player round records
	pk1, err := t.getPairingKey(p1, round)
	if err != nil {
		return nil, err
	}
	pk2, err := t.getPairingKey(p2, round)
	if err != nil {
		return nil, err
	}

	// Ensure that this is the current round
	if round < int(t.CurrentRound) && !amend {
		return nil, fmt.Errorf("submitted result for a past round (%d) that is not marked as an amendment", round)
	}

	if round > int(t.CurrentRound) && (!isByeOrForfeit(p1Result) || !isByeOrForfeit(p2Result)) {
		return nil, fmt.Errorf("submitted result for a future round (%d) that is not a bye or forfeit", round)
	}

	// Ensure that the pairing exists
	if pk1 == "" {
		return nil, fmt.Errorf("submitted result for a player with a null pairing: %s round (%d)", p1, round)
	}

	if pk2 == "" {
		return nil, fmt.Errorf("submitted result for a player with a null pairing: %s round (%d)", p2, round)
	}

	// Ensure the submitted results were for players that were paired
	if pk1 != pk2 {
		log.Debug().Interface("pr1", pk1).Interface("pri2", pk2).Msg("not-play")
		return nil, fmt.Errorf("submitted result for players that didn't play each other: %s (%s), %s (%s) round (%d)", p1, pk1, p2, pk2, round)
	}

	pairing, ok := t.PairingMap[pk1]
	if !ok {
		return nil, fmt.Errorf("pairing does not exist in the pairing map: %s", pk1)
	}
	pairingMethod := t.RoundControls[round].PairingMethod

	if pairing.Games == nil {
		return nil, fmt.Errorf("submitted result for a pairing with no initialized games: %s (%s), %s (%s) round (%d)", p1, pk1, p2, pk2, round)
	}

	// For Elimination tournaments only.
	// Could be a tiebreaking result or could be an out of range
	// game index
	if pairingMethod == realtime.PairingMethod_ELIMINATION && gameIndex >= int(t.RoundControls[round].GamesPerRound) {
		if gameIndex != len(pairing.Games) {
			return nil, fmt.Errorf("submitted tiebreaking result with invalid game index."+
				" Player 1: %s, Player 2: %s, Round: %d, GameIndex: %d", p1, p2, round, gameIndex)
		} else {
			pairing.Games = append(pairing.Games,
				&realtime.TournamentGame{Scores: []int32{0, 0},
					Results: []realtime.TournamentGameResult{realtime.TournamentGameResult_NO_RESULT,
						realtime.TournamentGameResult_NO_RESULT}})
		}
	}

	if gameIndex >= len(pairing.Games) {
		return nil, fmt.Errorf("submitted result where game index is out of range: %d >= %d", gameIndex, len(pairing.Games))
	}

	// If this is not an amendment, but attempts to amend a result, reject
	// this submission.
	if !amend && ((pairing.Outcomes[0] != realtime.TournamentGameResult_NO_RESULT &&
		pairing.Outcomes[1] != realtime.TournamentGameResult_NO_RESULT) ||

		pairing.Games[gameIndex].Results[0] != realtime.TournamentGameResult_NO_RESULT &&
			pairing.Games[gameIndex].Results[1] != realtime.TournamentGameResult_NO_RESULT) {
		return nil, fmt.Errorf("result is already submitted for round %d, %s vs. %s", round, p1, p2)
	}

	// If this claims to be an amendment and is not submitting forfeit
	// losses for players show up late, reject this submission.
	if amend && p1Result != realtime.TournamentGameResult_FORFEIT_LOSS &&
		p2Result != realtime.TournamentGameResult_FORFEIT_LOSS &&
		pairing.Games[gameIndex].Results[0] == realtime.TournamentGameResult_NO_RESULT &&
		pairing.Games[gameIndex].Results[1] == realtime.TournamentGameResult_NO_RESULT {
		return nil, fmt.Errorf("submitted amendment for a result that does not exist in round %d, %s vs. %s", round, p1, p2)
	}

	p1Index := 0
	if pairing.Players[1] == t.PlayerIndexMap[p1] {
		p1Index = 1
	}

	if pairingMethod == realtime.PairingMethod_ELIMINATION {
		pairing.Games[gameIndex].Scores[p1Index] = int32(p1Score)
		pairing.Games[gameIndex].Scores[1-p1Index] = int32(p2Score)
		pairing.Games[gameIndex].Results[p1Index] = p1Result
		pairing.Games[gameIndex].Results[1-p1Index] = p2Result
		pairing.Games[gameIndex].GameEndReason = reason
		pairing.Games[gameIndex].Id = id

		// Get elimination outcomes will take care of the indexing
		// for us because the newOutcomes are aligned with the data
		// in pairing.Games
		newOutcomes := getEliminationOutcomes(pairing.Games, t.RoundControls[round].GamesPerRound)

		pairing.Outcomes[0] = newOutcomes[0]
		pairing.Outcomes[1] = newOutcomes[1]
	} else {
		// Classic tournaments only ever have
		// one game per round
		pairing.Games[0].Scores[p1Index] = int32(p1Score)
		pairing.Games[0].Scores[1-p1Index] = int32(p2Score)
		pairing.Games[0].Results[p1Index] = p1Result
		pairing.Games[0].Results[1-p1Index] = p2Result
		pairing.Games[0].GameEndReason = reason
		pairing.Games[0].Id = id
		pairing.Outcomes[p1Index] = p1Result
		pairing.Outcomes[1-p1Index] = p2Result
	}

	roundComplete, err := t.IsRoundComplete(round)
	if err != nil {
		return nil, err
	}
	finished, err := t.IsFinished()
	if err != nil {
		return nil, err
	}

	response := make(map[int][]*realtime.Pairing)

	response[round] = []*realtime.Pairing{pairing}
	// Only pair if this round is complete and the tournament
	// is not over. Don't pair for standings independent pairings since those pairings
	// were made when the tournament was created.
	if roundComplete && !finished && !amend {
		if !pair.IsStandingsIndependent(t.RoundControls[round+1].PairingMethod) {
			pairRoundResponse, err := t.PairRound(round + 1)
			if err != nil {
				return nil, err
			}
			response = combinePairingsResponses(response, pairRoundResponse)
		}
		if t.DivisionControls.AutoStart {
			err = t.StartRound()
			if err != nil {
				return nil, err
			}
		}
	}
	return response, nil
}

func (t *ClassicDivision) PairRound(round int) (map[int][]*realtime.Pairing, error) {
	if round < 0 || round >= len(t.Matrix) {
		return nil, fmt.Errorf("round number out of range (PairRound): %d", round)
	}
	roundPairings := t.Matrix[round]
	pairingMethod := t.RoundControls[round].PairingMethod
	// This automatic pairing could be the result of an
	// amendment. Undo all the pairings so byes can be
	// properly assigned (bye assignment checks for nil pairing).
	for i := 0; i < len(roundPairings); i++ {
		t.clearPairingKey(t.PlayerIndexMap[t.Players.Persons[i].Id], round)
	}

	standingsRound := round
	if standingsRound == 0 {
		standingsRound = 1
	}

	standings, err := t.GetStandings(standingsRound - 1)
	if err != nil {
		return nil, err
	}

	poolMembers := []*entity.PoolMember{}

	// Round Robin must have the same ordering for each round
	var playerOrder []string
	if pairingMethod == realtime.PairingMethod_ROUND_ROBIN {
		for i := 0; i < len(t.Players.Persons); i++ {
			playerOrder = append(playerOrder, t.Players.Persons[i].Id)
		}
	} else {
		playerOrder = make([]string, len(standings))
		for i := 0; i < len(standings); i++ {
			playerOrder[i] = standings[i].PlayerId
		}
	}

	for i := 0; i < len(playerOrder); i++ {
		pm := &entity.PoolMember{Id: playerOrder[i]}
		// Wins do not matter for RoundRobin pairings
		if pairingMethod != realtime.PairingMethod_ROUND_ROBIN {
			pm.Wins = int(standings[i].Wins)
			pm.Draws = int(standings[i].Draws)
			pm.Spread = int(standings[i].Spread)
		} else {
			pm.Wins = 0
			pm.Draws = 0
			pm.Spread = 0
		}
		poolMembers = append(poolMembers, pm)
	}

	repeats, err := getRepeats(t, round-1)
	if err != nil {
		return nil, err
	}

	upm := &entity.UnpairedPoolMembers{RoundControls: t.RoundControls[round],
		PoolMembers: poolMembers,
		Repeats:     repeats}

	pairings, err := pair.Pair(upm)

	if err != nil {
		return nil, err
	}

	l := len(pairings)

	if l != len(playerOrder) {
		return nil, errors.New("pair did not return the correct number of pairings")
	}

	response := make(map[int][]*realtime.Pairing)

	for i := 0; i < l; i++ {
		// Player order might be a different order than the players in roundPairings
		playerIndex := t.PlayerIndexMap[playerOrder[i]]
		if pairingMethod != realtime.PairingMethod_ROUND_ROBIN &&
			t.Players.Persons[playerIndex].Suspended {
			return nil, fmt.Errorf("suspended player %s in the pool", t.Players.Persons[playerIndex].Id)
		}
		if roundPairings[playerIndex] == "" {

			var opponentIndex int32
			if pairings[i] < 0 {
				opponentIndex = playerIndex
			} else if pairings[i] >= l {
				return nil, fmt.Errorf("invalid pairing for round %d: %d", round, pairings[i])
			} else {
				opponentIndex = t.PlayerIndexMap[playerOrder[pairings[i]]]
			}

			playerId := t.Players.Persons[playerIndex].Id
			opponentId := t.Players.Persons[opponentIndex].Id
			nextResponse := make(map[int][]*realtime.Pairing)
			if pairingMethod == realtime.PairingMethod_ELIMINATION && round > 0 && i >= l>>round {
				pairingKey := t.makePairingKey()
				pairing := newEliminatedPairing(playerId, opponentId)
				t.PairingMap[pairingKey] = pairing
				roundPairings[playerIndex] = pairingKey
				nextResponse[round] = []*realtime.Pairing{pairing}
			} else {
				nextResponse, err = t.SetPairing(playerId, opponentId, round)
				if err != nil {
					return nil, err
				}
			}
			response = combinePairingsResponses(response, nextResponse)
		}
	}

	for i := 0; i < len(t.Players.Persons); i++ {
		player := t.Players.Persons[i]
		if pairingMethod != realtime.PairingMethod_ROUND_ROBIN && player.Suspended && roundPairings[i] != "" {
			return nil, fmt.Errorf("suspended player %s was paired with key %s", player.Id, roundPairings[i])
		}

		if !player.Suspended && roundPairings[i] == "" {
			return nil, fmt.Errorf("active player %s was not paired", player.Id)
		}
		if pairingMethod != realtime.PairingMethod_ROUND_ROBIN && player.Suspended {
			byeResponse, err := t.SetPairing(player.Id, player.Id, round)
			if err != nil {
				return nil, err
			}
			response = combinePairingsResponses(response, byeResponse)
		}
	}

	err = validatePairings(t, round)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (t *ClassicDivision) AddPlayers(players *realtime.TournamentPersons) (map[int][]*realtime.Pairing, error) {

	playerExists := false
	numNewPlayers := 0
	for _, player := range players.Persons {
		idx, ok := t.PlayerIndexMap[player.Id]
		playerExists = ok
		// If the player exists and is not suspended or the tournament hasn't started
		// throw an error
		if ok && (!t.Players.Persons[idx].Suspended || t.CurrentRound < 0) {
			return nil, fmt.Errorf("player %s already exists in the tournament", player)
		}
		if !ok {
			numNewPlayers++
		}
	}

	response := make(map[int][]*realtime.Pairing)

	if t.CurrentRound < 0 {
		t.Players.Persons = append(t.Players.Persons, players.Persons...)
		sort.Sort(PlayerSorter(t.Players.Persons))
		t.PlayerIndexMap = newPlayerIndexMap(t.Players.Persons)
		t.Matrix = newPairingMatrix(len(t.RoundControls), len(t.Players.Persons))
		newPairings, err := t.prepair()
		if err != nil {
			return nil, err
		}
		response = combinePairingsResponses(response, newPairings)
	} else {
		if int(t.CurrentRound) == len(t.Matrix)-1 {
			return nil, fmt.Errorf("cannot add players because the last round (%d) has already started", t.CurrentRound)
		}
		for _, player := range players.Persons {
			if !playerExists {
				t.Players.Persons = append(t.Players.Persons, player)
				t.PlayerIndexMap[player.Id] = int32(len(t.Players.Persons) - 1)
			} else {
				_, ok := t.PlayerIndexMap[player.Id]
				if !ok {
					return nil, fmt.Errorf("player %s marked as existing but not in the index map", player.Id)
				}
			}
			// When first adding players, first temporarily mark
			// them as suspended so that for all past rounds
			// they receive the proper suspended result for joining late
			t.Players.Persons[t.PlayerIndexMap[player.Id]].Suspended = true
		}

		for i := 0; i < len(t.Matrix); i++ {
			for k := 0; k < numNewPlayers; k++ {
				t.Matrix[i] = append(t.Matrix[i], "")
			}
		}

		for i := 0; i < len(t.Matrix); i++ {
			if i <= int(t.CurrentRound) {
				for _, player := range players.Persons {
					// Set the pairing
					// This also automatically submits a forfeit result
					newPairings, err := t.SetPairing(player.Id, player.Id, i)
					if i == int(t.CurrentRound) {
						// At this point all past rounds have been paired.
						// Now mark all new players as not suspended so that
						// for future pairings they don't get suspended results
						t.Players.Persons[t.PlayerIndexMap[player.Id]].Suspended = false
					}
					if err != nil {
						return nil, err
					}
					response = combinePairingsResponses(response, newPairings)
				}
			} else {
				pm := t.RoundControls[i].PairingMethod
				if (i == int(t.CurrentRound)+1 || pair.IsStandingsIndependent(pm)) && pm != realtime.PairingMethod_MANUAL {
					newPairings, err := t.PairRound(i)
					if err != nil {
						return nil, err
					}
					response = combinePairingsResponses(response, newPairings)
				}
			}
		}
	}
	return response, nil
}

func (t *ClassicDivision) RemovePlayers(persons *realtime.TournamentPersons) (map[int][]*realtime.Pairing, error) {
	for _, player := range persons.Persons {
		playerIndex, ok := t.PlayerIndexMap[player.Id]
		if !ok {
			return nil, fmt.Errorf("player %s does not exist in"+
				" classic division", player.Id)
		}
		if playerIndex < 0 || int(playerIndex) >= len(t.Players.Persons) {
			return nil, fmt.Errorf("player index %d for player %s is"+
				" out of range in classic division", playerIndex, player.Id)
		}
		if t.Players.Persons[playerIndex].Suspended {
			return nil, fmt.Errorf("player %s has already been removed", player.Id)
		}
	}

	response := make(map[int][]*realtime.Pairing)

	if t.CurrentRound < 0 {
		var err error
		t.Players, err = removeTournamentPersons(t.Players, persons, false)
		if err != nil {
			return nil, err
		}
		sort.Sort(PlayerSorter(t.Players.Persons))
		t.PlayerIndexMap = newPlayerIndexMap(t.Players.Persons)
		t.Matrix = newPairingMatrix(len(t.RoundControls), len(t.Players.Persons))
		newPairings, err := t.prepair()
		if err != nil {
			return nil, err
		}
		response = combinePairingsResponses(response, newPairings)
	} else {
		playersRemoved := 0
		for i := 0; i < len(t.Players.Persons); i++ {
			if t.Players.Persons[i].Suspended {
				playersRemoved++
			}
		}

		if playersRemoved+len(persons.Persons) >= len(t.Players.Persons) {
			return nil, fmt.Errorf("cannot remove players as tournament would be empty")
		}

		for _, player := range t.Players.Persons {
			for _, removedPlayer := range persons.Persons {
				if player.Id == removedPlayer.Id {
					player.Suspended = true
				}
			}
		}

		for i := int(t.CurrentRound + 1); i < len(t.Matrix); i++ {
			pm := t.RoundControls[i].PairingMethod
			if (i == int(t.CurrentRound)+1 || pair.IsStandingsIndependent(pm)) && pm != realtime.PairingMethod_MANUAL {
				newPairings, err := t.PairRound(i)
				if err != nil {
					return nil, err
				}
				response = combinePairingsResponses(response, newPairings)
			}
		}
	}

	return response, nil
}

func (t *ClassicDivision) GetCurrentRound() int {
	return int(t.CurrentRound)
}

func (t *ClassicDivision) GetStandings(round int) ([]*realtime.PlayerStanding, error) {
	if round < 0 || round >= len(t.Matrix) {
		return nil, fmt.Errorf("round number out of range (GetStandings): %d", round)
	}

	var wins int32 = 0
	var losses int32 = 0
	var draws int32 = 0
	var spread int32 = 0
	playerId := ""
	records := []*realtime.PlayerStanding{}

	for i := 0; i < len(t.Players.Persons); i++ {
		wins = 0
		losses = 0
		draws = 0
		spread = 0
		playerId = t.Players.Persons[i].Id
		if t.Players.Persons[i].Suspended {
			continue
		}
		for j := 0; j <= round; j++ {
			pairingKey := t.Matrix[j][i]
			pairing, ok := t.PairingMap[pairingKey]
			if ok && pairing != nil && pairing.Players != nil {
				playerIndex := 0
				if t.Players.Persons[pairing.Players[1]].Id == playerId {
					playerIndex = 1
				}
				if pairing.Outcomes[playerIndex] != realtime.TournamentGameResult_NO_RESULT {
					result := convertResult(pairing.Outcomes[playerIndex])
					if result == 2 {
						wins++
					} else if result == 0 {
						losses++
					} else {
						draws++
					}
					for k := 0; k < len(pairing.Games); k++ {
						spread += pairing.Games[k].Scores[playerIndex] -
							pairing.Games[k].Scores[1-playerIndex]
					}
				}
			}
		}
		records = append(records, &realtime.PlayerStanding{PlayerId: playerId,
			Wins:   wins,
			Losses: losses,
			Draws:  draws,
			Spread: spread})
	}

	pairingMethod := t.RoundControls[round].PairingMethod

	// The difference for Elimination is that the original order
	// of the player list must be preserved. This is how we keep
	// track of the "bracket", which is simply modeled by an
	// array in this implementation. To keep this order, the
	// index in the tournament matrix is used as a tie breaker
	// for wins. In this way, The groupings are preserved across
	// rounds.
	if pairingMethod == realtime.PairingMethod_ELIMINATION {
		sort.Slice(records,
			func(i, j int) bool {
				if records[i].Wins == records[j].Wins {
					return i < j
				} else {
					return records[i].Wins > records[j].Wins
				}
			})
	} else {
		sort.Slice(records,
			func(i, j int) bool {
				if records[i].Wins == records[j].Wins && records[i].Draws == records[j].Draws && records[i].Spread == records[j].Spread {
					// Tiebreak by rank to ensure determinism
					return t.PlayerIndexMap[records[j].PlayerId] > t.PlayerIndexMap[records[i].PlayerId]
				} else if records[i].Wins == records[j].Wins && records[i].Draws == records[j].Draws {
					return records[i].Spread > records[j].Spread
				} else if records[i].Wins == records[j].Wins {
					return records[i].Draws > records[j].Draws
				} else {
					return records[i].Wins > records[j].Wins
				}
			})
	}

	return records, nil
}

func (t *ClassicDivision) IsRoundReady(round int) (bool, error) {
	if round >= len(t.Matrix) || round < 0 {
		return false, fmt.Errorf("round number out of rang (IsRoundReady): %d", round)
	}
	// Check that everyone is paired
	for _, pairingKey := range t.Matrix[round] {
		if pairingKey == "" {
			return false, nil
		}
		_, ok := t.PairingMap[pairingKey]
		if !ok {
			return false, fmt.Errorf("pairing does not exist in the pairing map (IsRoundReady): %s", pairingKey)
		}
	}
	// Check that all previous rounds are complete
	for i := 0; i <= round-1; i++ {
		complete, err := t.IsRoundComplete(i)
		if err != nil || !complete {
			return false, err
		}
	}
	return true, nil
}

func (t *ClassicDivision) StartRound() error {
	if t.CurrentRound >= 0 {
		roundComplete, err := t.IsRoundComplete(int(t.CurrentRound))
		if err != nil {
			return err
		}
		if !roundComplete {
			return fmt.Errorf("cannot start the next round as round %d is not complete", t.CurrentRound)
		}
		isFinished, err := t.IsFinished()
		if err != nil {
			return err
		}
		if isFinished {
			return fmt.Errorf("cannot start round %d because the tournament is finished", t.CurrentRound+1)
		}
	} else if !t.IsStartable() {
		return errors.New("cannot start the the tournament with less than 2 players or less than 1 round")
	}

	ready, err := t.IsRoundReady(int(t.CurrentRound + 1))
	if err != nil {
		return err
	}
	if !ready {
		return fmt.Errorf("cannot start round %d because it is not ready", t.CurrentRound+1)
	}

	t.CurrentRound = t.CurrentRound + 1

	return nil
}

// SetReadyForGame sets the playerID with the given connID to be ready for the game
// with the given 0-based round (and gameIndex, optionally). If `unready` is
// passed in, we make the player unready.
// It returns a list of playerId:username:connIDs involved in the game, a boolean saying if they're ready,
// and an optional error.
func (t *ClassicDivision) SetReadyForGame(playerID, connID string, round, gameIndex int, unready bool) ([]string, bool, error) {
	if round >= len(t.Matrix) || round < 0 {
		return nil, false, fmt.Errorf("round number out of range (SetReadyForGame): %d", round)
	}
	toSet := connID
	if unready {
		toSet = ""
	}
	if int(t.CurrentRound) != round {
		return nil, false, errors.New("wrong round number")
	}
	// gameIndex is ignored for ClassicDivision?
	pairingKey, err := t.getPairingKey(playerID, round)
	if err != nil {
		return nil, false, err
	}

	if pairingKey != "" {
		pairing, ok := t.PairingMap[pairingKey]
		if !ok {
			return nil, false, fmt.Errorf("pairing does not exist in the pairing map (SetReadyForGame): %s", pairingKey)
		}

		for idx := range pairing.Players {
			if int(pairing.Players[idx]) >= len(t.Players.Persons) || pairing.Players[idx] < 0 {
				return nil, false, fmt.Errorf("cannot set ready for game, player %d index out of range", pairing.Players[idx])
			}
			pairingPlayerID := t.Players.Persons[pairing.Players[idx]].Id
			if playerID == pairingPlayerID {
				pairing.ReadyStates[idx] = toSet
			}
		}
		playerOneId := t.Players.Persons[pairing.Players[0]].Id
		playerTwoId := t.Players.Persons[pairing.Players[1]].Id
		// Check to see if both players are ready.
		involvedPlayers := []string{
			playerOneId + ":" + pairing.ReadyStates[0],
			playerTwoId + ":" + pairing.ReadyStates[1],
		}
		bothReady := pairing.ReadyStates[0] != "" && pairing.ReadyStates[1] != ""
		return involvedPlayers, bothReady, nil
	}
	return nil, false, nil
}

func (t *ClassicDivision) ClearReadyStates(playerID string, round, gameIndex int) (map[int][]*realtime.Pairing, error) {
	if round >= len(t.Matrix) || round < 0 {
		return nil, fmt.Errorf("round number out of range (ClearReadyStates): %d", round)
	}
	// ignore gameIndex for classicdivision
	p, err := t.getPairing(playerID, round)
	if err != nil {
		return nil, err
	}
	p.ReadyStates = []string{"", ""}
	return map[int][]*realtime.Pairing{round: []*realtime.Pairing{p}}, nil
}

func (t *ClassicDivision) IsRoundComplete(round int) (bool, error) {
	if round >= len(t.Matrix) || round < 0 {
		return false, fmt.Errorf("round number out of range (IsRoundComplete): %d", round)
	}
	for _, pairingKey := range t.Matrix[round] {
		pairing, ok := t.PairingMap[pairingKey]
		if !ok {
			return false, nil
		}
		if pairing.Outcomes[0] == realtime.TournamentGameResult_NO_RESULT ||
			pairing.Outcomes[1] == realtime.TournamentGameResult_NO_RESULT {
			return false, nil
		}
	}
	return true, nil
}

func (t *ClassicDivision) IsFinished() (bool, error) {
	if len(t.Matrix) < 1 {
		return false, nil
	}
	complete, err := t.IsRoundComplete(len(t.Matrix) - 1)
	if err != nil {
		return false, err
	}
	return complete, nil
}

func (t *ClassicDivision) IsStarted() bool {
	return t.CurrentRound >= 0
}

func (t *ClassicDivision) IsStartable() bool {
	return len(t.Players.Persons) >= 2 && len(t.Matrix) >= 1
}

func (t *ClassicDivision) GetXHRResponse() (*realtime.TournamentDivisionDataResponse, error) {
	return &realtime.TournamentDivisionDataResponse{Players: t.Players,
		Controls:      t.DivisionControls,
		RoundControls: t.RoundControls,
		PairingMap:    t.PairingMap,
		CurrentRound:  t.CurrentRound}, nil
}

func newPairingMatrix(numberOfRounds int, numberOfPlayers int) [][]string {
	pairings := [][]string{}
	for i := 0; i < numberOfRounds; i++ {
		roundPairings := []string{}
		for j := 0; j < numberOfPlayers; j++ {
			roundPairings = append(roundPairings, "")
		}
		pairings = append(pairings, roundPairings)
	}
	return pairings
}

func newClassicPairing(t *ClassicDivision,
	playerOne int32,
	playerTwo int32,
	round int) *realtime.Pairing {

	games := []*realtime.TournamentGame{}
	for i := 0; i < int(t.RoundControls[round].GamesPerRound); i++ {
		games = append(games, &realtime.TournamentGame{Scores: []int32{0, 0},
			Results: []realtime.TournamentGameResult{realtime.TournamentGameResult_NO_RESULT,
				realtime.TournamentGameResult_NO_RESULT},
			Id: ""})
	}

	playerGoingFirst := playerOne
	playerGoingSecond := playerTwo
	switchFirst := false
	firstMethod := t.RoundControls[round].FirstMethod

	if firstMethod != realtime.FirstMethod_MANUAL_FIRST {
		playerOneFS := getPlayerFirstsAndSeconds(t, playerGoingFirst, round-1)
		playerTwoFS := getPlayerFirstsAndSeconds(t, playerGoingSecond, round-1)
		if firstMethod == realtime.FirstMethod_RANDOM_FIRST {
			switchFirst = rand.Intn(2) == 0
		} else { // AutomaticFirst
			if playerOneFS[0] != playerTwoFS[0] {
				switchFirst = playerOneFS[0] > playerTwoFS[0]
			} else if playerOneFS[1] != playerTwoFS[1] {
				switchFirst = playerOneFS[1] < playerTwoFS[1]
			} else {
				// Might want to use head-to-head in the future to break this up
				switchFirst = rand.Intn(2) == 0
			}
		}
	}

	if switchFirst {
		playerGoingFirst, playerGoingSecond = playerGoingSecond, playerGoingFirst
	}

	return &realtime.Pairing{Players: []int32{playerGoingFirst, playerGoingSecond},
		Games: games,
		Outcomes: []realtime.TournamentGameResult{realtime.TournamentGameResult_NO_RESULT,
			realtime.TournamentGameResult_NO_RESULT},
		ReadyStates: []string{"", ""},
	}
}

func getPlayerFirstsAndSeconds(t *ClassicDivision, playerIndex int32, round int) []int {

	fs := []int{0, 0}

	// Maybe add error throwing later
	if round >= len(t.Matrix) || round < 0 {
		return fs
	}

	if int(playerIndex) >= len(t.Players.Persons) || playerIndex < 0 {
		return fs
	}

	for i := 0; i <= round; i++ {
		pairingKey := t.Matrix[i][int(playerIndex)]
		if pairingKey != "" {
			pairing, ok := t.PairingMap[pairingKey]
			if ok {
				playerPairingIndex := 0
				if pairing.Players[1] == playerIndex {
					playerPairingIndex = 1
				} else if pairing.Players[0] != playerIndex {
					return fs
				}
				outcome := pairing.Outcomes[playerPairingIndex]
				if outcome == realtime.TournamentGameResult_NO_RESULT ||
					outcome == realtime.TournamentGameResult_WIN ||
					outcome == realtime.TournamentGameResult_LOSS ||
					outcome == realtime.TournamentGameResult_DRAW {
					fs[playerPairingIndex]++
				}
			}
		}
	}
	return fs
}

func newEliminatedPairing(playerOne string, playerTwo string) *realtime.Pairing {
	return &realtime.Pairing{Outcomes: []realtime.TournamentGameResult{realtime.TournamentGameResult_ELIMINATED,
		realtime.TournamentGameResult_ELIMINATED}}
}

func newPlayerIndexMap(players []*realtime.TournamentPerson) map[string]int32 {
	m := make(map[string]int32)
	for i, player := range players {
		m[player.Id] = int32(i)
	}
	return m
}

func getRepeats(t *ClassicDivision, round int) (map[string]int, error) {
	if round >= len(t.Matrix) {
		return nil, fmt.Errorf("round number out of range (getRepeats): %d", round)
	}
	repeats := make(map[string]int)
	for i := 0; i <= round; i++ {
		roundPairings := t.Matrix[i]
		for _, pairingKey := range roundPairings {
			pairing, ok := t.PairingMap[pairingKey]
			if ok && pairing.Players != nil {
				playerOne := t.Players.Persons[pairing.Players[0]]
				playerTwo := t.Players.Persons[pairing.Players[1]]
				if playerOne != playerTwo {
					key := pair.GetRepeatKey(playerOne.Id, playerTwo.Id)
					repeats[key]++
				}
			}
		}
	}

	// All repeats have been counted twice at this point
	// so divide by two.
	for key, _ := range repeats {
		repeats[key] = repeats[key] / 2
	}
	return repeats, nil
}

func getEliminationOutcomes(games []*realtime.TournamentGame, gamesPerRound int32) []realtime.TournamentGameResult {
	// Determines if a player is eliminated for a given round in an
	// elimination tournament. The convertResult function gives 2 for a win,
	// 1 for a draw, and 0 otherwise. If a player's score is greater than
	// the games per round, they have won, unless there is a tie.
	var p1Wins int32 = 0
	var p2Wins int32 = 0
	var p1Spread int32 = 0
	var p2Spread int32 = 0
	for _, game := range games {
		p1Wins += convertResult(game.Results[0])
		p2Wins += convertResult(game.Results[1])
		p1Spread += game.Scores[0] - game.Scores[1]
		p2Spread += game.Scores[1] - game.Scores[0]
	}

	p1Outcome := realtime.TournamentGameResult_NO_RESULT
	p2Outcome := realtime.TournamentGameResult_NO_RESULT

	// In case of a tie by spread, more games need to be
	// submitted to break the tie. In the future we
	// might want to allow for Elimination tournaments
	// to disregard spread as a tiebreak entirely, but
	// this is an extreme edge case.
	if len(games) > int(gamesPerRound) { // Tiebreaking results are present
		if p1Wins > p2Wins ||
			(p1Wins == p2Wins && p1Spread > p2Spread) {
			p1Outcome = realtime.TournamentGameResult_WIN
			p2Outcome = realtime.TournamentGameResult_ELIMINATED
		} else if p2Wins > p1Wins ||
			(p2Wins == p1Wins && p2Spread > p1Spread) {
			p1Outcome = realtime.TournamentGameResult_ELIMINATED
			p2Outcome = realtime.TournamentGameResult_WIN
		}
	} else {
		if p1Wins > gamesPerRound ||
			(p1Wins == gamesPerRound && p2Wins == gamesPerRound && p1Spread > p2Spread) {
			p1Outcome = realtime.TournamentGameResult_WIN
			p2Outcome = realtime.TournamentGameResult_ELIMINATED
		} else if p2Wins > gamesPerRound ||
			(p1Wins == gamesPerRound && p2Wins == gamesPerRound && p1Spread < p2Spread) {
			p1Outcome = realtime.TournamentGameResult_ELIMINATED
			p2Outcome = realtime.TournamentGameResult_WIN
		}
	}
	return []realtime.TournamentGameResult{p1Outcome, p2Outcome}
}

func convertResult(result realtime.TournamentGameResult) int32 {
	var convertedResult int32 = 0
	if result == realtime.TournamentGameResult_WIN || result == realtime.TournamentGameResult_BYE || result == realtime.TournamentGameResult_FORFEIT_WIN {
		convertedResult = 2
	} else if result == realtime.TournamentGameResult_DRAW {
		convertedResult = 1
	}
	return convertedResult
}

func emptyRecord() []int {
	record := []int{}
	for i := 0; i < int(realtime.TournamentGameResult_ELIMINATED)+1; i++ {
		record = append(record, 0)
	}
	return record
}

func (t *ClassicDivision) getPairing(player string, round int) (*realtime.Pairing, error) {
	pk, err := t.getPairingKey(player, round)
	if err != nil {
		return nil, err
	}
	pairing, ok := t.PairingMap[pk]
	if !ok {
		return nil, fmt.Errorf("pairing does not exist in the pairing map: %s", pk)
	}
	return pairing, nil
}

func (t *ClassicDivision) getPairingKey(player string, round int) (string, error) {
	if round >= len(t.Matrix) || round < 0 {
		return "", fmt.Errorf("round number out of range (getPairingKey): %d", round)
	}

	playerIndex, ok := t.PlayerIndexMap[player]
	if !ok {
		return "", fmt.Errorf("player does not exist in the division: %s", player)
	}

	if playerIndex < 0 || int(playerIndex) >= len(t.Matrix[round]) {
		return "", fmt.Errorf("player index out of range: %d", playerIndex)
	}
	return t.Matrix[round][playerIndex], nil
}

func (t *ClassicDivision) setPairingKey(player string, round int, pairingKey string) error {
	if round >= len(t.Matrix) || round < 0 {
		return fmt.Errorf("round number out of range (setPairingKey): %d", round)
	}

	playerIndex, ok := t.PlayerIndexMap[player]
	if !ok {
		return fmt.Errorf("player does not exist in the division: %s", player)
	}
	t.Matrix[round][playerIndex] = pairingKey
	return nil
}

func (t *ClassicDivision) makePairingKey() string {
	key := fmt.Sprintf("%d", t.PairingKeyInt)
	t.PairingKeyInt++
	return key
}

func (t *ClassicDivision) clearPairingKey(playerIndex int32, round int) error {
	if round >= len(t.Matrix) || round < 0 {
		return fmt.Errorf("round number out of range (clearPairingKey): %d", round)
	}

	if int(playerIndex) >= len(t.Matrix[round]) || playerIndex < 0 {
		return fmt.Errorf("player index out of range: %d", playerIndex)
	}

	pairingKey := t.Matrix[round][playerIndex]
	delete(t.PairingMap, pairingKey)
	t.Matrix[round][playerIndex] = ""
	return nil
}

func (t *ClassicDivision) opponentOf(player string, round int) (string, error) {
	pairingKey, err := t.getPairingKey(player, round)

	if err != nil {
		return "", err
	}

	pairing, ok := t.PairingMap[pairingKey]
	if !ok {
		return "", nil
	}

	if int(pairing.Players[0]) >= len(t.Players.Persons) || pairing.Players[0] < 0 {
		return "", fmt.Errorf("player %s in round %d is out of range: %d",
			player, round, pairing.Players[0])
	}

	if int(pairing.Players[1]) >= len(t.Players.Persons) || pairing.Players[1] < 0 {
		return "", fmt.Errorf("player %s in round %d is out of range: %d",
			player, round, pairing.Players[0])
	}

	playerOne := t.Players.Persons[pairing.Players[0]].Id
	playerTwo := t.Players.Persons[pairing.Players[1]].Id

	if player != playerOne && player != playerTwo {
		return "", fmt.Errorf("player %s does not exist in the pairing (%s, %s)",
			player,
			playerOne,
			playerTwo)
	} else if player != playerOne {
		return playerOne, nil
	} else {
		return playerTwo, nil
	}
}

func combinePairingsResponses(pr1 map[int][]*realtime.Pairing, pr2 map[int][]*realtime.Pairing) map[int][]*realtime.Pairing {
	for key, _ := range pr1 {
		_, ok := pr2[key]
		if ok {
			pr1[key] = append(pr1[key], pr2[key]...)
		}
	}
	for key, pairings := range pr2 {
		_, ok := pr1[key]
		if !ok {
			pr1[key] = pairings
		}
	}
	return pr1
}

func isByeOrForfeit(r realtime.TournamentGameResult) bool {
	return r == realtime.TournamentGameResult_FORFEIT_WIN ||
		r == realtime.TournamentGameResult_FORFEIT_LOSS ||
		r == realtime.TournamentGameResult_BYE
}

func reverse(array []string) {
	for i, j := 0, len(array)-1; i < j; i, j = i+1, j-1 {
		array[i], array[j] = array[j], array[i]
	}
}

func validatePairings(tc *ClassicDivision, round int) error {
	// For each pairing, check that
	//   - Player's opponent is nonnull
	//   - Player's opponent's opponent is the player

	if round < 0 || round >= len(tc.Matrix) {
		return fmt.Errorf("round number out of range (validatePairings): %d", round)
	}

	for i, pairingKey := range tc.Matrix[round] {
		if pairingKey == "" {
			return fmt.Errorf("round %d player %d pairing nil", round, i)
		}
		pairing, ok := tc.PairingMap[pairingKey]
		if !ok {
			return fmt.Errorf("pairing key does not exist in pairing map: %s", pairingKey)
		}
		if pairing.Players == nil {
			// Some pairings can be nil for Elimination tournaments
			if tc.RoundControls[0].PairingMethod != realtime.PairingMethod_ELIMINATION {
				return fmt.Errorf("player %d is unpaired", i)
			} else {
				continue
			}
		}
		// Check that the pairing refs are correct
		opponent, err := tc.opponentOf(tc.Players.Persons[i].Id, round)
		if err != nil {
			return err
		}
		opponentOpponent, err := tc.opponentOf(opponent, round)
		if err != nil {
			return err
		}
		if tc.Players.Persons[i].Id != opponentOpponent {
			return fmt.Errorf("player %s's opponent's (%s) opponent (%s) is not themself",
				tc.Players.Persons[i].Id,
				opponent,
				opponentOpponent)
		}
	}
	return nil
}
