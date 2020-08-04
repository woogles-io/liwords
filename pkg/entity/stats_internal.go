package entity

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/domino14/macondo/alphabet"
	"github.com/domino14/macondo/gaddag"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
)

func makeAlphabetSubitems() map[string]int {
	alphabetSubitems := make(map[string]int)
	for i := 0; i < 26; i++ {
		alphabetSubitems[string('A'+rune(i))] = 0
	}
	alphabetSubitems[string(alphabet.BlankToken)] = 0
	return alphabetSubitems
}

func makeMistakeSubitems() map[string]int {
	mistakeSubitems := make(map[string]int)

	mistakeSubitems[KnowledgeMistakeType] = 0
	mistakeSubitems[FindingMistakeType] = 0
	mistakeSubitems[VisionMistakeType] = 0
	mistakeSubitems[TacticsMistakeType] = 0
	mistakeSubitems[StrategyMistakeType] = 0
	mistakeSubitems[TimeMistakeType] = 0
	mistakeSubitems[EndgameMistakeType] = 0
	mistakeSubitems[LargeMistakeMagnitude] = 0
	mistakeSubitems[MediumMistakeMagnitude] = 0
	mistakeSubitems[SmallMistakeMagnitude] = 0
	mistakeSubitems[UnspecifiedMistakeMagnitude] = 0

	return mistakeSubitems
}

func incrementStatItems(statItems []*StatItem, otherPlayerStatItems []*StatItem, history *pb.GameHistory, eventIndex int, id string, isPlayerOne bool) {
	for i := 0; i < len(statItems); i++ {
		statItem := statItems[i]
		if (statItem.IncrementType == EventType && eventIndex >= 0) ||
			(statItem.IncrementType == GameType && eventIndex < 0) {
			var otherPlayerStatItem *StatItem
			if otherPlayerStatItems != nil {
				otherPlayerStatItem = otherPlayerStatItems[i]
			}
			statItem.AddFunction(statItem, otherPlayerStatItem, history, eventIndex, id, isPlayerOne)
		}
	}
}

func incrementStatItem(statItem *StatItem, event *pb.GameEvent, id string) {
	if statItem.DataType == SingleType || event == nil {
		statItem.Total++
	} else if statItem.DataType == ListType {
		statItem.Total++
		statItem.List = append(statItem.List, &ListItem{Word: event.PlayedTiles, Probability: 1, Score: int(event.Score), GameId: id})
	}
}

func combineStatItemLists(statItems []*StatItem, otherStatItems []*StatItem) error {
	if len(statItems) != len(otherStatItems) {
		return errors.New("StatItem lists do not match in length")
	}
	for i := 0; i < len(statItems); i++ {
		item := statItems[i]
		itemOther := otherStatItems[i]
		if item.Name != itemOther.Name {
			return errors.New("StatItem names do not match")
		}
		combineItems(item, itemOther)
	}
	return nil
}

func confirmNotableItems(statItems []*StatItem, id string) {
	for i := 0; i < len(statItems); i++ {
		item := statItems[i]
		// For one player plays every _ stats
		// Player one adds to the total, player two subtracts
		// So we need the absolute value to account for both possibilities
		if item.Total < 0 {
			item.Total = item.Total * (-1)
		}
		if item.Total >= item.Minimum && item.Total <= item.Maximum {
			fmt.Printf("Notable confirmed: %s\n", item.Name)
			item.List = append(item.List, &ListItem{Word: "", Probability: 0, Score: 0, GameId: id})
		}
		item.Total = 0
	}
}

func combineItems(statItem *StatItem, otherStatItem *StatItem) {
	if statItem.DataType == SingleType {
		statItem.Total += otherStatItem.Total
	} else if statItem.DataType == ListType {
		statItem.Total += otherStatItem.Total
		statItem.List = append(statItem.List, otherStatItem.List...)
	} else if (statItem.DataType == MaximumType && otherStatItem.Total > statItem.Total) ||
		(statItem.DataType == MinimumType && otherStatItem.Total < statItem.Total) {
		statItem.Total = otherStatItem.Total
		statItem.List = otherStatItem.List
	}

	if statItem.Subitems != nil {
		for key, _ := range statItem.Subitems {
			statItem.Subitems[key] += otherStatItem.Subitems[key]
		}
	}
}

func finalize(statItems []*StatItem) {
	for _, statItem := range statItems {
		if statItem.IncrementType == FinalType {
			statItem.AddFunction(nil, nil, nil, -1, "", false)
		}
		var averages []float64
		for _, denominatorRef := range statItem.DenominatorList {
			averages = append(averages, float64(statItem.Total)/float64(denominatorRef.Total))
		}
		statItem.Averages = averages
	}
}
func addBingoNinesOrAbove(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	var succEvent *pb.GameEvent
	if i+1 < len(events) {
		succEvent = events[i+1]
	}
	if isBingoNineOrAbove(event) && (succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		incrementStatItem(statItem, event, id)
	}
}

func addBingos(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]

	var succEvent *pb.GameEvent
	if i+1 < len(events) {
		succEvent = events[i+1]
	}
	if event.IsBingo && (succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		incrementStatItem(statItem, event, id)
	}
}

func addBlanksPlayed(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	tiles := event.PlayedTiles
	for _, char := range tiles {
		if unicode.IsLower(char) {
			statItem.Total++
		}
	}
}

func addCombinedScoring(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	statItem.Total = int(history.FinalScores[0] + history.FinalScores[1])
}

func addConfidenceIntervals(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {

	// We'll need to decide if we even want this
	// We pull a sneaky here and overload some struct fields
	/*	tilesPlayedStat := statItem.Denominator
			gamesStat := tilesPlayedStat.Denominator
			totalGames := gamesStat.Total
			for tile, timesPlayed := range tilesPlayedStat.Subitems {
				bagSizes := 100 // Get real bag frequency somehow
				tileFrequency := 8 // Get real frequency somehow
		          my $tile_frequencies = Constants::TILE_FREQUENCIES;
		          my $f           = $tile_frequencies->{$subtitle};
		          my $P           = $average / 100;
		          my $n           = $f * $numgames;
		          my ($lower, $upper) = Utils::get_confidence_interval($P, $n);
		          my $prob        = sprintf "%.4f", $subaverage / $f;
			}*/
}

func addHighScoring(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	playerScore := int(history.FinalScores[0])
	if !isPlayerOne {
		playerScore = int(history.FinalScores[1])
	}
	if playerScore > statItem.Total {
		statItem.Total = playerScore
	}
}

func addExchanges(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.Type == pb.GameEvent_EXCHANGE {
		incrementStatItem(statItem, event, id)
	}
}

func addPhonies(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	isPhony := false // Add isPhony to GameEvent probably
	if isPhony {
		incrementStatItem(statItem, event, id)
	}
}

func addChallengedPhonies(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.Type == pb.GameEvent_PHONY_TILES_RETURNED {
		incrementStatItem(statItem, event, id)
		// Need to increment opp's challengesWon stat
	}
}

func addUnchallengedPhonies(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	isPhony := false // Add isPhony to GameEvent probably
	if isPhony {
		incrementStatItem(statItem, event, id)
	}
}

func addPlaysThatWereChallenged(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	var succEvent *pb.GameEvent
	if i+1 < len(events) {
		succEvent = events[i+1]
	}
	if succEvent != nil &&
		(succEvent.Type == pb.GameEvent_CHALLENGE_BONUS ||
			succEvent.Type == pb.GameEvent_PHONY_TILES_RETURNED) {
		incrementStatItem(statItem, event, id)
		// Need to increment opp's challengesLost stat
	}
}

func addChallenges(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.Type == pb.GameEvent_CHALLENGE_BONUS || // Opp's bonus
		event.Type == pb.GameEvent_PHONY_TILES_RETURNED { // Opp's phony tiles returned
		incrementStatItem(otherPlayerStatItem, events[i-1], id)
	} else if event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Player's turn loss
		incrementStatItem(statItem, events[i-1], id)
	}
}

func addChallengesWon(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.Type == pb.GameEvent_PHONY_TILES_RETURNED { // Opp's phony tiles returned
		incrementStatItem(otherPlayerStatItem, events[i-1], id)
	}
}

func addChallengesLost(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.Type == pb.GameEvent_CHALLENGE_BONUS { // Opp's bonus
		incrementStatItem(otherPlayerStatItem, events[i-1], id)
	} else if event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Player's turn loss
		incrementStatItem(statItem, events[i-1], id)
	}
}

func addComments(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.Note != "" {
		incrementStatItem(statItem, event, id)
	}
}

func addConsecutiveBingos(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	player := "player_one_streak"
	if !isPlayerOne && history.Players[1].Nickname == event.Nickname {
		player = "player_two_streak"
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
		event.Type == pb.GameEvent_PASS ||
		event.Type == pb.GameEvent_EXCHANGE {
		if event.IsBingo {
			statItem.Subitems[player]++
			if statItem.Subitems[player] > statItem.Total {
				statItem.Total = statItem.Subitems[player]
			}
		} else {
			statItem.Subitems[player] = 0
		}
	}
}

func addDraws(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	if history.Winner == -1 {
		incrementStatItem(statItem, nil, id)
	}
}

func addEveryE(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	var succEvent *pb.GameEvent
	if i+1 < len(events) {
		succEvent = events[i+1]
	}
	multiplier := 1
	if !isPlayerOne && history.Players[1].Nickname == event.Nickname {
		multiplier = -1
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE && (succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		for _, char := range event.PlayedTiles {
			if char == 'E' {
				statItem.Total += 1 * multiplier
			}
		}
	}
}

func addEveryPowerTile(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	var succEvent *pb.GameEvent
	if i+1 < len(events) {
		succEvent = events[i+1]
	}
	multiplier := 1
	if !isPlayerOne && history.Players[1].Nickname == event.Nickname {
		multiplier = -1
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE && (succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		for _, char := range event.PlayedTiles {
			if char == 'J' || char == 'Q' || char == 'X' || char == 'Z' || char == 'S' || unicode.IsLower(char) {
				statItem.Total += 1 * multiplier
				//fmt.Printf("therack: %s, the total: %d eventtype: %s, succ eventtype: %s\n", event.PlayedTiles, statItem.Total, event.Type, succEvent.Type)
			}
		}
	}
}

func addFirsts(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	if isPlayerOne {
		incrementStatItem(statItem, nil, id)
	}
}

func addGames(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	incrementStatItem(statItem, nil, id)
}

func addLosses(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	if (history.Winner == 1 && isPlayerOne) || (history.Winner == 0 && !isPlayerOne) {
		incrementStatItem(statItem, nil, id)
	}
}

func addManyChallenges(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.Type == pb.GameEvent_PHONY_TILES_RETURNED ||
		event.Type == pb.GameEvent_CHALLENGE_BONUS ||
		event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Do we want this last one?
		statItem.Total++
	}
}

func addMistakes(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	mistakeTypes := []string{KnowledgeMistakeType, FindingMistakeType, VisionMistakeType, TacticsMistakeType, StrategyMistakeType, TimeMistakeType, EndgameMistakeType}
	mistakeMagnitudes := []string{LargeMistakeMagnitude, MediumMistakeMagnitude, SmallMistakeMagnitude, "saddest", "sadder", "sad"}
	if event.Note != "" {
		note := strings.ToLower(event.Note) + " "
		for _, mType := range mistakeTypes {
			totalOccurences := 0
			for i := 0; i < len(mistakeMagnitudes); i++ {
				mMagnitude := mistakeMagnitudes[i]
				substring := "#" + mType + mMagnitude
				occurences := strings.Count(note, substring)
				note = strings.ReplaceAll(note, substring, "")
				unaliasedMistakeM := MistakeMagnitudeAliases[mMagnitude]
				statItem.Total += occurences
				statItem.Subitems[mType] += occurences
				statItem.Subitems[unaliasedMistakeM] += occurences
				totalOccurences += occurences
				for i := 0; i < occurences; i++ {
					statItem.List = append(statItem.List, &ListItem{Word: event.Note, Probability: MistakeTypeMapping[mType], Score: MistakeMagnitudeMapping[unaliasedMistakeM], GameId: id})
				}
			}
			unspecifiedOccurences := strings.Count(note, "#"+mType)
			note = strings.ReplaceAll(note, "#"+mType, "")
			statItem.Total += unspecifiedOccurences
			statItem.Subitems[mType] += unspecifiedOccurences
			statItem.Subitems[UnspecifiedMistakeMagnitude] += unspecifiedOccurences
			for i := 0; i < unspecifiedOccurences-totalOccurences; i++ {
				statItem.List = append(statItem.List, &ListItem{Word: event.Note, Probability: MistakeTypeMapping[mType], Score: MistakeMagnitudeMapping[UnspecifiedMistakeMagnitude], GameId: id})
			}
		}
	}
}

func addNoBingos(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	atLeastOneBingo := false
	// SAD! (have to loop through events again, should not do this, is the big not good)
	for i := 0; i < len(events); i++ {
		event := events[i]
		if (history.Players[0].Nickname == event.Nickname && isPlayerOne) ||
			(history.Players[1].Nickname == event.Nickname && !isPlayerOne) {
			atLeastOneBingo = true
			break
		}
	}
	if !atLeastOneBingo {
		incrementStatItem(statItem, nil, id)
	}
}

func addScore(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	if isPlayerOne {
		statItem.Total += int(history.FinalScores[0])
	} else {
		statItem.Total += int(history.FinalScores[1])
	}
}

func addTilesPlayed(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	var succEvent *pb.GameEvent
	if i+1 < len(events) {
		succEvent = events[i+1]
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE && (succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		for _, char := range event.PlayedTiles {
			if char != alphabet.ASCIIPlayedThrough {
				statItem.Total++
				if unicode.IsLower(char) {
					statItem.Subitems[string(alphabet.BlankToken)]++
				} else {
					statItem.Subitems[string(char)]++
				}
			}
		}
	}
}

func addTilesStuckWith(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[len(events)-1]
	if event.Type == pb.GameEvent_END_RACK_PTS &&
		((isPlayerOne && event.Nickname == history.Players[0].Nickname) ||
			(!isPlayerOne && event.Nickname == history.Players[1].Nickname)) {
		tilesStuckWith := event.Rack
		for _, char := range tilesStuckWith {
			statItem.Total++
			statItem.Subitems[string(char)]++
		}
	}
}

func addTripleTriples(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if isTripleTriple(event) {
		incrementStatItem(statItem, event, id)
	}
}

func addTurns(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
		event.Type == pb.GameEvent_PASS ||
		event.Type == pb.GameEvent_EXCHANGE ||
		event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Do we want this last one?
		incrementStatItem(statItem, event, id)
	}
}

func addTurnsWithBlank(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
		event.Type == pb.GameEvent_PASS ||
		event.Type == pb.GameEvent_EXCHANGE ||
		event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Do we want this last one?
		for _, char := range event.Rack {
			if char == alphabet.BlankToken {
				incrementStatItem(statItem, event, id)
				break
			}
		}
	}
}

func addVerticalOpenings(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[0]
	if isPlayerOne && events[0].Direction == pb.GameEvent_VERTICAL {
		incrementStatItem(statItem, event, id)
	}
}

func addWins(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	if (history.Winner == 0 && isPlayerOne) || (history.Winner == 1 && !isPlayerOne) {
		incrementStatItem(statItem, nil, id)
	}
}

func setHighGame(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	playerScore := int(history.FinalScores[0])
	if !isPlayerOne {
		playerScore = int(history.FinalScores[1])
	}
	if playerScore > statItem.Total {
		statItem.Total = playerScore
		statItem.List = []*ListItem{&ListItem{Word: "", Probability: 0, Score: 0, GameId: id}}
	}
}

func setLowGame(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	playerScore := int(history.FinalScores[0])
	if !isPlayerOne {
		playerScore = int(history.FinalScores[1])
	}
	if playerScore < statItem.Total {
		statItem.Total = playerScore
		statItem.List = []*ListItem{&ListItem{Word: "", Probability: 0, Score: 0, GameId: id}}
	}
}

func setHighTurn(statItem *StatItem, otherPlayerStatItem *StatItem, history *pb.GameHistory, i int, id string, isPlayerOne bool) {
	events := history.GetEvents()
	event := events[i]
	score := int(event.Score)
	if score > statItem.Total {
		statItem.Total = score
		statItem.List = []*ListItem{&ListItem{Word: "", Probability: 0, Score: 0, GameId: id}}
	}
}

func isTripleTriple(event *pb.GameEvent) bool {
	// Need to implement
	return false
}

func isBingoNineOrAbove(event *pb.GameEvent) bool {
	return event.IsBingo && len(event.PlayedTiles) >= 9
}

func validateWord(gd *gaddag.SimpleGaddag, word string) (bool, error) {
	alph := gd.GetAlphabet()
	machineWord, error := alphabet.ToMachineWord(word, alph)
	if error != nil {
		return false, error
	}
	return gaddag.FindMachineWord(gd, machineWord), nil
}
