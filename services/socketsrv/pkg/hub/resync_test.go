package sockets

import (
	"sort"
	"testing"

	"google.golang.org/protobuf/proto"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// After a NATS reconnect every registered socket asks the API for its realms
// again, exactly as it did when it first registered.
func TestRealmInitInfoForAll(t *testing.T) {
	h := &Hub{
		clients:         map[*Client][]Realm{},
		clientsByUserID: map[string]map[*Client]bool{},
		clientsByConnID: map[string]*Client{},
		realms:          map[Realm]map[*Client]bool{},
	}
	lobby := &Client{authenticated: true, userID: "u1", connID: "c1",
		tempRealms: []string{"lobby", "chat-lobby"}}
	game := &Client{userID: "anon1", connID: "c2", tempRealms: []string{"gametv-abc"}}
	for _, c := range []*Client{lobby, game} {
		h.clientsByUserID[c.userID] = map[*Client]bool{}
		h.clientsByUserID[c.userID][c] = true
		h.clientsByConnID[c.connID] = c
		h.addToRealm(c.tempRealms, c)
	}

	msgs := h.realmInitInfoForAll()
	sort.Slice(msgs, func(i, j int) bool { return msgs[i].subject < msgs[j].subject })
	want := []struct {
		subject string
		realms  []string
		userID  string
	}{
		{"ipc.pb.initRealmInfo.anon.anon1.c2", []string{"gametv-abc"}, "anon1"},
		{"ipc.pb.initRealmInfo.auth.u1.c1", []string{"lobby", "chat-lobby"}, "u1"},
	}
	if len(msgs) != len(want) {
		t.Fatalf("got %d messages, want %d", len(msgs), len(want))
	}
	for i, w := range want {
		if msgs[i].subject != w.subject {
			t.Errorf("subject %d = %q, want %q", i, msgs[i].subject, w.subject)
		}
		info := &pb.InitRealmInfo{}
		if err := proto.Unmarshal(msgs[i].data, info); err != nil {
			t.Fatal(err)
		}
		if info.UserId != w.userID || len(info.Realms) != len(w.realms) {
			t.Errorf("message %d = %v, want realms %v for %s", i, info, w.realms, w.userID)
			continue
		}
		for j := range w.realms {
			if info.Realms[j] != w.realms[j] {
				t.Errorf("message %d realms = %v, want %v", i, info.Realms, w.realms)
			}
		}
	}
}
