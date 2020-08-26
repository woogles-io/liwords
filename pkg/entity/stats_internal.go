package entity

import (
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/domino14/macondo/alphabet"
	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/macondo/gaddag"
	"github.com/domino14/macondo/game"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
	"strings"
	"unicode"
)

func makeAlphabetSubitems() map[string]int {
	alphabetSubitems := make(map[string]int)
	for i := 0; i < 26; i++ {
		alphabetSubitems[string('A'+rune(i))] = 0
	}
	alphabetSubitems[string(alphabet.BlankToken)] = 0
	return alphabetSubitems
}

func makeGameSubitems() map[string]int {
	gameSubitems := make(map[string]int)
	for _, value := range pb.ChallengeRule_name {
		gameSubitems[value] = 0
	}
	for _, value := range realtime.RatingMode_name {
		gameSubitems[value] = 0
	}
	return gameSubitems
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

func incrementStatItems(cfg *macondoconfig.Config,
	req *realtime.GameRequest,
	gameEndedEvent *realtime.GameEndedEvent,
	statItems map[string]*StatItem,
	otherPlayerStatItems map[string]*StatItem,
	history *pb.GameHistory,
	eventIndex int,
	id string,
	isPlayerOne bool) error {

	for key, statItem := range statItems {
		if (statItem.IncrementType == EventType && eventIndex >= 0) ||
			(statItem.IncrementType == GameType && eventIndex < 0) {
			var otherPlayerStatItem *StatItem
			if otherPlayerStatItems != nil {
				otherPlayerStatItem = otherPlayerStatItems[key]
			}
			info := &IncrementInfo{cfg: cfg,
				req:                 req,
				evt:                 gameEndedEvent,
				statItem:            statItem,
				otherPlayerStatItem: otherPlayerStatItem,
				history:             history,
				eventIndex:          eventIndex,
				id:                  id,
				isPlayerOne:         isPlayerOne}
			err := statItem.AddFunction(info)
			// LIST SEPARATION
			// Update the database here
			// if listItem
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func incrementStatItem(statItem *StatItem, event *pb.GameEvent, id string) {
	if statItem.DataType == SingleType || event == nil {
		statItem.Total++
	} else if statItem.DataType == ListType {
		statItem.Total++
		word := event.PlayedTiles
		if len(event.WordsFormed) > 0 {
			// We can have played tiles with words formed being empty
			// for phony_tiles_returned events, for example.
			word = event.WordsFormed[0]
		}
		statItem.List = append(statItem.List, &ListItem{Word: word, Probability: 1, Score: int(event.Score), GameId: id})
	}
}

func combineStatItemMaps(statItems map[string]*StatItem, otherStatItems map[string]*StatItem) error {
	for key, item := range statItems {
		itemOther := otherStatItems[key]
		combineItems(item, itemOther)
	}
	return nil
}

func confirmNotableItems(statItems map[string]*StatItem, id string) {
	for key, item := range statItems {
		// For one player plays every _ stats
		// Player one adds to the total, player two subtracts
		// So we need the absolute value to account for both possibilities
		if item.Total < 0 {
			item.Total = item.Total * (-1)
		}
		if item.Total >= item.Minimum && item.Total <= item.Maximum {
			log.Debug().Msgf("Notable confirmed: %s", key)
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
		// LIST SEPARATION
		// Eventually comment out the following line
		statItem.List = append(statItem.List, otherStatItem.List...)
	} else if (statItem.DataType == MaximumType && otherStatItem.Total > statItem.Total) ||
		(statItem.DataType == MinimumType && otherStatItem.Total < statItem.Total) {
		statItem.Total = otherStatItem.Total
		statItem.List = otherStatItem.List
	}

	if statItem.Subitems != nil {
		for key := range statItem.Subitems {
			statItem.Subitems[key] += otherStatItem.Subitems[key]
		}
	}
}

// This will need a list of GameIDS
func finalize(statItems map[string]*StatItem) {
	// For each statItem
	// LIST SEPARATION
	// If list is not empty
	// load the list from database
	return
}

func addBingos(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]

	var succEvent *pb.GameEvent
	if info.eventIndex+1 < len(events) {
		succEvent = events[info.eventIndex+1]
	}
	if event.IsBingo && (succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		log.Debug().Interface("evt", event).Interface("statitem", info.statItem).Interface("otherstat", info.otherPlayerStatItem).Msg("Add bing")
		incrementStatItem(info.statItem, event, info.id)
	}
	return nil
}

func addBlanksPlayed(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	tiles := event.PlayedTiles
	for _, char := range tiles {
		if unicode.IsLower(char) {
			info.statItem.Total++
		}
	}
	return nil
}

func addDoubleLetter(info *IncrementInfo) error {
	err := addBonusSquares(info, '\'')
	return err
}

func addDoubleWord(info *IncrementInfo) error {
	err := addBonusSquares(info, '-')
	return err
}

func addTripleLetter(info *IncrementInfo) error {
	err := addBonusSquares(info, '"')
	return err
}

func addTripleWord(info *IncrementInfo) error {
	err := addBonusSquares(info, '=')
	return err
}

func addCombinedScoring(info *IncrementInfo) error {
	info.statItem.Total = int(info.history.FinalScores[0] + info.history.FinalScores[1])
	return nil
}

func addHighScoring(info *IncrementInfo) error {
	playerScore := int(info.history.FinalScores[0])
	if !info.isPlayerOne {
		playerScore = int(info.history.FinalScores[1])
	}
	if playerScore > info.statItem.Total {
		info.statItem.Total = playerScore
	}
	return nil
}

func addExchanges(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	if event.Type == pb.GameEvent_EXCHANGE {
		incrementStatItem(info.statItem, event, info.id)
	}
	return nil
}

func addPhonies(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	isUnchallengedPhony, err := isUnchallengedPhonyEvent(event, info.history, info.cfg)
	if err != nil {
		return err
	}
	if event.Type == pb.GameEvent_PHONY_TILES_RETURNED || ((info.eventIndex+1 >= len(events) ||
		events[info.eventIndex+1].Type != pb.GameEvent_PHONY_TILES_RETURNED) && isUnchallengedPhony) {
		incrementStatItem(info.statItem, event, info.id)
	}
	return nil
}

func addChallengedPhonies(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	if event.Type == pb.GameEvent_PHONY_TILES_RETURNED {
		incrementStatItem(info.statItem, event, info.id)
	}
	return nil
}

func addUnchallengedPhonies(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	isUnchallengedPhony, err := isUnchallengedPhonyEvent(event, info.history, info.cfg)
	if err != nil {
		return err
	}
	if (info.eventIndex+1 >= len(events) || events[info.eventIndex+1].Type != pb.GameEvent_PHONY_TILES_RETURNED) && isUnchallengedPhony {
		incrementStatItem(info.statItem, event, info.id)
	}
	return nil
}

func addValidPlaysThatWereChallenged(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	var succEvent *pb.GameEvent
	if info.eventIndex+1 < len(events) {
		succEvent = events[info.eventIndex+1]
	}
	if succEvent != nil &&
		(succEvent.Type == pb.GameEvent_CHALLENGE_BONUS ||
			succEvent.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS) {
		incrementStatItem(info.statItem, event, info.id)
		// Need to increment opp's challengesLost stat
	}
	return nil
}

func addChallengesWon(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	if event.Type == pb.GameEvent_PHONY_TILES_RETURNED { // Opp's phony tiles returned
		incrementStatItem(info.otherPlayerStatItem, events[info.eventIndex-1], info.id)
	}
	return nil
}

func addChallengesLost(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	if event.Type == pb.GameEvent_CHALLENGE_BONUS { // Opp's bonus
		incrementStatItem(info.otherPlayerStatItem, events[info.eventIndex-1], info.id)
	} else if event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Player's turn loss
		incrementStatItem(info.statItem, events[info.eventIndex-1], info.id)
	}
	return nil
}

func addComments(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	if event.Note != "" {
		incrementStatItem(info.statItem, event, info.id)
	}
	return nil
}

func addConsecutiveBingos(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	player := "player_one_streak"
	if !info.isPlayerOne && info.history.Players[1].Nickname == event.Nickname {
		player = "player_two_streak"
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
		event.Type == pb.GameEvent_PASS ||
		event.Type == pb.GameEvent_EXCHANGE {
		if event.IsBingo {
			info.statItem.Subitems[player]++
			if info.statItem.Subitems[player] > info.statItem.Total {
				info.statItem.Total = info.statItem.Subitems[player]
			}
		} else {
			info.statItem.Subitems[player] = 0
		}
	}
	return nil
}

func addDraws(info *IncrementInfo) error {
	if info.history.Winner == -1 {
		incrementStatItem(info.statItem, nil, info.id)
	}
	return nil
}

func addEveryE(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	var succEvent *pb.GameEvent
	if info.eventIndex+1 < len(events) {
		succEvent = events[info.eventIndex+1]
	}
	multiplier := 1
	if !info.isPlayerOne && info.history.Players[1].Nickname == event.Nickname {
		multiplier = -1
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE && (succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		for _, char := range event.PlayedTiles {
			if char == 'E' {
				info.statItem.Total += 1 * multiplier
			}
		}
	}
	return nil
}

func addEveryPowerTile(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	var succEvent *pb.GameEvent
	if info.eventIndex+1 < len(events) {
		succEvent = events[info.eventIndex+1]
	}
	multiplier := 1
	if !info.isPlayerOne && info.history.Players[1].Nickname == event.Nickname {
		multiplier = -1
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE && (succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		for _, char := range event.PlayedTiles {
			if char == 'J' || char == 'Q' || char == 'X' || char == 'Z' || char == 'S' || unicode.IsLower(char) {
				info.statItem.Total += 1 * multiplier
			}
		}
	}
	return nil
}

func addFirsts(info *IncrementInfo) error {
	if info.isPlayerOne {
		incrementStatItem(info.statItem, nil, info.id)
	}
	return nil
}

func addGames(info *IncrementInfo) error {
	incrementStatItem(info.statItem, nil, info.id)
	info.statItem.Subitems[realtime.RatingMode_name[int32(info.req.RatingMode)]]++
	info.statItem.Subitems[pb.ChallengeRule_name[int32(info.req.ChallengeRule)]]++
	return nil
}

func addLosses(info *IncrementInfo) error {
	if (info.history.Winner == 1 && info.isPlayerOne) || (info.history.Winner == 0 && !info.isPlayerOne) {
		incrementStatItem(info.statItem, nil, info.id)
	}
	return nil
}

func addManyChallenges(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	if event.Type == pb.GameEvent_PHONY_TILES_RETURNED ||
		event.Type == pb.GameEvent_CHALLENGE_BONUS ||
		event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Do we want this last one?
		info.statItem.Total++
	}
	return nil
}

func addMistakes(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
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
				info.statItem.Total += occurences
				info.statItem.Subitems[mType] += occurences
				info.statItem.Subitems[unaliasedMistakeM] += occurences
				totalOccurences += occurences
				for k := 0; k < occurences; k++ {
					info.statItem.List = append(info.statItem.List, &ListItem{Word: event.Note, Probability: MistakeTypeMapping[mType], Score: MistakeMagnitudeMapping[unaliasedMistakeM], GameId: info.id})
				}
			}
			unspecifiedOccurences := strings.Count(note, "#"+mType)
			note = strings.ReplaceAll(note, "#"+mType, "")
			info.statItem.Total += unspecifiedOccurences
			info.statItem.Subitems[mType] += unspecifiedOccurences
			info.statItem.Subitems[UnspecifiedMistakeMagnitude] += unspecifiedOccurences
			for i := 0; i < unspecifiedOccurences-totalOccurences; i++ {
				info.statItem.List = append(info.statItem.List, &ListItem{Word: event.Note, Probability: MistakeTypeMapping[mType], Score: MistakeMagnitudeMapping[UnspecifiedMistakeMagnitude], GameId: info.id})
			}
		}
	}
	return nil
}

func addNoBingos(info *IncrementInfo) error {
	events := info.history.GetEvents()
	atLeastOneBingo := false
	// SAD! (have to loop through events again, should not do this, is the big not good)
	for i := 0; i < len(events); i++ {
		event := events[i]
		if (info.history.Players[0].Nickname == event.Nickname && info.isPlayerOne) ||
			(info.history.Players[1].Nickname == event.Nickname && !info.isPlayerOne) {
			atLeastOneBingo = true
			break
		}
	}
	if !atLeastOneBingo {
		incrementStatItem(info.statItem, nil, info.id)
	}
	return nil
}

func addRatings(info *IncrementInfo) error {
	var rating int32
	if info.isPlayerOne {
		rating = info.evt.NewRatings[info.history.Players[0].Nickname]
	} else {
		rating = info.evt.NewRatings[info.history.Players[1].Nickname]
	}
	info.statItem.Total++
	info.statItem.List = append(info.statItem.List, &ListItem{Word: "", Probability: 0, Score: int(rating), GameId: info.id})
	return nil
}

func addScore(info *IncrementInfo) error {
	if info.isPlayerOne {
		info.statItem.Total += int(info.history.FinalScores[0])
	} else {
		info.statItem.Total += int(info.history.FinalScores[1])
	}
	return nil
}

func addTilesPlayed(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	var succEvent *pb.GameEvent
	if info.eventIndex+1 < len(events) {
		succEvent = events[info.eventIndex+1]
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE && (succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		for _, char := range event.PlayedTiles {
			if char != alphabet.ASCIIPlayedThrough {
				info.statItem.Total++
				if unicode.IsLower(char) {
					info.statItem.Subitems[string(alphabet.BlankToken)]++
				} else {
					info.statItem.Subitems[string(char)]++
				}
			}
		}
	}
	return nil
}

func addTilesStuckWith(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[len(events)-1]
	if event.Type == pb.GameEvent_END_RACK_PTS &&
		((info.isPlayerOne && event.Nickname == info.history.Players[0].Nickname) ||
			(!info.isPlayerOne && event.Nickname == info.history.Players[1].Nickname)) {
		tilesStuckWith := event.Rack
		for _, char := range tilesStuckWith {
			info.statItem.Total++
			info.statItem.Subitems[string(char)]++
		}
	}
	return nil
}

func addTime(info *IncrementInfo) error {
	events := info.history.GetEvents()
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if (event.Nickname == info.history.Players[0].Nickname && info.isPlayerOne) ||
			(event.Nickname == info.history.Players[1].Nickname && !info.isPlayerOne) {
			info.statItem.Total += int((info.req.InitialTimeSeconds * 1000) - event.MillisRemaining)
		}
	}
	return nil
}

func addTripleTriples(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	var succEvent *pb.GameEvent
	if info.eventIndex+1 < len(events) {
		succEvent = events[info.eventIndex+1]
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE &&
		(succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) &&
		countBonusSquares(info, event, '=') >= 2 {
		incrementStatItem(info.statItem, event, info.id)
	}
	return nil
}

func addTurns(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
		event.Type == pb.GameEvent_PASS ||
		event.Type == pb.GameEvent_EXCHANGE ||
		event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Do we want this last one?
		incrementStatItem(info.statItem, event, info.id)
	}
	return nil
}

func addTurnsWithBlank(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
		event.Type == pb.GameEvent_PASS ||
		event.Type == pb.GameEvent_EXCHANGE ||
		event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Do we want this last one?
		for _, char := range event.Rack {
			if char == alphabet.BlankToken {
				incrementStatItem(info.statItem, event, info.id)
				break
			}
		}
	}
	return nil
}

func addVerticalOpenings(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[0]
	if info.isPlayerOne && events[0].Direction == pb.GameEvent_VERTICAL {
		incrementStatItem(info.statItem, event, info.id)
	}
	return nil
}

func addWins(info *IncrementInfo) error {
	if (info.history.Winner == 0 && info.isPlayerOne) || (info.history.Winner == 1 && !info.isPlayerOne) {
		incrementStatItem(info.statItem, nil, info.id)
	}
	return nil
}

func setHighGame(info *IncrementInfo) error {
	playerScore := int(info.history.FinalScores[0])
	if !info.isPlayerOne {
		playerScore = int(info.history.FinalScores[1])
	}
	if playerScore > info.statItem.Total {
		info.statItem.Total = playerScore
		info.statItem.List = []*ListItem{&ListItem{Word: "", Probability: 0, Score: 0, GameId: info.id}}
	}
	return nil
}

func setLowGame(info *IncrementInfo) error {
	playerScore := int(info.history.FinalScores[0])
	if !info.isPlayerOne {
		playerScore = int(info.history.FinalScores[1])
	}
	if playerScore < info.statItem.Total {
		info.statItem.Total = playerScore
		info.statItem.List = []*ListItem{&ListItem{Word: "", Probability: 0, Score: 0, GameId: info.id}}
	}
	return nil
}

func setHighTurn(info *IncrementInfo) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	score := int(event.Score)
	if score > info.statItem.Total {
		info.statItem.Total = score
		info.statItem.List = []*ListItem{&ListItem{Word: "", Probability: 0, Score: 0, GameId: info.id}}
	}
	return nil
}

func addBonusSquares(info *IncrementInfo, bonusSquare byte) error {
	events := info.history.GetEvents()
	event := events[info.eventIndex]
	var succEvent *pb.GameEvent
	if info.eventIndex+1 < len(events) {
		succEvent = events[info.eventIndex+1]
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE &&
		(succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		info.statItem.Total += countBonusSquares(info, event, bonusSquare)
	}
	return nil
}

func getOccupiedIndexes(event *pb.GameEvent) [][]int {
	occupied := [][]int{}
	row := int(event.Row)
	column := int(event.Column)
	for _, char := range event.PlayedTiles {
		if char != alphabet.ASCIIPlayedThrough {
			occupied = append(occupied, []int{row, column})
		}
		if event.Direction == pb.GameEvent_VERTICAL {
			row++
		} else {
			column++
		}
	}
	return occupied
}

func countBonusSquares(info *IncrementInfo, event *pb.GameEvent, bonusSquare byte) int {
	occupiedIndexes := getOccupiedIndexes(event)
	boardLayout, _ := game.HistoryToVariant(info.history)
	count := 0
	for j := 0; j < len(occupiedIndexes); j++ {
		rowAndColumn := occupiedIndexes[j]
		if boardLayout[rowAndColumn[0]][rowAndColumn[1]] == bonusSquare {
			count++
		}
	}
	return count
}

func isBingoNineOrAbove(event *pb.GameEvent) bool {
	return event.IsBingo && len(event.PlayedTiles) >= 9
}

func isUnchallengedPhonyEvent(event *pb.GameEvent, history *pb.GameHistory, cfg *macondoconfig.Config) (bool, error) {
	phony := false
	var err error
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE {
		gaddag, err := gaddag.GenericDawgCache.Get(cfg, history.Lexicon)
		if err != nil {
			return phony, err
		}
		phony, err = isPhony(gaddag, event.WordsFormed[0])
	}
	return phony, err
}

func isPhony(gd gaddag.GenericDawg, word string) (bool, error) {
	alph := gd.GetAlphabet()
	machineWord, err := alphabet.ToMachineWord(word, alph)
	if err != nil {
		return false, err
	}
	return !gaddag.FindMachineWord(gd, machineWord), nil
}
