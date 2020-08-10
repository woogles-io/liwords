package entity

func instantiatePlayerData() map[string]*StatItem {
	gamesStat := &StatItem{Total: 0,
		DataType:           SingleType,
		IncrementType:      GameType,
		DenominatorList:    nil,
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addGames}
	turnsStat := &StatItem{Total: 0,
		DataType:           SingleType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addTurns}

	scoreStat := &StatItem{Total: 0,
		DataType:           SingleType,
		IncrementType:      GameType,
		DenominatorList:    []*StatItem{gamesStat, turnsStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addScore}

	firstsStat := &StatItem{Total: 0,
		DataType:           SingleType,
		IncrementType:      GameType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addFirsts}
	verticalOpeningsStat := &StatItem{Total: 0,
		DataType:           SingleType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addVerticalOpenings}

	exchangesStat := &StatItem{Total: 0,
		DataType:           SingleType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addExchanges}

	phoniesStat := &StatItem{Total: 0,
		DataType:           SingleType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addPhonies}

	challengedPhoniesStat := &StatItem{Total: 0,
		DataType:           ListType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{phoniesStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addChallengedPhonies}

	unchallengedPhoniesStat := &StatItem{Total: 0,
		DataType:           ListType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{phoniesStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addUnchallengedPhonies}

	challengesStat := &StatItem{Total: 0,
		DataType:           SingleType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addChallenges}

	challengesWonStat := &StatItem{Total: 0,
		DataType:           ListType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{challengesStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addChallengesWon}

	challengesLostStat := &StatItem{Total: 0,
		DataType:           SingleType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{challengesStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addChallengesLost}

	playsThatWereChallengedStat := &StatItem{Total: 0,
		DataType:           ListType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{turnsStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addPlaysThatWereChallenged}

	winsStat := &StatItem{Total: 0,
		DataType:           SingleType,
		IncrementType:      GameType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addWins}
	lossesStat := &StatItem{Total: 0,
		DataType:           SingleType,
		IncrementType:      GameType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addLosses}
	drawsStat := &StatItem{Total: 0,
		DataType:           SingleType,
		IncrementType:      GameType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addDraws}
	bingosStat := &StatItem{Total: 0,
		DataType:           ListType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      true,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addBingos}

	noBingosStat := &StatItem{Total: 0,
		DataType:           ListType,
		IncrementType:      GameType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addNoBingos}

	tripleTriplesStat := &StatItem{Total: 0,
		DataType:           ListType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addTripleTriples}

	bingoNinesOrAboveStat := &StatItem{Total: 0,
		DataType:           ListType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addBingoNinesOrAbove}

	highGameStat := &StatItem{Total: 0,
		DataType:           MaximumType,
		IncrementType:      GameType,
		DenominatorList:    nil,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        setHighGame}

	lowGameStat := &StatItem{Total: MaxNotableInt,
		DataType:           MinimumType,
		IncrementType:      GameType,
		DenominatorList:    nil,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        setLowGame}

	highTurnStat := &StatItem{Total: 0,
		DataType:           MaximumType,
		IncrementType:      EventType,
		DenominatorList:    nil,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        setHighTurn}

	tilesPlayedStat := &StatItem{Total: 0,
		DataType:           SingleType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      false,
		List:               []*ListItem{},
		Subitems:           makeAlphabetSubitems(),
		HasMeaningfulTotal: true,
		AddFunction:        addTilesPlayed}

	turnsWithBlankStat := &StatItem{Total: 0,
		DataType:           SingleType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{turnsStat},
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addTurnsWithBlank}

	commentsStat := &StatItem{Total: 0,
		DataType:           ListType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addComments}

	mistakesStat := &StatItem{Total: 0,
		DataType:           ListType,
		IncrementType:      EventType,
		DenominatorList:    []*StatItem{gamesStat},
		IsProfileStat:      false,
		List:               []*ListItem{},
		Subitems:           makeMistakeSubitems(),
		HasMeaningfulTotal: true,
		AddFunction:        addMistakes}

	confidenceIntervalsStat := &StatItem{Total: 0,
		DataType:      SingleType,
		IncrementType: FinalType,
		// Not actually a denominatorList, just a needed ref
		// because this stat is special
		DenominatorList:    []*StatItem{tilesPlayedStat},
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: true,
		AddFunction:        addConfidenceIntervals}

	return map[string]*StatItem{BINGO_NINES_OR_ABOVE_STAT: bingoNinesOrAboveStat,
		BINGOS_STAT:                     bingosStat,
		CHALLENGED_PHONIES_STAT:         challengedPhoniesStat,
		CHALLENGES_STAT:                 challengesStat,
		CHALLENGES_LOST_STAT:            challengesLostStat,
		CHALLENGES_WON_STAT:             challengesWonStat,
		COMMENTS_STAT:                   commentsStat,
		CONFIDENCE_INTERVALS_STAT:       confidenceIntervalsStat, // Not tested
		DRAWS_STAT:                      drawsStat,
		EXCHANGES_STAT:                  exchangesStat,
		FIRSTS_STAT:                     firstsStat,
		GAMES_STAT:                      gamesStat,
		HIGH_GAME_STAT:                  highGameStat,
		HIGH_TURN_STAT:                  highTurnStat,
		LOSSES_STAT:                     lossesStat,
		LOW_GAME_STAT:                   lowGameStat,
		NO_BINGOS_STAT:                  noBingosStat,
		MISTAKES_STAT:                   mistakesStat,
		PHONIES_STAT:                    phoniesStat, // Not tested
		PLAYS_THAT_WERE_CHALLENGED_STAT: playsThatWereChallengedStat,
		SCORE_STAT:                      scoreStat,
		TILES_PLAYED_STAT:               tilesPlayedStat,
		TRIPLE_TRIPLES_STAT:             tripleTriplesStat, // Not tested
		TURNS_STAT:                      turnsStat,
		TURNS_WITH_BLANK_STAT:           turnsWithBlankStat,
		UNCHALLENGED_PHONIES_STAT:       unchallengedPhoniesStat, // Not tested
		VERTICAL_OPENINGS_STAT:          verticalOpeningsStat,
		WINS_STAT:                       winsStat,
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

func instantiateNotableData() map[string]*StatItem {
	return map[string]*StatItem{NO_BLANKS_PLAYED_STAT: &StatItem{Minimum: 0,
		Maximum:            0,
		Total:              0,
		DataType:           ListType,
		IncrementType:      EventType,
		IsProfileStat:      false,
		List:               []*ListItem{},
		HasMeaningfulTotal: false,
		AddFunction:        addBlanksPlayed},

		HIGH_SCORING_STAT: &StatItem{Minimum: 700,
			Maximum:            MaxNotableInt,
			Total:              0,
			DataType:           ListType,
			IncrementType:      GameType,
			IsProfileStat:      false,
			List:               []*ListItem{},
			HasMeaningfulTotal: false,
			AddFunction:        addHighScoring},

		COMBINED_HIGH_SCORING_STAT: &StatItem{Minimum: 1100,
			Maximum:            MaxNotableInt,
			Total:              0,
			DataType:           ListType,
			IncrementType:      GameType,
			IsProfileStat:      false,
			List:               []*ListItem{},
			HasMeaningfulTotal: false,
			AddFunction:        addCombinedScoring},

		COMBINED_LOW_SCORING_STAT: &StatItem{Minimum: 0,
			Maximum:            200,
			Total:              0,
			DataType:           ListType,
			IncrementType:      GameType,
			IsProfileStat:      false,
			List:               []*ListItem{},
			HasMeaningfulTotal: false,
			AddFunction:        addCombinedScoring},

		ONE_PLAYER_PLAYS_EVERY_POWER_TILE_STAT: &StatItem{Minimum: 10,
			Maximum:            10,
			Total:              0,
			DataType:           ListType,
			IncrementType:      EventType,
			IsProfileStat:      false,
			List:               []*ListItem{},
			HasMeaningfulTotal: false,
			AddFunction:        addEveryPowerTile},

		ONE_PLAYER_PLAYS_EVERY_E_STAT: &StatItem{Minimum: 12,
			Maximum:            12,
			Total:              0,
			DataType:           ListType,
			IncrementType:      EventType,
			IsProfileStat:      false,
			List:               []*ListItem{},
			HasMeaningfulTotal: false,
			AddFunction:        addEveryE},

		MANY_CHALLENGES_STAT: &StatItem{Minimum: 5,
			Maximum:            MaxNotableInt,
			Total:              0,
			DataType:           ListType,
			IncrementType:      EventType,
			IsProfileStat:      false,
			List:               []*ListItem{},
			HasMeaningfulTotal: false,
			AddFunction:        addManyChallenges},

		FOUR_OR_MORE_CONSECUTIVE_BINGOS_STAT: &StatItem{Minimum: 4,
			Maximum:            MaxNotableInt,
			Total:              0,
			DataType:           ListType,
			IncrementType:      EventType,
			IsProfileStat:      false,
			List:               []*ListItem{},
			Subitems:           map[string]int{"player_one_streak": 0, "player_two_streak": 0},
			HasMeaningfulTotal: false,
			AddFunction:        addConsecutiveBingos},
	}
}
