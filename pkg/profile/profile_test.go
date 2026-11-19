package profile

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/user"

	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

// personalInfoStore holds one user for UpdatePersonalInfo and keeps what
// SetPersonalInfo saves and what email changes are asked for. Any other
// method of user.Store panics.
type personalInfoStore struct {
	user.Store
	u        *entity.User
	saved    []string
	requests [][]string
}

func (s *personalInfoStore) Get(ctx context.Context, username string) (*entity.User, error) {
	return s.u, nil
}

// RequestEmailChange keeps the change asked for. The email itself does not
// change until the link sent to the new address is followed.
func (s *personalInfoStore) RequestEmailChange(ctx context.Context, uuid string, email string,
	token string, expiresAt time.Time) error {
	if token == "" || !expiresAt.After(time.Now()) {
		return errors.New("no token or already expired")
	}
	s.requests = append(s.requests, []string{uuid, email})
	return nil
}

// IsEmailOfSuspendedAccount knows only suspended@example.com.
func (s *personalInfoStore) IsEmailOfSuspendedAccount(ctx context.Context, email string) (bool, error) {
	return email == "suspended@example.com", nil
}

// GetByEmail finds only taken@example.com, which another account has.
func (s *personalInfoStore) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	if strings.EqualFold(email, "taken@example.com") {
		return &entity.User{UUID: "uuid-2", Email: "taken@example.com"}, nil
	}
	return nil, errors.New("user not found")
}

func (s *personalInfoStore) SetPersonalInfo(ctx context.Context, uuid string, firstName string,
	lastName string, birthDate string, countryCode string, about string) error {
	s.saved = []string{firstName, lastName, birthDate, countryCode, about}
	return nil
}

// A new email takes the current password; other edits do not.
func TestUpdatePersonalInfoPassword(t *testing.T) {
	hash, err := auth.GeneratePassword(auth.NewPasswordConfig(1, 1024, 1, 32), "correct horse")
	if err != nil {
		t.Fatal(err)
	}
	ctx := apiserver.PlaceInContext(context.Background(),
		&entity.Session{Username: "someone", UserUUID: "uuid-1"})

	for _, tc := range []struct {
		name     string
		email    string
		password string
		refused  bool
	}{
		{"email changed, no password", "new@example.com", "", true},
		{"email changed, wrong password", "new@example.com", "wrong horse", true},
		{"email changed, right password", "new@example.com", "correct horse", false},
		{"email kept, no password", "old@example.com", "", false},
		{"email taken by another account", "Taken@example.com", "correct horse", true},
		{"email not an address", "not an address", "correct horse", true},
		{"email with a display name", "New <new@example.com>", "correct horse", true},
		{"email with space around it", " new@example.com ", "correct horse", false},
		{"email of a suspended account", "suspended@example.com", "correct horse", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			store := &personalInfoStore{u: &entity.User{
				UUID: "uuid-1", Username: "someone", Email: "old@example.com", Password: hash}}
			ps := NewProfileService(store, nil, nil, true)

			resp, err := ps.UpdatePersonalInfo(ctx, connect.NewRequest(&pb.UpdatePersonalInfoRequest{
				Email: tc.email, Password: tc.password, About: "new bio"}))
			if tc.refused {
				is.Equal(connect.CodeOf(err), connect.CodeInvalidArgument)
				is.True(store.saved == nil) // nothing saved
				is.True(store.requests == nil)
				return
			}
			is.NoErr(err)
			is.Equal(store.saved[4], "new bio")
			// A changed email, and only a changed one, is asked for, to be
			// confirmed from the new address, and the response says so.
			email := strings.TrimSpace(tc.email)
			if email == "old@example.com" {
				is.True(store.requests == nil)
				is.Equal(resp.Msg.PendingEmail, "")
			} else {
				is.Equal(store.requests, [][]string{{"uuid-1", email}})
				is.Equal(resp.Msg.PendingEmail, email)
			}
		})
	}
}
