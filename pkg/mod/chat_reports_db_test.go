package mod_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/stores/models"
	mspb "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

func codeOf(err error) connect.Code {
	var ce *connect.Error
	if errors.As(err, &ce) {
		return ce.Code()
	}
	return 0
}

func TestChatReports(t *testing.T) {
	is := is.New(t)
	_, stores, _ := recreateDB()
	defer stores.Disconnect()
	ctx := DefaultConfig.WithContext(context.Background())
	ms := mod.NewModService(stores.UserStore, stores.ChatStore, stores.Queries)
	is.NoErr(stores.UserStore.New(ctx, &entity.User{Username: "mina", Email: "mina@woogles.io", UUID: "mina"}))
	cesar, err := stores.UserStore.Get(ctx, "cesar")
	is.NoErr(err)
	jesse, err := stores.UserStore.Get(ctx, "jesse")
	is.NoErr(err)
	mina, err := stores.UserStore.Get(ctx, "mina")
	is.NoErr(err)
	is.NoErr(stores.Queries.AssignRole(ctx, models.AssignRoleParams{
		Username: "mina", RoleName: string(rbac.Moderator),
	}))
	as := func(u *entity.User) context.Context {
		return apiserver.PlaceInContext(ctx, &entity.Session{
			ID: "s-" + u.Username, Username: u.Username, UserUUID: u.UUID, Expiry: time.Now().Add(time.Hour),
		})
	}
	report := func(u *entity.User, id string) error {
		_, err := ms.ReportChatMessage(as(u), connect.NewRequest(&mspb.ReportChatMessageRequest{MessageId: id}))
		return err
	}
	list := func(u *entity.User, handled bool) ([]*mspb.ChatReport, error) {
		resp, err := ms.GetChatReports(as(u), connect.NewRequest(&mspb.GetChatReportsRequest{Handled: handled}))
		if err != nil {
			return nil, err
		}
		return resp.Msg.Reports, nil
	}
	open := func(u *entity.User) ([]*mspb.ChatReport, error) {
		return list(u, false)
	}
	reopen := func(u *entity.User, id int64) error {
		_, err := ms.ReopenChatReport(as(u), connect.NewRequest(&mspb.ReopenChatReportRequest{ReportId: id}))
		return err
	}
	handle := func(u *entity.User, id int64) error {
		_, err := ms.HandleChatReport(as(u), connect.NewRequest(&mspb.HandleChatReportRequest{ReportId: id}))
		return err
	}

	lobby := "chat.lobby"
	rude, err := stores.ChatStore.AddChat(ctx, "cesar", cesar.UUID, "something rude", lobby, false)
	is.NoErr(err)

	// jesse reports cesar's message, twice: one report
	is.NoErr(report(jesse, rude.Id))
	is.NoErr(report(jesse, rude.Id))
	reports, err := open(mina)
	is.NoErr(err)
	is.Equal(len(reports), 1)
	r := reports[0]
	is.Equal(r.Message.Id, rude.Id)
	is.Equal(r.Message.Channel, lobby)
	is.Equal(r.Message.Message, "something rude")
	is.Equal(r.Message.Username, "cesar")
	is.Equal(r.Message.UserId, cesar.UUID)
	is.Equal(r.ReporterUsername, "jesse")
	is.Equal(r.ReporterId, jesse.UUID)
	is.True(!r.Deleted)
	// open: no one has handled it
	is.Equal(r.HandledByUsername, "")
	is.True(r.HandledAt == nil)
	// chat review marks the reported message
	page, err := ms.GetChannelChats(as(mina), connect.NewRequest(&mspb.GetChannelChatsRequest{Channel: lobby}))
	is.NoErr(err)
	is.Equal(page.Msg.ReportedIds, []string{rude.Id})

	// anyone may report any message: one's own, a line the server posts, a
	// private chat's; an id that is no message is refused
	is.NoErr(report(cesar, rude.Id))
	server, err := stores.ChatStore.AddChat(ctx, "", "", "Play was challenged off the board!", lobby, false)
	is.NoErr(err)
	is.NoErr(report(jesse, server.Id))
	pm := "chat.pm." + cesar.UUID + "_" + jesse.UUID
	whisper, err := stores.ChatStore.AddChat(ctx, "cesar", cesar.UUID, "a rude whisper", pm, false)
	is.NoErr(err)
	is.NoErr(report(jesse, whisper.Id))
	is.Equal(codeOf(report(jesse, "not an id")), connect.CodeInvalidArgument)
	is.True(report(jesse, uuid.NewString()) != nil)

	// newest first
	reports, err = open(mina)
	is.NoErr(err)
	is.Equal(len(reports), 4)
	is.Equal(reports[0].Message.Id, whisper.Id)
	is.Equal(reports[1].Message.Id, server.Id)
	is.Equal(reports[1].Message.Username, "")
	is.Equal(reports[2].ReporterUsername, "cesar")
	is.Equal(reports[3].Id, r.Id)

	// only moderators list or handle reports
	_, err = open(jesse)
	is.Equal(codeOf(err), connect.CodeUnauthenticated)
	_, err = list(jesse, true)
	is.Equal(codeOf(err), connect.CodeUnauthenticated)
	is.Equal(codeOf(handle(jesse, r.Id)), connect.CodeUnauthenticated)
	is.Equal(codeOf(reopen(jesse, r.Id)), connect.CodeUnauthenticated)

	// handling a report handles it alone, cesar's report of the same message
	// staying open; handled already is no error
	is.NoErr(handle(mina, r.Id))
	is.NoErr(handle(mina, r.Id))
	reports, err = open(mina)
	is.NoErr(err)
	is.Equal(len(reports), 3)
	is.Equal(reports[2].ReporterUsername, "cesar")
	handled, err := list(mina, true)
	is.NoErr(err)
	is.Equal(len(handled), 1)
	h := handled[0]
	is.Equal(h.Id, r.Id)
	is.Equal(h.HandledByUsername, "mina")
	is.Equal(h.HandledById, mina.UUID)
	is.True(h.HandledAt != nil)

	// reopened, as after a mistake: open again; open already is no error
	is.NoErr(reopen(mina, r.Id))
	is.NoErr(reopen(mina, r.Id))
	reports, err = open(mina)
	is.NoErr(err)
	is.Equal(len(reports), 4)
	handled, err = list(mina, true)
	is.NoErr(err)
	is.Equal(len(handled), 0)

	// a message a moderator deleted stays listed, marked deleted
	is.NoErr(stores.ChatStore.DeleteChat(ctx, pm, whisper.Id, mina.UUID))
	reports, err = open(mina)
	is.NoErr(err)
	is.Equal(reports[0].Message.Id, whisper.Id)
	is.True(reports[0].Deleted)
}
