package redis

// The presence scripts were rewritten in 2026-11 to ignore expired entries,
// keep the userpresences index for the sweep, and restore lost presence on
// renewal. These tests hold them to the scripts they replaced (kept as they
// were on master, in testdata/master) wherever nothing has expired, which is
// where their behavior must not change: the same results and the same data
// in Redis, operation after operation.

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
)

func masterScript(t *testing.T, name string) *redis.Script {
	t.Helper()
	src, err := os.ReadFile("testdata/master/" + name + ".lua")
	if err != nil {
		t.Fatal(err)
	}
	return redis.NewScript(0, string(src))
}

// dbConn returns a connection to Redis database db, emptied.
func dbConn(t *testing.T, db int) redis.Conn {
	t.Helper()
	if RedisURL == "" {
		t.Skip("REDIS_URL not set")
	}
	c, err := redis.DialURL(RedisURL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Do("SELECT", db); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Do("FLUSHDB"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Do("FLUSHDB"); c.Close() })
	return c
}

// dump is every sorted set in the database, member to score.
func dump(t *testing.T, c redis.Conn) map[string]map[string]float64 {
	t.Helper()
	keys, err := redis.Strings(c.Do("KEYS", "*"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]map[string]float64{}
	for _, k := range keys {
		if strings.HasPrefix(k, "closedconn:") {
			// New: closed tabs' tombstones (checked where tabs close).
			continue
		}
		typ, err := redis.String(c.Do("TYPE", k))
		if err != nil {
			t.Fatal(err)
		}
		if typ != "zset" {
			t.Fatalf("key %s is a %s", k, typ)
		}
		vals, err := redis.Strings(c.Do("ZRANGE", k, 0, -1, "WITHSCORES"))
		if err != nil {
			t.Fatal(err)
		}
		m := map[string]float64{}
		for i := 0; i < len(vals); i += 2 {
			var f float64
			fmt.Sscan(vals[i+1], &f)
			m[vals[i]] = f
		}
		out[k] = m
	}
	return out
}

// sameData compares two dumps. Active-game scores come from each script's
// own call to Redis's clock, so they may be a second apart.
func sameData(a, b map[string]map[string]float64) error {
	if len(a) != len(b) {
		return fmt.Errorf("keys: %v vs %v", sortedKeys(a), sortedKeys(b))
	}
	for k, am := range a {
		bm, ok := b[k]
		if !ok {
			return fmt.Errorf("key %s missing", k)
		}
		if len(am) != len(bm) {
			return fmt.Errorf("%s: %v vs %v", k, am, bm)
		}
		tolerance := 0.0
		if len(k) > 16 && k[:16] == "activeusergames:" {
			tolerance = 2
		}
		for m, as := range am {
			bs, ok := bm[m]
			if !ok || math.Abs(as-bs) > tolerance {
				return fmt.Errorf("%s %s: %v vs %v", k, m, as, bm[m])
			}
		}
	}
	return nil
}

func sortedKeys(m map[string]map[string]float64) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// strs reads a script's array of strings.
func strs(t *testing.T, v interface{}) []string {
	t.Helper()
	s, err := fromRedisArrayToArrayOfString(v)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

type presenceScripts struct {
	set, renew, clear, getChannels, updateActiveGame *redis.Script
}

func TestNewScriptsMatchMasterWhileNothingExpires(t *testing.T) {
	old := presenceScripts{
		set:              masterScript(t, "set_presence"),
		renew:            masterScript(t, "renew_presence"),
		clear:            masterScript(t, "clear_presence"),
		getChannels:      masterScript(t, "get_channels"),
		updateActiveGame: masterScript(t, "update_active_game"),
	}
	cur := presenceScripts{
		set:              redis.NewScript(0, SetPresenceScript),
		renew:            redis.NewScript(0, RenewPresenceScript),
		clear:            redis.NewScript(0, ClearPresenceScript),
		getChannels:      redis.NewScript(0, GetChannelsScript),
		updateActiveGame: redis.NewScript(0, UpdateActiveGameScript),
	}

	for seed := int64(1); seed <= 6; seed++ {
		t.Run(fmt.Sprint("seed", seed), func(t *testing.T) {
			oc, nc := dbConn(t, 12), dbConn(t, 13)
			runEquivalence(t, rand.New(rand.NewSource(seed)), old, cur, oc, nc)
		})
	}
}

func runEquivalence(t *testing.T, r *rand.Rand, old, cur presenceScripts, oc, nc redis.Conn) {
	type user struct{ uuid, name, auth string }
	users := []user{
		{"u1", "cesar", "auth"}, {"u2", "mina", "auth"}, {"u3", "zed", "auth"},
		{"u4", "ann", "auth"}, {"anon-x5", "anon-x5", "anon"}, {"anon-x6", "anon-x6", "anon"},
	}
	chans := []string{"chat.lobby", "chat.tournament.t1", "chat.game.g1", "chat.gametv.g2", "chat.puzzlelobby"}
	games := []string{"gA", "gB", "gC", "gD"}
	// What each connection has joined, to renew it with its channels the way
	// a socket server sending them does.
	joined := map[string]map[string]bool{}
	closed := map[string]bool{}

	t0 := time.Now().Unix()
	const ops = 1200
	for step := 0; step < ops; step++ {
		// Under 180 s in all: nothing expires.
		ts := t0 + int64(step)/8
		u := users[r.Intn(len(users))]
		conn := fmt.Sprintf("%s-c%d", u.uuid, r.Intn(3))
		for closed[conn] {
			conn += "x"
		}
		var desc string
		var ov, nv interface{}
		var oerr, nerr error

		switch op := r.Intn(100); {
		case op < 35:
			ch := chans[r.Intn(len(chans))]
			desc = fmt.Sprintf("set %s %s %s", u.uuid, conn, ch)
			ov, oerr = old.set.Do(oc, u.uuid, u.name, u.auth, conn, ch, ts)
			nv, nerr = cur.set.Do(nc, u.uuid, u.name, u.auth, conn, ch, ts)
			if joined[conn] == nil {
				joined[conn] = map[string]bool{}
			}
			joined[conn][ch] = true
		case op < 60:
			desc = fmt.Sprintf("renew %s %s", u.uuid, conn)
			ov, oerr = old.renew.Do(oc, u.uuid, u.name, u.auth, conn, ts)
			args := []interface{}{u.uuid, u.name, u.auth, conn, ts}
			if r.Intn(2) == 0 { // a socket server that sends the channels
				for ch := range joined[conn] {
					args = append(args, ch)
				}
				desc += " with channels"
			}
			nv, nerr = cur.renew.Do(nc, args...)
			if nerr == nil {
				// The new script also returns what it restored: nothing,
				// since nothing was lost.
				arr := nv.([]interface{})
				if len(strs(t, arr[2])) != 0 {
					t.Fatalf("step %d %s: restored %v", step, desc, strs(t, arr[2]))
				}
				nv = arr[:2]
			}
		case op < 75:
			desc = fmt.Sprintf("clear %s %s", u.uuid, conn)
			ov, oerr = old.clear.Do(oc, u.uuid, u.name, u.auth, conn, ts)
			nv, nerr = cur.clear.Do(nc, u.uuid, u.name, u.auth, conn, ts)
			delete(joined, conn)
			// The new script leaves a tombstone, for as long as an entry
			// lasts.
			if ttl, err := redis.Int(nc.Do("TTL", "closedconn:"+conn)); err != nil || ttl <= 0 || ttl > 180 {
				t.Fatalf("step %d %s: tombstone TTL %d, %v", step, desc, ttl, err)
			}
			// A tab never comes back once closed; use a fresh connection ID
			// from here on, as the API issues one per socket.
			closed[conn] = true
		case op < 87:
			desc = fmt.Sprintf("get channels %s", u.uuid)
			ov, oerr = old.getChannels.Do(oc, u.uuid)
			nv, nerr = cur.getChannels.Do(nc, ts, u.uuid)
		default:
			g := games[r.Intn(len(games))]
			ttl := []int{0, 43200}[r.Intn(2)]
			args := []interface{}{g, ttl, u.uuid}
			if r.Intn(2) == 0 {
				args = append(args, users[r.Intn(len(users))].uuid)
			}
			desc = fmt.Sprintf("active game %v", args)
			ov, oerr = old.updateActiveGame.Do(oc, args...)
			nv, nerr = cur.updateActiveGame.Do(nc, args...)
		}

		if oerr != nil || nerr != nil {
			t.Fatalf("step %d %s: errors %v / %v", step, desc, oerr, nerr)
		}
		if !reflect.DeepEqual(normalize(ov), normalize(nv)) {
			t.Fatalf("step %d %s: master returned %v, new %v", step, desc, normalize(ov), normalize(nv))
		}
		if err := sameData(dump(t, oc), dump(t, nc)); err != nil {
			t.Fatalf("step %d %s: data differs: %v", step, desc, err)
		}
	}
}

// normalize turns a script's nested arrays of bytes into strings.
func normalize(v interface{}) interface{} {
	switch x := v.(type) {
	case []interface{}:
		out := make([]interface{}, len(x))
		for i, e := range x {
			out[i] = normalize(e)
		}
		return out
	case []byte:
		return string(x)
	default:
		return x
	}
}
