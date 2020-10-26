package entity

import (
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/datatypes"
	"math/rand"
	"sort"
	"sync"
	"time"

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
)

type TournamentGame struct {
	Scores        []int                           `json:"s"`
	Results       []realtime.TournamentGameResult `json:"r"`
	GameEndReason realtime.GameEndReason          `json:"g"`
}

const Unpaired = -1

type Pairing struct {
	Players  []string                        `json:"p"`
	Games    []*TournamentGame               `json:"g"`
	Outcomes []realtime.TournamentGameResult `json:"o"`
}

type PlayerRoundInfo struct {
	Pairing *Pairing `json:"p"`
	Record  []int    `json:"r"`
	Spread  int      `json:"s"`
}

type Standing struct {
	Player string
	Wins   int
	Losses int
	Draws  int
	Spread int
}

type TournamentManager interface {
	GetPlayerRoundInfo(string, int) (*PlayerRoundInfo, error)
	SubmitResult(int, string, string, int, int, realtime.TournamentGameResult,
		realtime.TournamentGameResult, realtime.GameEndReason, bool, int) error
	GetStandings(int) ([]*Standing, error)
	SetPairing(string, string, int) error
	StartRound(int) error
	IsRoundComplete(int) (bool, error)
	IsFinished() (bool, error)
	Serialize() (datatypes.JSON, error)
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
	Matrix         [][]*PlayerRoundInfo `json:"t"`
	Players        []string             `json:"p"`
	PlayerIndexMap map[string]int       `json:"i"`
	PairingMethods []PairingMethod      `json:"m"`
	GamesPerRound  int                  `json:"g"`
}

type TournamentType int

const (
	ClassicTournamentType TournamentType = iota
	// It's gonna be lit:
	ArenaTournamentType
)

type TournamentPersons struct {
	Persons map[string]int `json:"p"`
}

type TournamentControls struct {
	GameRequest    *realtime.GameRequest `json:"r"`
	PairingMethods []PairingMethod       `json:"a"`
	NumberOfRounds int                   `json:"u"`
	GamesPerRound  int                   `json:"g"`
	Type           TournamentType        `json:"t"`
	StartTime      time.Time             `json:"s"`
}

type Tournament struct {
	sync.RWMutex
	UUID              string              `json:"u"`
	Name              string              `json:"n"`
	Description       string              `json:"e"`
	Directors         *TournamentPersons  `json:"d"`
	Players           *TournamentPersons  `json:"p"`
	Controls          *TournamentControls `json:"c"`
	IsStarted         bool                `json:"i"`
	TournamentManager TournamentManager   `json:"m"`
}

func NewTournamentClassic(players []string,
	numberOfRounds int,
	methods []PairingMethod,
	gamesPerRound int) (*TournamentClassic, error) {
	numberOfPlayers := len(players)

	if numberOfPlayers < 2 {
		return nil, errors.New("Classic Tournaments must have at least 2 players")
	}

	if numberOfRounds < 1 {
		return nil, errors.New("Classic Tournaments must have at least 1 round")
	}

	if numberOfRounds != len(methods) {
		return nil, errors.New("Pairing methods length does not match the number of rounds!")
	}

	isElimination := false
	for _, method := range methods {
		if method == Elimination {
			isElimination = true
		} else if isElimination && method != Elimination {
			return nil, errors.New("")
		}
	}

	// For now, assume we require exactly n round and 2 ^ n players for an elimination tournament

	if methods[0] == Elimination {
		expectedNumberOfPlayers := twoPower(numberOfRounds)
		if expectedNumberOfPlayers != numberOfPlayers {
			return nil, errors.New(fmt.Sprintf("Invalid number of players based on the number of rounds."+
				" Have %d players, expected %d players based on the number of rounds (%d)\n",
				expectedNumberOfPlayers, numberOfPlayers, numberOfRounds))
		}

	}

	pairings := newPairingMatrix(numberOfRounds, numberOfPlayers)
	playerIndexMap := newPlayerIndexMap(players)
	t := &TournamentClassic{Matrix: pairings,
		Players:        players,
		PlayerIndexMap: playerIndexMap,
		PairingMethods: methods,
		GamesPerRound:  gamesPerRound}
	pairRoundClassic(t, 0)

	// We can make all non-standings dependent pairings right now
	for i := 1; i < numberOfRounds; i++ {
		if methods[i] == RoundRobin || methods[i] == Random {
			pairRoundClassic(t, i)
			isElimination = true
		}
	}

	return t, nil
}

func (t *TournamentClassic) Serialize() (datatypes.JSON, error) {
	json, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	return json, err
}

func TournamentClassicUnserialize(blob datatypes.JSON) (*TournamentClassic, error) {
	var tc TournamentClassic
	err := json.Unmarshal(blob, &tc)
	if err != nil {
		return nil, err
	}
	return &tc, nil
}

func (t *TournamentClassic) StartRound(round int) error {
	// Not sure yet
	return nil
}

func (t *TournamentClassic) GetPlayerRoundInfo(player string, round int) (*PlayerRoundInfo, error) {
	if round >= len(t.Matrix) || round < 0 {
		return nil, errors.New(fmt.Sprintf("Round number out of range: %d\n", round))
	}
	roundPairings := t.Matrix[round]

	playerIndex, ok := t.PlayerIndexMap[player]
	if !ok {
		return nil, errors.New(fmt.Sprintf("Player does not exist in the tournament: %s\n", player))
	}
	return roundPairings[playerIndex], nil
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

	newPairing := newClassicPairing(playerOne, playerTwo, t.GamesPerRound)
	playerOneInfo.Pairing = newPairing
	playerTwoInfo.Pairing = newPairing
	return nil
}

func (t *TournamentClassic) SubmitResult(round int,
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
	pairingMethod := t.PairingMethods[round]

	// For Elimination tournaments only.
	// Could be a tiebreaking result or could be an out of range
	// game index
	if pairingMethod == Elimination && gameIndex >= t.GamesPerRound {
		if gameIndex != len(pairing.Games) {
			return errors.New(fmt.Sprintf("Submitted tiebreaking result with invalid game index."+
				" Player 1: %s, Player 2: %s, Round: %d, GameIndex: %d\n", p1, p2, round, gameIndex))
		} else {
			pairing.Games = append(pairing.Games,
				&TournamentGame{Scores: []int{0, 0},
					Results: []realtime.TournamentGameResult{realtime.TournamentGameResult_NO_RESULT,
						realtime.TournamentGameResult_NO_RESULT}})
		}
	}

	if !amend && ((pairing.Outcomes[0] != realtime.TournamentGameResult_NO_RESULT &&
		pairing.Outcomes[1] != realtime.TournamentGameResult_NO_RESULT) ||
		pairing.Games[gameIndex].Results[0] != realtime.TournamentGameResult_NO_RESULT &&
			pairing.Games[gameIndex].Results[1] != realtime.TournamentGameResult_NO_RESULT) {
		return errors.New("This result is already submitted")
	}

	if pairingMethod == Elimination {
		if amend {

			p1ScoreOld := pairing.Games[gameIndex].Scores[0]
			p2ScoreOld := pairing.Games[gameIndex].Scores[1]
			pri1.Spread -= p1ScoreOld - p2ScoreOld
			pri2.Spread -= p2ScoreOld - p1ScoreOld
		}

		pairing.Games[gameIndex].Scores[0] = p1Score
		pairing.Games[gameIndex].Scores[1] = p2Score
		pairing.Games[gameIndex].Results[0] = p1Result
		pairing.Games[gameIndex].Results[1] = p2Result
		pairing.Games[gameIndex].GameEndReason = reason
		pri1.Spread += p1Score - p2Score
		pri2.Spread += p2Score - p1Score

		// A possible amendment to the outcomes.
		// If the outcomes remain unchanged, this will
		// just be undone in the next if block.
		if pairing.Outcomes[0] != realtime.TournamentGameResult_NO_RESULT &&
			pairing.Outcomes[1] != realtime.TournamentGameResult_NO_RESULT {
			pri1.Record[pairing.Outcomes[0]] -= 1
			pri2.Record[pairing.Outcomes[1]] -= 1
		}

		newOutcomes := getEliminationOutcomes(pairing.Games, t.GamesPerRound)

		pairing.Outcomes[0] = newOutcomes[0]
		pairing.Outcomes[1] = newOutcomes[1]

		if pairing.Outcomes[0] != realtime.TournamentGameResult_NO_RESULT &&
			pairing.Outcomes[1] != realtime.TournamentGameResult_NO_RESULT {
			pri1.Record[pairing.Outcomes[0]] += 1
			pri2.Record[pairing.Outcomes[1]] += 1
		}
	} else {
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
	}

	complete, err := t.IsRoundComplete(round)
	if err != nil {
		return err
	}
	finished, err := t.IsFinished()
	if err != nil {
		return err
	}

	// Copy the records over to the next round is the round is
	// over.
	// Only pair if this round is complete and the tournament
	// is not over. Don't pair for round robin and random since those pairings
	// were made when the tournament was created.
	if !finished && complete {
		copyRecords(t.Matrix, round)
		nextPairingMethod := t.PairingMethods[round+1]
		if nextPairingMethod != RoundRobin && nextPairingMethod != Random {
			pairRoundClassic(t, round+1)
		}
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

	pairingMethod := t.PairingMethods[round]

	// The difference for Elimination is that the original order
	// of the player list must be preserved. This is how we keep
	// track of the "bracket", which is simply modeled by an
	// array in this implementation. To keep this order, the
	// index in the tournament matrix is used as a tie breaker
	// for wins. In this way, The groupings are preserved across
	// rounds.
	if pairingMethod == Elimination {
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

func (t *TournamentClassic) IsRoundComplete(round int) (bool, error) {
	if round >= len(t.Matrix) || round < 0 {
		return false, errors.New(fmt.Sprintf("Round number out of range: %d\n", round))
	}
	for _, pri := range t.Matrix[round] {
		if pri.Pairing == nil || pri.Pairing.Outcomes[0] == realtime.TournamentGameResult_NO_RESULT ||
			pri.Pairing.Outcomes[1] == realtime.TournamentGameResult_NO_RESULT {
			return false, nil
		}
	}
	return true, nil
}

func (t *TournamentClassic) IsFinished() (bool, error) {
	return t.IsRoundComplete(len(t.Matrix) - 1)
}

func AddDirectors(t *Tournament, directors *TournamentPersons) error {
	return addTournamentPersons(t, directors, false)
}

func RemoveDirectors(t *Tournament, directors *TournamentPersons) error {
	return removeTournamentPersons(t, directors, false)
}

func AddPlayers(t *Tournament, players *TournamentPersons) error {
	return addTournamentPersons(t, players, true)
}

func RemovePlayers(t *Tournament, players *TournamentPersons) error {
	return removeTournamentPersons(t, players, true)
}

func addTournamentPersons(t *Tournament, persons *TournamentPersons, isPlayers bool) error {

	var personsMap map[string]int
	if isPlayers {
		personsMap = t.Players.Persons
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
	return nil
}

func removeTournamentPersons(t *Tournament, persons *TournamentPersons, isPlayers bool) error {

	var personsMap map[string]int
	if isPlayers {
		personsMap = t.Players.Persons
	} else {
		personsMap = t.Directors.Persons
	}

	// Only perform the remove operation if all persons can be removed.
	for k, _ := range persons.Persons {
		_, ok := personsMap[k]
		if !ok {
			return errors.New(fmt.Sprintf("Person (%s, %d) does not.", k, personsMap[k]))
		}
	}

	for k, _ := range persons.Persons {
		delete(personsMap, k)
	}
	return nil
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

func newPairingMatrix(numberOfRounds int, numberOfPlayers int) [][]*PlayerRoundInfo {
	pairings := [][]*PlayerRoundInfo{}
	for i := 0; i < numberOfRounds; i++ {
		roundPairings := []*PlayerRoundInfo{}
		for j := 0; j < numberOfPlayers; j++ {
			roundPairings = append(roundPairings, &PlayerRoundInfo{Record: emptyRecord(), Spread: 0})
		}
		pairings = append(pairings, roundPairings)
	}
	return pairings
}

func newClassicPairing(playerOne string, playerTwo string, gamesPerRound int) *Pairing {
	games := []*TournamentGame{}
	for i := 0; i < gamesPerRound; i++ {
		games = append(games, &TournamentGame{Scores: []int{0, 0},
			Results: []realtime.TournamentGameResult{realtime.TournamentGameResult_NO_RESULT,
				realtime.TournamentGameResult_NO_RESULT}})
	}
	return &Pairing{Players: []string{playerOne, playerTwo},
		Games: games,
		Outcomes: []realtime.TournamentGameResult{realtime.TournamentGameResult_NO_RESULT,
			realtime.TournamentGameResult_NO_RESULT}}
}

func newEliminatedPairing(playerOne string, playerTwo string) *Pairing {
	return &Pairing{Outcomes: []realtime.TournamentGameResult{realtime.TournamentGameResult_ELIMINATED,
		realtime.TournamentGameResult_ELIMINATED}}
}

func newPlayerIndexMap(players []string) map[string]int {
	m := make(map[string]int)
	for i, player := range players {
		m[player] = i
	}
	return m
}

func pairRoundClassic(t *TournamentClassic, round int) error {
	if round < 0 || round >= len(t.Matrix) {
		return errors.New("Round number out of range")
	}

	roundPairings := t.Matrix[round]
	pairingMethod := t.PairingMethods[round]
	// This automatic pairing could be the result of an
	// amendment. Undo all the pairings so byes can be
	// properly assigned (bye assignment checks for nil pairing).
	// Do not do this for manual pairings
	if pairingMethod != Manual {
		for i := 0; i < len(roundPairings); i++ {
			roundPairings[i].Pairing = nil
		}
	}

	if pairingMethod == KingOfTheHill || pairingMethod == Elimination {
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
		l := len(standings)
		for i := 0; i < l-1; i += 2 {
			playerOne := standings[i].Player
			playerTwo := standings[i+1].Player
			var newPairing *Pairing
			// If we are past the first round in an elimination tournament,
			// the bottom half of the standings have been eliminated.
			// Each successive round eliminates half as many players,
			// hence the l / twoPower(l) determines which players are eliminated.
			if pairingMethod == Elimination && round > 0 && i >= l/twoPower(round) {
				newPairing = newEliminatedPairing(playerOne, playerTwo)
			} else {
				newPairing = newClassicPairing(playerOne, playerTwo, t.GamesPerRound)
			}
			roundPairings[t.PlayerIndexMap[playerOne]].Pairing = newPairing
			roundPairings[t.PlayerIndexMap[playerTwo]].Pairing = newPairing
		}
	} else if pairingMethod == Random {
		playerIndexes := []int{}
		for _, v := range t.PlayerIndexMap {
			playerIndexes = append(playerIndexes, v)
		}
		// rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(playerIndexes),
			func(i, j int) {
				playerIndexes[i], playerIndexes[j] = playerIndexes[j], playerIndexes[i]
			})
		for i := 0; i < len(playerIndexes)-1; i += 2 {
			newPairing := newClassicPairing(t.Players[playerIndexes[i]], t.Players[playerIndexes[i+1]], t.GamesPerRound)
			roundPairings[playerIndexes[i]].Pairing = newPairing
			roundPairings[playerIndexes[i+1]].Pairing = newPairing
		}
	} else if pairingMethod == RoundRobin {
		roundRobinPlayers := t.Players
		// The empty string represents the bye
		if len(roundRobinPlayers)%2 == 1 {
			roundRobinPlayers = append(roundRobinPlayers, "")
		}

		roundRobinPairings := getRoundRobinPairings(roundRobinPlayers, round)
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

			newPairing := newClassicPairing(playerOne, playerTwo, t.GamesPerRound)
			roundPairings[t.PlayerIndexMap[playerOne]].Pairing = newPairing
			roundPairings[t.PlayerIndexMap[playerTwo]].Pairing = newPairing
		}
	}
	// Give all unpaired players a bye
	// realtime.TournamentGameResult_BYEs are always designated as a player
	// paired with themselves
	if pairingMethod != Manual {
		for i := 0; i < len(roundPairings); i++ {
			pri := roundPairings[i]
			if pri.Pairing == nil {
				pri.Pairing = newClassicPairing(t.Players[i], t.Players[i], t.GamesPerRound)
			}
		}
	}
	return nil
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

func getEliminationOutcomes(games []*TournamentGame, gamesPerRound int) []realtime.TournamentGameResult {
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
			p1Wins == p2Wins && p1Spread > p2Spread {
			p1Outcome = realtime.TournamentGameResult_WIN
			p2Outcome = realtime.TournamentGameResult_ELIMINATED
		} else if p2Wins > p1Wins ||
			p2Wins == p1Wins && p2Spread > p1Spread {
			p1Outcome = realtime.TournamentGameResult_ELIMINATED
			p2Outcome = realtime.TournamentGameResult_WIN
		}
	} else {
		if p1Wins > gamesPerRound ||
			p1Wins == gamesPerRound && p2Wins == gamesPerRound && p1Spread > p2Spread {
			p1Outcome = realtime.TournamentGameResult_WIN
			p2Outcome = realtime.TournamentGameResult_ELIMINATED
		} else if p2Wins > gamesPerRound ||
			p1Wins == gamesPerRound && p2Wins == gamesPerRound && p1Spread < p2Spread {
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
	for i := 0; i < int(realtime.TournamentGameResult_ELIMINATED)+1; i++ {
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

func twoPower(power int) int {
	product := 1
	for i := 0; i < power; i++ {
		product = product * 2
	}
	return product
}
