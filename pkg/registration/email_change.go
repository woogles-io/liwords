package registration

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/rs/zerolog"

	"github.com/woogles-io/liwords/pkg/emailer"
	"github.com/woogles-io/liwords/pkg/user"
)

// EmailChangeExpiration is how long the link sent to a new address works.
const EmailChangeExpiration = 24 * time.Hour

const emailChangeTemplate = `Hi %s,

You asked to change the email address of your Woogles.io account to this one. To confirm, click the link below:

%s

This link will expire in 24 hours. Until you click it, your account keeps its current email address.

If you didn't ask for this, you can safely ignore this email.

The Woogles.io team
`

// RequestEmailChange starts changing a user's email: it keeps the change as
// pending and sends a link to the new address, which must already have
// passed user.CheckNewEmail. The email changes only when the link is
// followed, so an account's address is always one its owner can read.
func RequestEmailChange(ctx context.Context, us user.Store, username string, uuid string,
	email string, emailDebugMode bool) error {
	if err := user.CheckEmailAllowed(ctx, us, email); err != nil {
		return err
	}
	token, err := generateVerificationToken()
	if err != nil {
		return fmt.Errorf("failed to generate email change token: %w", err)
	}
	if err := us.RequestEmailChange(ctx, uuid, email, token, time.Now().Add(EmailChangeExpiration)); err != nil {
		return err
	}
	link := "https://woogles.io/confirm-email-change?token=" + url.QueryEscape(token)
	_, err = emailer.SendSimpleMessageCtx(ctx, emailDebugMode, email,
		"Confirm your new Woogles.io email", fmt.Sprintf(emailChangeTemplate, username, link))
	if err != nil {
		// Unlike at registration there is nothing to fall back on, so say so;
		// asking again sends a new link.
		return fmt.Errorf("failed to send the confirmation email: %w", err)
	}
	zerolog.Ctx(ctx).Info().Str("userID", uuid).Msg("email-change-requested")
	return nil
}

// ConfirmEmailChange makes the pending email change with this token. The
// address is checked again, since an account may have been suspended, or
// taken the address, since the change was asked for.
func ConfirmEmailChange(ctx context.Context, us user.Store, token string) (string, error) {
	if token == "" {
		return "", user.ErrEmailChangeNotFound
	}
	uuid, email, err := us.ConfirmEmailChange(ctx, token, func(email string) error {
		return user.CheckEmailAllowed(ctx, us, email)
	})
	if err != nil {
		return "", err
	}
	zerolog.Ctx(ctx).Info().Str("userID", uuid).Msg("email-change-confirmed")
	return email, nil
}
