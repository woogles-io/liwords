package registration

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/user"
)

// confirmStore confirms every token as the change it holds. Any other method
// of user.Store panics.
type confirmStore struct {
	user.Store
	change user.EmailChange
}

func (s *confirmStore) ConfirmEmailChange(ctx context.Context, token string,
	check func(string) error) (user.EmailChange, error) {
	if err := check(s.change.NewEmail); err != nil {
		return user.EmailChange{}, err
	}
	return s.change, nil
}

func (s *confirmStore) IsEmailOfSuspendedAccount(ctx context.Context, email string) (bool, error) {
	return false, nil
}

func (s *confirmStore) Username(ctx context.Context, uuid string) (string, error) {
	return "someone", nil
}

// captureStdout returns what f prints, which in email debug mode is every
// email it sends.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = stdout }()
	f()
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// A confirmed change is told to the old address, naming the new one.
func TestConfirmEmailChangeNotifiesOldEmail(t *testing.T) {
	is := is.New(t)
	store := &confirmStore{change: user.EmailChange{
		UserUUID: "uuid-1", OldEmail: "old@example.com", NewEmail: "new@example.com"}}
	var email string
	var err error
	out := captureStdout(t, func() {
		email, err = ConfirmEmailChange(context.Background(), store, "token", true)
	})
	is.NoErr(err)
	is.Equal(email, "new@example.com")
	is.Equal(strings.Count(out, "=== EMAIL DEBUG ==="), 1)
	is.True(strings.Contains(out, "To: old@example.com\n"))
	is.True(strings.Contains(out, "Hi someone,"))
	is.True(strings.Contains(out, "to new@example.com."))
}

// Confirming the address the user already has sends nothing.
func TestConfirmEmailChangeSameEmail(t *testing.T) {
	is := is.New(t)
	store := &confirmStore{change: user.EmailChange{
		UserUUID: "uuid-1", OldEmail: "same@example.com", NewEmail: "same@example.com"}}
	var err error
	out := captureStdout(t, func() {
		_, err = ConfirmEmailChange(context.Background(), store, "token", true)
	})
	is.NoErr(err)
	is.Equal(out, "")
}

// An empty token is not looked up.
func TestConfirmEmailChangeNoToken(t *testing.T) {
	is := is.New(t)
	_, err := ConfirmEmailChange(context.Background(), &confirmStore{}, "", true)
	is.Equal(err, user.ErrEmailChangeNotFound)
}
