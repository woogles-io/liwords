package user

import (
	"context"
	"errors"
	"net/mail"
	"strings"
)

var (
	ErrInvalidEmail = errors.New("please use a valid email address")
	ErrEmailTaken   = errors.New("that email address is already used by another account")
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
