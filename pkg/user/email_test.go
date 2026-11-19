package user

import (
	"context"
	"errors"
	"testing"

	"github.com/matryer/is"
)

// checkStore answers CheckEmailAllowed's lookups from fixed sets, or fails
// them all. Any other method of Store panics.
type checkStore struct {
	Store
	taken, suspended map[string]bool
	fail             bool
}

func (s *checkStore) IsEmailKeyTaken(ctx context.Context, uuid string, email string) (bool, error) {
	if s.fail {
		return false, errors.New("down")
	}
	return s.taken[email], nil
}

func (s *checkStore) IsEmailOfSuspendedAccount(ctx context.Context, email string) (bool, error) {
	if s.fail {
		return false, errors.New("down")
	}
	return s.suspended[email], nil
}

type fakeDisposable struct{ fail bool }

func (d fakeDisposable) Domains(ctx context.Context) (map[string]struct{}, error) {
	if d.fail {
		return nil, errors.New("list unavailable")
	}
	return map[string]struct{}{"mailinator.com": {}}, nil
}

func TestCheckEmailAllowed(t *testing.T) {
	ctx := context.Background()
	store := &checkStore{
		taken:     map[string]bool{"taken@example.com": true, "both@mailinator.com": true},
		suspended: map[string]bool{"banned@example.com": true, "banned@mailinator.com": true},
	}
	for _, tc := range []struct {
		email      string
		disposable DisposableDomains
		want       error
	}{
		{"fine@example.com", fakeDisposable{}, nil},
		{"taken@example.com", fakeDisposable{}, ErrEmailTaken},
		{"x@mailinator.com", fakeDisposable{}, ErrDisposableEmail},
		{"x@sub.mailinator.com", fakeDisposable{}, ErrDisposableEmail},
		{"banned@example.com", fakeDisposable{}, ErrEmailNotAllowed},
		// Taken comes first, then disposable, then suspended.
		{"both@mailinator.com", fakeDisposable{}, ErrEmailTaken},
		{"banned@mailinator.com", fakeDisposable{}, ErrDisposableEmail},
		// With no list, or a list that can't be had, no domain is refused.
		{"x@mailinator.com", nil, nil},
		{"x@mailinator.com", fakeDisposable{fail: true}, nil},
	} {
		t.Run(tc.email, func(t *testing.T) {
			is := is.New(t)
			is.Equal(CheckEmailAllowed(ctx, store, tc.disposable, "", tc.email), tc.want)
		})
	}

	// When the database lookups fail, the address is let through.
	is := is.New(t)
	is.NoErr(CheckEmailAllowed(ctx, &checkStore{fail: true}, fakeDisposable{}, "", "taken@example.com"))
}
