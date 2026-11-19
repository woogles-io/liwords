package user

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/rs/zerolog"
)

var (
	ErrInvalidEmail = errors.New("please use a valid email address")
	ErrEmailTaken   = errors.New("that email address is already used by another account")
	// ErrEmailNotAllowed does not say why, so as not to tell anyone which
	// addresses belong to suspended accounts.
	ErrEmailNotAllowed     = errors.New("that email address cannot be used, please use another")
	ErrEmailChangeNotFound = errors.New("this email change link is invalid or has expired")
)

// CheckNewEmail checks an email address that the user with the given UUID is
// to have: it must be a bare address, and no other account may have it in any
// case. It returns the address with surrounding space trimmed.
func CheckNewEmail(ctx context.Context, us Store, uuid string, email string) (string, error) {
	email = strings.TrimSpace(email)
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", ErrInvalidEmail
	}
	// A lookup error is taken to mean no account has it; the unique index on
	// the email column still refuses a duplicate.
	if other, err := us.GetByEmail(ctx, email); err == nil && other.UUID != uuid {
		return "", ErrEmailTaken
	}
	return email, nil
}

// EmailChange is a change of a user's email that was made.
type EmailChange struct {
	UserUUID string
	OldEmail string
	NewEmail string
}

// CheckEmailAllowed refuses, with ErrEmailNotAllowed, an address that a
// suspended account has or had, so that a suspended user cannot reuse it for
// a new account or move it onto another one. If the check itself fails, the
// address is allowed: this guards against evasion and should not block
// sign-ups while the database is struggling.
func CheckEmailAllowed(ctx context.Context, us Store, email string) error {
	suspended, err := us.IsEmailOfSuspendedAccount(ctx, email)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("check-email-of-suspended-account")
		return nil
	}
	if suspended {
		return ErrEmailNotAllowed
	}
	return nil
}
