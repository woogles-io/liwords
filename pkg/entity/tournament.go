package entity

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
)

type Result int

const (
	Win Result = iota
	Loss
	Draw
	Bye
	ForfeitLoss
	ForfeitWin
	Eliminated
	None
)

type TournamentGame struct {
	Scores        []int
	Results       []Result
	GameEndReason realtime.GameEndReason
}

const Unpaired = -1

type Pairing struct {
	Players  []string
	Games    []*TournamentGame
	Outcomes []Result
}

type PlayerRoundInfo struct {
	Pairing *Pairing
	Record  []int
	Spread  int
}

type Standing struct {
	Player string
	Wins   int
	Losses int
	Draws  int
	Spread int
}

type Tournament interface {
	GetPlayerRoundInfo(string, int) (*PlayerRoundInfo, error)
	SubmitResult(int, string, string, int, int, Result, Result, realtime.GameEndReason, bool) error
	GetStandings(int) ([]*Standing, error)
	SetPairing(string, string, int) error
	IsRoundComplete(int) (bool, error)
	IsFinished() (bool, error)
}

type TournamentManager struct {
	Tournament Tournament
	// StartTime time.Time
	// RoundDuration time.Time
}

type PairingMethod int

const (
	Random PairingMethod = iota
	RoundRobin
	KingOfTheHill
	Elimination
	// Need to implement eventually
	// Swiss
	// Performance

	// Manual simply does not make any
	// pairings at all. The director
	// has to make all the pairings themselves.
	Manual
)

type TournamentClassic struct {
	Matrix         [][]*PlayerRoundInfo
	Players        []string
	PlayerIndexMap map[string]int
	PairingMethod  PairingMethod
	GamesPerRound  int
}

func NewTournamentClassic(players []string, numberOfRounds int, method PairingMethod) (*TournamentClassic, error) {

	if len(players) < 2 {
		return nil, errors.New("Classic Tournaments must have at least 2 players")
	}

	if numberOfRounds < 1 {
		return nil, errors.New("Classic Tournaments must have at least 1 round")
	}

	pairings := newPairingMatrix(numberOfRounds, len(players))
	playerIndexMap := newPlayerIndexMap(players)
	t := &TournamentClassic{Matrix: pairings,
		Players:        players,
		PlayerIndexMap: playerIndexMap,
		PairingMethod:  method}
	pairRoundClassic(t, 0)

	// For round robins, we can pair the whole
	// tournament right now.
	if method == RoundRobin {
		for i := 1; i < len(pairings); i++ {
			pairRoundClassic(t, i)
		}
	}

	return t, nil
}

func (t *TournamentClassic) GetPlayerRoundInfo(player string, round int) (*PlayerRoundInfo, error) {
	return getPlayerRoundInfo(t.Matrix, round, player, t.PlayerIndexMap)
}

func (t *TournamentClassic) SetPairing(playerOne string, playerTwo string, round int) error {
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

	newPairing := newClassicPairing(playerOne, playerTwo)
	playerOneInfo.Pairing = newPairing
	playerTwoInfo.Pairing = newPairing
	return nil
}

func (t *TournamentClassic) SubmitResult(round int,
	p1 string,
	p2 string,
	p1Score int,
	p2Score int,
	p1Result Result,
	p2Result Result,
	reason realtime.GameEndReason,
	amend bool) error {

	// Fetch the player round records
	pri1, err := getPlayerRoundInfo(t.Matrix, round, p1, t.PlayerIndexMap)
	if err != nil {
		return err
	}
	pri2, err := getPlayerRoundInfo(t.Matrix, round, p2, t.PlayerIndexMap)
	if err != nil {
		return err
	}

	// Ensure that the pairing exists
	if pri1.Pairing == nil {
		return errors.New(fmt.Sprintf("Submitted result for a player with a null pairing: %s round (%d)\n", p1, round))
	}

	if pri2.Pairing == nil {
		return errors.New(fmt.Sprintf("Submitted result for a player with a null pairing: %s round (%d)\n", p2, round))
	}

	// Ensure the submitted results were for players that were paired
	if pri1.Pairing != pri2.Pairing {
		return errors.New(fmt.Sprintf("Submitted result for players that didn't player each other: %s, %s round (%d)\n", p1, p2, round))
	}

	pairing := pri1.Pairing

	if pairing.Outcomes[0] != None && pairing.Outcomes[1] != None && !amend {
		return errors.New("This result is already submitted")
	}

	if amend {
		pri1.Record[pairing.Outcomes[0]] -= 1
		pri2.Record[pairing.Outcomes[1]] -= 1
		p1ScoreOld := pairing.Games[0].Scores[0]
		p2ScoreOld := pairing.Games[0].Scores[1]
		pri1.Spread -= p1ScoreOld - p2ScoreOld
		pri2.Spread -= p2ScoreOld - p1ScoreOld
	}

	// Classic tournaments only ever have
	// one game per round
	pairing.Games[0].Scores[0] = p1Score
	pairing.Games[0].Scores[1] = p2Score
	pairing.Games[0].Results[0] = p1Result
	pairing.Games[0].Results[1] = p2Result
	pairing.Games[0].GameEndReason = reason
	pairing.Outcomes[0] = p1Result
	pairing.Outcomes[1] = p2Result

	// Update the player records
	pri1.Record[pairing.Outcomes[0]] += 1
	pri2.Record[pairing.Outcomes[1]] += 1
	pri1.Spread += p1Score - p2Score
	pri2.Spread += p2Score - p1Score

	// Pair the next round if this round is complete
	complete, err := t.IsRoundComplete(round)
	if err != nil {
		return err
	}
	finished, err := t.IsFinished()
	if err != nil {
		return err
	}
	// Only pair if this round is complete and the tournament
	// is not over. Don't pair for round robin since all pairings
	// were made when the tournament was created
	if !finished && complete && t.PairingMethod != RoundRobin {
		pairRoundClassic(t, round+1)
	}

	return nil
}

func (t *TournamentClassic) GetStandings(round int) ([]*Standing, error) {
	if round < 0 || round >= len(t.Matrix) {
		return nil, errors.New("Round number out of range")
	}
	records := []*Standing{}
	for i, pri := range t.Matrix[round] {
		wins, losses, draws := resultsToScores(pri.Record)
		records = append(records, &Standing{Player: t.Players[i],
			Wins:   wins,
			Losses: losses,
			Draws:  draws,
			Spread: pri.Spread})
	}

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

	return records, nil
}

func resultsToScores(results []int) (int, int, int) {
	wins := results[int(Win)] + results[int(Bye)] + results[int(ForfeitWin)]
	losses := results[int(Loss)] + results[int(ForfeitLoss)]
	draws := results[int(Draw)]

	return wins, losses, draws
}

func newPairingMatrix(numberOfRounds int, numberOfPlayers int) [][]*PlayerRoundInfo {
	pairings := [][]*PlayerRoundInfo{}
	for i := 0; i < numberOfRounds; i++ {
		roundPairings := []*PlayerRoundInfo{}
		for j := 0; j < numberOfPlayers; j++ {
			roundPairings = append(roundPairings, &PlayerRoundInfo{Record: emptyRecord()})
		}
		pairings = append(pairings, roundPairings)
	}
	return pairings
}

func newClassicPairing(playerOne string, playerTwo string) *Pairing {
	return &Pairing{Players: []string{playerOne, playerTwo},
		Games:    []*TournamentGame{&TournamentGame{Scores: []int{0, 0}, Results: []Result{None, None}}},
		Outcomes: []Result{None, None}}
}

func newPlayerIndexMap(players []string) map[string]int {
	m := make(map[string]int)
	for i, player := range players {
		m[player] = i
	}
	return m
}

func getPlayerRoundInfo(matrix [][]*PlayerRoundInfo, round int, player string, playerIndexMap map[string]int) (*PlayerRoundInfo, error) {
	if round >= len(matrix) || round < 0 {
		return nil, errors.New(fmt.Sprintf("Round number out of range: %d\n", round))
	}
	roundPairings := matrix[round]

	playerIndex, ok := playerIndexMap[player]
	if !ok {
		return nil, errors.New(fmt.Sprintf("Player does not exist in the tournament: %s\n", player))
	}
	return roundPairings[playerIndex], nil
}

func pairRoundClassic(t *TournamentClassic, round int) error {
	if round < 0 || round >= len(t.Matrix) {
		return errors.New("Round number out of range")
	}

	copyRecords(t.Matrix, round-1)

	roundPairings := t.Matrix[round]
	if t.PairingMethod == KingOfTheHill || t.PairingMethod == Elimination {
		standingsRound := round - 1
		// If this is the first round, just pair
		// based on the order of the player list,
		// which GetStandings should return for a tournament
		// with no results
		if standingsRound < 0 {
			standingsRound = 0
		}
		standings, err := t.GetStandings(standingsRound)
		if err != nil {
			return err
		}
		for i := 0; i < len(standings)-1; i += 2 {
			playerOne := standings[i].Player
			playerTwo := standings[i+1].Player
			newPairing := newClassicPairing(playerOne, playerTwo)
			roundPairings[t.PlayerIndexMap[playerOne]].Pairing = newPairing
			roundPairings[t.PlayerIndexMap[playerTwo]].Pairing = newPairing
		}
	} else if t.PairingMethod == Random {
		playerIndexes := []int{}
		for _, v := range t.PlayerIndexMap {
			playerIndexes = append(playerIndexes, v)
		}
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(playerIndexes),
			func(i, j int) {
				playerIndexes[i], playerIndexes[j] = playerIndexes[j], playerIndexes[i]
			})

		for i := 0; i < len(playerIndexes)-1; i += 2 {
			newPairing := newClassicPairing(t.Players[playerIndexes[i]], t.Players[playerIndexes[i+1]])
			roundPairings[playerIndexes[i]].Pairing = newPairing
			roundPairings[playerIndexes[i+1]].Pairing = newPairing
		}
	} else if t.PairingMethod == RoundRobin {
		roundRobinPlayers := t.Players
		// The empty string represents the bye
		if len(roundRobinPlayers)%2 == 1 {
			roundRobinPlayers = append(roundRobinPlayers, "")
		}

		roundRobinPairings := getRoundRobinPairings(roundRobinPlayers, round)
		// fmt.Sprintf("the pairings: %s\n", strings.Join(roundRobinPairings, ", "))
		for i := 0; i < len(roundRobinPairings)-1; i += 2 {
			playerOne := roundRobinPairings[i]
			playerTwo := roundRobinPairings[i+1]

			if playerOne == "" && playerTwo == "" {
				return errors.New(fmt.Sprintf("Two byes playing each other in round %d\n", round))
			}

			// The blank string represents a bye in the
			// getRoundRobinPairings algorithm, but in a Tournament
			// is represented by a player paired with themselves,
			// so convert it here.

			if playerOne == "" {
				playerOne = playerTwo
			}

			if playerTwo == "" {
				playerTwo = playerOne
			}

			newPairing := newClassicPairing(playerOne, playerTwo)
			roundPairings[t.PlayerIndexMap[playerOne]].Pairing = newPairing
			roundPairings[t.PlayerIndexMap[playerTwo]].Pairing = newPairing
		}
	}
	// Give all unpaired players a bye
	// Byes are always designated as a player
	// paired with themselves
	if t.PairingMethod != Manual {
		for i := 0; i < len(roundPairings); i++ {
			pri := roundPairings[i]
			if pri.Pairing == nil {
				pri.Pairing = newClassicPairing(t.Players[i], t.Players[i])
			}
		}
	}
	return nil
}

func (t *TournamentClassic) IsRoundComplete(round int) (bool, error) {
	if round >= len(t.Matrix) || round < 0 {
		return false, errors.New(fmt.Sprintf("Round number out of range: %d\n", round))
	}
	for _, pri := range t.Matrix[round] {
		if pri.Pairing == nil || pri.Pairing.Outcomes[0] == None || pri.Pairing.Outcomes[1] == None {
			return false, nil
		}
	}
	return true, nil
}

func (t *TournamentClassic) IsFinished() (bool, error) {
	return t.IsRoundComplete(len(t.Matrix) - 1)
}

func getRoundRobinPairings(players []string, round int) []string {

	/* Round Robin pairing algorithm (from stackoverflow, where else?):

	Players are numbered 1..n. In this example, there are 8 players

	Write all the players in two rows.

	1 2 3 4
	8 7 6 5

	The columns show which players will play in that round (1 vs 8, 2 vs 7, ...).

	Now, keep 1 fixed, but rotate all the other players. In round 2, you get

	1 8 2 3
	7 6 5 4

	and in round 3, you get

	1 7 8 2
	6 5 4 3

	This continues through round n-1, in this case,

	1 3 4 5
	2 8 7 6

	The following algorithm captures the pairings for a certain rotation
	based on the round. The length of players will always be even
	since a bye will be added for any odd length players.

	*/

	rotatedPlayers := players[1:len(players)]

	l := len(rotatedPlayers)
	rotationIndex := l - (round % l)

	rotatedPlayers = append(rotatedPlayers[rotationIndex:l], rotatedPlayers[0:rotationIndex]...)
	rotatedPlayers = append([]string{players[0]}, rotatedPlayers...)

	l = len(rotatedPlayers)
	topHalf := rotatedPlayers[0 : l/2]
	bottomHalf := rotatedPlayers[l/2 : l]
	reverse(bottomHalf)

	pairings := []string{}
	for i := 0; i < len(players)/2; i++ {
		pairings = append(pairings, topHalf[i])
		pairings = append(pairings, bottomHalf[i])
	}
	return pairings
}

/*func outcomeForRound(results []Result, gamesPerRound int) int {
	wins := 0
	losses := 0
	for _, result := range results {
		convertedResult := convertResult(result)
		if convertedResult == 1 {
			wins += 2
		} else if convertedResult == -1 {
			wins += 1
			losses += 1
		} else {
			losses += 2
		}
	}
	// Win -> 1
	// Draw -> -1
	// Loss -> 0
	outcome := -1
	if wins > gamesPerRound {
		outcome = 1
	} else if losses > gamesPerRound {
		outcome = 0
	}
	return outcome
}

func convertResult(result Result) int {
	if result == Win || result == Bye || result == ForfeitWin {
		return 1
	} else if result == Draw {
		return -1
	} else {
		return 0
	}
}*/

func copyRecords(m [][]*PlayerRoundInfo, round int) error {
	if round < 0 {
		return nil
	}
	if round+1 >= len(m) {
		return errors.New(fmt.Sprintf("Copying records to an out of range round: %d\n", round+1))
	}
	roundInfo := m[round]
	nextRoundInfo := m[round+1]
	for i := 0; i < len(m[round]); i++ {
		nextRoundInfo[i].Record = roundInfo[i].Record
		nextRoundInfo[i].Spread = roundInfo[i].Spread
	}
	return nil
}

func emptyRecord() []int {
	record := []int{}
	for i := 0; i < int(None)+1; i++ {
		record = append(record, 0)
	}
	return record
}

func opponentOf(pairing *Pairing, player string) (string, error) {
	if pairing == nil {
		return "", nil
	}
	if player != pairing.Players[0] && player != pairing.Players[1] {
		return "", errors.New(fmt.Sprintf("Player %s does not exist in the pairing (%s, %s)\n",
			player,
			pairing.Players[0],
			pairing.Players[1]))
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
