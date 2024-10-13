package testutils

import (
	"strconv"
	"strings"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func AddRoundPairings(request *pb.PairRequest, pairingsStr string) {
	pairings := strings.Fields(pairingsStr)
	roundPairings := &pb.RoundPairings{}
	for _, pairing := range pairings {
		pairingInt, err := strconv.Atoi(pairing)
		if err != nil {
			panic(err)
		}
		roundPairings.Pairings = append(roundPairings.Pairings, int32(pairingInt))
	}
	request.DivisionPairings = append(request.DivisionPairings, roundPairings)
}

func AddRoundResults(request *pb.PairRequest, resultsStr string) {
	results := strings.Fields(resultsStr)
	roundResults := &pb.RoundResults{}
	for _, result := range results {
		resultInt, err := strconv.Atoi(result)
		if err != nil {
			panic(err)
		}
		roundResults.Results = append(roundResults.Results, int32(resultInt))
	}
	request.DivisionResults = append(request.DivisionResults, roundResults)
}

func AddRoundResultsAndPairings(request *pb.PairRequest, combinedStr string) {
	fields := strings.Fields(combinedStr)

	roundPairings := &pb.RoundPairings{}
	roundResults := &pb.RoundResults{}

	if len(fields)%2 != 0 {
		panic("Input string must contain pairs of <opp_index> <player_score>")
	}

	for i := 0; i < len(fields); i += 2 {
		oppIndex, err := strconv.Atoi(fields[i])
		if err != nil {
			panic(err)
		}
		playerScore, err := strconv.Atoi(fields[i+1])
		if err != nil {
			panic(err)
		}

		roundPairings.Pairings = append(roundPairings.Pairings, int32(oppIndex))
		roundResults.Results = append(roundResults.Results, int32(playerScore))
	}

	request.DivisionPairings = append(request.DivisionPairings, roundPairings)
	request.DivisionResults = append(request.DivisionResults, roundResults)
}

func CreateDefaultPairRequest() *pb.PairRequest {
	request := &pb.PairRequest{
		PairMethod:           pb.PairMethod_COP,
		PlayerNames:          []string{"Alice", "Bob", "Charlie", "Dave", "Eric", "Frank", "Grace", "Holly"},
		DivisionPairings:     []*pb.RoundPairings{},
		DivisionResults:      []*pb.RoundResults{},
		Classes:              []int32{4},
		ClassPrizes:          []int32{2},
		GibsonSpreads:        []int32{300, 250, 200},
		ControlLossThreshold: 0.25,
		HopefulnessThreshold: 0.02,
		Players:              8,
		Rounds:               10,
		PlacePrizes:          2,
		DivisionSims:         1000,
		ControlLossSims:      1000,
		UseControlLoss:       false,
		AllowRepeatByes:      false,
	}
	return request
}

func CreateDefaultOddPairRequest() *pb.PairRequest {
	request := &pb.PairRequest{
		PairMethod:           pb.PairMethod_COP,
		PlayerNames:          []string{"Alice", "Bob", "Charlie", "Dave", "Eric", "Frank", "Grace"},
		DivisionPairings:     []*pb.RoundPairings{},
		DivisionResults:      []*pb.RoundResults{},
		Classes:              []int32{4},
		ClassPrizes:          []int32{2},
		GibsonSpreads:        []int32{300, 250, 200},
		ControlLossThreshold: 0.25,
		HopefulnessThreshold: 0.02,
		Players:              7,
		Rounds:               10,
		PlacePrizes:          2,
		DivisionSims:         1000,
		ControlLossSims:      1000,
		UseControlLoss:       false,
		AllowRepeatByes:      false,
	}
	return request
}

func CreateKingston2023AfterRound15PairRequest() *pb.PairRequest {
	request := &pb.PairRequest{
		PairMethod:  pb.PairMethod_COP,
		Players:     26,
		Rounds:      23,
		PlayerNames: []string{"Jackson Smylie", "Joshua Sokol", "Arie Sinke", "Michael Fagen", "Max Panitch", "Lou Cornelis", "Eric Goldstein", "Jason Li", "Christopher Sykes", "Chloe Fatsis", "Deen Hergott", "Stefan Fatsis", "Noah Kalus", "Steve Ozorio", "Anna Miransky", "Lydia Keras", "Andre Popadynec", "Marcela Kadanka", "Agnes Kramer", "Richard Kirsch", "Joan Buma", "Gregg Bigourdin", "Sharmaine Farini", "Lilla Sinanan", "Mad Palazzo", "Trevor Sealy"},
		DivisionPairings: []*pb.RoundPairings{
			{Pairings: []int32{20, 22, 23, 25, 21, 24, 10, 14, 16, 15, 6, 17, 19, 18, 7, 9, 8, 11, 13, 12, 0, 4, 1, 2, 5, 3}},
			{Pairings: []int32{16, 13, 17, 19, 14, 15, 18, 21, 20, 24, 22, 23, 25, 1, 4, 5, 0, 2, 6, 3, 8, 7, 10, 11, 9, 12}},
			{Pairings: []int32{8, 6, 11, 12, 7, 9, 1, 4, 0, 5, 13, 2, 3, 10, 21, 24, 20, 23, 22, 25, 16, 14, 18, 17, 15, 19}},
			{Pairings: []int32{3, 18, 10, 0, 19, 7, 14, 5, 24, 16, 2, 22, 21, 20, 6, 23, 9, 25, 1, 4, 13, 12, 11, 15, 8, 17}},
			{Pairings: []int32{10, 7, 3, 2, 24, 18, 21, 1, 19, 25, 0, 15, 14, 22, 12, 11, 17, 16, 5, 8, 23, 6, 13, 20, 4, 9}},
			{Pairings: []int32{11, 19, 4, 15, 2, 14, 24, 8, 7, 13, 12, 0, 10, 9, 5, 3, 22, 18, 17, 1, 21, 20, 16, 25, 6, 23}},
			{Pairings: []int32{19, 12, 5, 14, 18, 2, 13, 17, 10, 11, 8, 9, 1, 6, 3, 22, 25, 7, 4, 0, 24, 23, 15, 21, 20, 16}},
			{Pairings: []int32{2, 17, 0, 4, 3, 19, 9, 10, 12, 6, 7, 20, 8, 15, 16, 13, 14, 1, 21, 5, 11, 18, 23, 22, 25, 24}},
			{Pairings: []int32{1, 0, 7, 17, 6, 8, 4, 2, 5, 12, 23, 18, 9, 19, 24, 16, 15, 3, 11, 13, 22, 25, 20, 10, 14, 21}},
			{Pairings: []int32{5, 8, 12, 9, 16, 0, 17, 13, 1, 3, 20, 21, 2, 7, 23, 19, 4, 6, 24, 15, 10, 11, 25, 14, 18, 22}},
			{Pairings: []int32{4, 3, 9, 1, 0, 17, 8, 18, 6, 2, 15, 13, 16, 11, 22, 10, 12, 5, 7, 23, 25, 24, 14, 19, 21, 20}},
			{Pairings: []int32{7, 9, 8, 11, 12, 6, 5, 0, 2, 1, 19, 3, 4, 25, 17, 20, 21, 14, 23, 10, 15, 16, 24, 18, 22, 13}},
			{Pairings: []int32{14, 5, 6, 21, 17, 1, 2, 16, 9, 8, 24, 12, 11, 23, 0, 25, 7, 4, 20, 22, 18, 3, 19, 13, 10, 15}},
			{Pairings: []int32{6, 2, 1, 8, 5, 4, 0, 9, 3, 7, 25, 14, 24, 17, 11, 18, 23, 13, 15, 20, 19, 22, 21, 16, 12, 10}},
			{Pairings: []int32{8, 6, 19, 9, 22, 11, 1, 12, 0, 3, 18, 5, 7, 16, 25, 21, 13, 20, 10, 2, 17, 15, 4, 24, 23, 14}},
		},
		DivisionResults: []*pb.RoundResults{
			{Results: []int32{572, 430, 504, 392, 413, 401, 427, 322, 422, 449, 376, 332, 355, 416, 438, 367, 224, 475, 389, 435, 324, 408, 397, 307, 355, 378}},
			{Results: []int32{395, 465, 502, 382, 440, 416, 445, 483, 530, 449, 420, 347, 533, 302, 287, 336, 329, 301, 300, 361, 316, 320, 379, 343, 341, 336}},
			{Results: []int32{388, 484, 494, 334, 396, 450, 467, 378, 352, 396, 409, 353, 392, 398, 364, 267, 477, 419, 551, 332, 347, 410, 289, 309, 398, 408}},
			{Results: []int32{402, 437, 441, 503, 340, 361, 456, 452, 421, 415, 301, 366, 404, 471, 420, 407, 267, 419, 413, 452, 403, 355, 314, 280, 414, 373}},
			{Results: []int32{378, 415, 407, 412, 568, 374, 453, 384, 412, 464, 444, 381, 339, 408, 479, 315, 388, 456, 417, 334, 457, 351, 402, 344, 324, 354}},
			{Results: []int32{460, 449, 417, 406, 379, 441, 332, 377, 455, 387, 405, 262, 469, 351, 345, 393, 467, 418, 360, 400, 381, 418, 256, 352, 482, 358}},
			{Results: []int32{342, 395, 414, 437, 393, 529, 494, 466, 392, 535, 356, 333, 426, 370, 406, 388, 353, 363, 419, 344, 404, 390, 286, 343, 409, 330}},
			{Results: []int32{513, 448, 413, 354, 498, 471, 458, 469, 486, 246, 341, 593, 328, 381, 308, 286, 425, 493, 379, 394, 344, 570, 378, 342, 427, 334}},
			{Results: []int32{410, 531, 476, 489, 353, 380, 433, 350, 385, 454, 314, 403, 425, 388, 380, 383, 387, 375, 503, 334, 401, 344, 404, 454, 327, 349}},
			{Results: []int32{434, 393, 479, 478, 459, 410, 433, 498, 387, 484, 430, 473, 329, 340, 566, 322, 440, 319, 414, 365, 312, 407, 393, 311, 431, 345}},
			{Results: []int32{453, 439, 405, 359, 386, 450, 327, 414, 488, 369, 496, 423, 452, 187, 446, 350, 387, 226, 316, 418, 359, 455, 389, 380, 394, 448}},
			{Results: []int32{436, 360, 386, 384, 444, 358, 433, 423, 445, 470, 330, 378, 452, 333, 380, 435, 407, 358, 448, 344, 376, 372, 243, 437, 437, 406}},
			{Results: []int32{470, 444, 353, 340, 381, 350, 432, 413, 435, 301, 378, 439, 367, 523, 367, 378, 370, 442, 442, 343, 331, 328, 329, 322, 365, 314}},
			{Results: []int32{400, 482, 374, 451, 409, 424, 442, 510, 413, 323, 514, 356, 414, 326, 359, 382, 496, 370, 326, 444, 414, 293, 445, 324, 355, 282}},
			{Results: []int32{442, 414, 477, 469, 360, 537, 504, 382, 400, 431, 386, 249, 405, 428, 402, 358, 267, 325, 428, 404, 408, 481, 356, 427, 368, 378}},
		},
		Classes:              []int32{4},
		ClassPrizes:          []int32{2},
		GibsonSpreads:        []int32{300, 250, 200},
		ControlLossThreshold: 0.25,
		HopefulnessThreshold: 0.02,
		PlacePrizes:          2,
		DivisionSims:         1000,
		ControlLossSims:      1000,
		UseControlLoss:       false,
		AllowRepeatByes:      false,
	}
	return request
}

func CreateAlbanyjuly4th2024AfterRound21PairRequest() *pb.PairRequest {
	request := &pb.PairRequest{
		PairMethod:  pb.PairMethod_COP,
		Players:     30,
		Rounds:      27,
		PlayerNames: []string{"Wellington Jighere", "Adam Logan", "Will Anderson", "Dennis Ikekeregor", "Austin Shin", "Matthew O'Connor", "Chris Lipe", "Joshua Castellano", "Josh Sokol", "Jason Keller", "Ben Schoenbrun", "Erickson Smith", "Bright Idahosa", "Robert Linn", "Jason Ubeika", "Tim Weiss", "Richard Buck", "Anthony Ikolo", "Daniel Blake", "Terry Kang", "Carmel Dodd", "Niel Gan", "Steve Ozorio", "Thomas Stumpf", "Joe Roberdeau", "Cheryl Melvin", "Iliana Filby", "Ivan Sentongo", "Edgar Odongkara", "Mohamed Kamara"},
		DivisionPairings: []*pb.RoundPairings{
			{Pairings: []int32{25, 20, 26, 22, 21, 13, 23, 24, 27, 29, 15, 17, 19, 5, 18, 10, 28, 11, 14, 12, 1, 4, 3, 6, 7, 0, 2, 8, 16, 9}},
			{Pairings: []int32{29, 19, 14, 16, 17, 18, 15, 27, 24, 25, 23, 21, 20, 26, 2, 6, 3, 4, 5, 1, 12, 11, 28, 10, 8, 9, 13, 7, 22, 0}},
			{Pairings: []int32{9, 12, 5, 28, 11, 2, 10, 8, 7, 0, 6, 4, 1, 14, 13, 23, 22, 21, 26, 20, 19, 17, 16, 15, 27, 29, 18, 24, 3, 25}},
			{Pairings: []int32{4, 13, 28, 17, 0, 16, 7, 6, 12, 10, 9, 18, 8, 1, 24, 26, 5, 3, 11, 27, 29, 25, 23, 22, 14, 21, 15, 19, 2, 20}},
			{Pairings: []int32{11, 2, 1, 9, 20, 28, 17, 23, 10, 3, 8, 0, 22, 18, 16, 29, 14, 6, 13, 24, 4, 27, 12, 7, 19, 26, 25, 21, 5, 15}},
			{Pairings: []int32{3, 10, 8, 0, 23, 6, 5, 17, 2, 11, 1, 9, 24, 20, 25, 27, 26, 7, 28, 29, 13, 22, 21, 4, 12, 14, 16, 15, 18, 19}},
			{Pairings: []int32{1, 0, 7, 10, 18, 23, 8, 2, 6, 28, 3, 12, 11, 17, 26, 20, 25, 13, 4, 21, 15, 19, 24, 5, 22, 16, 14, 29, 9, 27}},
			{Pairings: []int32{6, 3, 11, 1, 28, 29, 0, 10, 20, 19, 7, 2, 14, 23, 12, 21, 24, 18, 17, 9, 8, 15, 26, 13, 16, 27, 22, 25, 4, 5}},
			{Pairings: []int32{10, 6, 4, 7, 2, 19, 1, 3, 14, 12, 0, 28, 9, 24, 8, 18, 23, 29, 15, 5, 21, 20, 25, 16, 13, 22, 27, 26, 11, 17}},
			{Pairings: []int32{7, 28, 12, 6, 8, 21, 3, 0, 4, 15, 11, 10, 2, 16, 29, 9, 13, 20, 19, 18, 17, 5, 27, 26, 25, 24, 23, 22, 1, 14}},
			{Pairings: []int32{12, 8, 17, 11, 6, 27, 4, 15, 1, 14, 28, 3, 0, 19, 9, 7, 20, 2, 21, 13, 16, 18, 29, 25, 26, 23, 24, 5, 10, 22}},
			{Pairings: []int32{15, 11, 20, 4, 3, 22, 28, 29, 17, 13, 12, 1, 10, 9, 27, 0, 18, 8, 16, 25, 2, 26, 5, 24, 23, 19, 21, 14, 6, 7}},
			{Pairings: []int32{28, 15, 29, 12, 10, 25, 11, 13, 9, 8, 4, 6, 3, 7, 23, 1, 17, 16, 22, 26, 27, 24, 18, 14, 21, 5, 19, 20, 0, 2}},
			{Pairings: []int32{8, 16, 3, 2, 7, 11, 20, 4, 0, 24, 17, 5, 13, 12, 22, 19, 1, 10, 25, 15, 6, 23, 14, 21, 9, 18, 29, 28, 27, 26}},
			{Pairings: []int32{2, 17, 0, 13, 15, 7, 9, 5, 11, 6, 20, 8, 16, 3, 28, 4, 12, 1, 24, 22, 10, 29, 19, 27, 18, 26, 25, 23, 14, 21}},
			{Pairings: []int32{13, 7, 10, 29, 27, 17, 28, 1, 16, 26, 2, 22, 15, 0, 21, 12, 8, 5, 20, 23, 18, 14, 11, 19, 25, 24, 9, 4, 6, 3}},
			{Pairings: []int32{5, 20, 6, 18, 10, 0, 2, 11, 29, 22, 4, 7, 28, 21, 19, 16, 15, 27, 3, 14, 1, 13, 9, 25, 26, 23, 24, 17, 12, 8}},
			{Pairings: []int32{10, 4, 27, 14, 1, 12, 20, 28, 15, 17, 0, 19, 5, 25, 3, 8, 21, 9, 23, 11, 6, 16, 26, 18, 29, 13, 22, 2, 7, 24}},
			{Pairings: []int32{20, 2, 1, 24, 9, 26, 27, 10, 18, 4, 7, 14, 17, 22, 11, 28, 19, 12, 8, 16, 0, 25, 13, 29, 3, 21, 5, 6, 15, 23}},
			{Pairings: []int32{17, 9, 10, 21, 12, 14, 18, 20, 13, 1, 2, 24, 4, 8, 5, 22, 27, 0, 6, 25, 7, 3, 15, 26, 11, 19, 23, 16, 29, 28}},
			{Pairings: []int32{18, 5, 9, 23, 7, 1, 12, 4, 10, 2, 8, 27, 6, 28, 20, 24, 29, 19, 0, 17, 14, 26, 25, 3, 15, 22, 21, 11, 13, 16}},
		},
		DivisionResults: []*pb.RoundResults{
			{Results: []int32{509, 431, 503, 426, 438, 454, 452, 533, 466, 506, 443, 219, 413, 487, 373, 399, 382, 503, 552, 404, 366, 379, 392, 405, 327, 356, 371, 356, 437, 345}},
			{Results: []int32{533, 428, 521, 423, 479, 503, 444, 435, 514, 526, 453, 459, 397, 554, 321, 427, 423, 405, 406, 352, 392, 431, 408, 382, 389, 325, 437, 353, 421, 356}},
			{Results: []int32{400, 486, 448, 432, 363, 406, 420, 407, 469, 506, 483, 495, 358, 409, 426, 370, 397, 401, 424, 421, 474, 309, 440, 388, 385, 349, 411, 451, 394, 506}},
			{Results: []int32{429, 493, 491, 397, 329, 515, 424, 390, 527, 356, 415, 578, 344, 392, 417, 464, 350, 305, 342, 474, 520, 498, 381, 458, 462, 246, 354, 402, 364, 421}},
			{Results: []int32{595, 439, 375, 494, 436, 349, 564, 523, 361, 462, 595, 408, 460, 465, 485, 415, 389, 466, 490, 480, 355, 493, 416, 272, 325, 304, 396, 253, 436, 338}},
			{Results: []int32{481, 368, 475, 361, 497, 340, 497, 448, 426, 422, 474, 469, 494, 357, 552, 437, 460, 414, 431, 379, 436, 419, 397, 446, 360, 412, 369, 423, 531, 465}},
			{Results: []int32{481, 503, 423, 575, 467, 556, 444, 478, 413, 380, 346, 425, 337, 391, 444, 410, 499, 505, 294, 407, 442, 395, 323, 296, 414, 372, 367, 421, 497, 489}},
			{Results: []int32{436, 409, 326, 342, 365, 411, 378, 425, 447, 482, 428, 506, 443, 400, 314, 505, 564, 523, 380, 461, 378, 372, 415, 358, 448, 442, 351, 483, 494, 523}},
			{Results: []int32{405, 454, 389, 436, 494, 321, 392, 422, 553, 426, 492, 522, 443, 461, 419, 545, 429, 482, 374, 471, 422, 319, 404, 364, 283, 367, 393, 490, 469, 338}},
			{Results: []int32{521, 327, 439, 395, 506, 424, 474, 479, 466, 410, 519, 360, 469, 412, 494, 430, 449, 472, 469, 440, 317, 368, 340, 352, 463, 277, 490, 450, 564, 422}},
			{Results: []int32{423, 447, 474, 470, 393, 436, 452, 333, 362, 407, 404, 364, 430, 474, 391, 387, 316, 417, 411, 327, 562, 366, 353, 416, 433, 448, 420, 598, 390, 445}},
			{Results: []int32{430, 495, 374, 386, 487, 481, 473, 554, 434, 350, 518, 343, 387, 379, 401, 386, 469, 442, 391, 441, 396, 448, 382, 419, 358, 377, 304, 408, 456, 353}},
			{Results: []int32{430, 476, 427, 383, 524, 480, 333, 466, 399, 460, 410, 473, 535, 441, 525, 311, 415, 424, 395, 479, 418, 427, 375, 367, 367, 264, 397, 409, 399, 406}},
			{Results: []int32{351, 490, 491, 481, 376, 524, 375, 537, 475, 420, 570, 409, 474, 492, 308, 572, 337, 375, 410, 343, 395, 364, 481, 369, 414, 373, 320, 456, 404, 428}},
			{Results: []int32{436, 362, 459, 442, 421, 389, 536, 437, 470, 401, 484, 364, 395, 401, 417, 412, 336, 513, 396, 382, 330, 310, 396, 409, 401, 402, 472, 511, 570, 499}},
			{Results: []int32{586, 581, 449, 399, 473, 383, 466, 312, 535, 537, 559, 499, 493, 327, 419, 437, 398, 381, 459, 424, 369, 377, 412, 363, 421, 372, 267, 377, 427, 441}},
			{Results: []int32{454, 412, 424, 373, 477, 383, 399, 475, 505, 465, 378, 392, 443, 416, 458, 398, 485, 487, 447, 440, 453, 315, 367, 370, 470, 414, 373, 493, 492, 292}},
			{Results: []int32{492, 325, 503, 503, 466, 429, 519, 355, 514, 494, 492, 460, 387, 545, 383, 375, 410, 326, 469, 422, 415, 413, 378, 404, 352, 303, 403, 373, 500, 496}},
			{Results: []int32{459, 450, 451, 397, 489, 594, 470, 451, 455, 438, 422, 357, 421, 360, 484, 449, 396, 435, 479, 569, 442, 533, 356, 420, 400, 321, 287, 341, 401, 487}},
			{Results: []int32{480, 469, 445, 476, 460, 407, 409, 437, 503, 498, 466, 507, 377, 401, 403, 375, 352, 313, 406, 465, 485, 330, 436, 410, 319, 379, 434, 438, 514, 422}},
			{Results: []int32{497, 390, 296, 585, 419, 417, 327, 438, 443, 595, 449, 337, 517, 526, 538, 379, 374, 338, 421, 326, 421, 507, 465, 333, 411, 398, 374, 502, 357, 426}},
		},
		Classes:              []int32{4},
		ClassPrizes:          []int32{2},
		GibsonSpreads:        []int32{300, 250, 200},
		ControlLossThreshold: 0.25,
		HopefulnessThreshold: 0.02,
		PlacePrizes:          2,
		DivisionSims:         1000,
		ControlLossSims:      1000,
		UseControlLoss:       false,
		AllowRepeatByes:      false,
	}
	return request
}
