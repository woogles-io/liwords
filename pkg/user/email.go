package user

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/rs/zerolog"

	"github.com/woogles-io/liwords/pkg/disposableemail"
)

var (
	ErrInvalidEmail = errors.New("please use a valid email address")
	ErrEmailTaken   = errors.New("that email address is already used by another account")
	// ErrEmailNotAllowed does not say why, so as not to tell anyone which
	// addresses belong to suspended accounts.
	ErrEmailNotAllowed     = errors.New("that email address cannot be used, please use another")
	ErrDisposableEmail     = errors.New("please use a permanent email address, not a disposable one")
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

// DisposableDomains gives the known disposable-email domains;
// *disposableemail.List is one.
type DisposableDomains interface {
	Domains(ctx context.Context) (map[string]struct{}, error)
}

// CheckEmailAllowed checks an address that the account with the given UUID
// (empty for a new account) is to have, beyond its form and exact
// uniqueness. It refuses, in this order:
//   - with ErrEmailTaken, an address that reaches the same inbox as another
//     account's, such as name+x@ when name@ has an account;
//   - with ErrDisposableEmail, one on a disposable domain, if disposable is
//     not nil;
//   - with ErrEmailNotAllowed, one that a suspended account has or had, so a
//     suspended user cannot reuse it for a new account or move it onto
//     another one.
//
// If a lookup fails, that check lets the address through: these checks guard
// against evasion and should not block sign-ups while something is down.
func CheckEmailAllowed(ctx context.Context, us Store, disposable DisposableDomains,
	uuid string, email string) error {
	log := zerolog.Ctx(ctx)
	if taken, err := us.IsEmailKeyTaken(ctx, uuid, email); err != nil {
		log.Err(err).Msg("check-email-key-taken")
	} else if taken {
		return ErrEmailTaken
	}
	if disposable != nil {
		if domains, err := disposable.Domains(ctx); err != nil {
			log.Err(err).Msg("check-disposable-email")
		} else if disposableemail.Listed(domains, email) {
			return ErrDisposableEmail
		}
	}
	if suspended, err := us.IsEmailOfSuspendedAccount(ctx, email); err != nil {
		log.Err(err).Msg("check-email-of-suspended-account")
	} else if suspended {
		return ErrEmailNotAllowed
	}
	return nil
}
