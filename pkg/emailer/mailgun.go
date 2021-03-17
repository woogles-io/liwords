package emailer

import (
	"context"
	"time"

	"github.com/mailgun/mailgun-go/v4"
)

// Send emails using mailgun.

const domain = "mg.woogles.io"
const WooglesAdministratorAddress = "woogles@woogles.io"

func SendSimpleMessage(apiKey, recipient, subject, body string) (string, error) {
	mg := mailgun.NewMailgun(domain, apiKey)
	m := mg.NewMessage(
		"Woogles <woogles@"+domain+">",
		subject,
		body,
		recipient,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, id, err := mg.Send(ctx, m)
	return id, err
}
