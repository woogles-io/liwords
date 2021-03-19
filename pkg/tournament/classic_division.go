package tournament

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"

	"gorm.io/datatypes"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/pair"
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/rs/zerolog/log"
)

type ClassicDivision struct {
	Matrix     [][]string                           `json:"matrix"`
	PairingMap map[string]*realtime.PlayerRoundInfo `json:"pairingMap"`
	// By convention, players should look like userUUID:username
	Players           []string                                 `json:"players"`
	PlayersProperties []*realtime.PlayerProperties             `json:"playerProperties"`
	PlayerIndexMap    map[string]int32                         `json:"pidxMap"`
	RoundControls     []*realtime.RoundControl                 `json:"roundCtrls"`
	CurrentRound      int                                      `json:"currentRound"`
	AutoStart         bool                                     `json:"autoStart"`
	LastStarted       *realtime.TournamentRoundStarted         `json:"lastStarted"`
	Response          *realtime.TournamentDivisionDataResponse `json:"response"`
}

func NewClassicDivision(players []string,
	playerRatings *realtime.TournamentPersons,
	roundControls []*realtime.RoundControl,
	autoStart bool) (*ClassicDivision, error) {
	numberOfPlayers := len(players)
	numberOfRounds := len(roundControls)
	pairingMap := make(map[string]*realtime.PlayerRoundInfo)

	playersProperties := []*realtime.PlayerProperties{}
	for i := 0; i < numberOfPlayers; i++ {
		prop := newPlayerProperties()
		rating, ok := playerRatings.Persons[players[i]]
		if !ok {
			return nil, fmt.Errorf("player in player list does not exist in division players: %s", players[i])
		}
		prop.Rating = rating
		playersProperties = append(playersProperties, prop)
	}

	if numberOfPlayers < 2 || numberOfRounds < 1 {
		pairings := newPairingMatrix(numberOfRounds, numberOfPlayers)
		playerIndexMap := newPlayerIndexMap(players)
		t := &ClassicDivision{Matrix: pairings,
			PairingMap:        pairingMap,
			Players:           players,
			PlayersProperties: playersProperties,
			PlayerIndexMap:    playerIndexMap,
			RoundControls:     roundControls,
			AutoStart:         autoStart,
			CurrentRound:      -1}
		err := t.writeResponse(0)
		if err != nil {
			return nil, err
		}
		return t, nil
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
			return nil, errors.New("cannot mix Elimination pairings with any other pairing method")
		} else if i != 0 {
			if control.PairingMethod == realtime.PairingMethod_INITIAL_FONTES &&
				roundControls[i-1].PairingMethod != realtime.PairingMethod_INITIAL_FONTES {
				return nil, errors.New("cannot use Initial Fontes pairing when an earlier round used a different pairing method")
			} else if control.PairingMethod != realtime.PairingMethod_INITIAL_FONTES &&
				roundControls[i-1].PairingMethod == realtime.PairingMethod_INITIAL_FONTES {
				initialFontes = int32(i)
			}
		}
	}

	if initialFontes > 0 && initialFontes%2 == 0 {
		return nil, fmt.Errorf("number of rounds paired with Initial Fontes must be odd, have %d instead", initialFontes)
	}

	// For now, assume we require exactly n rounds and 2 ^ n players for an elimination tournament

	if roundControls[0].PairingMethod == realtime.PairingMethod_ELIMINATION {
		expectedNumberOfPlayers := 1 << numberOfRounds
		if expectedNumberOfPlayers != numberOfPlayers {
			return nil, fmt.Errorf("invalid number of players based on the number of rounds: "+
				" have %d players, expected %d players based on the number of rounds which is %d",
				numberOfPlayers, expectedNumberOfPlayers, numberOfRounds)
		}
	}

	for i := 0; i < numberOfRounds; i++ {
		roundControls[i].InitialFontes = initialFontes
		roundControls[i].Round = int32(i)
	}

	pairings := newPairingMatrix(numberOfRounds, numberOfPlayers)
	playerIndexMap := newPlayerIndexMap(players)
	t := &ClassicDivision{Matrix: pairings,
		PairingMap:        pairingMap,
		Players:           players,
		PlayersProperties: playersProperties,
		PlayerIndexMap:    playerIndexMap,
		RoundControls:     roundControls,
		AutoStart:         autoStart,
		CurrentRound:      -1}
	if roundControls[0].PairingMethod != realtime.PairingMethod_MANUAL {
		err := t.PairRound(0)
		if err != nil {
			return nil, err
		}
	}

	// We can make all standings independent pairings right now
	for i := 1; i < numberOfRounds; i++ {
		pm := roundControls[i].PairingMethod
		if pair.IsStandingsIndependent(pm) && pm != realtime.PairingMethod_MANUAL {
			err := t.PairRound(i)
			if err != nil {
				return nil, err
			}
		}
	}
	err := t.writeResponse(0)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *ClassicDivision) SetSingleRoundControls(round int, controls *realtime.RoundControl) error {
	if round >= len(t.Matrix) || round < 0 {
		return fmt.Errorf("round number out of range: %d", round)
	}
	err := t.writeResponse(round)
	if err != nil {
		return err
	}
	t.RoundControls[round] = controls
	return nil
}

func (t *ClassicDivision) SetPairing(playerOne string, playerTwo string, round int, isForfeit bool) error {
	if playerOne != playerTwo && isForfeit {
		return fmt.Errorf("forfeit results require that player one and two are identical, instead have: %s, %s", playerOne, playerTwo)
	}

	playerOneOpponent, err := t.opponentOf(playerOne, round)
	if err != nil {
		return err
	}

	playerTwoOpponent, err := t.opponentOf(playerTwo, round)
	if err != nil {
		return err
	}

	if playerOneOpponent != "" {
		err = t.clearPairingKey(playerOneOpponent, round)
		if err != nil {
			return err
		}
	}

	if playerTwoOpponent != "" {
		err = t.clearPairingKey(playerTwoOpponent, round)
		if err != nil {
			return err
		}
	}

	// The opponentOf calls protect against
	// out-of-range indexes
	playerOneProperties := t.PlayersProperties[t.PlayerIndexMap[playerOne]]
	playerTwoProperties := t.PlayersProperties[t.PlayerIndexMap[playerTwo]]

	newPairing := newClassicPairing(t, playerOne, playerTwo, round)
	pairingKey := makePairingKey(playerOne, playerTwo, round)
	t.PairingMap[pairingKey] = newPairing

	err = t.setPairingKey(playerOne, round, pairingKey)
	if err != nil {
		return err
	}

	err = t.setPairingKey(playerTwo, round, pairingKey)
	if err != nil {
		return err
	}

	// This pairing is a bye or forfeit, the result
	// can be submitted immediately
	if playerOne == playerTwo {

		score := entity.ByeScore
		tgr := realtime.TournamentGameResult_BYE
		if isForfeit {
			score = entity.ForfeitScore
			tgr = realtime.TournamentGameResult_FORFEIT_LOSS
		}
		// Use round < t.CurrentRound to satisfy
		// amendment checking. These results always need
		// to be submitted.
		err = t.SubmitResult(round,
			playerOne,
			playerOne,
			score,
			0,
			tgr,
			tgr,
			realtime.GameEndReason_NONE,
			round < t.CurrentRound,
			0,
			"")
		if err != nil {
			return err
		}
	} else if playerOneProperties.Removed || playerTwoProperties.Removed {
		err = t.SetPairing(playerOne, playerOne, round, playerOneProperties.Removed)
		if err != nil {
			return err
		}
		err = t.SetPairing(playerTwo, playerTwo, round, playerTwoProperties.Removed)
		if err != nil {
			return err
		}
	}
	err = t.writeResponse(round)
	if err != nil {
		return err
	}
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
	gameIndex int,
	id string) error {

	// Fetch the player round records
	pk1, err := t.getPairingKey(p1, round)
	if err != nil {
		return err
	}
	pk2, err := t.getPairingKey(p2, round)
	if err != nil {
		return err
	}

	// Ensure that this is the current round
	if round < t.CurrentRound && !amend {
		return fmt.Errorf("submitted result for a past round (%d) that is not marked as an amendment", round)
	}

	if round > t.CurrentRound && p1 != p2 {
		return fmt.Errorf("submitted result for a future round (%d) that is not a bye or forfeit", round)
	}

	// Ensure that the pairing exists
	if pk1 == "" {
		return fmt.Errorf("submitted result for a player with a null pairing: %s round (%d)", p1, round)
	}

	if pk2 == "" {
		return fmt.Errorf("submitted result for a player with a null pairing: %s round (%d)", p2, round)
	}

	// Ensure the submitted results were for players that were paired
	if pk1 != pk2 {
		log.Debug().Interface("pr1", pk1).Interface("pri2", pk2).Msg("not-play")
		return fmt.Errorf("submitted result for players that didn't play each other: %s (%s), %s (%s) round (%d)", p1, pk1, p2, pk2, round)
	}

	pairing, ok := t.PairingMap[pk1]
	if !ok {
		return fmt.Errorf("pairing does not exist in the pairing map: %s", pk1)
	}
	pairingMethod := t.RoundControls[round].PairingMethod

	if pairing.Games == nil {
		return fmt.Errorf("submitted result for a pairing with no initialized games: %s (%s), %s (%s) round (%d)", p1, pk1, p2, pk2, round)
	}

	// For Elimination tournaments only.
	// Could be a tiebreaking result or could be an out of range
	// game index
	if pairingMethod == realtime.PairingMethod_ELIMINATION && gameIndex >= int(t.RoundControls[round].GamesPerRound) {
		if gameIndex != len(pairing.Games) {
			return fmt.Errorf("submitted tiebreaking result with invalid game index."+
				" Player 1: %s, Player 2: %s, Round: %d, GameIndex: %d", p1, p2, round, gameIndex)
		} else {
			pairing.Games = append(pairing.Games,
				&realtime.TournamentGame{Scores: []int32{0, 0},
					Results: []realtime.TournamentGameResult{realtime.TournamentGameResult_NO_RESULT,
						realtime.TournamentGameResult_NO_RESULT}})
		}
	}

	if gameIndex >= len(pairing.Games) {
		return fmt.Errorf("submitted result where game index is out of range: %d >= %d", gameIndex, len(pairing.Games))
	}

	// If this is not an amendment, but attempts to amend a result, reject
	// this submission.
	if !amend && ((pairing.Outcomes[0] != realtime.TournamentGameResult_NO_RESULT &&
		pairing.Outcomes[1] != realtime.TournamentGameResult_NO_RESULT) ||

		pairing.Games[gameIndex].Results[0] != realtime.TournamentGameResult_NO_RESULT &&
			pairing.Games[gameIndex].Results[1] != realtime.TournamentGameResult_NO_RESULT) {
		return fmt.Errorf("result is already submitted for round %d, %s vs. %s", round, p1, p2)
	}

	// If this claims to be an amendment and is not submitting forfeit
	// losses for players show up late, reject this submission.
	if amend && p1Result != realtime.TournamentGameResult_FORFEIT_LOSS &&
		p2Result != realtime.TournamentGameResult_FORFEIT_LOSS &&
		(pairing.Games[gameIndex].Results[0] == realtime.TournamentGameResult_NO_RESULT &&
			pairing.Games[gameIndex].Results[1] == realtime.TournamentGameResult_NO_RESULT) {
		return fmt.Errorf("submitted amendment for a result that does not exist in round %d, %s vs. %s", round, p1, p2)
	}

	p1Index := 0
	if pairing.Players[1] == p1 {
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
		return err
	}
	finished, err := t.IsFinished()
	if err != nil {
		return err
	}

	// Only pair if this round is complete and the tournament
	// is not over. Don't pair for standings independent pairings since those pairings
	// were made when the tournament was created.
	if roundComplete && !finished && !amend {
		if !pair.IsStandingsIndependent(t.RoundControls[round+1].PairingMethod) {
			err = t.PairRound(round + 1)
			if err != nil {
				// Should probably use defer
				errWrite := t.writeResponse(round)
				if errWrite != nil {
					return errWrite
				}
				return err
			}
		}
		if t.AutoStart {
			err = t.StartRound()
			if err != nil {
				// Should probably use defer
				errWrite := t.writeResponse(round)
				if errWrite != nil {
					return errWrite
				}
				return err
			}
		}
	}
	err = t.writeResponse(round)
	if err != nil {
		return err
	}
	return nil
}

func (t *ClassicDivision) GetCurrentRound() int {
	return int(t.CurrentRound)
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
		t.clearPairingKey(t.Players[i], round)
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
	if pairingMethod == realtime.PairingMethod_ROUND_ROBIN {
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
		if pairingMethod != realtime.PairingMethod_ROUND_ROBIN {
			pm.Wins = int(standings[i].Wins)
			pm.Draws = int(standings[i].Draws)
			pm.Spread = int(standings[i].Spread)
		} else {
			pm.Wins = 0
			pm.Draws = 0
			pm.Spread = 0
		}
		playerIndex, ok := t.PlayerIndexMap[playerOrder[i]]
		if !ok {
			return fmt.Errorf("player %s does not exist in the player index map (PairRound)", playerOrder[i])
		}
		pm.Removed = t.PlayersProperties[playerIndex].Removed
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

		if roundPairings[playerIndex] == "" {

			var opponentIndex int32
			if pairings[i] < 0 {
				opponentIndex = playerIndex
			} else if pairings[i] >= l {
				fmt.Println(pairings)
				return fmt.Errorf("invalid pairing for round %d: %d", round, pairings[i])
			} else {
				opponentIndex = t.PlayerIndexMap[playerOrder[pairings[i]]]
			}

			playerName := t.Players[playerIndex]
			opponentName := t.Players[opponentIndex]

			if pairingMethod == realtime.PairingMethod_ELIMINATION && round > 0 && i >= l>>round {
				pairingKey := makePairingKey(playerName, opponentName, round)
				t.PairingMap[pairingKey] = newEliminatedPairing(playerName, opponentName)
				roundPairings[playerIndex] = pairingKey
			} else {
				err = t.SetPairing(playerName, opponentName, round, (playerName == opponentName) && t.PlayersProperties[playerIndex].Removed)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (t *ClassicDivision) AddPlayers(persons *realtime.TournamentPersons) error {

	// Redundant players have already been checked for
	for personId, personRating := range persons.Persons {
		t.Players = append(t.Players, personId)
		prop := newPlayerProperties()
		prop.Rating = personRating
		t.PlayersProperties = append(t.PlayersProperties, prop)
		t.PlayerIndexMap[personId] = int32(len(t.Players) - 1)
	}

	for i := 0; i < len(t.Matrix); i++ {
		for _, _ = range persons.Persons {
			t.Matrix[i] = append(t.Matrix[i], "")
		}
	}

	for i := 0; i < len(t.Matrix); i++ {
		if i <= t.CurrentRound {
			for personId, _ := range persons.Persons {
				// Set the pairing
				// This also automatically submits a forfeit result
				err := t.SetPairing(personId, personId, i, true)
				if err != nil {
					return err
				}
			}
		} else {
			pm := t.RoundControls[i].PairingMethod
			if (i == t.CurrentRound || pair.IsStandingsIndependent(pm)) && pm != realtime.PairingMethod_MANUAL {
				err := t.PairRound(i)
				if err != nil {
					return err
				}
			}
		}
	}
	err := t.writeResponse(t.CurrentRound)
	if err != nil {
		return err
	}
	return nil
}

func (t *ClassicDivision) RemovePlayers(persons *realtime.TournamentPersons) error {
	for personId, _ := range persons.Persons {
		playerIndex, ok := t.PlayerIndexMap[personId]
		if !ok {
			return fmt.Errorf("player %s does not exist in"+
				" classic division RemovePlayers", personId)
		}
		if playerIndex < 0 || int(playerIndex) >= len(t.Players) {
			return fmt.Errorf("player index %d for player %s is"+
				" out of range in classic division RemovePlayers", playerIndex, personId)
		}
	}

	playersRemaining := len(t.Players)
	for i := 0; i < len(t.PlayersProperties); i++ {
		if t.PlayersProperties[i].Removed {
			playersRemaining--
		}
	}

	if playersRemaining-len(persons.Persons) <= 0 {
		return fmt.Errorf("cannot remove players as tournament would be empty")
	}

	for personId, _ := range persons.Persons {
		t.PlayersProperties[t.PlayerIndexMap[personId]].Removed = true
	}

	for i := t.CurrentRound; i < len(t.Matrix); i++ {
		if i > t.CurrentRound {
			pm := t.RoundControls[i].PairingMethod
			if (i == t.CurrentRound || pair.IsStandingsIndependent(pm)) && pm != realtime.PairingMethod_MANUAL {
				err := t.PairRound(i)
				if err != nil {
					return err
				}
			}
		}
	}
	err := t.writeResponse(t.CurrentRound)
	if err != nil {
		return err
	}
	return nil
}

func (t *ClassicDivision) GetStandings(round int) ([]*realtime.PlayerStanding, error) {
	if round < 0 || round >= len(t.Matrix) {
		return nil, fmt.Errorf("round number out of range: %d", round)
	}

	var wins int32 = 0
	var losses int32 = 0
	var draws int32 = 0
	var spread int32 = 0
	player := ""
	records := []*realtime.PlayerStanding{}

	for i := 0; i < len(t.Players); i++ {
		wins = 0
		losses = 0
		draws = 0
		spread = 0
		player = t.Players[i]
		for j := 0; j <= round; j++ {
			pairingKey := t.Matrix[j][i]
			pairing, ok := t.PairingMap[pairingKey]
			if ok && pairing != nil && pairing.Players != nil {
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
		records = append(records, &realtime.PlayerStanding{Player: player,
			Wins:    wins,
			Losses:  losses,
			Draws:   draws,
			Spread:  spread,
			Removed: t.PlayersProperties[i].Removed})
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
				// If players were removed, they are listed last
				if (records[i].Removed && !records[j].Removed) || (!records[i].Removed && records[j].Removed) {
					return records[j].Removed
				} else if records[i].Wins == records[j].Wins && records[i].Draws == records[j].Draws && records[i].Spread == records[j].Spread {
					// Tiebreak alphabetically to ensure determinism
					return t.PlayerIndexMap[records[j].Player] > t.PlayerIndexMap[records[i].Player]
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
		return false, fmt.Errorf("round number out of range: %d", round)
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
		roundComplete, err := t.IsRoundComplete(t.CurrentRound)
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
			// Simply do nothing
			return nil
		}
	} else {
		if len(t.Players) < 2 || len(t.Matrix) < 1 {
			return errors.New("cannot start the the tournament with less than 2 players or less than 1 round")
		}
	}

	ready, err := t.IsRoundReady(t.CurrentRound + 1)
	if err != nil {
		return err
	}
	if !ready {
		return fmt.Errorf("cannot start round %d because it is not ready", t.CurrentRound+1)
	}

	t.CurrentRound = t.CurrentRound + 1

	err = t.writeResponse(t.CurrentRound)
	if err != nil {
		return err
	}
	return nil
}

func (t *ClassicDivision) SetLastStarted(ls *realtime.TournamentRoundStarted) error {
	t.LastStarted = ls
	err := t.writeResponse(t.CurrentRound)
	if err != nil {
		return err
	}
	return nil
}

// SetReadyForGame sets the playerID with the given connID to be ready for the game
// with the given 0-based round (and gameIndex, optionally). If `unready` is
// passed in, we make the player unready.
// It returns a list of playerId:username:connIDs involved in the game, a boolean saying if they're ready,
// and an optional error.
func (t *ClassicDivision) SetReadyForGame(playerID, connID string, round, gameIndex int, unready bool) ([]string, bool, error) {
	if round >= len(t.Matrix) || round < 0 {
		return nil, false, fmt.Errorf("round number out of range: %d", round)
	}
	toSet := connID
	if unready {
		toSet = ""
	}
	if t.CurrentRound != round {
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
			if playerID == pairing.Players[idx] {
				pairing.ReadyStates[idx] = toSet
			}
		}

		// Check to see if both players are ready.
		involvedPlayers := []string{
			pairing.Players[0] + ":" + pairing.ReadyStates[0],
			pairing.Players[1] + ":" + pairing.ReadyStates[1],
		}
		bothReady := pairing.ReadyStates[0] != "" && pairing.ReadyStates[1] != ""
		err = t.writeResponse(t.CurrentRound)
		if err != nil {
			return nil, false, err
		}
		return involvedPlayers, bothReady, nil
	}
	err = t.writeResponse(t.CurrentRound)
	if err != nil {
		return nil, false, err
	}
	return nil, false, nil
}

func (t *ClassicDivision) ClearReadyStates(playerID string, round, gameIndex int) error {
	if round >= len(t.Matrix) || round < 0 {
		return fmt.Errorf("round number out of range: %d", round)
	}
	// ignore gameIndex for classicdivision
	p, err := t.getPairing(playerID, round)
	if err != nil {
		return err
	}
	p.ReadyStates = []string{"", ""}
	err = t.writeResponse(round)
	if err != nil {
		return err
	}
	return nil
}

func (t *ClassicDivision) IsRoundComplete(round int) (bool, error) {
	if round >= len(t.Matrix) || round < 0 {
		return false, fmt.Errorf("round number out of range: %d", round)
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

func (t *ClassicDivision) ToResponse() (*realtime.TournamentDivisionDataResponse, error) {
	return t.Response, nil
}

func (t *ClassicDivision) writeResponse(round int) error {
	if len(t.Matrix) > 0 && (round >= len(t.Matrix) || round < 0) {
		return fmt.Errorf("round number out of range (writeResponse): %d", round)
	}
	if t.Response == nil {
		t.Response = &realtime.TournamentDivisionDataResponse{Standings: make(map[int32]*realtime.RoundStandings)}
	}

	realtimeTournamentControls := &realtime.TournamentControls{RoundControls: []*realtime.RoundControl{}}
	for i := 0; i < len(t.RoundControls); i++ {
		realtimeTournamentControls.RoundControls = append(realtimeTournamentControls.RoundControls, t.RoundControls[i])
	}

	division := []string{}

	newKeyInt := 0
	skinnyPairingsMap := make(map[string]*realtime.PlayerRoundInfo)
	skinniedMappings := make(map[string]string)

	for i := 0; i < len(t.Matrix); i++ {
		for j := 0; j < len(t.Matrix[i]); j++ {
			oldKey := t.Matrix[i][j]
			newKey := ""
			if oldKey != "" {
				var ok bool
				newKey, ok = skinniedMappings[oldKey]
				if !ok {
					oldPRI := t.PairingMap[oldKey]
					if oldPRI != nil {
						newKey = fmt.Sprintf("%d", newKeyInt)
						var newPlayers []string
						if oldPRI.Players != nil {
							playerOneIndex, ok := t.PlayerIndexMap[oldPRI.Players[0]]
							if !ok {
								return fmt.Errorf("player %s does not exist in pairing map (writeResponse)", oldPRI.Players[0])
							}
							playerTwoIndex, ok := t.PlayerIndexMap[oldPRI.Players[1]]
							if !ok {
								return fmt.Errorf("player %s does not exist in pairing map (writeResponse)", oldPRI.Players[1])
							}
							newPlayers = []string{fmt.Sprintf("%d", playerOneIndex), fmt.Sprintf("%d", playerTwoIndex)}
						}
						newPRI := &realtime.PlayerRoundInfo{
							Games:       oldPRI.Games,
							Outcomes:    oldPRI.Outcomes,
							ReadyStates: oldPRI.ReadyStates,
							Players:     newPlayers}
						skinnyPairingsMap[newKey] = newPRI
						skinniedMappings[oldKey] = newKey
						newKeyInt++
					}
				}
			}
			division = append(division, newKey)
		}
	}

	standingsResponse := []*realtime.PlayerStanding{}

	if len(t.Matrix) > 0 && len(t.Players) > 0 {
		standings, err := t.GetStandings(round)
		if err != nil {
			return err
		}

		for i := 0; i < len(standings); i++ {
			standingResponse := &realtime.PlayerStanding{Player: standings[i].Player,
				Wins:   int32(standings[i].Wins),
				Losses: int32(standings[i].Losses),
				Draws:  int32(standings[i].Draws),
				Spread: int32(standings[i].Spread)}
			standingsResponse = append(standingsResponse, standingResponse)
		}
	}

	playersProperties := []*realtime.PlayerProperties{}
	for i := 0; i < len(t.PlayersProperties); i++ {
		playersProperties = append(playersProperties, &realtime.PlayerProperties{Removed: t.PlayersProperties[i].Removed})
	}

	isFinished, err := t.IsFinished()
	if err != nil {
		return err
	}

	t.Response.Players = t.Players
	// t.Response.Controls = realtimeTournamentControls
	t.Response.Division = division
	t.Response.PairingMap = skinnyPairingsMap
	// t.Response.PlayerIndexMap = t.PlayerIndexMap
	t.Response.PlayersProperties = playersProperties
	t.Response.CurrentRound = int32(t.CurrentRound)
	t.Response.Finished = isFinished
	t.Response.Standings[int32(round)] = &realtime.RoundStandings{Standings: standingsResponse}

	return nil
}

func (t *ClassicDivision) Serialize() (datatypes.JSON, error) {
	json, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	return json, err
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
	playerOne string,
	playerTwo string,
	round int) *realtime.PlayerRoundInfo {

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
		playerOneFS := getPlayerFS(t, playerGoingFirst, round-1)
		playerTwoFS := getPlayerFS(t, playerGoingSecond, round-1)
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

	return &realtime.PlayerRoundInfo{Players: []string{playerGoingFirst, playerGoingSecond},
		Games: games,
		Outcomes: []realtime.TournamentGameResult{realtime.TournamentGameResult_NO_RESULT,
			realtime.TournamentGameResult_NO_RESULT},
		ReadyStates: []string{"", ""},
	}
}

func getPlayerFS(t *ClassicDivision, player string, round int) []int {

	playerMatrixIndex := t.PlayerIndexMap[player]

	fs := []int{0, 0}
	for i := 0; i <= round; i++ {
		pairingKey := t.Matrix[i][playerMatrixIndex]
		if pairingKey != "" {
			pairing, ok := t.PairingMap[pairingKey]
			if ok {
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
	}
	return fs
}

func newEliminatedPairing(playerOne string, playerTwo string) *realtime.PlayerRoundInfo {
	return &realtime.PlayerRoundInfo{Outcomes: []realtime.TournamentGameResult{realtime.TournamentGameResult_ELIMINATED,
		realtime.TournamentGameResult_ELIMINATED}}
}

func newPlayerIndexMap(players []string) map[string]int32 {
	m := make(map[string]int32)
	for i, player := range players {
		m[player] = int32(i)
	}
	return m
}

func newPlayerProperties() *realtime.PlayerProperties {
	return &realtime.PlayerProperties{Removed: false}
}

func getRepeats(t *ClassicDivision, round int) (map[string]int, error) {
	if round >= len(t.Matrix) {
		return nil, fmt.Errorf("round number out of range: %d", round)
	}
	repeats := make(map[string]int)
	for i := 0; i <= round; i++ {
		roundPairings := t.Matrix[i]
		for _, pairingKey := range roundPairings {
			pairing, ok := t.PairingMap[pairingKey]
			if ok && pairing.Players != nil {
				playerOne := pairing.Players[0]
				playerTwo := pairing.Players[1]
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

func (t *ClassicDivision) getPairing(player string, round int) (*realtime.PlayerRoundInfo, error) {
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
		return "", fmt.Errorf("round number out of range: %d", round)
	}

	playerIndex, ok := t.PlayerIndexMap[player]
	if !ok {
		return "", fmt.Errorf("player does not exist in the division: %s", player)
	}
	return t.Matrix[round][playerIndex], nil
}

func (t *ClassicDivision) setPairingKey(player string, round int, pairingKey string) error {
	if round >= len(t.Matrix) || round < 0 {
		return fmt.Errorf("round number out of range: %d", round)
	}

	playerIndex, ok := t.PlayerIndexMap[player]
	if !ok {
		return fmt.Errorf("player does not exist in the division: %s", player)
	}
	t.Matrix[round][playerIndex] = pairingKey
	return nil
}

func makePairingKey(playerOne string, playerTwo string, round int) string {
	if playerTwo < playerOne {
		playerOne, playerTwo = playerTwo, playerOne
	}
	return playerOne + "::" + playerTwo + "::" + fmt.Sprintf("%d", round)
}

func (t *ClassicDivision) clearPairingKey(player string, round int) error {
	if round >= len(t.Matrix) || round < 0 {
		return fmt.Errorf("round number out of range: %d", round)
	}

	playerIndex, ok := t.PlayerIndexMap[player]
	if !ok {
		return fmt.Errorf("player does not exist in the tournament: %s", player)
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
