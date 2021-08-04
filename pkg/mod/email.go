package mod

import (
	"bytes"
	_ "embed"
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
	IsDeletion        bool
}

const TermsOfServiceURL = "https://woogles.io/terms"
const AddressToContact = "conduct@woogles.io"
const EmailTemplateName = "email"

//go:embed email_template
var EmailTemplate string

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
		IsCheater:         emailType == ms.EmailType_CHEATING,
		IsDeletion:        emailType == ms.EmailType_DELETION})
	if err != nil {
		return "", err
	}
	return emailContentBuffer.String(), nil

}
