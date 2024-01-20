package common

import (
	"math"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const DefaultLexicon = "CSW21"
const DefaultLetterDistribution = "english"
const DefaultVariantName = "classic"

// var DefaultMacondoConfig = macondoconfig.DefaultConfig()

// macondoconfig.Config{
// 	LexiconPath:               os.Getenv("LEXICON_PATH"),
// 	LetterDistributionPath:    os.Getenv("LETTER_DISTRIBUTION_PATH"),
// 	DefaultLexicon:            DefaultLexicon,
// 	DefaultLetterDistribution: DefaultLetterDistribution,
// 	DataPath:                  os.Getenv("DATA_PATH"),
// }

var DefaultGameReq = &ipc.GameRequest{Lexicon: DefaultLexicon,
	Rules: &ipc.GameRules{BoardLayoutName: entity.CrosswordGame,
		LetterDistributionName: DefaultLetterDistribution,
		VariantName:            DefaultVariantName},
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

var DefaultSeeker = ipc.MatchUser{
	UserId:         "seeker_id",
	RelevantRating: "1500",
	IsAnonymous:    false,
	DisplayName:    "seeker",
}

var DefaultReceiver = ipc.MatchUser{
	UserId:         "receiver_id",
	RelevantRating: "2000",
	IsAnonymous:    false,
	DisplayName:    "receiver",
}

var DefaultSeekRequest = ipc.SeekRequest{
	GameRequest:          DefaultGameReq,
	User:                 &DefaultSeeker,
	MinimumRatingRange:   0,
	MaximumRatingRange:   3000,
	SeekerConnectionId:   "seeker_conn_id",
	ReceivingUser:        &DefaultReceiver,
	UserState:            ipc.SeekState_ABSENT,
	ReceiverState:        ipc.SeekState_ABSENT,
	ReceiverConnectionId: "receiver_conn_id",
}

const epsilon = 1e-4

func WithinEpsilon(a, b float64) bool {
	return math.Abs(float64(a-b)) < float64(epsilon)
}
