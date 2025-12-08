package registration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	goaway "github.com/TwiN/go-away"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/auth"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/emailer"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/user"
	"github.com/woogles-io/liwords/pkg/utilities"
)

const (
	// VerificationTokenExpiration is how long a verification token is valid
	VerificationTokenExpiration = 48 * time.Hour
)

const VerificationEmailTemplate = `
Dear %s,

Welcome to Woogles.io! To complete your registration, please verify your email address by clicking the link below:

%s

This link will expire in 48 hours.

If you didn't create this account, you can safely ignore this email.

Love,

The Woogles.io team
`

// generateVerificationToken creates a random token for email verification
func generateVerificationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// RegisterUser registers a user.
func RegisterUser(ctx context.Context, username string, password string, email string,
	firstName string, lastName string, birthDate string, countryCode string,
	userStore user.Store, bot bool, argonConfig config.ArgonConfig, emailDebugMode bool, skipEmailVerification bool) error {
	// username = strings.Rep
	if len(username) < 3 || len(username) > 20 {
		return errors.New("username must be between 3 and 20 letters in length")
	}
	if strings.IndexFunc(username, func(c rune) bool {
		return !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '.' || c == '_')
	}) != -1 {
		return errors.New("username can only contain letters, digits, period, hyphen or underscore")
	}
	// Should we have other unacceptable usernames?
	if strings.EqualFold(username, "anonymous") ||
		strings.EqualFold(username, utilities.CensoredUsername) ||
		strings.EqualFold(username, utilities.AnotherCensoredUsername) ||
		strings.EqualFold(username, utilities.YetAnotherCensoredUsername) {
		return errors.New("username is not acceptable")
	}
	if strings.HasPrefix(username, "-") || strings.HasPrefix(username, ".") || strings.HasPrefix(username, "_") {
		return errors.New("username must start with a number or a letter")
	}
	if strings.HasSuffix(username, "-") || strings.HasSuffix(username, ".") || strings.HasSuffix(username, "_") {
		return errors.New("username must end with a number or a letter")
	}
	if strings.HasSuffix(strings.ToLower(username), "bot") {
		return errors.New("username is not acceptable")
	}
	if goaway.IsProfane(username) {
		return errors.New("username is not acceptable")
	}

	if len(password) < 8 {
		return errors.New("your new password is too short, use 8 or more characters")
	}
	if len(email) < 3 {
		return errors.New("please use a valid email address")
	}
	email = strings.TrimSpace(email)

	config := auth.NewPasswordConfig(argonConfig.Time, argonConfig.Memory, argonConfig.Threads, argonConfig.Keylen)
	hashPass, err := auth.GeneratePassword(config, password)
	if err != nil {
		return err
	}

	// Generate verification token for non-bot, non-skip users
	var verificationToken string
	var verificationExpiresAt time.Time
	verified := bot || skipEmailVerification // Bots and dev mode are auto-verified

	if !verified {
		verificationToken, err = generateVerificationToken()
		if err != nil {
			return fmt.Errorf("failed to generate verification token: %w", err)
		}
		verificationExpiresAt = time.Now().Add(VerificationTokenExpiration)
	}

	err = userStore.New(ctx, &entity.User{
		Username: username,
		Password: hashPass,
		Email:    email,
		Profile: &entity.Profile{
			FirstName:   firstName,
			LastName:    lastName,
			BirthDate:   birthDate,
			CountryCode: countryCode,
		},
		IsBot:                 bot,
		Verified:              verified,
		VerificationToken:     verificationToken,
		VerificationExpiresAt: verificationExpiresAt,
	})
	if err != nil {
		if err, ok := err.(*pgconn.PgError); ok {
			// https://www.postgresql.org/docs/current/errcodes-appendix.html
			if err.Code == "23505" {
				if err.ConstraintName == "username_idx" {
					return errors.New("That username has already been signed up, please log in")
				} else if err.ConstraintName == "email_idx" {
					return errors.New("That email address has already been signed up, please log in with your existing username")
				}
			}
		}
		return err
	}

	// Send verification email for non-bot, non-skip users
	if !verified {
		verificationURL := fmt.Sprintf("https://woogles.io/verify-email?token=%s", verificationToken)
		emailBody := fmt.Sprintf(VerificationEmailTemplate, username, verificationURL)

		_, err = emailer.SendSimpleMessage(emailDebugMode, email, "Verify your Woogles.io email", emailBody)
		if err != nil {
			log.Error().Err(err).Str("email", email).Str("emailBody", emailBody).Msg("failed to send verification email")
			// Don't fail registration if email sending fails - user can resend later
		} else {
			log.Info().Str("email", email).Str("username", username).Msg("verification email sent")
		}
	}

	return nil
}

// VerifyUserEmail verifies a user's email using the token
func VerifyUserEmail(ctx context.Context, token string, userStore user.Store) error {
	if token == "" {
		return errors.New("verification token is required")
	}

	// Get user by verification token
	u, err := userStore.GetByVerificationToken(ctx, token)
	if err != nil {
		return errors.New("invalid or expired verification token")
	}

	// Check if already verified - treat as success (idempotent)
	if u.Verified {
		log.Info().Str("username", u.Username).Str("email", u.Email).Msg("user-email-already-verified")
		return nil // Success - already verified
	}

	// Check if token expired
	if time.Now().After(u.VerificationExpiresAt) {
		return errors.New("verification token has expired. Please request a new one")
	}

	// Mark user as verified and clear token
	err = userStore.SetEmailVerified(ctx, u.UUID, true)
	if err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	log.Info().Str("username", u.Username).Str("email", u.Email).Msg("user-email-verified")
	return nil
}

// ResendVerificationEmail generates a new token and resends the verification email
func ResendVerificationEmail(ctx context.Context, email string, userStore user.Store, emailDebugMode bool) error {
	if email == "" {
		return errors.New("email is required")
	}

	email = strings.TrimSpace(email)

	// Get user by email
	u, err := userStore.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if email exists or not for security
		return errors.New("if this email is registered and unverified, a verification email will be sent")
	}

	// Check if already verified
	if u.Verified {
		return errors.New("email is already verified. Please log in")
	}

	// Generate new verification token
	verificationToken, err := generateVerificationToken()
	if err != nil {
		return fmt.Errorf("failed to generate verification token: %w", err)
	}
	verificationExpiresAt := time.Now().Add(VerificationTokenExpiration)

	// Update user with new token
	err = userStore.UpdateVerificationToken(ctx, u.UUID, verificationToken, verificationExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to update verification token: %w", err)
	}

	// Send verification email
	verificationURL := fmt.Sprintf("https://woogles.io/verify-email?token=%s", verificationToken)
	emailBody := fmt.Sprintf(VerificationEmailTemplate, u.Username, verificationURL)

	_, err = emailer.SendSimpleMessage(emailDebugMode, email, "Verify your Woogles.io email", emailBody)
	if err != nil {
		log.Error().Err(err).Str("email", email).Str("emailBody", emailBody).Msg("failed to send verification email")
		return errors.New("failed to send verification email. Please try again later")
	}

	log.Info().Str("email", email).Str("username", u.Username).Msg("verification-email-resent")
	return nil
}
