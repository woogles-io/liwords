package config

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matryer/is"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/disposableemail"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/config_service"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

func TestDisposableEmailCheck(t *testing.T) {
	is := is.New(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("mailinator.com\n"))
	}))
	defer srv.Close()
	cs := &ConfigService{disposable: disposableemail.NewList(srv.URL)}
	check := cs.disposableEmailCheck(context.Background())

	is.Equal(check("someone@mailinator.com"), pb.DisposableEmail_DISPOSABLE_EMAIL_YES)
	is.Equal(check("someone@gmail.com"), pb.DisposableEmail_DISPOSABLE_EMAIL_NO)
}

func TestDisposableEmailCheckWithoutTheListIsUnknown(t *testing.T) {
	is := is.New(t)
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	for _, cs := range []*ConfigService{
		{disposable: disposableemail.NewList(srv.URL)},
		{},
	} {
		check := cs.disposableEmailCheck(context.Background())
		is.Equal(check("someone@mailinator.com"), pb.DisposableEmail_DISPOSABLE_EMAIL_UNKNOWN)
		is.Equal(check("someone@gmail.com"), pb.DisposableEmail_DISPOSABLE_EMAIL_UNKNOWN)
	}
}

func suspend(end *timestamppb.Timestamp) map[string]*ms.ModAction {
	return map[string]*ms.ModAction{
		ms.ModActionType_SUSPEND_ACCOUNT.String(): {
			Type:      ms.ModActionType_SUSPEND_ACCOUNT,
			StartTime: timestamppb.New(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)),
			EndTime:   end,
			Note:      "spam",
		},
	}
}

func TestSuspensionOf(t *testing.T) {
	is := is.New(t)
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	later := timestamppb.New(now.Add(48 * time.Hour))

	permanent := suspensionOf(suspend(nil), now)
	is.True(permanent != nil)
	is.Equal(permanent.Type, "SUSPEND_ACCOUNT")
	is.True(permanent.End == nil)
	is.Equal(permanent.Note, "spam")

	temporary := suspensionOf(suspend(later), now)
	is.True(temporary != nil)
	is.True(temporary.End.AsTime().Equal(later.AsTime()))

	is.True(suspensionOf(suspend(timestamppb.New(now)), now) == nil)
	is.True(suspensionOf(suspend(timestamppb.New(now.Add(-time.Hour))), now) == nil)
	is.True(suspensionOf(map[string]*ms.ModAction{
		ms.ModActionType_MUTE.String(): {Type: ms.ModActionType_MUTE},
	}, now) == nil)
	is.True(suspensionOf(nil, now) == nil)
}

// actionsBatchStore answers GetActionsBatch only; any other user.Store
// method would panic on the nil embedded interface.
type actionsBatchStore struct {
	user.Store
	actions map[string]map[string]*ms.ModAction
	calls   [][]string
}

func (s *actionsBatchStore) GetActionsBatch(ctx context.Context, uuids []string) (map[string]map[string]*ms.ModAction, error) {
	s.calls = append(s.calls, uuids)
	return s.actions, nil
}

func TestFillSuspensions(t *testing.T) {
	is := is.New(t)
	later := timestamppb.New(time.Now().Add(48 * time.Hour))
	store := &actionsBatchStore{actions: map[string]map[string]*ms.ModAction{
		"banned": suspend(nil),
		"muted": {
			ms.ModActionType_MUTE.String(): {Type: ms.ModActionType_MUTE},
		},
		"suspended": suspend(later),
	}}
	cs := &ConfigService{userStore: store}
	users := []*pb.UserDetailsResponse{
		{Uuid: "banned"}, {Uuid: "muted"}, {Uuid: "suspended"}, {Uuid: "clean"},
	}

	is.NoErr(cs.fillSuspensions(context.Background(), users))
	is.Equal(len(store.calls), 1)
	is.Equal(store.calls[0], []string{"banned", "muted", "suspended", "clean"})
	is.True(users[0].Suspension != nil && users[0].Suspension.End == nil)
	is.True(users[1].Suspension == nil)
	is.True(users[2].Suspension != nil && users[2].Suspension.End.AsTime().Equal(later.AsTime()))
	is.True(users[3].Suspension == nil)

	is.NoErr(cs.fillSuspensions(context.Background(), nil))
	is.Equal(len(store.calls), 1)
}
