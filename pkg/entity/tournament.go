package entity

import (
	"errors"
)

type Pairing struct {
	Player   int
	Opponent int
	Results  []Result
	Record   []int
}

type Result int

const (
	Unset Result = iota
	Win
	Loss
	Draw
	Bye
	ForfeitLoss
	ForfeitWin
	None
)

type Tournament interface {
	GetPairing(int, int) (Pairing, error)
	SubmitResult(int, int, int, Result, bool) error
	GetStandings(int) ([][]int, error)
}

type TournamentManager struct {
	Players    []string
	Tournament Tournament
	// StartTime time.Time
	// RoundDuration time.Time
}

type ClassicPairingMethod int

const (
	Random ClassicPairingMethod = iota
	KingOfTheHill
	Swiss
	Performance
)

type TournamentClassic struct {
	Matrix               [][]*Pairing
	ClassicPairingMethod ClassicPairingMethod
}

type TournamentElimination struct {
	Matrix          [][]*Pairing
	GamesPerRound   int
	NumberOfPlayers int
}

func NewTournamentClassic(players []string, numberOfRounds int, method ClassicPairingMethod) (*TournamentClassic, error) {
	pairings := [][]*Pairing{}
	for i := 0; i < numberOfRounds; i++ {
		roundPairings := []*Pairing{}
		for j := 0; j < len(players); j++ {
			pairing := &Pairing{Player: j, Opponent: -1, Results: []Result{Unset}, Record: emptyRecord()}
			roundPairings = append(roundPairings, pairing)
		}
		pairings = append(pairings, roundPairings)
	}

	return &TournamentClassic{Matrix: pairings, ClassicPairingMethod: method}, nil
}

func (t TournamentClassic) GetPairing(r int, p int) (*Pairing, error) {
	if r >= len(t.Matrix) || r < 0 {
		return nil, errors.New("Round number out of range")
	}
	roundPairings := t.Matrix[r]

	if p >= len(roundPairings) || p < 0 {
		return nil, errors.New("Player number out of range")
	}
	return roundPairings[p], nil
}

func (t TournamentClassic) SubmitResult(round int, player int, i int, result Result, amend bool) error {

	pairing, err := t.GetPairing(round, player)

	if err != nil {
		return err
	}

	if pairing.Opponent < 0 {
		return errors.New("This pairing has not been made yet")
	}

	if pairing.Results[0] != Unset && !amend {
		return errors.New("This result is already")
	}

	if amend {
		pairing.Record[pairing.Results[0]] -= 1
	}

	pairing.Results[0] = result
	pairing.Record[pairing.Results[0]] += 1

	if round+1 < len(t.Matrix) && t.roundIsComplete(round) {
		t.pairRound(round + 1)
	}

	return nil
}

func (t TournamentClassic) GetStandings(r int) ([][]int, error) {
	records := [][]int{}
	if r < 0 || r >= len(t.Matrix) {
		return nil, errors.New("Round number out of range")
	}
	for _, player := range t.Matrix[r] {
		records = append(records, player.Record)
	}
	return records, nil
}

func (t TournamentClassic) pairRound(r int) error {
	if r < 0 || r >= len(t.Matrix) {
		return errors.New("Round number out of range")
	}
	round := t.Matrix[r]
	if t.ClassicPairingMethod == KingOfTheHill {
		for _, pairing := range round {
			// 0 1 yes
			// 1 0 yes
			// 2 3 yes
			// 3 2
			// 4 5
			// 5 4
			pairing.Opponent = pairing.Player - (((pairing.Player % 2) * 2) - 1)
			previousPairing, err := t.GetPairing(r-1, pairing.Player)
			if err != nil {
				return err
			}
			pairing.Record = previousPairing.Record
		}
	}
	return nil
}

func (t TournamentClassic) roundIsComplete(r int) bool {
	for _, player := range t.Matrix[r] {
		if player.Results[0] == Unset {
			return false
		}
	}
	return true
}

func NewTournamentElimination(players []string, gamesPerRound int) (*TournamentElimination, error) {
	pairings := [][]*Pairing{}
	numberOfPlayersThisRound := len(players)
	for numberOfPlayersThisRound > 0 {
		if numberOfPlayersThisRound%2 != 0 {
			return nil, errors.New("The number of players for an elimination tournament must be a power of two")
		}
		roundPairings := []*Pairing{}
		for j := 0; j < numberOfPlayersThisRound; j++ {
			results := []Result{}
			for i := 0; i < gamesPerRound+1; i++ {
				results = append(results, Unset)
			}
			player := j
			if numberOfPlayersThisRound != len(players) {
				player = -1
			}
			pairing := &Pairing{Player: player, Opponent: -1, Results: results, Record: emptyRecord()}
			roundPairings = append(roundPairings, pairing)
		}
		pairings = append(pairings, roundPairings)
		numberOfPlayersThisRound = numberOfPlayersThisRound / 2
	}

	return &TournamentElimination{Matrix: pairings, GamesPerRound: gamesPerRound, NumberOfPlayers: len(players)}, nil
}

func (t TournamentElimination) GetPairing(r int, p int) (*Pairing, error) {
	if r >= len(t.Matrix) || r < 0 {
		return nil, errors.New("Round number out of range")
	}
	roundPairings := t.Matrix[r]

	for _, pairing := range roundPairings {
		if pairing.Player == p {
			return pairing, nil
		}
	}

	return nil, errors.New("Pairing not found")
}

func (t TournamentElimination) SubmitResult(round int, player int, result Result, i int, amend bool) error {

	pairing, err := t.GetPairing(round, player)

	if err != nil {
		return err
	}

	if pairing.Opponent < 0 {
		return errors.New("This pairing has not been made yet")
	}

	if pairing.Results[i] != Unset && !amend {
		return errors.New("This result is already")
	}

	pairing.Results[i] = result
	outcome := outcomeForRound(pairing.Results, t.GamesPerRound)

	if outcome != -1 {
		// Only a Win or Loss is possible for elimination
		if outcome == 1 {
			pairing.Record[Win] += 1
		} else {
			pairing.Record[Loss] += 1
		}
		//
		if round+1 < len(t.Matrix) && len(t.Matrix[round+1]) > 2 { // If true, we know we have another round to pair
			relativeIndex := i % 4
			var adjacentMatchOffset int
			var nextMatchOffset int
			if relativeIndex < 2 {
				adjacentMatchOffset = 2
				nextMatchOffset = 0
			} else {
				adjacentMatchOffset = -2
				nextMatchOffset = 1
			}
			adjacentPairing := t.Matrix[round][i+adjacentMatchOffset]
			adjacentOutcome := outcomeForRound(adjacentPairing.Results, t.GamesPerRound)
			if adjacentOutcome != -1 {
				// Adjacent pairing is done, we get to pair the next match
				var nextOpponent int
				var nextOpponentRecord []int
				if adjacentOutcome == 1 {
					nextOpponent = adjacentPairing.Player
					nextOpponentRecord = pairing.Record
				} else {
					nextOpponent = adjacentPairing.Opponent
					nextOpponentRecord = adjacentPairing.Record
				}
				baseIndexNextRound := (i / 4) * 2
				nextPlayerPairing := t.Matrix[round+1][baseIndexNextRound+nextMatchOffset]
				nextOpponentPairing := t.Matrix[round+1][baseIndexNextRound+(1-nextMatchOffset)]
				nextPlayerPairing.Player = player
				nextPlayerPairing.Opponent = nextOpponent
				nextPlayerPairing.Record = pairing.Record
				nextOpponentPairing.Player = nextOpponent
				nextOpponentPairing.Opponent = player
				nextOpponentPairing.Record = nextOpponentRecord
			}
		}
	}

	return nil
}

func (t TournamentElimination) GetStandings(r int) ([][]int, error) {
	return nil, nil
}

func outcomeForRound(results []Result, gamesPerRound int) int {
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
}

func emptyRecord() []int {
	record := []int{}
	for i := 0; i < int(None)+1; i++ {
		record = append(record, 0)
	}
	return record
}
