package entity

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/domino14/macondo/board"
	"github.com/domino14/macondo/game"
	"github.com/lithammer/shortuuid/v4"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type SoughtGameType int

const (
	TypeSeek SoughtGameType = iota
	TypeNone
)

type SoughtGame struct {
	SeekRequest *pb.SeekRequest
}

func NewSoughtGame(seekRequest *pb.SeekRequest) *SoughtGame {
	sg := &SoughtGame{
		SeekRequest: seekRequest,
	}

	sg.SeekRequest.GameRequest.RequestId = shortuuid.New()
	// Note that even though sought games are never rematches,
	// we must set the OriginalRequestId since they can start
	// a streak of rematches, from which an OriginalRequestId
	// is needed.
	if sg.SeekRequest.GameRequest.OriginalRequestId == "" {
		sg.SeekRequest.GameRequest.OriginalRequestId =
			sg.SeekRequest.GameRequest.RequestId
	}
	return sg
}

func (sg *SoughtGame) ID() (string, error) {
	sr, err := getSeekRequest(sg)
	if err != nil {
		return "", err
	}
	if sr.GameRequest == nil {
		return "", errors.New("nil game request on seek request")
	}
	return sr.GameRequest.RequestId, nil
}

func (sg *SoughtGame) SeekerConnID() (string, error) {
	sr, err := getSeekRequest(sg)
	if err != nil {
		return "", err
	}
	return sr.SeekerConnectionId, nil
}

func (sg *SoughtGame) ReceiverConnID() (string, error) {
	sr, err := getSeekRequest(sg)
	if err != nil {
		return "", err
	}
	return sr.ReceiverConnectionId, nil
}

func (sg *SoughtGame) SeekerUserID() (string, error) {
	sr, err := getSeekRequest(sg)
	if err != nil {
		return "", err
	}
	if sr.User == nil {
		return "", errors.New("nil user for seek request")
	}
	return sr.User.UserId, nil
}

func (sg *SoughtGame) ReceiverUserID() (string, error) {
	sr, err := getSeekRequest(sg)
	if err != nil {
		return "", err
	}
	if sr.ReceivingUser == nil {
		return "", errors.New("nil receiving user on seek request")
	}
	return sr.ReceivingUser.UserId, nil
}

func (sg *SoughtGame) ReceiverDisplayName() (string, error) {
	sr, err := getSeekRequest(sg)
	if err != nil {
		return "", err
	}
	if sr.ReceivingUser == nil {
		return "", errors.New("nil receiving user on seek request")
	}
	return sr.ReceivingUser.DisplayName, nil
}

func (sg *SoughtGame) ReceiverIsPermanent() (bool, error) {
	sr, err := getSeekRequest(sg)
	if err != nil {
		return false, err
	}
	return sr.ReceiverIsPermanent, nil
}

func getSeekRequest(sg *SoughtGame) (*pb.SeekRequest, error) {
	if sg.SeekRequest != nil {
		return sg.SeekRequest, nil
	}
	return nil, errors.New("nil seek request on sought game")
}

func isEnglish(lexicon string) bool {
	return strings.HasPrefix(lexicon, "NWL") ||
		strings.HasPrefix(lexicon, "CSW") ||
		strings.HasPrefix(lexicon, "ECWL")
}

// ValidateGameRequest validates a generic game request.
func ValidateGameRequest(ctx context.Context, req *pb.GameRequest) error {
	if req == nil {
		return errors.New("game request is missing")
	}

	// Correspondence games have different time limits
	if req.GameMode == pb.GameMode_CORRESPONDENCE {
		if req.InitialTimeSeconds < 3600 { // 1 hour minimum
			return errors.New("correspondence games must have at least 1 hour initial time")
		}
		if req.InitialTimeSeconds > 432000 { // 5 days maximum
			return errors.New("correspondence games cannot exceed 5 days per turn")
		}
		// For correspondence, increment should equal initial time (reset logic)
		// and max overtime should be 0
		if req.MaxOvertimeMinutes != 0 {
			return errors.New("correspondence games cannot use max overtime")
		}
	} else {
		// Real-time game validation
		if req.InitialTimeSeconds < 15 {
			return errors.New("the initial time must be at least 15 seconds")
		}
		if req.MaxOvertimeMinutes < 0 || req.MaxOvertimeMinutes > 10 {
			return errors.New("overtime minutes must be between 0 and 10")
		}
		if req.IncrementSeconds < 0 {
			return errors.New("you cannot have a negative time increment")
		}
		if req.IncrementSeconds > 60 {
			return errors.New("time increment must be at most 60 seconds")
		}
		if req.MaxOvertimeMinutes > 0 && req.IncrementSeconds > 0 {
			return errors.New("you can have increments or max overtime, but not both")
		}
	}

	// Modify the game request if the variant calls for it.
	if req.Rules.VariantName == game.VarClassicSuper {
		if !isEnglish(req.Lexicon) {
			return errors.New("non-english lexica not supported for this variant")
		}
		req.Rules.BoardLayoutName = board.SuperCrosswordGameLayout
		req.Rules.LetterDistributionName = "english_super"
	}

	for _, lex := range AllowedNewGameLexica {
		if req.Lexicon == lex {
			return nil
		}
	}

	return fmt.Errorf("%s is not a supported lexicon", req.Lexicon)
}

func (sg *SoughtGame) Value() (driver.Value, error) {
	return json.Marshal(sg)
}

func (sg *SoughtGame) Scan(value interface{}) error {
	fmt.Println("tryna scan", value)
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for sought game")
	}

	return json.Unmarshal(b, &sg)
}
