package stats

import (
	"context"
	"errors"
	"strings"
	"unicode"

	"github.com/rs/zerolog/log"

	"github.com/domino14/macondo/board"
	"github.com/domino14/macondo/game"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
	wglconfig "github.com/domino14/word-golib/config"
	"github.com/domino14/word-golib/kwg"
	"github.com/domino14/word-golib/tilemapping"

	"github.com/woogles-io/liwords/pkg/entity"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type ListStatStore interface {
	AddListItem(ctx context.Context, gameId string, playerId string, statType int, time int64, item entity.ListDatum) error
	GetListItems(ctx context.Context, statType int, gameIds []string, playerId string) ([]*entity.ListItem, error)
}

type IncrementInfo struct {
	Ctx                 context.Context
	Cfg                 *wglconfig.Config
	Req                 *ipc.GameRequest
	Evt                 *ipc.GameEndedEvent
	Lss                 ListStatStore
	StatName            string
	StatItem            *entity.StatItem
	OtherPlayerStatItem *entity.StatItem
	History             *pb.GameHistory
	EventIndex          int
	GameId              string
	PlayerId            string
	OtherPlayerId       string
	IsPlayerOne         bool
}

var StatNameToAddFunction = map[string]func(*IncrementInfo) error{
	entity.ALL_TRIPLE_LETTERS_COVERED_STAT:        addTripleLetter,
	entity.ALL_TRIPLE_WORDS_COVERED_STAT:          addTripleWord,
	entity.BINGOS_STAT:                            addBingos,
	entity.CHALLENGED_PHONIES_STAT:                addChallengedPhonies,
	entity.CHALLENGES_LOST_STAT:                   addChallengesLost,
	entity.CHALLENGES_WON_STAT:                    addChallengesWon,
	entity.COMMENTS_STAT:                          addComments,
	entity.DRAWS_STAT:                             addDraws,
	entity.EXCHANGES_STAT:                         addExchanges,
	entity.FIRSTS_STAT:                            addFirsts,
	entity.GAMES_STAT:                             addGames,
	entity.HIGH_GAME_STAT:                         setHighGame,
	entity.HIGH_TURN_STAT:                         setHighTurn,
	entity.LOSSES_STAT:                            addLosses,
	entity.LOW_GAME_STAT:                          setLowGame,
	entity.NO_BINGOS_STAT:                         addNoBingos,
	entity.MANY_DOUBLE_LETTERS_COVERED_STAT:       addDoubleLetter,
	entity.MANY_DOUBLE_WORDS_COVERED_STAT:         addDoubleWord,
	entity.MISTAKES_STAT:                          addMistakes,
	entity.SCORE_STAT:                             addScore,
	entity.RATINGS_STAT:                           addRatings,
	entity.TILES_PLAYED_STAT:                      addTilesPlayed,
	entity.TIME_STAT:                              addTime,
	entity.TRIPLE_TRIPLES_STAT:                    addTripleTriples,
	entity.TURNS_STAT:                             addTurns,
	entity.TURNS_WITH_BLANK_STAT:                  addTurnsWithBlank,
	entity.UNCHALLENGED_PHONIES_STAT:              addUnchallengedPhonies,
	entity.VALID_PLAYS_THAT_WERE_CHALLENGED_STAT:  addValidPlaysThatWereChallenged,
	entity.VERTICAL_OPENINGS_STAT:                 addVerticalOpenings,
	entity.WINS_STAT:                              addWins,
	entity.NO_BLANKS_PLAYED_STAT:                  addBlanksPlayed,
	entity.HIGH_SCORING_STAT:                      addHighScoring,
	entity.COMBINED_HIGH_SCORING_STAT:             addCombinedScoring,
	entity.COMBINED_LOW_SCORING_STAT:              addCombinedScoring,
	entity.ONE_PLAYER_PLAYS_EVERY_POWER_TILE_STAT: addEveryPowerTile,
	entity.ONE_PLAYER_PLAYS_EVERY_E_STAT:          addEveryE,
	entity.MANY_CHALLENGES_STAT:                   addManyChallenges,
	entity.FOUR_OR_MORE_CONSECUTIVE_BINGOS_STAT:   addConsecutiveBingos,
	entity.HIGH_LOSS_STAT:                         setHighLoss,
	entity.LOW_WIN_STAT:                           setLowWin,
	entity.UPSET_WIN_STAT:                         setUpsetWin,
}

var StatNameToDataType = map[string]entity.StatItemType{
	entity.ALL_TRIPLE_LETTERS_COVERED_STAT:        entity.ListType,
	entity.ALL_TRIPLE_WORDS_COVERED_STAT:          entity.ListType,
	entity.BINGOS_STAT:                            entity.ListType,
	entity.CHALLENGED_PHONIES_STAT:                entity.ListType,
	entity.CHALLENGES_LOST_STAT:                   entity.ListType,
	entity.CHALLENGES_WON_STAT:                    entity.ListType,
	entity.COMMENTS_STAT:                          entity.ListType,
	entity.DRAWS_STAT:                             entity.SingleType,
	entity.EXCHANGES_STAT:                         entity.SingleType,
	entity.FIRSTS_STAT:                            entity.SingleType,
	entity.GAMES_STAT:                             entity.SingleType,
	entity.HIGH_GAME_STAT:                         entity.MaximumType,
	entity.HIGH_TURN_STAT:                         entity.MaximumType,
	entity.LOSSES_STAT:                            entity.SingleType,
	entity.LOW_GAME_STAT:                          entity.MinimumType,
	entity.NO_BINGOS_STAT:                         entity.ListType,
	entity.MANY_DOUBLE_LETTERS_COVERED_STAT:       entity.ListType,
	entity.MANY_DOUBLE_WORDS_COVERED_STAT:         entity.ListType,
	entity.MISTAKES_STAT:                          entity.ListType,
	entity.SCORE_STAT:                             entity.SingleType,
	entity.RATINGS_STAT:                           entity.ListType,
	entity.TILES_PLAYED_STAT:                      entity.SingleType,
	entity.TIME_STAT:                              entity.SingleType,
	entity.TRIPLE_TRIPLES_STAT:                    entity.ListType,
	entity.TURNS_STAT:                             entity.SingleType,
	entity.TURNS_WITH_BLANK_STAT:                  entity.SingleType,
	entity.UNCHALLENGED_PHONIES_STAT:              entity.ListType,
	entity.VALID_PLAYS_THAT_WERE_CHALLENGED_STAT:  entity.ListType,
	entity.VERTICAL_OPENINGS_STAT:                 entity.SingleType,
	entity.WINS_STAT:                              entity.SingleType,
	entity.NO_BLANKS_PLAYED_STAT:                  entity.ListType,
	entity.HIGH_SCORING_STAT:                      entity.ListType,
	entity.COMBINED_HIGH_SCORING_STAT:             entity.ListType,
	entity.COMBINED_LOW_SCORING_STAT:              entity.ListType,
	entity.ONE_PLAYER_PLAYS_EVERY_POWER_TILE_STAT: entity.ListType,
	entity.ONE_PLAYER_PLAYS_EVERY_E_STAT:          entity.ListType,
	entity.MANY_CHALLENGES_STAT:                   entity.ListType,
	entity.FOUR_OR_MORE_CONSECUTIVE_BINGOS_STAT:   entity.ListType,
	entity.HIGH_LOSS_STAT:                         entity.MaximumType,
	entity.LOW_WIN_STAT:                           entity.MinimumType,
	entity.UPSET_WIN_STAT:                         entity.MaximumType,
}

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

func AddGame(ctx context.Context, stats *entity.Stats, lss ListStatStore, history *pb.GameHistory, req *ipc.GameRequest,
	cfg *wglconfig.Config, gameEndedEvent *ipc.GameEndedEvent, gameId string) error {
	// Josh, plz fix these asinine calls to incrementStatItem
	events := history.GetEvents()

	info := &IncrementInfo{Ctx: ctx,
		Cfg:     cfg,
		Req:     req,
		Evt:     gameEndedEvent,
		Lss:     lss,
		History: history,
		GameId:  gameId}

	for i := 0; i < len(events); i++ {
		event := events[i]
		if event.PlayerIndex == 0 {
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

	confirmNotableItems(info.Ctx, lss, gameId, gameEndedEvent.Time, stats.NotableData)
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

// Finalize will be needed in the future when we want to retrieve stats.
func Finalize(ctx context.Context, stats *entity.Stats, lss ListStatStore, gameIDs []string,
	playerOneID string, playerTwoID string) error {
	err := finalize(ctx, stats.PlayerOneData, lss, gameIDs, playerOneID)
	if err != nil {
		return err
	}

	err = finalize(ctx, stats.PlayerTwoData, lss, gameIDs, playerTwoID)

	if err != nil {
		return err
	}

	err = finalize(ctx, stats.NotableData, lss, gameIDs, "")
	if err != nil {
		return err
	}

	return nil
}

func finalize(ctx context.Context, statItems map[string]*entity.StatItem, lss ListStatStore,
	gameIDs []string, playerID string) error {
	for statName, statItem := range statItems {
		if StatNameToDataType[statName] == entity.ListType {
			newList, err := lss.GetListItems(ctx, entity.StatName_value[statName], gameIDs, playerID)
			if err != nil {
				return err
			}
			if len(newList) > 0 {
				statItem.List = newList
			}
		}
	}
	return nil
}

func incrementStatItems(info *IncrementInfo,
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

			err := StatNameToAddFunction[info.StatName](info)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func incrementStatItem(ctx context.Context,
	statItem *entity.StatItem,
	lss ListStatStore,
	event *pb.GameEvent,
	gameId string,
	playerId string,
	time int64) {

	if StatNameToDataType[statItem.Name] == entity.SingleType || event == nil {
		statItem.Total++
	} else if StatNameToDataType[statItem.Name] == entity.ListType {
		statItem.Total++
		word := event.PlayedTiles
		if len(event.WordsFormed) > 0 {
			// We can have played tiles with words formed being empty
			// for phony_tiles_returned events, for example.
			word = event.WordsFormed[0]
		}
		lss.AddListItem(ctx,
			gameId,
			playerId,
			entity.StatName_value[statItem.Name],
			time, entity.ListDatum{Word: word, Probability: 1, Score: int(event.Score)})
	}
}

func combineStatItemMaps(statItems map[string]*entity.StatItem,
	otherStatItems map[string]*entity.StatItem) error {
	for key, item := range statItems {
		itemOther := otherStatItems[key]
		if itemOther != nil {
			combineItems(item, itemOther)
		}
	}
	return nil
}

func confirmNotableItems(ctx context.Context, lss ListStatStore, gameId string, time int64, statItems map[string]*entity.StatItem) {
	for key, item := range statItems {
		// For one player plays every _ stats
		// Player one adds to the total, player two subtracts
		// So we need the absolute value to account for both possibilities
		if item.Total < 0 {
			item.Total = item.Total * (-1)
		}
		if item.Total >= item.Minimum && item.Total <= item.Maximum {
			log.Debug().Msgf("Notable confirmed: %s", key)
			lss.AddListItem(ctx, gameId, "", entity.StatName_value[key], time, entity.ListDatum{})
		}
		item.Total = 0
	}
}

func combineItems(statItem *entity.StatItem, otherStatItem *entity.StatItem) {
	if StatNameToDataType[statItem.Name] == entity.SingleType {
		statItem.Total += otherStatItem.Total
	} else if StatNameToDataType[statItem.Name] == entity.ListType {
		statItem.Total += otherStatItem.Total
	} else if (StatNameToDataType[statItem.Name] == entity.MaximumType &&
		otherStatItem.Total > statItem.Total) ||
		(StatNameToDataType[statItem.Name] == entity.MinimumType &&
			otherStatItem.Total < statItem.Total) {
		statItem.Total = otherStatItem.Total
		statItem.List = otherStatItem.List
	} else if otherStatItem.Total == statItem.Total &&
		(StatNameToDataType[statItem.Name] == entity.MaximumType || StatNameToDataType[statItem.Name] == entity.MinimumType) {
		statItem.List = append(statItem.List, otherStatItem.List...)
	}

	if statItem.Subitems != nil {
		for key := range statItem.Subitems {
			statItem.Subitems[key] += otherStatItem.Subitems[key]
		}
	}
}

func addBingos(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]

	var succEvent *pb.GameEvent
	if info.EventIndex+1 < len(events) {
		succEvent = events[info.EventIndex+1]
	}
	if event.IsBingo && (succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		log.Debug().Interface("evt", event).Interface("statitem", info.StatItem).Interface("otherstat", info.OtherPlayerStatItem).Msg("Add bing")
		incrementStatItem(info.Ctx, info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addBlanksPlayed(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	// XXX: this needs to be fixed or removed for multi-char tiles
	tiles := event.PlayedTiles

	for _, char := range tiles {
		if unicode.IsLower(char) {
			info.StatItem.Total++
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
	info.StatItem.Total = int(info.History.FinalScores[0] + info.History.FinalScores[1])
	return nil
}

func addHighScoring(info *IncrementInfo) error {
	playerScore := int(info.History.FinalScores[0])
	if !info.IsPlayerOne {
		playerScore = int(info.History.FinalScores[1])
	}
	if playerScore > info.StatItem.Total {
		info.StatItem.Total = playerScore
	}
	return nil
}

func addExchanges(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Type == pb.GameEvent_EXCHANGE {
		incrementStatItem(info.Ctx, info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addChallengedPhonies(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Type == pb.GameEvent_PHONY_TILES_RETURNED {
		incrementStatItem(info.Ctx, info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addUnchallengedPhonies(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	isUnchallengedPhony, err := isUnchallengedPhonyEvent(event, info.History, info.Cfg)
	if err != nil {
		return err
	}
	if (info.EventIndex+1 >= len(events) ||
		events[info.EventIndex+1].Type != pb.GameEvent_PHONY_TILES_RETURNED) &&
		isUnchallengedPhony {
		incrementStatItem(info.Ctx, info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addValidPlaysThatWereChallenged(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	var succEvent *pb.GameEvent
	if info.EventIndex+1 < len(events) {
		succEvent = events[info.EventIndex+1]
	}
	if succEvent != nil &&
		(succEvent.Type == pb.GameEvent_CHALLENGE_BONUS ||
			succEvent.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS) {
		incrementStatItem(info.Ctx, info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
		// Need to increment opp's challengesLost stat
	}
	return nil
}

func addChallengesWon(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Type == pb.GameEvent_PHONY_TILES_RETURNED { // Opp's phony tiles returned
		incrementStatItem(info.Ctx, info.OtherPlayerStatItem,
			info.Lss,
			events[info.EventIndex-1],
			info.GameId,
			info.OtherPlayerId,
			info.Evt.Time)
	}
	return nil
}

func addChallengesLost(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Type == pb.GameEvent_CHALLENGE_BONUS { // Opp's bonus
		incrementStatItem(info.Ctx, info.OtherPlayerStatItem,
			info.Lss,
			events[info.EventIndex-1],
			info.GameId,
			info.OtherPlayerId,
			info.Evt.Time)
	} else if event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Player's turn loss
		incrementStatItem(info.Ctx, info.StatItem,
			info.Lss, events[info.EventIndex-1],
			info.GameId,
			info.PlayerId,
			info.Evt.Time)
	}
	return nil
}

func addComments(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Note != "" {
		incrementStatItem(info.Ctx, info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addConsecutiveBingos(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	player := "player_one_streak"
	if !info.IsPlayerOne && event.PlayerIndex == 1 {
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

func addDraws(info *IncrementInfo) error {
	if info.History.Winner == -1 {
		incrementStatItem(info.Ctx, info.StatItem, info.Lss, nil, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addEveryE(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	var succEvent *pb.GameEvent
	if info.EventIndex+1 < len(events) {
		succEvent = events[info.EventIndex+1]
	}
	multiplier := 1
	if !info.IsPlayerOne && event.PlayerIndex == 1 {
		multiplier = -1
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE &&
		(succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		// XXX: this needs to be changed/removed for multi-char tiles
		for _, char := range event.PlayedTiles {
			if char == 'E' {
				info.StatItem.Total += 1 * multiplier
			}
		}
	}
	return nil
}

func addEveryPowerTile(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	var succEvent *pb.GameEvent
	if info.EventIndex+1 < len(events) {
		succEvent = events[info.EventIndex+1]
	}
	multiplier := 1
	if !info.IsPlayerOne && event.PlayerIndex == 1 {
		multiplier = -1
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE &&
		(succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		// XXX: this needs to be changed or removed for multi-char tiles
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

func addFirsts(info *IncrementInfo) error {
	if info.IsPlayerOne {
		incrementStatItem(info.Ctx, info.StatItem, info.Lss, nil, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addGames(info *IncrementInfo) error {
	incrementStatItem(info.Ctx, info.StatItem, info.Lss, nil, info.GameId, info.PlayerId, info.Evt.Time)
	info.StatItem.Subitems[ipc.RatingMode_name[int32(info.Req.RatingMode)]]++
	info.StatItem.Subitems[pb.ChallengeRule_name[int32(info.Req.ChallengeRule)]]++
	return nil
}

func addLosses(info *IncrementInfo) error {
	if (info.History.Winner == 1 && info.IsPlayerOne) || (info.History.Winner == 0 && !info.IsPlayerOne) {
		incrementStatItem(info.Ctx, info.StatItem, info.Lss, nil, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addManyChallenges(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Type == pb.GameEvent_PHONY_TILES_RETURNED ||
		event.Type == pb.GameEvent_CHALLENGE_BONUS ||
		event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Do we want this last one?
		info.StatItem.Total++
	}
	return nil
}

func addMistakes(info *IncrementInfo) error {
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
					info.Lss.AddListItem(info.Ctx, info.GameId,
						info.PlayerId,
						entity.StatName_value[entity.MISTAKES_STAT],
						info.Evt.Time,
						entity.ListDatum{MistakeType: entity.MistakeTypeMapping[mType],
							MistakeSize: entity.MistakeMagnitudeMapping[unaliasedMistakeM]})
				}
			}
			unspecifiedOccurences := strings.Count(note, "#"+mType)
			note = strings.ReplaceAll(note, "#"+mType, "")
			info.StatItem.Total += unspecifiedOccurences
			info.StatItem.Subitems[mType] += unspecifiedOccurences
			info.StatItem.Subitems[entity.UnspecifiedMistakeMagnitude] += unspecifiedOccurences
			for i := 0; i < unspecifiedOccurences-totalOccurences; i++ {
				info.Lss.AddListItem(info.Ctx, info.GameId,
					info.PlayerId,
					entity.StatName_value[entity.MISTAKES_STAT],
					info.Evt.Time,
					entity.ListDatum{MistakeType: entity.MistakeTypeMapping[mType],
						MistakeSize: entity.MistakeMagnitudeMapping[entity.UnspecifiedMistakeMagnitude]})
			}
		}
	}
	return nil
}

func addNoBingos(info *IncrementInfo) error {
	events := info.History.GetEvents()
	atLeastOneBingo := false
	// SAD! (have to loop through events again, should not do this, is the big not good)
	for i := 0; i < len(events); i++ {
		event := events[i]
		if (event.PlayerIndex == 0 && info.IsPlayerOne) ||
			(event.PlayerIndex == 1 && !info.IsPlayerOne) {
			atLeastOneBingo = true
			break
		}
	}
	if !atLeastOneBingo {
		incrementStatItem(info.Ctx, info.StatItem, info.Lss, nil, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addRatings(info *IncrementInfo) error {
	var rating int32
	if info.IsPlayerOne {
		rating = info.Evt.NewRatings[info.History.Players[0].Nickname]
	} else {
		rating = info.Evt.NewRatings[info.History.Players[1].Nickname]
	}
	info.StatItem.Total++
	timeControl, variant, err := entity.VariantFromGameReq(info.Req)
	if err != nil {
		return err
	}
	variantKey := entity.ToVariantKey(info.Req.Lexicon, variant, timeControl)
	info.Lss.AddListItem(info.Ctx, info.GameId,
		info.PlayerId,
		entity.StatName_value[entity.RATINGS_STAT],
		info.Evt.Time,
		entity.ListDatum{Rating: int(rating), Variant: string(variantKey)})
	return nil
}

func addScore(info *IncrementInfo) error {
	if info.IsPlayerOne {
		info.StatItem.Total += int(info.History.FinalScores[0])
	} else {
		info.StatItem.Total += int(info.History.FinalScores[1])
	}
	return nil
}

func addTilesPlayed(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	var succEvent *pb.GameEvent
	if info.EventIndex+1 < len(events) {
		succEvent = events[info.EventIndex+1]
	}
	// XXX: this needs to be fixed for multi-char tiles and non-english lexica
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE &&
		(succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {
		for _, char := range event.PlayedTiles {
			if char != tilemapping.ASCIIPlayedThrough {
				info.StatItem.Total++
				if unicode.IsLower(char) {
					info.StatItem.Subitems[string(tilemapping.BlankToken)]++
				} else {
					info.StatItem.Subitems[string(char)]++
				}
			}
		}
	}
	return nil
}

func addTilesStuckWith(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[len(events)-1]
	if event.Type == pb.GameEvent_END_RACK_PTS &&
		((info.IsPlayerOne && event.PlayerIndex == 0) ||
			(!info.IsPlayerOne && event.PlayerIndex == 1)) {
		tilesStuckWith := event.Rack
		for _, char := range tilesStuckWith {
			info.StatItem.Total++
			info.StatItem.Subitems[string(char)]++
		}
	}
	return nil
}

func addTime(info *IncrementInfo) error {
	events := info.History.GetEvents()
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if (event.PlayerIndex == 0 && info.IsPlayerOne) ||
			(event.PlayerIndex == 1 && !info.IsPlayerOne) {
			info.StatItem.Total += int((info.Req.InitialTimeSeconds * 1000) - event.MillisRemaining)
		}
	}
	return nil
}

func addTripleTriples(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	var succEvent *pb.GameEvent
	if info.EventIndex+1 < len(events) {
		succEvent = events[info.EventIndex+1]
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE &&
		(succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {

		bonuses, err := countBonusSquares(info, event, '=')
		if err != nil {
			return err
		}
		if bonuses >= 2 {
			incrementStatItem(info.Ctx, info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
		}
	}
	return nil
}

func addTurns(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
		event.Type == pb.GameEvent_PASS ||
		event.Type == pb.GameEvent_EXCHANGE ||
		event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Do we want this last one?
		incrementStatItem(info.Ctx, info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addTurnsWithBlank(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
		event.Type == pb.GameEvent_PASS ||
		event.Type == pb.GameEvent_EXCHANGE ||
		event.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS { // Do we want this last one?
		for _, char := range event.Rack {
			if char == tilemapping.BlankToken {
				incrementStatItem(info.Ctx, info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
				break
			}
		}
	}
	return nil
}

func addVerticalOpenings(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[0]
	if info.IsPlayerOne && events[0].Direction == pb.GameEvent_VERTICAL {
		incrementStatItem(info.Ctx, info.StatItem, info.Lss, event, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func addWins(info *IncrementInfo) error {
	if (info.History.Winner == 0 && info.IsPlayerOne) || (info.History.Winner == 1 && !info.IsPlayerOne) {
		incrementStatItem(info.Ctx, info.StatItem, info.Lss, nil, info.GameId, info.PlayerId, info.Evt.Time)
	}
	return nil
}

func setHighGame(info *IncrementInfo) error {
	playerScore := int(info.History.FinalScores[0])
	if !info.IsPlayerOne {
		playerScore = int(info.History.FinalScores[1])
	}
	if playerScore > info.StatItem.Total {
		info.StatItem.Total = playerScore
		info.StatItem.List = []*entity.ListItem{{GameId: info.GameId,
			PlayerId: info.PlayerId,
			Time:     info.Evt.Time,
			Item:     entity.ListDatum{Score: playerScore}}}
	}
	return nil
}

func setLowGame(info *IncrementInfo) error {
	playerScore := int(info.History.FinalScores[0])
	if !info.IsPlayerOne {
		playerScore = int(info.History.FinalScores[1])
	}
	if playerScore < info.StatItem.Total {
		info.StatItem.Total = playerScore
		info.StatItem.List = []*entity.ListItem{{GameId: info.GameId,
			PlayerId: info.PlayerId,
			Time:     info.Evt.Time,
			Item:     entity.ListDatum{Score: playerScore}}}
	}
	return nil
}

func setHighTurn(info *IncrementInfo) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	score := int(event.Score)
	if score > info.StatItem.Total {
		info.StatItem.Total = score
		info.StatItem.List = []*entity.ListItem{{GameId: info.GameId,
			PlayerId: info.PlayerId,
			Time:     info.Evt.Time,
			Item:     entity.ListDatum{Score: score}}}
	}
	return nil
}

func setHighLoss(info *IncrementInfo) error {
	playerScore := int(info.History.FinalScores[0])
	isWin := info.History.Winner == 0
	if !info.IsPlayerOne {
		playerScore = int(info.History.FinalScores[1])
		isWin = info.History.Winner == 1
	}

	if !isWin && playerScore > info.StatItem.Total {
		info.StatItem.Total = playerScore
		info.StatItem.List = []*entity.ListItem{{GameId: info.GameId,
			PlayerId: info.PlayerId,
			Time:     info.Evt.Time,
			Item:     entity.ListDatum{Score: playerScore}}}
	}
	return nil
}

func setLowWin(info *IncrementInfo) error {
	playerScore := int(info.History.FinalScores[0])
	isWin := info.History.Winner == 0
	if !info.IsPlayerOne {
		playerScore = int(info.History.FinalScores[1])
		isWin = info.History.Winner == 1
	}

	if isWin && playerScore < info.StatItem.Total {
		info.StatItem.Total = playerScore
		info.StatItem.List = []*entity.ListItem{{GameId: info.GameId,
			PlayerId: info.PlayerId,
			Time:     info.Evt.Time,
			Item:     entity.ListDatum{Score: playerScore}}}
	}
	return nil
}

func setUpsetWin(info *IncrementInfo) error {
	isWin := info.History.Winner == 0
	oldRatings := map[string]int{}

	for p := range info.Evt.NewRatings {
		oldRatings[p] = int(info.Evt.NewRatings[p] - info.Evt.RatingDeltas[p])
	}

	winnerRatDiff := 0
	if info.History.Winner != -1 {
		// A winner is set; i.e. it's not a tie.
		winnerRatDiff = oldRatings[info.History.Players[info.History.Winner].Nickname] -
			oldRatings[info.History.Players[1-info.History.Winner].Nickname]
	}

	if !info.IsPlayerOne {
		isWin = info.History.Winner == 1
	}

	if isWin && winnerRatDiff < 0 && (-winnerRatDiff) > info.StatItem.Total {
		info.StatItem.Total = -winnerRatDiff
		info.StatItem.List = []*entity.ListItem{{GameId: info.GameId,
			PlayerId: info.PlayerId,
			Time:     info.Evt.Time,
			Item:     entity.ListDatum{Score: -winnerRatDiff}}}
	}
	return nil
}

func addBonusSquares(info *IncrementInfo, bonusSquare byte) error {
	events := info.History.GetEvents()
	event := events[info.EventIndex]
	var succEvent *pb.GameEvent
	if info.EventIndex+1 < len(events) {
		succEvent = events[info.EventIndex+1]
	}
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE &&
		(succEvent == nil || succEvent.Type != pb.GameEvent_PHONY_TILES_RETURNED) {

		bonuses, err := countBonusSquares(info, event, bonusSquare)
		if err != nil {
			return err
		}
		info.StatItem.Total += bonuses
	}
	return nil
}

func getOccupiedIndexes(info *IncrementInfo, event *pb.GameEvent) [][]int {
	occupied := [][]int{}
	row := int(event.Row)
	column := int(event.Column)

	ldname := info.History.LetterDistribution
	if ldname == "" {
		ldname = "english"
	}

	ld, err := tilemapping.GetDistribution(info.Cfg, ldname)
	if err != nil {
		log.Err(err).Str("dist", ldname).Msg("get-occupied-indexes-get-dist-err")
		return occupied
	}
	mls, err := tilemapping.ToMachineLetters(event.PlayedTiles, ld.TileMapping())
	if err != nil {
		log.Err(err).Msg("get-occupied-indexes-conv-err")
		return occupied
	}
	for _, ml := range mls {
		if ml != 0 {
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

func countBonusSquares(info *IncrementInfo,
	event *pb.GameEvent,
	bonusSquare byte) (int, error) {
	occupiedIndexes := getOccupiedIndexes(info, event)
	boardLayout, _, _ := game.HistoryToVariant(info.History)
	var bd []string
	switch boardLayout {
	case "", board.CrosswordGameLayout:
		bd = board.CrosswordGameBoard
	case board.SuperCrosswordGameLayout:
		bd = board.SuperCrosswordGameBoard
	default:
		return 0, errors.New("board not supported")
	}
	count := 0
	for j := 0; j < len(occupiedIndexes); j++ {
		rowAndColumn := occupiedIndexes[j]
		if bd[rowAndColumn[0]][rowAndColumn[1]] == bonusSquare {
			count++
		}
	}
	return count, nil
}

func isUnchallengedPhonyEvent(event *pb.GameEvent,
	history *pb.GameHistory,
	cfg *wglconfig.Config) (bool, error) {
	phony := false
	if event.Type == pb.GameEvent_TILE_PLACEMENT_MOVE {
		kwg, err := kwg.Get(cfg, history.Lexicon)
		if err != nil {
			return phony, err
		}
		for _, word := range event.WordsFormed {
			phony, err := isPhony(kwg, word, history.Variant)
			if err != nil {
				return false, err
			}
			if phony {
				return phony, nil
			}
		}

	}
	return false, nil
}

func isPhony(gd *kwg.KWG, word, variant string) (bool, error) {
	lex := kwg.Lexicon{KWG: *gd}
	machineWord, err := tilemapping.ToMachineWord(word, lex.GetAlphabet())
	if err != nil {
		return false, err
	}
	var valid bool
	switch string(variant) {
	case string(game.VarWordSmog):
		valid = lex.HasAnagram(machineWord)
	default:
		valid = lex.HasWord(machineWord)
	}
	return !valid, nil
}

func instantiatePlayerData() map[string]*entity.StatItem {

	gamesStat := &entity.StatItem{Name: entity.GAMES_STAT,
		Total:         0,
		IncrementType: entity.GameType,
		Subitems:      makeGameSubitems()}

	turnsStat := &entity.StatItem{Name: entity.TURNS_STAT,
		Total:         0,
		IncrementType: entity.EventType}

	scoreStat := &entity.StatItem{Name: entity.SCORE_STAT,
		Total:         0,
		IncrementType: entity.GameType}

	firstsStat := &entity.StatItem{Name: entity.FIRSTS_STAT,
		Total:         0,
		IncrementType: entity.GameType}

	verticalOpeningsStat := &entity.StatItem{Name: entity.VERTICAL_OPENINGS_STAT,
		Total:         0,
		IncrementType: entity.EventType}

	exchangesStat := &entity.StatItem{Name: entity.EXCHANGES_STAT,
		Total:         0,
		IncrementType: entity.EventType}

	challengedPhoniesStat :=
		&entity.StatItem{Name: entity.CHALLENGED_PHONIES_STAT,
			Total:         0,
			IncrementType: entity.EventType}

	unchallengedPhoniesStat :=
		&entity.StatItem{Name: entity.UNCHALLENGED_PHONIES_STAT,
			Total:         0,
			IncrementType: entity.EventType}

	validPlaysThatWereChallengedStat :=
		&entity.StatItem{Name: entity.VALID_PLAYS_THAT_WERE_CHALLENGED_STAT,
			Total:         0,
			IncrementType: entity.EventType}

	challengesWonStat := &entity.StatItem{Name: entity.CHALLENGES_WON_STAT,
		Total:         0,
		IncrementType: entity.EventType}

	challengesLostStat := &entity.StatItem{Name: entity.CHALLENGES_LOST_STAT,
		Total:         0,
		IncrementType: entity.EventType}

	winsStat := &entity.StatItem{Name: entity.WINS_STAT,
		Total:         0,
		IncrementType: entity.GameType}

	lossesStat := &entity.StatItem{Name: entity.LOSSES_STAT,
		Total:         0,
		IncrementType: entity.GameType}

	drawsStat := &entity.StatItem{Name: entity.DRAWS_STAT,
		Total:         0,
		IncrementType: entity.GameType}

	bingosStat := &entity.StatItem{Name: entity.BINGOS_STAT,
		Total:         0,
		IncrementType: entity.EventType}

	noBingosStat := &entity.StatItem{Name: entity.NO_BINGOS_STAT,
		Total:         0,
		IncrementType: entity.GameType}

	tripleTriplesStat := &entity.StatItem{Name: entity.TRIPLE_TRIPLES_STAT,
		Total:         0,
		IncrementType: entity.EventType}

	highGameStat := &entity.StatItem{Name: entity.HIGH_GAME_STAT,
		Total:         0,
		IncrementType: entity.GameType,
		List:          []*entity.ListItem{}}

	lowGameStat := &entity.StatItem{Name: entity.LOW_GAME_STAT,
		Total:         entity.MaxNotableInt,
		IncrementType: entity.GameType,
		List:          []*entity.ListItem{}}

	highTurnStat := &entity.StatItem{Name: entity.HIGH_TURN_STAT,
		Total:         0,
		IncrementType: entity.EventType,
		List:          []*entity.ListItem{}}

	tilesPlayedStat := &entity.StatItem{Name: entity.TILES_PLAYED_STAT,
		Total:         0,
		IncrementType: entity.EventType,
		Subitems:      makeAlphabetSubitems()}

	turnsWithBlankStat := &entity.StatItem{Name: entity.TURNS_WITH_BLANK_STAT,
		Total:         0,
		IncrementType: entity.EventType}

	commentsStat := &entity.StatItem{Name: entity.COMMENTS_STAT,
		Total:         0,
		IncrementType: entity.EventType}

	timeStat := &entity.StatItem{Name: entity.TIME_STAT,
		Total:         0,
		IncrementType: entity.GameType}

	mistakesStat := &entity.StatItem{Name: entity.MISTAKES_STAT,
		Total:         0,
		IncrementType: entity.EventType,
		Subitems:      makeMistakeSubitems()}

	ratingsStat := &entity.StatItem{Name: entity.RATINGS_STAT,
		Total:         0,
		IncrementType: entity.GameType}

	highLossStat := &entity.StatItem{Name: entity.HIGH_LOSS_STAT,
		Total:         0,
		IncrementType: entity.GameType,
		List:          []*entity.ListItem{}}

	lowWinStat := &entity.StatItem{Name: entity.LOW_WIN_STAT,
		Total:         entity.MaxNotableInt,
		IncrementType: entity.GameType,
		List:          []*entity.ListItem{}}

	upsetWinStat := &entity.StatItem{Name: entity.UPSET_WIN_STAT,
		Total:         0,
		IncrementType: entity.GameType,
		List:          []*entity.ListItem{}}

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
		entity.HIGH_LOSS_STAT:                        highLossStat,
		entity.LOW_WIN_STAT:                          lowWinStat,
		entity.UPSET_WIN_STAT:                        upsetWinStat,
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

	return map[string]*entity.StatItem{entity.NO_BLANKS_PLAYED_STAT: {Name: entity.NO_BLANKS_PLAYED_STAT,
		Minimum:       0,
		Maximum:       0,
		Total:         0,
		IncrementType: entity.EventType},

		// All is 24
		entity.MANY_DOUBLE_LETTERS_COVERED_STAT: {Name: entity.MANY_DOUBLE_LETTERS_COVERED_STAT,
			Minimum:       20,
			Maximum:       entity.MaxNotableInt,
			Total:         0,
			IncrementType: entity.EventType},

		// All is 17
		entity.MANY_DOUBLE_WORDS_COVERED_STAT: {Name: entity.MANY_DOUBLE_WORDS_COVERED_STAT,
			Minimum:       15,
			Maximum:       entity.MaxNotableInt,
			Total:         0,
			IncrementType: entity.EventType},

		entity.ALL_TRIPLE_LETTERS_COVERED_STAT: {Name: entity.ALL_TRIPLE_LETTERS_COVERED_STAT,
			Minimum:       12,
			Maximum:       entity.MaxNotableInt,
			Total:         0,
			IncrementType: entity.EventType},

		entity.ALL_TRIPLE_WORDS_COVERED_STAT: {Name: entity.ALL_TRIPLE_WORDS_COVERED_STAT,
			Minimum:       8,
			Maximum:       entity.MaxNotableInt,
			Total:         0,
			IncrementType: entity.EventType},

		entity.COMBINED_HIGH_SCORING_STAT: {Name: entity.COMBINED_HIGH_SCORING_STAT,
			Minimum:       1100,
			Maximum:       entity.MaxNotableInt,
			Total:         0,
			IncrementType: entity.GameType},

		entity.COMBINED_LOW_SCORING_STAT: {Name: entity.COMBINED_LOW_SCORING_STAT,
			Minimum:       0,
			Maximum:       200,
			Total:         0,
			IncrementType: entity.GameType},

		entity.ONE_PLAYER_PLAYS_EVERY_POWER_TILE_STAT: {Name: entity.ONE_PLAYER_PLAYS_EVERY_POWER_TILE_STAT,
			Minimum:       10,
			Maximum:       10,
			Total:         0,
			IncrementType: entity.EventType},

		entity.ONE_PLAYER_PLAYS_EVERY_E_STAT: {Name: entity.ONE_PLAYER_PLAYS_EVERY_E_STAT,
			Minimum:       12,
			Maximum:       12,
			Total:         0,
			IncrementType: entity.EventType},

		entity.MANY_CHALLENGES_STAT: {Name: entity.MANY_CHALLENGES_STAT,
			Minimum:       5,
			Maximum:       entity.MaxNotableInt,
			Total:         0,
			IncrementType: entity.EventType},

		entity.FOUR_OR_MORE_CONSECUTIVE_BINGOS_STAT: {Name: entity.FOUR_OR_MORE_CONSECUTIVE_BINGOS_STAT,
			Minimum:       4,
			Maximum:       entity.MaxNotableInt,
			Total:         0,
			IncrementType: entity.EventType,
			Subitems:      map[string]int{"player_one_streak": 0, "player_two_streak": 0}}}
}

// XXX: This needs to be re-done to support non-English alphabets.
func makeAlphabetSubitems() map[string]int {
	alphabetSubitems := make(map[string]int)
	for i := 0; i < 26; i++ {
		alphabetSubitems[string('A'+rune(i))] = 0
	}
	alphabetSubitems[string(tilemapping.BlankToken)] = 0
	return alphabetSubitems
}

func makeGameSubitems() map[string]int {
	gameSubitems := make(map[string]int)
	for _, value := range pb.ChallengeRule_name {
		gameSubitems[value] = 0
	}
	for _, value := range ipc.RatingMode_name {
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
