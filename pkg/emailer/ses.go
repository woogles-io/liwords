package emailer

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

const (
	region        = "us-east-2"
	senderAddress = "Woogles <noreply@woogles.io>"
	charset       = "UTF-8"
)

// SendSimpleMessage sends an email via Amazon SES.
// The debugMode parameter controls whether to actually send or just log.
func SendSimpleMessage(debugMode bool, recipient, subject, body string) (string, error) {
	if debugMode {
		fmt.Printf("=== EMAIL DEBUG ===\nTo: %s\nSubject: %s\nBody:\n%s\n=== END EMAIL ===\n", recipient, subject, body)
		return "debug-mode", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := sesv2.NewFromConfig(cfg)

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(senderAddress),
		Destination: &types.Destination{
			ToAddresses: []string{recipient},
		},
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

	return aws.ToString(result.MessageId), nil
}
