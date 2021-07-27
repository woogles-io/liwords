package mod

import (
	"bytes"
	"text/template"
	"time"

	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EmailInfo struct {
	Username          string
	TermsOfServiceURL string
	Action            string
	StartTime         string
	EndTime           string
	AddressToContact  string
	IsCheater         bool
}

const TermsOfServiceURL = "https://woogles.io/terms"
const AddressToContact = "conduct@woogles.io"
const EmailTemplateName = "email"

const EmailTemplate = `
Dear Woogles.io user,

The account associated with {{.Username}} has violated the Woogles Terms of Service, which can be found at {{.TermsOfServiceURL}}.{{if .IsCheater}} You are receiving this email because the anti-cheating detection algorithm has flagged the account's play as suspicious. Upon further review of the evidence, it was determined that the account has been cheating on our platform.{{- end}} As such, the following action has been taken against the account:

Action:     {{.Action}}
Start Time: {{.StartTime}}
{{if .EndTime}}End Time:   {{.EndTime}}{{- end}}
{{if .IsCheater}}
All cheating determinations are final and non-negotiable. However, the length of the action can be appealed. We will not consider or reply to an appeal unless it includes the following:

1. An admission of cheating.
2. An explanation of when and how you cheated on Woogles.
3. A promise to not cheat on our platform again.

{{- end}}

If you think this was done in error or would like to appeal, contact {{.AddressToContact}}. Do not reply directly to this email. Contacting Woogles team members privately (by email or on social media) may result in a lengthier ban.

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

func instantiateEmail(username, actionTaken, note string, starttime, endtime *timestamppb.Timestamp, emailType ms.EmailType) (string, error) {

	golangStartTime, err := ptypes.Timestamp(starttime)
	if err != nil {
		return "", err
	}
	startTimeString := golangStartTime.UTC().Format(time.UnixDate)

	golangEndTime, err := ptypes.Timestamp(endtime)
	var endTimeString string
	if err == nil {
		endTimeString = golangEndTime.UTC().Format(time.UnixDate)
	}

	emailTemplate, err := template.New(EmailTemplateName).Parse(EmailTemplate)
	if err != nil {
		return "", err
	}

	emailContentBuffer := &bytes.Buffer{}
	err = emailTemplate.Execute(emailContentBuffer, &EmailInfo{Username: username,
		TermsOfServiceURL: TermsOfServiceURL,
		Action:            actionTaken,
		StartTime:         startTimeString,
		EndTime:           endTimeString,
		AddressToContact:  AddressToContact,
		IsCheater:         emailType == ms.EmailType_CHEATING})
	if err != nil {
		return "", err
	}
	return emailContentBuffer.String(), nil

}
