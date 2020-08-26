package entity

func instantiatePlayerData() map[string]*StatItem {

	gamesStat := &StatItem{Total: 0,
		DataType:      SingleType,
		IncrementType: GameType,
		List:          []*ListItem{},
		Subitems:      makeGameSubitems(),
		AddFunction:   addGames}

	turnsStat := &StatItem{Total: 0,
		DataType:      SingleType,
		IncrementType: EventType,
		List:          []*ListItem{},
		AddFunction:   addTurns}

	scoreStat := &StatItem{Total: 0,
		DataType:      SingleType,
		IncrementType: GameType,
		List:          []*ListItem{},
		AddFunction:   addScore}

	firstsStat := &StatItem{Total: 0,
		DataType:      SingleType,
		IncrementType: GameType,
		List:          []*ListItem{},
		AddFunction:   addFirsts}

	verticalOpeningsStat := &StatItem{Total: 0,
		DataType:      SingleType,
		IncrementType: EventType,
		List:          []*ListItem{},
		AddFunction:   addVerticalOpenings}

	exchangesStat := &StatItem{Total: 0,
		DataType:      SingleType,
		IncrementType: EventType,
		List:          []*ListItem{},
		AddFunction:   addExchanges}

	challengedPhoniesStat := &StatItem{Total: 0,
		DataType:      ListType,
		IncrementType: EventType,
		List:          []*ListItem{},
		AddFunction:   addChallengedPhonies}

	unchallengedPhoniesStat := &StatItem{Total: 0,
		DataType:      ListType,
		IncrementType: EventType,
		List:          []*ListItem{},
		AddFunction:   addUnchallengedPhonies}

	validPlaysThatWereChallengedStat := &StatItem{Total: 0,
		DataType:      ListType,
		IncrementType: EventType,
		List:          []*ListItem{},
		AddFunction:   addValidPlaysThatWereChallenged}

	challengesWonStat := &StatItem{Total: 0,
		DataType:      ListType,
		IncrementType: EventType,
		List:          []*ListItem{},
		AddFunction:   addChallengesWon}

	challengesLostStat := &StatItem{Total: 0,
		DataType:      ListType,
		IncrementType: EventType,
		List:          []*ListItem{},
		AddFunction:   addChallengesLost}

	winsStat := &StatItem{Total: 0,
		DataType:      SingleType,
		IncrementType: GameType,
		List:          []*ListItem{},
		AddFunction:   addWins}

	lossesStat := &StatItem{Total: 0,
		DataType:      SingleType,
		IncrementType: GameType,
		List:          []*ListItem{},
		AddFunction:   addLosses}

	drawsStat := &StatItem{Total: 0,
		DataType:      SingleType,
		IncrementType: GameType,
		List:          []*ListItem{},
		AddFunction:   addDraws}

	bingosStat := &StatItem{Total: 0,
		DataType:      ListType,
		IncrementType: EventType,
		List:          []*ListItem{},
		AddFunction:   addBingos}

	noBingosStat := &StatItem{Total: 0,
		DataType:      ListType,
		IncrementType: GameType,
		List:          []*ListItem{},
		AddFunction:   addNoBingos}

	tripleTriplesStat := &StatItem{Total: 0,
		DataType:      ListType,
		IncrementType: EventType,
		List:          []*ListItem{},
		AddFunction:   addTripleTriples}

	highGameStat := &StatItem{Total: 0,
		DataType:      MaximumType,
		IncrementType: GameType,
		List:          []*ListItem{},
		AddFunction:   setHighGame}

	lowGameStat := &StatItem{Total: MaxNotableInt,
		DataType:      MinimumType,
		IncrementType: GameType,
		List:          []*ListItem{},
		AddFunction:   setLowGame}

	highTurnStat := &StatItem{Total: 0,
		DataType:      MaximumType,
		IncrementType: EventType,
		List:          []*ListItem{},
		AddFunction:   setHighTurn}

	tilesPlayedStat := &StatItem{Total: 0,
		DataType:      SingleType,
		IncrementType: EventType,
		List:          []*ListItem{},
		Subitems:      makeAlphabetSubitems(),
		AddFunction:   addTilesPlayed}

	turnsWithBlankStat := &StatItem{Total: 0,
		DataType:      SingleType,
		IncrementType: EventType,
		List:          []*ListItem{},
		AddFunction:   addTurnsWithBlank}

	commentsStat := &StatItem{Total: 0,
		DataType:      ListType,
		IncrementType: EventType,
		List:          []*ListItem{},
		AddFunction:   addComments}

	timeStat := &StatItem{Total: 0,
		DataType:      SingleType,
		IncrementType: GameType,
		List:          []*ListItem{},
		AddFunction:   addTime}

	mistakesStat := &StatItem{Total: 0,
		DataType:      ListType,
		IncrementType: EventType,
		List:          []*ListItem{},
		Subitems:      makeMistakeSubitems(),
		AddFunction:   addMistakes}

	ratingsStat := &StatItem{Total: 0,
		DataType:      ListType,
		IncrementType: GameType,
		List:          []*ListItem{},
		AddFunction:   addRatings}

	return map[string]*StatItem{BINGOS_STAT: bingosStat,
		CHALLENGED_PHONIES_STAT:               challengedPhoniesStat,
		CHALLENGES_LOST_STAT:                  challengesLostStat,
		CHALLENGES_WON_STAT:                   challengesWonStat,
		COMMENTS_STAT:                         commentsStat,
		DRAWS_STAT:                            drawsStat,
		EXCHANGES_STAT:                        exchangesStat,
		FIRSTS_STAT:                           firstsStat,
		GAMES_STAT:                            gamesStat,
		HIGH_GAME_STAT:                        highGameStat,
		HIGH_TURN_STAT:                        highTurnStat,
		LOSSES_STAT:                           lossesStat,
		LOW_GAME_STAT:                         lowGameStat,
		NO_BINGOS_STAT:                        noBingosStat,
		MISTAKES_STAT:                         mistakesStat,
		VALID_PLAYS_THAT_WERE_CHALLENGED_STAT: validPlaysThatWereChallengedStat,
		RATINGS_STAT:                          ratingsStat,
		SCORE_STAT:                            scoreStat,
		TILES_PLAYED_STAT:                     tilesPlayedStat,
		TIME_STAT:                             timeStat,
		TRIPLE_TRIPLES_STAT:                   tripleTriplesStat,
		TURNS_STAT:                            turnsStat,
		TURNS_WITH_BLANK_STAT:                 turnsWithBlankStat,
		UNCHALLENGED_PHONIES_STAT:             unchallengedPhoniesStat,
		VERTICAL_OPENINGS_STAT:                verticalOpeningsStat,
		WINS_STAT:                             winsStat,
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
		Maximum:       0,
		Total:         0,
		DataType:      ListType,
		IncrementType: EventType,
		List:          []*ListItem{},
		AddFunction:   addBlanksPlayed},

		// All is 24
		MANY_DOUBLE_LETTERS_COVERED_STAT: &StatItem{Minimum: 20,
			Maximum:       MaxNotableInt,
			Total:         0,
			DataType:      ListType,
			IncrementType: EventType,
			List:          []*ListItem{},
			AddFunction:   addDoubleLetter},

		// All is 17
		MANY_DOUBLE_WORDS_COVERED_STAT: &StatItem{Minimum: 15,
			Maximum:       MaxNotableInt,
			Total:         0,
			DataType:      ListType,
			IncrementType: EventType,
			List:          []*ListItem{},
			AddFunction:   addDoubleWord},

		ALL_TRIPLE_LETTERS_COVERED_STAT: &StatItem{Minimum: 12,
			Maximum:       MaxNotableInt,
			Total:         0,
			DataType:      ListType,
			IncrementType: EventType,
			List:          []*ListItem{},
			AddFunction:   addTripleLetter},

		ALL_TRIPLE_WORDS_COVERED_STAT: &StatItem{Minimum: 8,
			Maximum:       MaxNotableInt,
			Total:         0,
			DataType:      ListType,
			IncrementType: EventType,
			List:          []*ListItem{},
			AddFunction:   addTripleWord},

		COMBINED_HIGH_SCORING_STAT: &StatItem{Minimum: 1100,
			Maximum:       MaxNotableInt,
			Total:         0,
			DataType:      ListType,
			IncrementType: GameType,
			List:          []*ListItem{},
			AddFunction:   addCombinedScoring},

		COMBINED_LOW_SCORING_STAT: &StatItem{Minimum: 0,
			Maximum:       200,
			Total:         0,
			DataType:      ListType,
			IncrementType: GameType,
			List:          []*ListItem{},
			AddFunction:   addCombinedScoring},

		ONE_PLAYER_PLAYS_EVERY_POWER_TILE_STAT: &StatItem{Minimum: 10,
			Maximum:       10,
			Total:         0,
			DataType:      ListType,
			IncrementType: EventType,
			List:          []*ListItem{},
			AddFunction:   addEveryPowerTile},

		ONE_PLAYER_PLAYS_EVERY_E_STAT: &StatItem{Minimum: 12,
			Maximum:       12,
			Total:         0,
			DataType:      ListType,
			IncrementType: EventType,
			List:          []*ListItem{},
			AddFunction:   addEveryE},

		MANY_CHALLENGES_STAT: &StatItem{Minimum: 5,
			Maximum:       MaxNotableInt,
			Total:         0,
			DataType:      ListType,
			IncrementType: EventType,
			List:          []*ListItem{},
			AddFunction:   addManyChallenges},

		FOUR_OR_MORE_CONSECUTIVE_BINGOS_STAT: &StatItem{Minimum: 4,
			Maximum:       MaxNotableInt,
			Total:         0,
			DataType:      ListType,
			IncrementType: EventType,
			List:          []*ListItem{},
			Subitems:      map[string]int{"player_one_streak": 0, "player_two_streak": 0},
			AddFunction:   addConsecutiveBingos},
	}
}
