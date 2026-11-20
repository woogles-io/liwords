package sockets

import (
	"sort"
	"testing"

	"google.golang.org/protobuf/proto"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func followersOf(fi *followIndex, user string) []string {
	var got []string
	for f := range fi.followersOf(user) {
		got = append(got, f)
	}
	sort.Strings(got)
	return got
}

func assertFollowers(t *testing.T, fi *followIndex, user string, want ...string) {
	t.Helper()
	got := followersOf(fi, user)
	if len(got) != len(want) {
		t.Fatalf("followers of %s = %v, want %v", user, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("followers of %s = %v, want %v", user, got, want)
		}
	}
}

func TestFollowIndex(t *testing.T) {
	fi := newFollowIndex()
	fi.set("bob", []string{"x", "y"})
	fi.set("carol", []string{"x"})
	assertFollowers(t, fi, "x", "bob", "carol")
	assertFollowers(t, fi, "y", "bob")

	// A newer list replaces the old one.
	fi.set("bob", []string{"y", "z"})
	assertFollowers(t, fi, "x", "carol")
	assertFollowers(t, fi, "z", "bob")

	fi.follow("carol", "z")
	fi.unfollow("bob", "y")
	assertFollowers(t, fi, "z", "bob", "carol")
	assertFollowers(t, fi, "y")

	// Changes for users who aren't connected here are ignored.
	fi.follow("dave", "x")
	assertFollowers(t, fi, "x", "carol")

	fi.remove("bob")
	fi.remove("carol")
	if users, follows := fi.size(); users != 0 || follows != 0 {
		t.Fatalf("size after removing everyone = %d, %d", users, follows)
	}
	if len(fi.followers) != 0 {
		t.Fatalf("followers left behind: %v", fi.followers)
	}
}

// A user who follows nobody is still recorded, so a later follow reaches them.
func TestFollowIndexEmptyList(t *testing.T) {
	fi := newFollowIndex()
	fi.set("bob", []string{})
	fi.follow("bob", "x")
	assertFollowers(t, fi, "x", "bob")
}

func testHub() *Hub {
	return &Hub{
		clients:         map[*Client][]Realm{},
		clientsByUserID: map[string]map[*Client]bool{},
		clientsByConnID: map[string]*Client{},
		realms:          map[Realm]map[*Client]bool{},
		follows:         newFollowIndex(),
	}
}

func connect(h *Hub, userID, connID string) *Client {
	c := &Client{authenticated: true, userID: userID, connID: connID,
		send: make(chan []byte, 8)}
	if h.clientsByUserID[userID] == nil {
		h.clientsByUserID[userID] = map[*Client]bool{}
	}
	h.clientsByUserID[userID][c] = true
	h.clientsByConnID[connID] = c
	h.addToRealm([]string{"lobby"}, c)
	return c
}

func received(c *Client) int {
	n := 0
	for {
		select {
		case <-c.send:
			n++
		default:
			return n
		}
	}
}

func TestDeliverToFollowers(t *testing.T) {
	h := testHub()
	bob1 := connect(h, "bob", "b1")
	bob2 := connect(h, "bob", "b2")
	carol := connect(h, "carol", "c1")
	dave := connect(h, "dave", "d1")
	h.follows.set("bob", []string{"x"})
	h.follows.set("carol", []string{"y"})
	h.follows.set("dave", []string{})

	h.deliverToFollowers(FollowersMessage{userID: "x", msg: []byte("x is here")})
	if received(bob1) != 1 || received(bob2) != 1 {
		t.Fatal("each of bob's sockets should get x's presence")
	}
	if received(carol) != 0 || received(dave) != 0 {
		t.Fatal("users who don't follow x got its presence")
	}

	h.applyFollowChange(FollowChange{follower: "dave", followed: "x", following: true})
	h.applyFollowChange(FollowChange{follower: "bob", followed: "x", following: false})
	h.deliverToFollowers(FollowersMessage{userID: "x", msg: []byte("x is here")})
	if received(dave) != 1 {
		t.Fatal("dave followed x and should get its presence")
	}
	if received(bob1) != 0 || received(bob2) != 0 {
		t.Fatal("bob unfollowed x and shouldn't get its presence")
	}
}

// A follower whose socket is too backed up is removed while being delivered
// to; the rest still get the message.
func TestDeliverToFollowersRemovesStuckSocket(t *testing.T) {
	h := testHub()
	// removeClient publishes leave messages; with no connection they fail.
	h.pubsub = &PubSub{}
	stuck := connect(h, "bob", "b1")
	stuck.send = make(chan []byte) // full: nothing reads it
	carol := connect(h, "carol", "c1")
	h.follows.set("bob", []string{"x"})
	h.follows.set("carol", []string{"x"})

	h.deliverToFollowers(FollowersMessage{userID: "x", msg: []byte("x is here")})
	if received(carol) != 1 {
		t.Fatal("carol should get x's presence")
	}
	if !stuck.removed {
		t.Fatal("bob's stuck socket should be removed")
	}
	if _, ok := h.follows.follows["bob"]; ok {
		t.Fatal("bob's follows outlived his last socket")
	}
	assertFollowers(t, h.follows, "x", "carol")
}

func TestParseFollowChange(t *testing.T) {
	for _, tc := range []struct {
		subject string
		want    FollowChange
		ok      bool
	}{
		{"follows.added.bob.x", FollowChange{"bob", "x", true}, true},
		{"follows.removed.bob.x", FollowChange{"bob", "x", false}, true},
		{"follows.changed.bob.x", FollowChange{}, false},
		{"follows.added.bob", FollowChange{}, false},
		{"follows.added.bob.", FollowChange{}, false},
		{"follows.added.bob.x.y", FollowChange{}, false},
	} {
		got, ok := parseFollowChange(tc.subject)
		if ok != tc.ok || got != tc.want {
			t.Errorf("parseFollowChange(%q) = %v, %v; want %v, %v", tc.subject, got, ok, tc.want, tc.ok)
		}
	}
}

func TestFollowedPlayersSeek(t *testing.T) {
	seek := &pb.SeekRequest{OnlyFollowedPlayers: true, RatingKey: "x"}
	data, err := proto.Marshal(&pb.FollowedPlayersSeek{
		SeekRequest:      seek,
		RecipientUserIds: []string{"bob", "alice", "carol"},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := followedPlayersSeek("alice", data)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alice", "bob", "carol"}
	if len(m.userIDs) != len(want) {
		t.Fatalf("recipients = %v, want %v", m.userIDs, want)
	}
	for i := range want {
		if m.userIDs[i] != want[i] {
			t.Fatalf("recipients = %v, want %v", m.userIDs, want)
		}
	}
	if m.msg[2] != byte(pb.MessageType_SEEK_REQUEST) {
		t.Fatalf("message type = %d, want SEEK_REQUEST", m.msg[2])
	}
	got := &pb.SeekRequest{}
	if err := proto.Unmarshal(m.msg[3:], got); err != nil || !proto.Equal(got, seek) {
		t.Fatalf("seek = %v (%v), want %v", got, err, seek)
	}
}
