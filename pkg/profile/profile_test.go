package profile

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/user"

	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

// personalInfoStore holds one user for UpdatePersonalInfo and keeps what
// SetPersonalInfo saves. Any other method of user.Store panics.
type personalInfoStore struct {
	user.Store
	u     *entity.User
	saved []string
}

func (s *personalInfoStore) Get(ctx context.Context, username string) (*entity.User, error) {
	return s.u, nil
}

func (s *personalInfoStore) SetPersonalInfo(ctx context.Context, uuid string, email string, firstName string,
	lastName string, birthDate string, countryCode string, about string) error {
	s.saved = []string{email, firstName, lastName, birthDate, countryCode, about}
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			store := &personalInfoStore{u: &entity.User{
				UUID: "uuid-1", Username: "someone", Email: "old@example.com", Password: hash}}
			ps := NewProfileService(store, nil, nil)

			_, err := ps.UpdatePersonalInfo(ctx, connect.NewRequest(&pb.UpdatePersonalInfoRequest{
				Email: tc.email, Password: tc.password, About: "new bio"}))
			if tc.refused {
				is.Equal(connect.CodeOf(err), connect.CodeInvalidArgument)
				is.True(store.saved == nil) // nothing saved
				return
			}
			is.NoErr(err)
			is.Equal(store.saved[0], tc.email)
			is.Equal(store.saved[5], "new bio")
		})
	}
}
