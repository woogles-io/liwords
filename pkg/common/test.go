package common

import (
	"math"
	"os"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	macondoconfig "github.com/domino14/macondo/config"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

const PuzzleVariant = "puzzle"
const DefaultLexicon = "CSW21"

var DefaultConfig = macondoconfig.Config{
	LexiconPath:               os.Getenv("LEXICON_PATH"),
	LetterDistributionPath:    os.Getenv("LETTER_DISTRIBUTION_PATH"),
	DefaultLexicon:            "CSW21",
	DefaultLetterDistribution: "English",
}

var DefaultGameReq = &ipc.GameRequest{Lexicon: DefaultLexicon,
	Rules: &ipc.GameRules{BoardLayoutName: entity.CrosswordGame,
		LetterDistributionName: "English",
		VariantName:            "classic"},
	InitialTimeSeconds: 25 * 60,
	IncrementSeconds:   0,
	ChallengeRule:      macondopb.ChallengeRule_FIVE_POINT,
	GameMode:           ipc.GameMode_REAL_TIME,
	RatingMode:         ipc.RatingMode_RATED,
	RequestId:          "puzzlereq",
	OriginalRequestId:  "puzzlereq",
	MaxOvertimeMinutes: 0}

var DefaultPlayerOneInfo = ipc.PlayerInfo{
	UserId:   "puzzlePlayerOne",
	Nickname: "p1",
	FullName: "Puzzle Player One",
	Rating:   "0",
	IsBot:    true,
	First:    true,
}

var DefaultPlayerTwoInfo = ipc.PlayerInfo{
	UserId:   "puzzlePlayerTwo",
	Nickname: "p2",
	FullName: "Puzzle Player Two",
	Rating:   "0",
	IsBot:    true,
	First:    false,
}

const epsilon = 1e-4

func WithinEpsilon(a, b float64) bool {
	return math.Abs(float64(a-b)) < float64(epsilon)
}
