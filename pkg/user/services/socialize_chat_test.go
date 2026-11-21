package services

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/chat"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

const pkg = "socialize"

type noProfiles struct{ user.Store }

func (noProfiles) GetBriefProfiles(context.Context, []string) (map[string]*pb.BriefProfile, error) {
	return nil, nil
}

func setupChat(t *testing.T) (*SocializeService, *chat.DBStore) {
	t.Helper()
	if err := common.RecreateTestDB(pkg); err != nil {
		t.Fatal(err)
	}
	db, err := common.OpenTestingDB(pkg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	for _, u := range []string{"cesar", "mina", "zed"} {
		if _, err := db.Exec(context.Background(),
			`INSERT INTO users (username, uuid, email, password, created_at)
			 VALUES ($1::text, 'uuid-' || $1::text, $1::text || '@x.test', '', now())`, u); err != nil {
			t.Fatal(err)
		}
	}
	cs := chat.NewDBStore(db)
	return NewSocializeService(noProfiles{}, cs, nil, nil), cs
}

func as(uid string) context.Context {
	return apiserver.PlaceInContext(context.Background(), &entity.Session{UserUUID: uid})
}

// A client that doesn't page marks a private chat read by fetching it, as
// it fetches it when the user opens it. A paging client marks reads itself.
func TestFetchMarksPrivateChatRead(t *testing.T) {
	is := is.New(t)
	ss, cs := setupChat(t)
	ch := "chat.pm.uuid-cesar_uuid-mina"
	m, err := cs.AddChat(context.Background(), "cesar", "uuid-cesar", "hi", ch, false)
	is.NoErr(err)

	unread := func() bool {
		resp, err := ss.GetActiveChatChannels(as("uuid-mina"), connect.NewRequest(&pb.GetActiveChatChannelsRequest{Number: 20}))
		is.NoErr(err)
		is.Equal(len(resp.Msg.Channels), 1)
		return resp.Msg.Channels[0].HasUpdate
	}
	is.True(unread())

	resp, err := ss.GetChatsForChannel(as("uuid-mina"), connect.NewRequest(&pb.GetChatsRequest{Channel: ch, AroundLastRead: true}))
	is.NoErr(err)
	is.Equal(len(resp.Msg.Messages), 1)
	is.Equal(resp.Msg.LastReadId, "")
	is.True(unread()) // a paging client: not marked

	resp, err = ss.GetChatsForChannel(as("uuid-mina"), connect.NewRequest(&pb.GetChatsRequest{Channel: ch}))
	is.NoErr(err)
	is.Equal(resp.Msg.LastReadId, m.Id)
	is.True(!unread())

	// Someone outside the chat can't read it or mark it.
	_, err = ss.GetChatsForChannel(as("uuid-zed"), connect.NewRequest(&pb.GetChatsRequest{Channel: ch}))
	is.True(err != nil)
	_, err = ss.MarkChannelRead(as("uuid-zed"), connect.NewRequest(&pb.MarkChannelReadRequest{Channel: ch, MessageId: m.Id}))
	is.True(err != nil)
}

// GetChatsForChannel pages with one cursor at a time and opens around the
// user's read position.
func TestGetChatsForChannelPaging(t *testing.T) {
	is := is.New(t)
	ss, cs := setupChat(t)
	ch := "chat.tournament.t1"
	var ids []string
	for _, s := range []string{"a", "b", "c", "d", "e"} {
		m, err := cs.AddChat(context.Background(), "zed", "uuid-zed", s, ch, false)
		is.NoErr(err)
		ids = append(ids, m.Id)
	}
	_, err := ss.MarkChannelRead(as("uuid-mina"), connect.NewRequest(&pb.MarkChannelReadRequest{Channel: ch, MessageId: ids[1]}))
	is.NoErr(err)

	resp, err := ss.GetChatsForChannel(as("uuid-mina"), connect.NewRequest(&pb.GetChatsRequest{
		Channel: ch, AroundLastRead: true, Limit: 3}))
	is.NoErr(err)
	is.Equal(resp.Msg.LastReadId, ids[1])
	is.Equal(len(resp.Msg.Messages), 3) // a, b (read), c
	is.Equal(resp.Msg.Messages[1].Id, ids[1])
	is.True(resp.Msg.HasNewer)
	is.True(!resp.Msg.HasOlder)

	resp, err = ss.GetChatsForChannel(context.Background(), connect.NewRequest(&pb.GetChatsRequest{
		Channel: ch, AfterId: ids[2]}))
	is.NoErr(err) // anonymous readers can read room chats
	is.Equal(len(resp.Msg.Messages), 2)
	is.Equal(resp.Msg.LastReadId, "")

	_, err = ss.GetChatsForChannel(as("uuid-mina"), connect.NewRequest(&pb.GetChatsRequest{
		Channel: ch, BeforeId: ids[2], AfterId: ids[0]}))
	is.Equal(connect.CodeOf(err), connect.CodeInvalidArgument)
	_, err = ss.GetChatsForChannel(as("uuid-mina"), connect.NewRequest(&pb.GetChatsRequest{Channel: ch, Limit: 500}))
	is.Equal(connect.CodeOf(err), connect.CodeInvalidArgument)
}
