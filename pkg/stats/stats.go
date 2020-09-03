package stats

import (
	"errors"
	"github.com/domino14/liwords/pkg/entity"
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

// InstantiateNewStats instantiates a new stats object. playerOneId MUST
// have gone first in the game.
func InstantiateNewStats(playerOneId string, playerTwoId string) *entity.Stats {
	log.Debug().Str("p1id", playerOneId).Str("p2id", playerTwoId).Msg("instantiating new stats.")
	return &entity.Stats{
		PlayerOneId:   playerOneId,
		PlayerTwoId:   playerTwoId,
		PlayerOneData: instantiatePlayerData(),
		PlayerTwoData: instantiatePlayerData(),
		NotableData:   instantiateNotableData()}
}

func AddGame(stats *entity.Stats, lss entity.ListStatStore, history *pb.GameHistory, req *realtime.GameRequest,
	cfg *macondoconfig.Config, gameEndedEvent *realtime.GameEndedEvent, gameId string) error {
	// Josh, plz fix these asinine calls to incrementStatItem
	events := history.GetEvents()

	info := &entity.IncrementInfo{Cfg: cfg,
		Req:     req,
		Evt:     gameEndedEvent,
		Lss:     lss,
		History: history,
		GameId:  gameId}

	for i := 0; i < len(events); i++ {
		event := events[i]
		if history.Players[0].Nickname == event.Nickname {
			err := incrementStatItems(info, stats.PlayerOneData, stats.PlayerTwoData, i, true)
			if err != nil {
				return err
			}
		} else {
			err := incrementStatItems(info, stats.PlayerTwoData, stats.PlayerOneData, i, false)
			if err != nil {
				return err
			}
		}
		err := incrementStatItems(info, stats.NotableData, nil, i, false)
		if err != nil {
			return err
		}
	}

	err := incrementStatItems(info, stats.PlayerOneData, stats.PlayerTwoData, -1, true)
	if err != nil {
		return err
	}

	err = incrementStatItems(info, stats.PlayerTwoData, stats.PlayerOneData, -1, false)
	if err != nil {
		return err
	}

	err = incrementStatItems(info, stats.NotableData, nil, -1, false)
	if err != nil {
		return err
	}

	confirmNotableItems(lss, gameId, gameEndedEvent.Time, stats.NotableData)
	return nil
}

func AddStats(stats *entity.Stats, otherStats *entity.Stats) error {

	if stats.PlayerOneId != otherStats.PlayerOneId &&
		stats.PlayerOneId != otherStats.PlayerTwoId {
		return errors.New("Stats do not share an identical PlayerOneId")
	}

	otherPlayerOneData := otherStats.PlayerOneData
	otherPlayerTwoData := otherStats.PlayerTwoData

	if stats.PlayerOneId == otherStats.PlayerTwoId {
		otherPlayerOneData = otherStats.PlayerTwoData
		otherPlayerTwoData = otherStats.PlayerOneData
	}

	err := combineStatItemMaps(stats.PlayerOneData, otherPlayerOneData)

	if err != nil {
		return err
	}

	err = combineStatItemMaps(stats.PlayerTwoData, otherPlayerTwoData)

	if err != nil {
		return err
	}

	err = combineStatItemMaps(stats.NotableData, otherStats.NotableData)

	if err != nil {
		return err
	}

	return nil
}

func Finalize(stats *entity.Stats, lss entity.ListStatStore, gameIds []string,
	playerOneId string, playerTwoId string) error {
	err := finalize(stats.PlayerOneData, lss, gameIds, playerOneId)
	if err != nil {
		return err
	}

	err = finalize(stats.PlayerTwoData, lss, gameIds, playerTwoId)

	if err != nil {
		return err
	}
	return nil
}

func finalize(statItems map[string]*entity.StatItem, lss entity.ListStatStore,
	gameIds []string, playerId string) error {
	for statName, statItem := range statItems {
		newList, err := lss.GetListItems(entity.StatName_value[statName], gameIds, playerId)
		if err != nil {
			return err
		}
		if len(newList) > 0 {
			statItem.List = newList
		}
	}
	return nil
}

func incrementStatItems(info *entity.IncrementInfo,
	statItems map[string]*entity.StatItem,
	otherPlayerStatItems map[string]*entity.StatItem,
	eventIndex int,
	isPlayerOne bool) error {

	playerId := info.History.Players[0].UserId
	otherPlayerId := info.History.Players[1].UserId

	if !isPlayerOne {
		playerId, otherPlayerId = otherPlayerId, playerId
	}

	info.EventIndex = eventIndex
	info.PlayerId = playerId
	info.OtherPlayerId = otherPlayerId
	info.IsPlayerOne = isPlayerOne

	for key, statItem := range statItems {
		if (statItem.IncrementType == entity.EventType && eventIndex >= 0) ||
			(statItem.IncrementType == entity.GameType && eventIndex < 0) {

			var otherPlayerStatItem *entity.StatItem
			if otherPlayerStatItems != nil {
				otherPlayerStatItem = otherPlayerStatItems[key]
			}

			info.StatName = key
			info.StatItem = statItem
			info.OtherPlayerStatItem = otherPlayerStatItem

			err := statItem.AddFunction(info)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func incrementStatItem(statItem *entity.StatItem,
	lss entity.ListStatStore,
	event *pb.GameEvent,
	gameId string,
	playerId string,
	time int64) {
	if statItem.DataType == entity.SingleType || event == nil {
		statItem.Total++
	} else if statItem.DataType == entity.ListType {
		statItem.Total++
		word := event.PlayedTiles
		if len(event.WordsFormed) > 0 {
			// We can have played tiles with words formed being empty
			// for phony_tiles_returned events, for example.
			word = event.WordsFormed[0]
		}
		lss.AddListItem(gameId,
			playerId,
			entity.StatName_value[statItem.Name],
			time, &entity.ListWord{Word: word, Probability: 1, Score: int(event.Score)})
	}
}

func combineStatItemMaps(statItems map[string]*entity.StatItem,
	otherStatItems map[string]*entity.StatItem) error {
	for key, item := range statItems {
		itemOther := otherStatItems[key]
		combineItems(item, itemOther)
	}
	return nil
}

func confirmNotableItems(lss entity.ListStatStore, gameId string, time int64, statItems map[string]*entity.StatItem) {
	for key, item := range statItems {
		// For one player plays every _ stats
		// Player one adds to the total, player two subtracts
		// So we need the absolute value to account for both possibilities
		if item.Total < 0 {
			item.Total = item.Total * (-1)
		}
		if item.Total >= item.Minimum && item.Total <= item.Maximum {
			log.Debug().Msgf("Notable confirmed: %s", key)
			lss.AddListItem(gameId, "", entity.StatName_value[key], time, &entity.ListGame{})
		}
		item.Total = 0
	}
}

func combineItems(statItem *entity.StatItem, otherStatItem *entity.StatItem) {
	if statItem.DataType == entity.SingleType {
		statItem.Total += otherStatItem.Total
	} else if statItem.DataType == entity.ListType {
		statItem.Total += otherStatItem.Total
	} else if (statItem.DataType == entity.MaximumType &&
		otherStatItem.Total > statItem.Total) ||
		(statItem.DataType == entity.MinimumType &&
			otherStatItem.Total < statItem.Total) {
		statItem.Total = otherStatItem.Total
		statItem.List = otherStatItem.List
	}

	if statItem.Subitems != nil {
		for key := range statItem.Subitems {
			statItem.Subitems[key] += otherStatItem.Subitems[key]
		}
	}
}

func addBingos(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]

	var succEvent *pb.GameEvent
	if info.EventIndex+1 < len(events) {
		succEvent = events[info.EventIndex+1]
	}
	if event.IsBingo && (succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		log.Debug().Interface("evt", event).Interface("statitem", info.StatItem).Interface("otherstat", info.OtherPlayerStatItem).Msg("Add bing")
		incrementStatItem(info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addBlanksPlayed(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	tiles := event.PlayedTiles
	for _, char := range tiles {
		if unicode.IsLower(char) {
			info.StatItem.Total++
		}
	}
	return nil
}

func addDoubleLetter(info *entity.IncrementInfo) error {
	err := addBonusSquares(info, '\'')
	return err
}

func addDoubleWord(info *entity.IncrementInfo) error {
	err := addBonusSquares(info, '-')
	return err
}

func addTripleLetter(info *entity.IncrementInfo) error {
	err := addBonusSquares(info, '"')
	return err
}

func addTripleWord(info *entity.IncrementInfo) error {
	err := addBonusSquares(info, '=')
	return err
}

func addCombinedScoring(info *entity.IncrementInfo) error {
	info.StatItem.Total = int(info.History.FinalScores[0] + info.History.FinalScores[1])
	return nil
}

func addHighScoring(info *entity.IncrementInfo) error {
	playerScore := int(info.History.FinalScores[0])
	if !info.IsPlayerOne {
		playerScore = int(info.History.FinalScores[1])
	}
	if playerScore > info.StatItem.Total {
		info.StatItem.Total = playerScore
	}
	return nil
}

func addExchanges(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Type == pb.GameEvent_EXCHANGE {
		incrementStatItem(info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addChallengedPhonies(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Type == pb.GameEvent_PHONY_TILES_RETURNED {
		incrementStatItem(info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addUnchallengedPhonies(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	isUnchallengedPhony, err := isUnchallengedPhonyEvent(event, info.History, info.Cfg)
	if err != nil {
		return err
	}
	if (info.EventIndex+1 >= len(events) ||
		events[info.EventIndex+1].Type != pb.GameEvent_PHONY_TILES_RETURNED) &&
		isUnchallengedPhony {
		incrementStatItem(info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addValidPlaysThatWereChallenged(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	var succEvent *pb.GameEvent
	if info.EventIndex+1 < len(events) {
		succEvent = events[info.EventIndex+1]
	}
	if succEvent != nil &&
		(succEvent.Type == pb.GameEvent_CHALLENGE_BONUS ||
			succEvent.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS) {
		incrementStatItem(info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
		// Need to increment opp's challengesLost stat
	}
	return nil
}

func addChallengesWon(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Type == pb.GameEvent_PHONY_TILES_RETURNED { // Opp's phony tiles returned
		incrementStatItem(info.OtherPlayerStatItem,
			info.Lss,
			events[info.EventIndex-1],
			info.GameId,
			info.OtherPlayerId,
			info.Evt.Time)
	}
	return nil
}

func addChallengesLost(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Type == pb.GameEvent_CHALLENGE_BONUS { // Opp's bonus
		incrementStatItem(info.OtherPlayerStatItem,
			info.Lss,
			events[info.EventIndex-1],
			info.GameId,
			info.OtherPlayerId,
			info.Evt.Time)
	} else if event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Player's turn loss
		incrementStatItem(info.StatItem,
			info.Lss, events[info.EventIndex-1],
			info.GameId,
			info.PlayerId,
			info.Evt.Time)
	}
	return nil
}

func addComments(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Note != "" {
		incrementStatItem(info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addConsecutiveBingos(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	player := "player_one_streak"
	if !info.IsPlayerOne && info.History.Players[1].Nickname == event.Nickname {
		player = "player_two_streak"
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
		event.Type == pb.GameEvent_PASS ||
		event.Type == pb.GameEvent_EXCHANGE {
		if event.IsBingo {
			info.StatItem.Subitems[player]++
			if info.StatItem.Subitems[player] > info.StatItem.Total {
				info.StatItem.Total = info.StatItem.Subitems[player]
			}
		} else {
			info.StatItem.Subitems[player] = 0
		}
	}
	return nil
}

func addDraws(info *entity.IncrementInfo) error {
	if info.History.Winner == -1 {
		incrementStatItem(info.StatItem, info.Lss, nil, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addEveryE(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	var succEvent *pb.GameEvent
	if info.EventIndex+1 < len(events) {
		succEvent = events[info.EventIndex+1]
	}
	multiplier := 1
	if !info.IsPlayerOne &&
		info.History.Players[1].Nickname == event.Nickname {
		multiplier = -1
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE &&
		(succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		for _, char := range event.PlayedTiles {
			if char == 'E' {
				info.StatItem.Total += 1 * multiplier
			}
		}
	}
	return nil
}

func addEveryPowerTile(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	var succEvent *pb.GameEvent
	if info.EventIndex+1 < len(events) {
		succEvent = events[info.EventIndex+1]
	}
	multiplier := 1
	if !info.IsPlayerOne && info.History.Players[1].Nickname == event.Nickname {
		multiplier = -1
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE &&
		(succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		for _, char := range event.PlayedTiles {
			if char == 'J' ||
				char == 'Q' ||
				char == 'X' ||
				char == 'Z' ||
				char == 'S' ||
				unicode.IsLower(char) {
				info.StatItem.Total += 1 * multiplier
			}
		}
	}
	return nil
}

func addFirsts(info *entity.IncrementInfo) error {
	if info.IsPlayerOne {
		incrementStatItem(info.StatItem, info.Lss, nil, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addGames(info *entity.IncrementInfo) error {
	incrementStatItem(info.StatItem, info.Lss, nil, info.GameId, info.PlayerId, info.Evt.Time)
	info.StatItem.Subitems[realtime.RatingMode_name[int32(info.Req.RatingMode)]]++
	info.StatItem.Subitems[pb.ChallengeRule_name[int32(info.Req.ChallengeRule)]]++
	return nil
}

func addLosses(info *entity.IncrementInfo) error {
	if (info.History.Winner == 1 && info.IsPlayerOne) || (info.History.Winner == 0 && !info.IsPlayerOne) {
		incrementStatItem(info.StatItem, info.Lss, nil, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addManyChallenges(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Type == pb.GameEvent_PHONY_TILES_RETURNED ||
		event.Type == pb.GameEvent_CHALLENGE_BONUS ||
		event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Do we want this last one?
		info.StatItem.Total++
	}
	return nil
}

func addMistakes(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	mistakeTypes := []string{entity.KnowledgeMistakeType,
		entity.FindingMistakeType,
		entity.VisionMistakeType,
		entity.TacticsMistakeType,
		entity.StrategyMistakeType,
		entity.TimeMistakeType,
		entity.EndgameMistakeType}
	mistakeMagnitudes := []string{entity.LargeMistakeMagnitude,
		entity.MediumMistakeMagnitude,
		entity.SmallMistakeMagnitude,
		"saddest",
		"sadder",
		"sad"}
	if event.Note != "" {
		note := strings.ToLower(event.Note) + " "
		for _, mType := range mistakeTypes {
			totalOccurences := 0
			for i := 0; i < len(mistakeMagnitudes); i++ {
				mMagnitude := mistakeMagnitudes[i]
				substring := "#" + mType + mMagnitude
				occurences := strings.Count(note, substring)
				note = strings.ReplaceAll(note, substring, "")
				unaliasedMistakeM := entity.MistakeMagnitudeAliases[mMagnitude]
				info.StatItem.Total += occurences
				info.StatItem.Subitems[mType] += occurences
				info.StatItem.Subitems[unaliasedMistakeM] += occurences
				totalOccurences += occurences
				for k := 0; k < occurences; k++ {
					info.Lss.AddListItem(info.GameId,
						info.PlayerId,
						entity.StatName_value[entity.MISTAKES_STAT],
						info.Evt.Time,
						&entity.ListMistake{Type: entity.MistakeTypeMapping[mType],
							Size: entity.MistakeMagnitudeMapping[unaliasedMistakeM]})
				}
			}
			unspecifiedOccurences := strings.Count(note, "#"+mType)
			note = strings.ReplaceAll(note, "#"+mType, "")
			info.StatItem.Total += unspecifiedOccurences
			info.StatItem.Subitems[mType] += unspecifiedOccurences
			info.StatItem.Subitems[entity.UnspecifiedMistakeMagnitude] += unspecifiedOccurences
			for i := 0; i < unspecifiedOccurences-totalOccurences; i++ {
				info.Lss.AddListItem(info.GameId,
					info.PlayerId,
					entity.StatName_value[entity.MISTAKES_STAT],
					info.Evt.Time,
					&entity.ListMistake{Type: entity.MistakeTypeMapping[mType],
						Size: entity.MistakeMagnitudeMapping[entity.UnspecifiedMistakeMagnitude]})
			}
		}
	}
	return nil
}

func addNoBingos(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	atLeastOneBingo := false
	// SAD! (have to loop through events again, should not do this, is the big not good)
	for i := 0; i < len(events); i++ {
		event := events[i]
		if (info.History.Players[0].Nickname == event.Nickname && info.IsPlayerOne) ||
			(info.History.Players[1].Nickname == event.Nickname && !info.IsPlayerOne) {
			atLeastOneBingo = true
			break
		}
	}
	if !atLeastOneBingo {
		incrementStatItem(info.StatItem, info.Lss, nil, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addRatings(info *entity.IncrementInfo) error {
	var rating int32
	if info.IsPlayerOne {
		rating = info.Evt.NewRatings[info.History.Players[0].Nickname]
	} else {
		rating = info.Evt.NewRatings[info.History.Players[1].Nickname]
	}
	info.StatItem.Total++
	info.Lss.AddListItem(info.GameId,
		info.PlayerId,
		entity.StatName_value[entity.RATINGS_STAT],
		info.Evt.Time,
		&entity.ListRating{Rating: int(rating)})
	return nil
}

func addScore(info *entity.IncrementInfo) error {
	if info.IsPlayerOne {
		info.StatItem.Total += int(info.History.FinalScores[0])
	} else {
		info.StatItem.Total += int(info.History.FinalScores[1])
	}
	return nil
}

func addTilesPlayed(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	var succEvent *pb.GameEvent
	if info.EventIndex+1 < len(events) {
		succEvent = events[info.EventIndex+1]
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE &&
		(succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		for _, char := range event.PlayedTiles {
			if char != alphabet.ASCIIPlayedThrough {
				info.StatItem.Total++
				if unicode.IsLower(char) {
					info.StatItem.Subitems[string(alphabet.BlankToken)]++
				} else {
					info.StatItem.Subitems[string(char)]++
				}
			}
		}
	}
	return nil
}

func addTilesStuckWith(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[len(events)-1]
	if event.Type == pb.GameEvent_END_RACK_PTS &&
		((info.IsPlayerOne && event.Nickname == info.History.Players[0].Nickname) ||
			(!info.IsPlayerOne && event.Nickname == info.History.Players[1].Nickname)) {
		tilesStuckWith := event.Rack
		for _, char := range tilesStuckWith {
			info.StatItem.Total++
			info.StatItem.Subitems[string(char)]++
		}
	}
	return nil
}

func addTime(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if (event.Nickname == info.History.Players[0].Nickname && info.IsPlayerOne) ||
			(event.Nickname == info.History.Players[1].Nickname && !info.IsPlayerOne) {
			info.StatItem.Total += int((info.Req.InitialTimeSeconds * 1000) - event.MillisRemaining)
		}
	}
	return nil
}

func addTripleTriples(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	var succEvent *pb.GameEvent
	if info.EventIndex+1 < len(events) {
		succEvent = events[info.EventIndex+1]
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE &&
		(succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) &&
		countBonusSquares(info, event, '=') >= 2 {
		incrementStatItem(info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addTurns(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
		event.Type == pb.GameEvent_PASS ||
		event.Type == pb.GameEvent_EXCHANGE ||
		event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Do we want this last one?
		incrementStatItem(info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addTurnsWithBlank(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
		event.Type == pb.GameEvent_PASS ||
		event.Type == pb.GameEvent_EXCHANGE ||
		event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Do we want this last one?
		for _, char := range event.Rack {
			if char == alphabet.BlankToken {
				incrementStatItem(info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
				break
			}
		}
	}
	return nil
}

func addVerticalOpenings(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[0]
	if info.IsPlayerOne && events[0].Direction == pb.GameEvent_VERTICAL {
		incrementStatItem(info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addWins(info *entity.IncrementInfo) error {
	if (info.History.Winner == 0 && info.IsPlayerOne) || (info.History.Winner == 1 && !info.IsPlayerOne) {
		incrementStatItem(info.StatItem, info.Lss, nil, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func setHighGame(info *entity.IncrementInfo) error {
	playerScore := int(info.History.FinalScores[0])
	if !info.IsPlayerOne {
		playerScore = int(info.History.FinalScores[1])
	}
	if playerScore > info.StatItem.Total {
		info.StatItem.Total = playerScore
		info.StatItem.List = []*entity.ListItem{&entity.ListItem{GameId: info.GameId,
			PlayerId: info.PlayerId,
			Time:     info.Evt.Time,
			Item:     &entity.ListGame{Score: playerScore}}}
	}
	return nil
}

func setLowGame(info *entity.IncrementInfo) error {
	playerScore := int(info.History.FinalScores[0])
	if !info.IsPlayerOne {
		playerScore = int(info.History.FinalScores[1])
	}
	if playerScore < info.StatItem.Total {
		info.StatItem.Total = playerScore
		info.StatItem.List = []*entity.ListItem{&entity.ListItem{GameId: info.GameId,
			PlayerId: info.PlayerId,
			Time:     info.Evt.Time,
			Item:     &entity.ListGame{Score: playerScore}}}
	}
	return nil
}

func setHighTurn(info *entity.IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	score := int(event.Score)
	if score > info.StatItem.Total {
		info.StatItem.Total = score
		info.StatItem.List = []*entity.ListItem{&entity.ListItem{GameId: info.GameId,
			PlayerId: info.PlayerId,
			Time:     info.Evt.Time,
			Item:     &entity.ListGame{Score: score}}}
	}
	return nil
}

func addBonusSquares(info *entity.IncrementInfo, bonusSquare byte) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	var succEvent *pb.GameEvent
	if info.EventIndex+1 < len(events) {
		succEvent = events[info.EventIndex+1]
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE &&
		(succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		info.StatItem.Total += countBonusSquares(info, event, bonusSquare)
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

func countBonusSquares(info *entity.IncrementInfo,
	event *pb.GameEvent,
	bonusSquare byte) int {
	occupiedIndexes := getOccupiedIndexes(event)
	boardLayout, _ := game.HistoryToVariant(info.History)
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

func isUnchallengedPhonyEvent(event *pb.GameEvent,
	history *pb.GameHistory,
	cfg *macondoconfig.Config) (bool, error) {
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

func instantiatePlayerData() map[string]*entity.StatItem {

	gamesStat := &entity.StatItem{Name: entity.GAMES_STAT,
		Total:         0,
		DataType:      entity.SingleType,
		IncrementType: entity.GameType,
		Subitems:      makeGameSubitems(),
		AddFunction:   addGames}

	turnsStat := &entity.StatItem{Name: entity.TURNS_STAT,
		Total:         0,
		DataType:      entity.SingleType,
		IncrementType: entity.EventType,
		AddFunction:   addTurns}

	scoreStat := &entity.StatItem{Name: entity.SCORE_STAT,
		Total:         0,
		DataType:      entity.SingleType,
		IncrementType: entity.GameType,
		AddFunction:   addScore}

	firstsStat := &entity.StatItem{Name: entity.FIRSTS_STAT,
		Total:         0,
		DataType:      entity.SingleType,
		IncrementType: entity.GameType,
		AddFunction:   addFirsts}

	verticalOpeningsStat := &entity.StatItem{Name: entity.VERTICAL_OPENINGS_STAT,
		Total:         0,
		DataType:      entity.SingleType,
		IncrementType: entity.EventType,
		AddFunction:   addVerticalOpenings}

	exchangesStat := &entity.StatItem{Name: entity.EXCHANGES_STAT,
		Total:         0,
		DataType:      entity.SingleType,
		IncrementType: entity.EventType,
		AddFunction:   addExchanges}

	challengedPhoniesStat :=
		&entity.StatItem{Name: entity.CHALLENGED_PHONIES_STAT,
			Total:         0,
			DataType:      entity.ListType,
			IncrementType: entity.EventType,
			AddFunction:   addChallengedPhonies}

	unchallengedPhoniesStat :=
		&entity.StatItem{Name: entity.UNCHALLENGED_PHONIES_STAT,
			Total:         0,
			DataType:      entity.ListType,
			IncrementType: entity.EventType,
			AddFunction:   addUnchallengedPhonies}

	validPlaysThatWereChallengedStat :=
		&entity.StatItem{Name: entity.VALID_PLAYS_THAT_WERE_CHALLENGED_STAT,
			Total:         0,
			DataType:      entity.ListType,
			IncrementType: entity.EventType,
			AddFunction:   addValidPlaysThatWereChallenged}

	challengesWonStat := &entity.StatItem{Name: entity.CHALLENGES_WON_STAT,
		Total:         0,
		DataType:      entity.ListType,
		IncrementType: entity.EventType,
		AddFunction:   addChallengesWon}

	challengesLostStat := &entity.StatItem{Name: entity.CHALLENGES_LOST_STAT,
		Total:         0,
		DataType:      entity.ListType,
		IncrementType: entity.EventType,
		AddFunction:   addChallengesLost}

	winsStat := &entity.StatItem{Name: entity.WINS_STAT,
		Total:         0,
		DataType:      entity.SingleType,
		IncrementType: entity.GameType,
		AddFunction:   addWins}

	lossesStat := &entity.StatItem{Name: entity.LOSSES_STAT,
		Total:         0,
		DataType:      entity.SingleType,
		IncrementType: entity.GameType,
		AddFunction:   addLosses}

	drawsStat := &entity.StatItem{Name: entity.DRAWS_STAT,
		Total:         0,
		DataType:      entity.SingleType,
		IncrementType: entity.GameType,
		AddFunction:   addDraws}

	bingosStat := &entity.StatItem{Name: entity.BINGOS_STAT,
		Total:         0,
		DataType:      entity.ListType,
		IncrementType: entity.EventType,
		AddFunction:   addBingos}

	noBingosStat := &entity.StatItem{Name: entity.NO_BINGOS_STAT,
		Total:         0,
		DataType:      entity.ListType,
		IncrementType: entity.GameType,
		AddFunction:   addNoBingos}

	tripleTriplesStat := &entity.StatItem{Name: entity.TRIPLE_TRIPLES_STAT,
		Total:         0,
		DataType:      entity.ListType,
		IncrementType: entity.EventType,
		AddFunction:   addTripleTriples}

	highGameStat := &entity.StatItem{Name: entity.HIGH_GAME_STAT,
		Total:         0,
		DataType:      entity.MaximumType,
		IncrementType: entity.GameType,
		List:          []*entity.ListItem{},
		AddFunction:   setHighGame}

	lowGameStat := &entity.StatItem{Name: entity.LOW_GAME_STAT,
		Total:         entity.MaxNotableInt,
		DataType:      entity.MinimumType,
		IncrementType: entity.GameType,
		List:          []*entity.ListItem{},
		AddFunction:   setLowGame}

	highTurnStat := &entity.StatItem{Name: entity.HIGH_TURN_STAT,
		Total:         0,
		DataType:      entity.MaximumType,
		IncrementType: entity.EventType,
		List:          []*entity.ListItem{},
		AddFunction:   setHighTurn}

	tilesPlayedStat := &entity.StatItem{Name: entity.TILES_PLAYED_STAT,
		Total:         0,
		DataType:      entity.SingleType,
		IncrementType: entity.EventType,
		Subitems:      makeAlphabetSubitems(),
		AddFunction:   addTilesPlayed}

	turnsWithBlankStat := &entity.StatItem{Name: entity.TURNS_WITH_BLANK_STAT,
		Total:         0,
		DataType:      entity.SingleType,
		IncrementType: entity.EventType,
		AddFunction:   addTurnsWithBlank}

	commentsStat := &entity.StatItem{Name: entity.COMMENTS_STAT,
		Total:         0,
		DataType:      entity.ListType,
		IncrementType: entity.EventType,
		AddFunction:   addComments}

	timeStat := &entity.StatItem{Name: entity.TIME_STAT,
		Total:         0,
		DataType:      entity.SingleType,
		IncrementType: entity.GameType,
		AddFunction:   addTime}

	mistakesStat := &entity.StatItem{Name: entity.MISTAKES_STAT,
		Total:         0,
		DataType:      entity.ListType,
		IncrementType: entity.EventType,
		Subitems:      makeMistakeSubitems(),
		AddFunction:   addMistakes}

	ratingsStat := &entity.StatItem{Name: entity.RATINGS_STAT,
		Total:         0,
		DataType:      entity.ListType,
		IncrementType: entity.GameType,
		AddFunction:   addRatings}

	return map[string]*entity.StatItem{entity.BINGOS_STAT: bingosStat,
		entity.CHALLENGED_PHONIES_STAT:               challengedPhoniesStat,
		entity.CHALLENGES_LOST_STAT:                  challengesLostStat,
		entity.CHALLENGES_WON_STAT:                   challengesWonStat,
		entity.COMMENTS_STAT:                         commentsStat,
		entity.DRAWS_STAT:                            drawsStat,
		entity.EXCHANGES_STAT:                        exchangesStat,
		entity.FIRSTS_STAT:                           firstsStat,
		entity.GAMES_STAT:                            gamesStat,
		entity.HIGH_GAME_STAT:                        highGameStat,
		entity.HIGH_TURN_STAT:                        highTurnStat,
		entity.LOSSES_STAT:                           lossesStat,
		entity.LOW_GAME_STAT:                         lowGameStat,
		entity.NO_BINGOS_STAT:                        noBingosStat,
		entity.MISTAKES_STAT:                         mistakesStat,
		entity.VALID_PLAYS_THAT_WERE_CHALLENGED_STAT: validPlaysThatWereChallengedStat,
		entity.RATINGS_STAT:                          ratingsStat,
		entity.SCORE_STAT:                            scoreStat,
		entity.TILES_PLAYED_STAT:                     tilesPlayedStat,
		entity.TIME_STAT:                             timeStat,
		entity.TRIPLE_TRIPLES_STAT:                   tripleTriplesStat,
		entity.TURNS_STAT:                            turnsStat,
		entity.TURNS_WITH_BLANK_STAT:                 turnsWithBlankStat,
		entity.UNCHALLENGED_PHONIES_STAT:             unchallengedPhoniesStat,
		entity.VERTICAL_OPENINGS_STAT:                verticalOpeningsStat,
		entity.WINS_STAT:                             winsStat,
	}
	/*
		Missing stats:
			Full rack per turn
			Bonus square coverage
			Triple Triples
			Comments word length
			Dynamic Mistakes
			Confidence Intervals
			Bingo Probabilities? (kinda)
	*/
}

func instantiateNotableData() map[string]*entity.StatItem {

	return map[string]*entity.StatItem{entity.NO_BLANKS_PLAYED_STAT: &entity.StatItem{Name: entity.NO_BLANKS_PLAYED_STAT,
		Minimum:       0,
		Maximum:       0,
		Total:         0,
		DataType:      entity.ListType,
		IncrementType: entity.EventType,
		AddFunction:   addBlanksPlayed},

		// All is 24
		entity.MANY_DOUBLE_LETTERS_COVERED_STAT: &entity.StatItem{Name: entity.MANY_DOUBLE_LETTERS_COVERED_STAT,
			Minimum:       20,
			Maximum:       entity.MaxNotableInt,
			Total:         0,
			DataType:      entity.ListType,
			IncrementType: entity.EventType,
			AddFunction:   addDoubleLetter},

		// All is 17
		entity.MANY_DOUBLE_WORDS_COVERED_STAT: &entity.StatItem{Name: entity.MANY_DOUBLE_WORDS_COVERED_STAT,
			Minimum:       15,
			Maximum:       entity.MaxNotableInt,
			Total:         0,
			DataType:      entity.ListType,
			IncrementType: entity.EventType,
			AddFunction:   addDoubleWord},

		entity.ALL_TRIPLE_LETTERS_COVERED_STAT: &entity.StatItem{Name: entity.ALL_TRIPLE_LETTERS_COVERED_STAT,
			Minimum:       12,
			Maximum:       entity.MaxNotableInt,
			Total:         0,
			DataType:      entity.ListType,
			IncrementType: entity.EventType,
			AddFunction:   addTripleLetter},

		entity.ALL_TRIPLE_WORDS_COVERED_STAT: &entity.StatItem{Name: entity.ALL_TRIPLE_WORDS_COVERED_STAT,
			Minimum:       8,
			Maximum:       entity.MaxNotableInt,
			Total:         0,
			DataType:      entity.ListType,
			IncrementType: entity.EventType,
			AddFunction:   addTripleWord},

		entity.COMBINED_HIGH_SCORING_STAT: &entity.StatItem{Name: entity.COMBINED_HIGH_SCORING_STAT,
			Minimum:       1100,
			Maximum:       entity.MaxNotableInt,
			Total:         0,
			DataType:      entity.ListType,
			IncrementType: entity.GameType,
			AddFunction:   addCombinedScoring},

		entity.COMBINED_LOW_SCORING_STAT: &entity.StatItem{Name: entity.COMBINED_LOW_SCORING_STAT,
			Minimum:       0,
			Maximum:       200,
			Total:         0,
			DataType:      entity.ListType,
			IncrementType: entity.GameType,
			AddFunction:   addCombinedScoring},

		entity.ONE_PLAYER_PLAYS_EVERY_POWER_TILE_STAT: &entity.StatItem{Name: entity.ONE_PLAYER_PLAYS_EVERY_POWER_TILE_STAT,
			Minimum:       10,
			Maximum:       10,
			Total:         0,
			DataType:      entity.ListType,
			IncrementType: entity.EventType,
			AddFunction:   addEveryPowerTile},

		entity.ONE_PLAYER_PLAYS_EVERY_E_STAT: &entity.StatItem{Name: entity.ONE_PLAYER_PLAYS_EVERY_E_STAT,
			Minimum:       12,
			Maximum:       12,
			Total:         0,
			DataType:      entity.ListType,
			IncrementType: entity.EventType,
			AddFunction:   addEveryE},

		entity.MANY_CHALLENGES_STAT: &entity.StatItem{Name: entity.MANY_CHALLENGES_STAT,
			Minimum:       5,
			Maximum:       entity.MaxNotableInt,
			Total:         0,
			DataType:      entity.ListType,
			IncrementType: entity.EventType,
			AddFunction:   addManyChallenges},

		entity.FOUR_OR_MORE_CONSECUTIVE_BINGOS_STAT: &entity.StatItem{Name: entity.FOUR_OR_MORE_CONSECUTIVE_BINGOS_STAT,
			Minimum:       4,
			Maximum:       entity.MaxNotableInt,
			Total:         0,
			DataType:      entity.ListType,
			IncrementType: entity.EventType,
			Subitems:      map[string]int{"player_one_streak": 0, "player_two_streak": 0},
			AddFunction:   addConsecutiveBingos},
	}
}

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

	mistakeSubitems[entity.KnowledgeMistakeType] = 0
	mistakeSubitems[entity.FindingMistakeType] = 0
	mistakeSubitems[entity.VisionMistakeType] = 0
	mistakeSubitems[entity.TacticsMistakeType] = 0
	mistakeSubitems[entity.StrategyMistakeType] = 0
	mistakeSubitems[entity.TimeMistakeType] = 0
	mistakeSubitems[entity.EndgameMistakeType] = 0
	mistakeSubitems[entity.LargeMistakeMagnitude] = 0
	mistakeSubitems[entity.MediumMistakeMagnitude] = 0
	mistakeSubitems[entity.SmallMistakeMagnitude] = 0
	mistakeSubitems[entity.UnspecifiedMistakeMagnitude] = 0

	return mistakeSubitems
}
