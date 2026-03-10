package emailer

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/rs/zerolog/log"
)

const (
	region        = "us-east-2"
	senderAddress = "Woogles <noreply@woogles.io>"
	charset       = "UTF-8"
)

// SendSimpleMessage sends an email via Amazon SES.
// The debugMode parameter controls whether to actually send or just log.
// The bccAddress parameter optionally adds a BCC recipient.
func SendSimpleMessage(debugMode bool, recipient, subject, body string, bccAddress ...string) (string, error) {
	if debugMode {
		bccInfo := ""
		if len(bccAddress) > 0 {
			bccInfo = fmt.Sprintf("\nBCC: %s", bccAddress[0])
		}
		fmt.Printf("=== EMAIL DEBUG ===\nTo: %s%s\nSubject: %s\nBody:\n%s\n=== END EMAIL ===\n", recipient, bccInfo, subject, body)
		return "debug-mode", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := sesv2.NewFromConfig(cfg)

	destination := &types.Destination{
		ToAddresses: []string{recipient},
	}
	if len(bccAddress) > 0 && bccAddress[0] != "" {
		destination.BccAddresses = []string{bccAddress[0]}
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(senderAddress),
		Destination:      destination,
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    aws.String(subject),
					Charset: aws.String(charset),
				},
				Body: &types.Body{
					Text: &types.Content{
						Data:    aws.String(body),
						Charset: aws.String(charset),
					},
				},
			},
		},
	}

	result, err := client.SendEmail(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to send email via SES: %w", err)
	}
	log.Debug().Str("message_id", aws.ToString(result.MessageId)).
		Str("recipient", recipient).
		Str("subject", subject).
		Msg("Email sent via SES")

	return aws.ToString(result.MessageId), nil
}
