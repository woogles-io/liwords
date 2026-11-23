package redis

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Two simulated hours of tabs joining chats, renewing, closing, and dying
// without a word, with games starting and ending and the sweep running every
// 30 s, as in production. Presence must not pile up:
//   - every key but the userpresences index has a TTL;
//   - nothing stays more than one sweep past its expiry;
//   - the chat lists show exactly the tabs that are open;
//   - once every tab is gone and every game over, Redis is empty.
func TestPresenceDoesNotLeak(t *testing.T) {
	for _, withChannels := range []bool{true, false} {
		// Socket servers that send their tabs' channels with pongs, and
		// older ones that don't.
		t.Run(fmt.Sprint("channels-in-pongs=", withChannels), func(t *testing.T) {
			simulatePresence(t, rand.New(rand.NewSource(42)), withChannels)
		})
	}
}

type simTab struct {
	uuid, name string
	anon       bool
	conn       string
	chans      []string
	nextRenew  time.Time
	// expires is when the tab's entries lapse unless renewed.
	expires time.Time
	dead    bool // stopped renewing, without clearing
}

func simulatePresence(t *testing.T, r *rand.Rand, withChannels bool) {
	is := is.New(t)
	ps, pool, now := clockedStore(t)
	ctx := context.Background()
	c := pool.Get()
	defer c.Close()

	chans := []string{"chat.lobby", "chat.tournament.t1", "chat.game.g1", "chat.gametv.g2", "chat.puzzlelobby"}
	var tabs []*simTab
	games := map[string][]string{} // game to players
	start := *now
	lastSweep := *now
	nextID := 0

	sweep := func() {
		for {
			n, _, err := ps.SweepPresence(ctx, 50)
			is.NoErr(err)
			if n < 50 {
				break
			}
		}
		lastSweep = *now
	}

	for step := 0; now.Sub(start) < 2*time.Hour; step++ {
		*now = now.Add(5 * time.Second)

		// A few tabs open.
		for i := r.Intn(3); i > 0; i-- {
			nextID++
			u := r.Intn(20)
			tab := &simTab{uuid: fmt.Sprintf("u%d", u), name: fmt.Sprintf("user%d", u),
				conn: fmt.Sprintf("conn%d", nextID), nextRenew: now.Add(50 * time.Second),
				expires: now.Add(3 * time.Minute)}
			if u >= 16 {
				tab.uuid, tab.name, tab.anon = fmt.Sprintf("anon-%d", u), fmt.Sprintf("anon-%d", u), true
			}
			for _, ch := range r.Perm(len(chans))[:1+r.Intn(2)] {
				tab.chans = append(tab.chans, chans[ch])
				_, _, err := ps.SetPresence(ctx, tab.uuid, tab.name, tab.anon, chans[ch], tab.conn)
				is.NoErr(err)
			}
			tabs = append(tabs, tab)
		}

		// Tabs renew, close, or die. A dead tab is forgotten once its
		// entries have lapsed: the sweep should have removed them.
		kept := tabs[:0]
		for _, tab := range tabs {
			switch x := r.Intn(1000); {
			case tab.dead:
				if !tab.expires.After(*now) {
					continue
				}
			case x < 8:
				_, _, _, err := ps.ClearPresence(ctx, tab.uuid, tab.name, tab.anon, tab.conn)
				is.NoErr(err)
				continue
			case x < 12:
				tab.dead = true
			case !now.Before(tab.nextRenew):
				var renewWith []string
				if withChannels {
					renewWith = tab.chans
				}
				_, _, _, err := ps.RenewPresence(ctx, tab.uuid, tab.name, tab.anon, tab.conn, renewWith)
				is.NoErr(err)
				tab.nextRenew = now.Add(time.Duration(45+r.Intn(10)) * time.Second)
				tab.expires = now.Add(3 * time.Minute)
			}
			kept = append(kept, tab)
		}
		tabs = kept

		// Games start and end.
		if r.Intn(10) == 0 {
			g := fmt.Sprintf("g%d", step)
			players := []string{fmt.Sprintf("u%d", r.Intn(16)), fmt.Sprintf("u%d", r.Intn(16))}
			_, err := ps.UpdateActiveGame(ctx, activeGameEntry(g, 43200, players))
			is.NoErr(err)
			games[g] = players
		}
		for g, players := range games {
			if r.Intn(40) == 0 {
				_, err := ps.UpdateActiveGame(ctx, activeGameEntry(g, 0, players))
				is.NoErr(err)
				delete(games, g)
			}
		}

		if now.Sub(lastSweep) >= 30*time.Second {
			sweep()
			checkNoLeaks(t, c, *now, tabs)
		}
	}

	// Everyone leaves: some tabs close, the rest die; every game ends.
	for _, tab := range tabs {
		if !tab.dead && r.Intn(2) == 0 {
			_, _, _, err := ps.ClearPresence(ctx, tab.uuid, tab.name, tab.anon, tab.conn)
			is.NoErr(err)
		}
	}
	for g, players := range games {
		_, err := ps.UpdateActiveGame(ctx, activeGameEntry(g, 0, players))
		is.NoErr(err)
	}
	*now = now.Add(4 * time.Minute)
	sweep()
	keys, err := redis.Strings(c.Do("KEYS", "*"))
	is.NoErr(err)
	is.Equal(keys, []string{})
}

func activeGameEntry(id string, ttl int64, players []string) *pb.ActiveGameEntry {
	e := &pb.ActiveGameEntry{Id: id, Ttl: ttl}
	for _, p := range players {
		e.Player = append(e.Player, &pb.ActiveGamePlayer{UserId: p})
	}
	return e
}

// checkNoLeaks checks the invariants, just after a sweep: every key but the
// index has a TTL, nothing in any key has expired, and each chat's list holds
// exactly the tabs whose entries haven't lapsed (open ones, and dead ones
// not yet expired).
func checkNoLeaks(t *testing.T, c redis.Conn, now time.Time, tabs []*simTab) {
	t.Helper()
	listed := map[string]int{}
	for _, tab := range tabs {
		if tab.expires.After(now) {
			for _, ch := range tab.chans {
				listed["channelpresence:"+ch]++
			}
		}
	}
	keys, err := redis.Strings(c.Do("KEYS", "*"))
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range keys {
		if k != "userpresences" {
			ttl, err := redis.Int(c.Do("TTL", k))
			if err != nil {
				t.Fatal(err)
			}
			if ttl < 0 {
				t.Fatalf("%v: %s has no TTL", now, k)
			}
		}
		stale, err := redis.Int(c.Do("ZCOUNT", k, "-inf", now.Unix()))
		if err != nil {
			t.Fatal(err)
		}
		if stale > 0 {
			t.Fatalf("%v: %s has %d expired members after a sweep", now, k, stale)
		}
	}
	for _, k := range keys {
		if strings.HasPrefix(k, "channelpresence:") {
			if _, ok := listed[k]; !ok {
				listed[k] = 0
			}
		}
	}
	for k, want := range listed {
		got, err := redis.Int(c.Do("ZCARD", k))
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%v: %s lists %d tabs, want %d", now, k, got, want)
		}
	}
}
