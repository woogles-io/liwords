package mod

import (
	"fmt"
	"strings"

	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const TermsOfServiceURL = "https://woogles.io/terms"
const AddressToContact = "conduct@woogles.io"
const CheatingKeyword = "cheat"

const DefaultEmailTemplate = `
Dear Woogles.io user,

The account associated with %s has violated the our <a href="%s">Terms of Service</a>. The following action was taken against your account:

Action:     %s
Start Time: %s
End Time:   %s

If you think this was an error, please contact %s.

The Woogles.io team
`

const CheatingEmailTemplate = `
Dear Woogles.io user,

The account associated with %s has violated the our <a href="%s">Terms of Service</a>. You are receiving this email because our anti-cheating detection algorithm has flagged your play as suspicious. Upon further review of the evidence, we have determined that you have been cheating on our platform. As such, we have taken the following action against your account:

Action:     %s
Start Time: %s
End Time:   %s

If your suspension is not permanent, you will be allowed back on woogles.io but you will not be allowed to play rated games until we feel confident that you will not cheat again (this includes any club or tournament that rates their games.) All cheating determinations are final and non-negotiable. However, you may appeal the length of your suspension. We will not consider or reply to an appeal unless it includes the following:

1. An admission of cheating.
2. An explanation of when and how you cheated on Woogles (the more details you give the better a case we can make for being lenient.)
3. A promise to not cheat on our platform again.

Please send any appeals to %s. Do not reply directly to this email. Contacting Woogles team members privately (by email or on social media) may result in a lengthier ban.

Sincerely,
The Woogles Team
`

var ModActionEmailMap = map[ms.ModActionType]string{
	ms.ModActionType_MUTE:                    "Disable Chat",
	ms.ModActionType_SUSPEND_ACCOUNT:         "Account Suspension",
	ms.ModActionType_SUSPEND_RATED_GAMES:     "Suspend Rated Games",
	ms.ModActionType_SUSPEND_GAMES:           "Suspend Games",
	ms.ModActionType_RESET_RATINGS:           "Reset Ratings",
	ms.ModActionType_RESET_STATS:             "Reset Statistics",
	ms.ModActionType_RESET_STATS_AND_RATINGS: "Reset Ratings and Statistics",
}

func instantiateEmail(username, actionTaken, note string, starttime, endtime *timestamppb.Timestamp) (string, error) {

	golangStartTime, err := ptypes.Timestamp(starttime)
	if err != nil {
		return "", err
	}
	startTimeString := golangStartTime.UTC().String()

	golangEndTime, err := ptypes.Timestamp(endtime)
	var endTimeString string
	if err != nil {
		endTimeString = "None (this action is permanent)"
		return "", err
	} else {
		endTimeString = golangEndTime.UTC().String()
	}

	emailTemplate := DefaultEmailTemplate

	if strings.Contains(strings.ToLower(note), CheatingKeyword) {
		emailTemplate = CheatingEmailTemplate
	}

	return fmt.Sprint(emailTemplate, TermsOfServiceURL, username, actionTaken, startTimeString, endTimeString, AddressToContact), nil
}
