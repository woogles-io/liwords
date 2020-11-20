package tournament

import (
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/datatypes"
	"math/rand"
	"sort"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/pair"
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
)

type ClassicDivision struct {
	Matrix         [][]*entity.PlayerRoundInfo `json:"t"`
	Players        []string                    `json:"p"`
	PlayerIndexMap map[string]int              `json:"i"`
	RoundControls  []*entity.RoundControls     `json:"r"`
}

func NewClassicDivision(players []string,
	numberOfRounds int,
	roundControls []*entity.RoundControls) (*ClassicDivision, error) {
	numberOfPlayers := len(players)

	if numberOfPlayers < 2 {
		return nil, errors.New("classic Tournaments must have at least 2 players")
	}

	if numberOfRounds < 1 {
		return nil, errors.New("classic Tournaments must have at least 1 round")
	}

	if numberOfRounds != len(roundControls) {
		return nil, errors.New("round controls length does not match the number of rounds")
	}

	isElimination := false
	for _, control := range roundControls {
		if control.PairingMethod == entity.Elimination {
			isElimination = true
		} else if isElimination && control.PairingMethod != entity.Elimination {
			return nil, errors.New("cannot mix Elimination pairings with any other pairing method")
		}
	}

	// For now, assume we require exactly n round and 2 ^ n players for an elimination tournament

	if roundControls[0].PairingMethod == entity.Elimination {
		expectedNumberOfPlayers := 1 << numberOfRounds
		if expectedNumberOfPlayers != numberOfPlayers {
			return nil, fmt.Errorf("invalid number of players based on the number of rounds."+
				" Have %d players, expected %d players based on the number of rounds (%d)",
				numberOfPlayers, expectedNumberOfPlayers, numberOfRounds)
		}

	}

	for i := 0; i < numberOfRounds; i++ {
		roundControls[i].Round = i
	}

	pairings := newPairingMatrix(numberOfRounds, numberOfPlayers)
	playerIndexMap := newPlayerIndexMap(players)
	t := &ClassicDivision{Matrix: pairings,
		Players:        players,
		PlayerIndexMap: playerIndexMap,
		RoundControls:  roundControls}
	if roundControls[0].PairingMethod != entity.Manual {
		err := t.PairRound(0)
		if err != nil {
			return nil, err
		}
	}
	// We can make all non-standings dependent pairings right now
	for i := 1; i < numberOfRounds; i++ {
		pm := roundControls[i].PairingMethod
		if pm == entity.RoundRobin || pm == entity.Random {
			err := t.PairRound(i)
			if err != nil {
				return nil, err
			}
		}
	}

	return t, nil
}

func (t *ClassicDivision) GetPlayerRoundInfo(player string, round int) (*entity.PlayerRoundInfo, error) {
	if round >= len(t.Matrix) || round < 0 {
		return nil, fmt.Errorf("round number out of range: %d", round)
	}
	roundPairings := t.Matrix[round]

	playerIndex, ok := t.PlayerIndexMap[player]
	if !ok {
		return nil, fmt.Errorf("player does not exist in the tournament: %s", player)
	}
	return roundPairings[playerIndex], nil
}

func (t *ClassicDivision) SetPairing(playerOne string, playerTwo string, round int) error {
	playerOneInfo, err := t.GetPlayerRoundInfo(playerOne, round)
	if err != nil {
		return err
	}

	playerTwoInfo, err := t.GetPlayerRoundInfo(playerTwo, round)
	if err != nil {
		return err
	}

	playerOneOpponent, err := opponentOf(playerOneInfo.Pairing, playerOne)
	if err != nil {
		return err
	}

	playerTwoOpponent, err := opponentOf(playerTwoInfo.Pairing, playerTwo)
	if err != nil {
		return err
	}

	playerOneInfo.Pairing = nil
	playerTwoInfo.Pairing = nil

	// If playerOne was already paired, unpair their opponent
	if playerOneOpponent != "" {
		playerOneOpponentInfo, err := t.GetPlayerRoundInfo(playerOneOpponent, round)
		if err != nil {
			return err
		}
		playerOneOpponentInfo.Pairing = nil
	}

	// If playerTwo was already paired, unpair their opponent
	if playerTwoOpponent != "" {
		playerTwoOpponentInfo, err := t.GetPlayerRoundInfo(playerTwoOpponent, round)
		if err != nil {
			return err
		}
		playerTwoOpponentInfo.Pairing = nil
	}

	newPairing := newClassicPairing(t, playerOne, playerTwo, round)
	playerOneInfo.Pairing = newPairing
	playerTwoInfo.Pairing = newPairing
	return nil
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
	gameIndex int) error {

	// Fetch the player round records
	pri1, err := t.GetPlayerRoundInfo(p1, round)
	if err != nil {
		return err
	}
	pri2, err := t.GetPlayerRoundInfo(p2, round)
	if err != nil {
		return err
	}

	// Ensure that the pairing exists
	if pri1.Pairing == nil {
		return fmt.Errorf("submitted result for a player with a null pairing: %s round (%d)", p1, round)
	}

	if pri2.Pairing == nil {
		return fmt.Errorf("submitted result for a player with a null pairing: %s round (%d)", p2, round)
	}

	// Ensure the submitted results were for players that were paired
	if pri1.Pairing != pri2.Pairing {
		return fmt.Errorf("submitted result for players that didn't player each other: %s (%p), %s (%p) round (%d)", p1, pri1.Pairing, p2, pri2.Pairing, round)
	}

	pairing := pri1.Pairing
	pairingMethod := t.RoundControls[round].PairingMethod

	// For Elimination tournaments only.
	// Could be a tiebreaking result or could be an out of range
	// game index
	if pairingMethod == entity.Elimination && gameIndex >= t.RoundControls[round].GamesPerRound {
		if gameIndex != len(pairing.Games) {
			return fmt.Errorf("submitted tiebreaking result with invalid game index."+
				" Player 1: %s, Player 2: %s, Round: %d, GameIndex: %d", p1, p2, round, gameIndex)
		} else {
			pairing.Games = append(pairing.Games,
				&entity.TournamentGame{Scores: []int{0, 0},
					Results: []realtime.TournamentGameResult{realtime.TournamentGameResult_NO_RESULT,
						realtime.TournamentGameResult_NO_RESULT}})
		}
	}

	if !amend && ((pairing.Outcomes[0] != realtime.TournamentGameResult_NO_RESULT &&
		pairing.Outcomes[1] != realtime.TournamentGameResult_NO_RESULT) ||
		pairing.Games[gameIndex].Results[0] != realtime.TournamentGameResult_NO_RESULT &&
			pairing.Games[gameIndex].Results[1] != realtime.TournamentGameResult_NO_RESULT) {
		return fmt.Errorf("result is already submitted for round %d, %s vs. %s", round, p1, p2)
	}

	p1Index := 0
	if pairing.Players[1] == p1 {
		p1Index = 1
	}

	if pairingMethod == entity.Elimination {
		pairing.Games[gameIndex].Scores[p1Index] = p1Score
		pairing.Games[gameIndex].Scores[1-p1Index] = p2Score
		pairing.Games[gameIndex].Results[p1Index] = p1Result
		pairing.Games[gameIndex].Results[1-p1Index] = p2Result
		pairing.Games[gameIndex].GameEndReason = reason

		// Get elimination outcomes will take care of the indexing
		// for us because the newOutcomes are aligned with the data
		// in pairing.Games
		newOutcomes := getEliminationOutcomes(pairing.Games, t.RoundControls[round].GamesPerRound)

		pairing.Outcomes[0] = newOutcomes[0]
		pairing.Outcomes[1] = newOutcomes[1]
	} else {
		// Classic tournaments only ever have
		// one game per round
		pairing.Games[0].Scores[p1Index] = p1Score
		pairing.Games[0].Scores[1-p1Index] = p2Score
		pairing.Games[0].Results[p1Index] = p1Result
		pairing.Games[0].Results[1-p1Index] = p2Result
		pairing.Games[0].GameEndReason = reason
		pairing.Outcomes[p1Index] = p1Result
		pairing.Outcomes[1-p1Index] = p2Result
	}

	complete, err := t.IsRoundComplete(round)
	if err != nil {
		return err
	}
	finished, err := t.IsFinished()
	if err != nil {
		return err
	}

	// Only pair if this round is complete and the tournament
	// is not over. Don't pair for round robin and random since those pairings
	// were made when the tournament was created.
	if !finished && complete &&
		t.RoundControls[round+1].PairingMethod != entity.RoundRobin &&
		t.RoundControls[round+1].PairingMethod != entity.Random &&
		t.RoundControls[round+1].PairingMethod != entity.Manual {
		err = t.PairRound(round + 1)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *ClassicDivision) PairRound(round int) error {
	if round < 0 || round >= len(t.Matrix) {
		return fmt.Errorf("round number out of range: %d", round)
	}
	roundPairings := t.Matrix[round]
	pairingMethod := t.RoundControls[round].PairingMethod
	// This automatic pairing could be the result of an
	// amendment. Undo all the pairings so byes can be
	// properly assigned (bye assignment checks for nil pairing).
	for i := 0; i < len(roundPairings); i++ {
		roundPairings[i].Pairing = nil
	}

	standingsRound := round
	if standingsRound == 0 {
		standingsRound = 1
	}

	standings, err := t.GetStandings(standingsRound - 1)
	if err != nil {
		return err
	}

	poolMembers := []*entity.PoolMember{}

	// Round Robin must have the same ordering for each round
	var playerOrder []string
	if pairingMethod == entity.RoundRobin {
		playerOrder = t.Players
	} else {
		playerOrder = make([]string, len(standings))
		for i := 0; i < len(standings); i++ {
			playerOrder[i] = standings[i].Player
		}
	}

	for i := 0; i < len(playerOrder); i++ {
		pm := &entity.PoolMember{Id: playerOrder[i]}
		// Wins do not matter for RoundRobin pairings
		if pairingMethod != entity.RoundRobin {
			pm.Wins = standings[i].Wins
			pm.Draws = standings[i].Draws
			pm.Spread = standings[i].Spread
		} else {
			pm.Wins = 0
			pm.Draws = 0
			pm.Spread = 0
		}
		poolMembers = append(poolMembers, pm)
	}

	repeats, err := getRepeats(t, round-1)
	if err != nil {
		return err
	}

	upm := &entity.UnpairedPoolMembers{RoundControls: t.RoundControls[round],
		PoolMembers: poolMembers,
		Repeats:     repeats}

	pairings, err := pair.Pair(upm)

	if err != nil {
		return err
	}

	l := len(pairings)

	if l != len(roundPairings) {
		return errors.New("pair did not return the correct number of pairings")
	}

	for i := 0; i < l; i++ {
		// Player order might be a different order than the players in roundPairings
		playerIndex := t.PlayerIndexMap[playerOrder[i]]

		if roundPairings[playerIndex].Pairing == nil {

			var opponentIndex int
			if pairings[i] < 0 {
				opponentIndex = playerIndex
			} else if pairings[i] >= l {
				return fmt.Errorf("invalid pairing for round %d: %d", round, pairings[i])
			} else {
				opponentIndex = t.PlayerIndexMap[playerOrder[pairings[i]]]
			}

			playerName := t.Players[playerIndex]
			opponentName := t.Players[opponentIndex]

			var newPairing *entity.Pairing
			if pairingMethod == entity.Elimination && round > 0 && i >= l>>round {
				newPairing = newEliminatedPairing(playerName, opponentName)
			} else {
				newPairing = newClassicPairing(t, playerName, opponentName, round)
			}
			roundPairings[playerIndex].Pairing = newPairing
			roundPairings[opponentIndex].Pairing = newPairing
		}
	}
	return nil
}

func (t *ClassicDivision) GetStandings(round int) ([]*entity.Standing, error) {
	if round < 0 || round >= len(t.Matrix) {
		return nil, errors.New("round number out of range")
	}

	wins := 0
	losses := 0
	draws := 0
	spread := 0
	player := ""
	records := []*entity.Standing{}

	for i := 0; i < len(t.Players); i++ {
		wins = 0
		losses = 0
		draws = 0
		spread = 0
		player = t.Players[i]
		for j := 0; j <= round; j++ {
			pairing := t.Matrix[j][i].Pairing
			if pairing != nil && pairing.Players != nil {
				playerIndex := 0
				if pairing.Players[1] == player {
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
		records = append(records, &entity.Standing{Player: player,
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
	if pairingMethod == entity.Elimination {
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
				if records[i].Wins == records[j].Wins && records[i].Draws == records[j].Draws {
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
		return false, fmt.Errorf("round number out of range: %d", round)
	}
	// Check that everyone is paired
	for _, pri := range t.Matrix[round] {
		if pri.Pairing == nil {
			return false, nil
		}
	}
	// Check that all previous round are complete
	for i := 0; i <= round-1; i++ {
		complete, err := t.IsRoundComplete(i)
		if err != nil || !complete {
			return false, err
		}
	}
	return true, nil
}

func (t *ClassicDivision) IsRoundComplete(round int) (bool, error) {
	if round >= len(t.Matrix) || round < 0 {
		return false, fmt.Errorf("round number out of range: %d", round)
	}
	for _, pri := range t.Matrix[round] {
		if pri.Pairing == nil || pri.Pairing.Outcomes[0] == realtime.TournamentGameResult_NO_RESULT ||
			pri.Pairing.Outcomes[1] == realtime.TournamentGameResult_NO_RESULT {
			return false, nil
		}
	}
	return true, nil
}

func (t *ClassicDivision) IsFinished() (bool, error) {
	return t.IsRoundComplete(len(t.Matrix) - 1)
}

func (t *ClassicDivision) Serialize() (datatypes.JSON, error) {
	json, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	return json, err
}

func newPairingMatrix(numberOfRounds int, numberOfPlayers int) [][]*entity.PlayerRoundInfo {
	pairings := [][]*entity.PlayerRoundInfo{}
	for i := 0; i < numberOfRounds; i++ {
		roundPairings := []*entity.PlayerRoundInfo{}
		for j := 0; j < numberOfPlayers; j++ {
			roundPairings = append(roundPairings, &entity.PlayerRoundInfo{})
		}
		pairings = append(pairings, roundPairings)
	}
	return pairings
}

func newClassicPairing(t *ClassicDivision,
	playerOne string,
	playerTwo string,
	round int) *entity.Pairing {

	games := []*entity.TournamentGame{}
	for i := 0; i < t.RoundControls[round].GamesPerRound; i++ {
		games = append(games, &entity.TournamentGame{Scores: []int{0, 0},
			Results: []realtime.TournamentGameResult{realtime.TournamentGameResult_NO_RESULT,
				realtime.TournamentGameResult_NO_RESULT}})
	}

	playerGoingFirst := playerOne
	playerGoingSecond := playerTwo
	switchFirst := false
	firstMethod := t.RoundControls[round].FirstMethod

	if firstMethod != entity.ManualFirst {
		playerOneFS := getPlayerFS(t, playerGoingFirst, round-1)
		playerTwoFS := getPlayerFS(t, playerGoingSecond, round-1)
		if firstMethod == entity.RandomFirst {
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

	return &entity.Pairing{Players: []string{playerGoingFirst, playerGoingSecond},
		Games: games,
		Outcomes: []realtime.TournamentGameResult{realtime.TournamentGameResult_NO_RESULT,
			realtime.TournamentGameResult_NO_RESULT}}
}

func getPlayerFS(t *ClassicDivision, player string, round int) []int {

	fs := []int{0, 0}
	for i := 0; i <= round; i++ {
		pairing := t.Matrix[i][t.PlayerIndexMap[player]].Pairing
		if pairing != nil {
			playerIndex := 0
			if pairing.Players[1] == player {
				playerIndex = 1
			}
			outcome := pairing.Outcomes[playerIndex]
			if outcome == realtime.TournamentGameResult_NO_RESULT ||
				outcome == realtime.TournamentGameResult_WIN ||
				outcome == realtime.TournamentGameResult_LOSS ||
				outcome == realtime.TournamentGameResult_DRAW {
				fs[playerIndex]++
			}
		}
	}
	return fs
}

func newEliminatedPairing(playerOne string, playerTwo string) *entity.Pairing {
	return &entity.Pairing{Outcomes: []realtime.TournamentGameResult{realtime.TournamentGameResult_ELIMINATED,
		realtime.TournamentGameResult_ELIMINATED}}
}

func newPlayerIndexMap(players []string) map[string]int {
	m := make(map[string]int)
	for i, player := range players {
		m[player] = i
	}
	return m
}

func getRepeats(t *ClassicDivision, round int) (map[string]int, error) {
	if round >= len(t.Matrix) {
		return nil, fmt.Errorf("round number out of range: %d", round)
	}
	repeats := make(map[string]int)
	for i := 0; i <= round; i++ {
		roundPairings := t.Matrix[i]
		for _, pri := range roundPairings {
			if pri.Pairing != nil && pri.Pairing.Players != nil {
				playerOne := pri.Pairing.Players[0]
				playerTwo := pri.Pairing.Players[1]
				if playerOne != playerTwo {
					key := pair.GetRepeatKey(playerOne, playerTwo)
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

func getEliminationOutcomes(games []*entity.TournamentGame, gamesPerRound int) []realtime.TournamentGameResult {
	// Determines if a player is eliminated for a given round in an
	// elimination tournament. The convertResult function gives 2 for a win,
	// 1 for a draw, and 0 otherwise. If a player's score is greater than
	// the games per round, they have won, unless there is a tie.
	p1Wins := 0
	p2Wins := 0
	p1Spread := 0
	p2Spread := 0
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
	if len(games) > gamesPerRound { // Tiebreaking results are present
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

func convertResult(result realtime.TournamentGameResult) int {
	convertedResult := 0
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

func opponentOf(pairing *entity.Pairing, player string) (string, error) {
	if pairing == nil {
		return "", nil
	}
	if player != pairing.Players[0] && player != pairing.Players[1] {
		return "", fmt.Errorf("player %s does not exist in the pairing (%s, %s)",
			player,
			pairing.Players[0],
			pairing.Players[1])
	} else if player != pairing.Players[0] {
		return pairing.Players[0], nil
	} else {
		return pairing.Players[1], nil
	}
}

func reverse(array []string) {
	for i, j := 0, len(array)-1; i < j; i, j = i+1, j-1 {
		array[i], array[j] = array[j], array[i]
	}
}
