package redis

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	NullPresenceChannel = "NULL"
)

// RedisPresenceStore implements a Redis store for user presence.
type RedisPresenceStore struct {
	redisPool *redis.Pool

	setPresenceScript      *redis.Script
	clearPresenceScript    *redis.Script
	renewPresenceScript    *redis.Script
	getChannelsScript      *redis.Script
	updateActiveGameScript *redis.Script
	sweepPresenceScript    *redis.Script

	eventChan chan *entity.EventWrapper
	// now is the clock; tests move it.
	now func() time.Time
}

// startSpan starts a span for one presence operation. end records err on it
// and ends it.
func startSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (trace.Span, func(error)) {
	_, span := otel.Tracer("presence-store").Start(ctx, name, trace.WithAttributes(attrs...))
	return span, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}

// channelChangeAttrs describes a before/after pair of a user's channels.
func channelChangeAttrs(before, after []string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int("presence.channels_before", len(before)),
		attribute.Int("presence.channels_after", len(after)),
	}
}

// SetPresenceScript is a Lua script that handles presence in an atomic way.
// We may want to move this to a separate file if we start adding more Lua
// scripts.
//
//go:embed set_presence.lua
var SetPresenceScript string

// ClearPresenceScript clears the presence and returns the channel(s) it was in.
//
//go:embed clear_presence.lua
var ClearPresenceScript string

// RenewPresenceScript renews the presence
//
//go:embed renew_presence.lua
var RenewPresenceScript string

// GetChannelsScript gets the channels
//
//go:embed get_channels.lua
var GetChannelsScript string

// updateActiveGameScript gets the channels
//
//go:embed update_active_game.lua
var UpdateActiveGameScript string

// SweepPresenceScript removes the presence of connections that went away
// without clearing it.
//
//go:embed sweep_presence.lua
var SweepPresenceScript string

func NewRedisPresenceStore(r *redis.Pool) *RedisPresenceStore {

	return &RedisPresenceStore{
		redisPool:              r,
		setPresenceScript:      redis.NewScript(0, SetPresenceScript),
		clearPresenceScript:    redis.NewScript(0, ClearPresenceScript),
		renewPresenceScript:    redis.NewScript(0, RenewPresenceScript),
		getChannelsScript:      redis.NewScript(0, GetChannelsScript),
		updateActiveGameScript: redis.NewScript(0, UpdateActiveGameScript),
		sweepPresenceScript:    redis.NewScript(0, SweepPresenceScript),
		eventChan:              nil,
		now:                    time.Now,
	}
}

func fromRedisToArrayOfLengthN(ret interface{}, n int) ([]interface{}, error) {
	arr, ok := ret.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result: %T", ret)
	}
	if len(arr) != n {
		return nil, fmt.Errorf("unexpected length (want %v, got %v): %v", n, len(arr), arr)
	}
	return arr, nil
}

func fromRedisArrayToArrayOfString(v interface{}) ([]string, error) {
	arr, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected subresult: %T", v)
	}
	cvt := make([]string, len(arr))
	for i, el := range arr {
		barr, ok := el.([]byte)
		if !ok {
			return nil, fmt.Errorf("unexpected interface type: %T", el)
		}
		cvt[i] = string(barr)
	}
	return cvt, nil
}

// getChannels returns each user's channels, in one call.
func (s *RedisPresenceStore) getChannels(conn redis.Conn, uuids []string) ([][]string, error) {
	args := make([]interface{}, 0, 1+len(uuids))
	args = append(args, s.now().Unix())
	for _, u := range uuids {
		args = append(args, u)
	}
	ret, err := s.getChannelsScript.Do(conn, args...)
	if err != nil {
		return nil, err
	}
	arr, err := fromRedisToArrayOfLengthN(ret, len(uuids))
	if err != nil {
		return nil, err
	}
	channels := make([][]string, len(arr))
	for i, a := range arr {
		if channels[i], err = fromRedisArrayToArrayOfString(a); err != nil {
			return nil, err
		}
	}
	return channels, nil
}

// SetPresence sets the user's presence channel.
func (s *RedisPresenceStore) SetPresence(ctx context.Context, uuid, username string, anon bool,
	channel, connID string) (oldChannels, newChannels []string, err error) {
	// We try to map channels closely to the pubsub NATS channels (and realms),
	// with some exceptions.
	// If the user is online in two different tabs, we go in priority order,
	// as we only want to show them in one place.
	// Priority (from lowest to highest):
	// 	- lobby - The "base" channel.
	//  - usertv.<user_id> - Following a user's games
	//  - gametv.<game_id> - Watching a game
	//  - game.<game_id> - Playing in a game
	span, end := startSpan(ctx, "presence.Set", attribute.String("presence.channel", channel))
	defer func() { span.SetAttributes(channelChangeAttrs(oldChannels, newChannels)...); end(err) }()

	conn := s.redisPool.Get()
	defer conn.Close()
	log.Debug().Str("username", username).Str("connID", connID).Msg("set-presence")

	ret, err := s.setPresenceScript.Do(conn, uuid, username, authOrAnon(anon), connID, channel, s.now().Unix())
	if err != nil {
		return nil, nil, err
	}
	return beforeAfter(ret)
}

func authOrAnon(anon bool) string {
	if anon {
		return "anon"
	}
	return "auth"
}

// beforeAfter reads a script's {before, after} channel lists.
func beforeAfter(ret interface{}) ([]string, []string, error) {
	arr, err := fromRedisToArrayOfLengthN(ret, 2)
	if err != nil {
		return nil, nil, err
	}
	before, err := fromRedisArrayToArrayOfString(arr[0])
	if err != nil {
		return nil, nil, err
	}
	after, err := fromRedisArrayToArrayOfString(arr[1])
	if err != nil {
		return nil, nil, err
	}
	return before, after, nil
}

func (s *RedisPresenceStore) ClearPresence(ctx context.Context, uuid, username string,
	anon bool, connID string) (oldChannels, newChannels, removedChannels []string, err error) {
	span, end := startSpan(ctx, "presence.Clear")
	defer func() {
		span.SetAttributes(channelChangeAttrs(oldChannels, newChannels)...)
		span.SetAttributes(attribute.Int("presence.channels_removed", len(removedChannels)))
		end(err)
	}()

	conn := s.redisPool.Get()
	defer conn.Close()

	log.Debug().Str("username", username).Str("connID", connID).Msg("clear-presence")

	ret, err := s.clearPresenceScript.Do(conn, uuid, username, authOrAnon(anon), connID, s.now().Unix())
	if err != nil {
		return nil, nil, nil, err
	}
	arr, err := fromRedisToArrayOfLengthN(ret, 3)
	if err != nil {
		return nil, nil, nil, err
	}
	if oldChannels, newChannels, err = beforeAfter(arr[:2]); err != nil {
		return nil, nil, nil, err
	}
	if removedChannels, err = fromRedisArrayToArrayOfString(arr[2]); err != nil {
		return nil, nil, nil, err
	}
	return oldChannels, newChannels, removedChannels, nil
}

// live is the lowest score of a live (unexpired) member: a member's score is
// when it expires.
func (s *RedisPresenceStore) live() string {
	return fmt.Sprintf("(%d", s.now().Unix())
}

func (s *RedisPresenceStore) GetInChannel(ctx context.Context, channel string) (users []*entity.User, err error) {
	span, end := startSpan(ctx, "presence.GetInChannel", attribute.String("presence.channel", channel))
	defer func() { span.SetAttributes(attribute.Int("presence.users", len(users))); end(err) }()

	conn := s.redisPool.Get()
	defer conn.Close()

	vals, err := redis.Strings(conn.Do("ZRANGEBYSCORE", "channelpresence:"+channel, s.live(), "+inf"))
	if err != nil {
		return nil, err
	}
	users = make([]*entity.User, len(vals))

	for idx, member := range vals {
		splitmember := strings.Split(member, "#")
		anon := false
		if len(splitmember) > 2 {
			anon = splitmember[2] == "anon"
		}

		users[idx] = &entity.User{
			UUID:      splitmember[0],
			Username:  splitmember[1],
			Anonymous: anon,
		}
	}

	return users, nil
}

// GetPresence returns the channels the given user's live connections are
// in. Return empty for no channel.
func (s *RedisPresenceStore) GetPresence(ctx context.Context, uuid string) ([]string, error) {
	conn := s.redisPool.Get()
	defer conn.Close()

	m, err := redis.Strings(conn.Do("ZRANGEBYSCORE", "userpresence:"+uuid, s.live(), "+inf"))
	if err != nil {
		return nil, err
	}

	if m == nil {
		return nil, nil
	}

	ret := make([]string, len(m))
	for i, el := range m {
		// Format should be: conn_id#channel
		p := strings.Split(el, "#")
		if len(p) != 2 {
			return nil, fmt.Errorf("unexpected presence member: %v (%v)", el, m)
		}
		ret[i] = p[1]
	}
	return ret, nil
}

// RenewPresence renews the connection's presence. Given the channels the
// connection is in, it also puts back the entries it's missing, and returns
// those channels as restored.
func (s *RedisPresenceStore) RenewPresence(ctx context.Context, userID, username string,
	anon bool, connID string, channels []string) (oldChannels, newChannels, restored []string, err error) {
	// Renewals come with every tenth pong of every socket, the busiest
	// presence call: no span of their own, just these on the pong's.
	span := trace.SpanFromContext(ctx)
	defer func() {
		span.SetAttributes(channelChangeAttrs(oldChannels, newChannels)...)
		span.SetAttributes(attribute.Int("presence.channels_restored", len(restored)))
		if err != nil {
			span.RecordError(err)
		}
	}()

	conn := s.redisPool.Get()
	defer conn.Close()

	args := make([]interface{}, 0, 5+len(channels))
	args = append(args, userID, username, authOrAnon(anon), connID, s.now().Unix())
	for _, c := range channels {
		args = append(args, c)
	}
	ret, err := s.renewPresenceScript.Do(conn, args...)
	if err != nil {
		return nil, nil, nil, err
	}
	arr, err := fromRedisToArrayOfLengthN(ret, 3)
	if err != nil {
		return nil, nil, nil, err
	}
	if oldChannels, newChannels, err = beforeAfter(arr[:2]); err != nil {
		return nil, nil, nil, err
	}
	if restored, err = fromRedisArrayToArrayOfString(arr[2]); err != nil {
		return nil, nil, nil, err
	}
	return oldChannels, newChannels, restored, nil
}

// CountInChannel counts the live connections in channel.
func (s *RedisPresenceStore) CountInChannel(ctx context.Context, channel string) (int, error) {
	conn := s.redisPool.Get()
	defer conn.Close()

	return redis.Int(conn.Do("ZCOUNT", "channelpresence:"+channel, s.live(), "+inf"))
}

func (s *RedisPresenceStore) SetEventChan(c chan *entity.EventWrapper) {
	s.eventChan = c
}

func (s *RedisPresenceStore) EventChan() chan *entity.EventWrapper {
	return s.eventChan
}

func (s *RedisPresenceStore) BatchGetChannels(ctx context.Context, uuids []string) (channels [][]string, err error) {
	_, end := startSpan(ctx, "presence.BatchGetChannels", attribute.Int("presence.users", len(uuids)))
	defer func() { end(err) }()
	if len(uuids) == 0 {
		return [][]string{}, nil
	}

	conn := s.redisPool.Get()
	defer conn.Close()
	return s.getChannels(conn, uuids)
}

// FollowChangeTopic is the NATS subject on which a follow or unfollow is
// announced: follows.added.<follower>.<followed> or
// follows.removed.<follower>.<followed>.
func FollowChangeTopic(follower, followed string, following bool) string {
	verb := "removed"
	if following {
		verb = "added"
	}
	return "follows." + verb + "." + follower + "." + followed
}

func (s *RedisPresenceStore) UpdateFollower(ctx context.Context, followee, follower *entity.User, following bool) error {
	var channels []string

	if following {
		var err error
		func() {
			conn := s.redisPool.Get()
			defer conn.Close()

			var chans [][]string
			chans, err = s.getChannels(conn, []string{followee.UUID})
			if err == nil {
				channels = chans[0]
			}
		}()
		if err != nil {
			return err
		}
	}

	// when following, send the (possibly empty) current list of channels.
	// when unfollowing, always send [].

	evtChan := s.EventChan()
	if evtChan != nil {
		wrapped := entity.WrapEvent(&pb.PresenceEntry{
			Username: followee.Username,
			UserId:   followee.UUID,
			Channel:  channels,
		}, pb.MessageType_PRESENCE_ENTRY)
		wrapped.AddAudience(entity.AudUser, follower.UUID)
		// The socket servers keep, for each user connected to them, whom
		// they follow; this tells them about the change.
		wrapped.AddAudience(entity.EventAudienceType(FollowChangeTopic(follower.UUID, followee.UUID, following)), "")
		evtChan <- wrapped
	}

	return nil
}

func (s *RedisPresenceStore) UpdateActiveGame(ctx context.Context, activeGameEntry *pb.ActiveGameEntry) (channels [][][]string, err error) {
	_, end := startSpan(ctx, "presence.UpdateActiveGame",
		attribute.String("game.id", activeGameEntry.Id),
		attribute.Int64("presence.ttl", activeGameEntry.Ttl))
	defer func() { end(err) }()

	args := make([]interface{}, 0, 2+len(activeGameEntry.Player))
	args = append(args, activeGameEntry.Id)
	args = append(args, activeGameEntry.Ttl)
	for _, player := range activeGameEntry.Player {
		args = append(args, player.UserId)
	}

	conn := s.redisPool.Get()
	defer conn.Close()

	ret, err := s.updateActiveGameScript.Do(conn, args...)
	if err != nil {
		return nil, err
	}

	// Script returns an array of [before, after] channels, both are sorted unique arrays.
	arr, err := fromRedisToArrayOfLengthN(ret, len(activeGameEntry.Player))
	if err != nil {
		return nil, err
	}
	channels = make([][][]string, 0, len(arr))
	for _, redisElt := range arr {
		oldChannels, newChannels, err := beforeAfter(redisElt)
		if err != nil {
			return nil, err
		}
		channels = append(channels, [][]string{oldChannels, newChannels})
	}

	return channels, nil
}

// SweptUser is a user some of whose connections the sweep found gone.
type SweptUser struct {
	UUID     string
	Username string
	Anon     bool
	// Removed are the channels none of the user's live connections is in
	// any more: the ones to announce they left.
	Removed []string
	// Before and After are the user's channels, active games included.
	Before, After []string
}

// SweepPresence removes up to limit expired presence entries: those of
// connections that went away without clearing them. It returns how many it
// removed and the users they belonged to.
func (s *RedisPresenceStore) SweepPresence(ctx context.Context, limit int) (swept int, users []SweptUser, err error) {
	span, end := startSpan(ctx, "presence.Sweep", attribute.Int("presence.limit", limit))
	defer func() {
		span.SetAttributes(attribute.Int("presence.swept", swept), attribute.Int("presence.users", len(users)))
		end(err)
	}()

	conn := s.redisPool.Get()
	defer conn.Close()

	ret, err := s.sweepPresenceScript.Do(conn, s.now().Unix(), limit)
	if err != nil {
		return 0, nil, err
	}
	arr, ok := ret.([]interface{})
	if !ok || len(arr) == 0 {
		return 0, nil, fmt.Errorf("unexpected sweep result: %T", ret)
	}
	n, ok := arr[0].(int64)
	if !ok {
		return 0, nil, fmt.Errorf("unexpected sweep count: %T", arr[0])
	}
	for _, elt := range arr[1:] {
		row, err := fromRedisToArrayOfLengthN(elt, 6)
		if err != nil {
			return 0, nil, err
		}
		strs := make([]string, 3)
		for i := range strs {
			b, ok := row[i].([]byte)
			if !ok {
				return 0, nil, fmt.Errorf("unexpected sweep field: %T", row[i])
			}
			strs[i] = string(b)
		}
		u := SweptUser{UUID: strs[0], Username: strs[1], Anon: strs[2] == "anon"}
		if u.Removed, err = fromRedisArrayToArrayOfString(row[3]); err != nil {
			return 0, nil, err
		}
		if u.Before, u.After, err = beforeAfter(row[4:]); err != nil {
			return 0, nil, err
		}
		users = append(users, u)
	}
	return int(n), users, nil
}

// ActiveGame is a game recorded as being played by a user.
type ActiveGame struct {
	UserID string
	GameID string
	// Expires is when the entry lapses, if the game hasn't ended by then.
	Expires time.Time
}

// ListActiveGames returns every live active-game entry. On the way, it
// drops expired entries and gives each key that has none a TTL, until its
// last game expires: keys written before they got one would otherwise stay.
func (s *RedisPresenceStore) ListActiveGames(ctx context.Context) (games []ActiveGame, err error) {
	span, end := startSpan(ctx, "presence.ListActiveGames")
	var purged, keysExpiring int
	defer func() {
		span.SetAttributes(attribute.Int("presence.entries", len(games)),
			attribute.Int("presence.expired_removed", purged),
			attribute.Int("presence.keys_given_ttl", keysExpiring))
		end(err)
	}()

	conn := s.redisPool.Get()
	defer conn.Close()

	const prefix = "activeusergames:"
	cursor := "0"
	for {
		vals, err := redis.Values(conn.Do("SCAN", cursor, "MATCH", prefix+"*", "COUNT", 1000))
		if err != nil {
			return nil, err
		}
		if cursor, err = redis.String(vals[0], nil); err != nil {
			return nil, err
		}
		keys, err := redis.Strings(vals[1], nil)
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			n, err := redis.Int(conn.Do("ZREMRANGEBYSCORE", k, "-inf", s.now().Unix()))
			if err != nil {
				return nil, err
			}
			purged += n
			m, err := redis.StringMap(conn.Do("ZRANGEBYSCORE", k, s.live(), "+inf", "WITHSCORES"))
			if err != nil {
				return nil, err
			}
			var last int64
			for gid, score := range m {
				var exp int64
				if _, err := fmt.Sscan(score, &exp); err != nil {
					return nil, err
				}
				last = max(last, exp)
				games = append(games, ActiveGame{UserID: strings.TrimPrefix(k, prefix), GameID: gid,
					Expires: time.Unix(exp, 0)})
			}
			if len(m) > 0 {
				// EXPIRE NX would do, but not before Redis 7.
				ttl, err := redis.Int(conn.Do("TTL", k))
				if err != nil {
					return nil, err
				}
				if ttl == -1 {
					if _, err := conn.Do("EXPIREAT", k, last); err != nil {
						return nil, err
					}
					keysExpiring++
				}
			}
		}
		if cursor == "0" {
			break
		}
	}
	return games, nil
}
