package soughtgame

// We may never need a redis store for this, but let's keep this code for a
// little while.

// import (
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"strconv"

// 	"github.com/domino14/liwords/pkg/entity"
// 	"github.com/garyburd/redigo/redis"
// 	"github.com/rs/zerolog/log"

// 	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
// )

// const (
// 	// Expire sought games after some time in case they don't get removed cleanly.
// 	SoughtGameExpiration = 60 * 60 * 2
// )

// type RedisStore struct {
// 	redisPool *redis.Pool
// }

// func NewRedisSoughtGameStore(r *redis.Pool) *RedisStore {
// 	return &RedisStore{
// 		redisPool: r,
// 	}
// }

// func sgFromMap(m map[string]string) (*entity.SoughtGame, error) {
// 	if len(m) == 0 {
// 		return nil, errNotFound
// 	}
// 	seekType, err := strconv.Atoi(m["type"])
// 	if err != nil {
// 		return nil, err
// 	}

// 	if seekType == int(entity.TypeSeek) {
// 		sr := &pb.SeekRequest{}
// 		err = json.Unmarshal([]byte(m["request"]), sr)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return entity.NewSoughtGame(sr), nil
// 	} else if seekType == int(entity.TypeMatch) {
// 		mr := &pb.MatchRequest{}
// 		err = json.Unmarshal([]byte(m["request"]), mr)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return entity.NewMatchRequest(mr), nil
// 	}
// 	log.Error().Int("seekType", seekType).Interface("redisMap", m).Msg("unexpected-seek-type")
// 	return nil,
// }

// // Get gets the sought game with the given ID.
// func (s *RedisStore) Get(ctx context.Context, id string) (*entity.SoughtGame, error) {

// 	conn := s.redisPool.Get()
// 	defer conn.Close()

// 	sg, err := redis.StringMap(conn.Do("HGETALL", "soughtgame:"+id))
// 	if err != nil {
// 		return nil, err
// 	}
// 	return sgFromMap(sg)
// }

// // GetByConnID gets the sought game with the given socket connection ID.
// func (s *RedisStore) GetByConnID(ctx context.Context, connID string) (*entity.SoughtGame, error) {
// 	conn := s.redisPool.Get()
// 	defer conn.Close()

// 	sg, err := redis.StringMap(conn.Do("HGETALL", "soughtgamebyconnid:"+connID))
// 	if err != nil {
// 		return nil, err
// 	}
// 	return sgFromMap(sg)
// }

// func (s *RedisStore) getByUser(ctx context.Context, userID string) (*entity.SoughtGame, error) {
// 	conn := s.redisPool.Get()
// 	defer conn.Close()

// 	sg, err := redis.StringMap(conn.Do("HGETALL", "soughtgamebyuser:"+userID))
// 	if err != nil {
// 		return nil, err
// 	}
// 	return sgFromMap(sg)
// }

// // Set sets the game in the store.
// func (s *RedisStore) Set(ctx context.Context, game *entity.SoughtGame) error {

// 	redistype := strconv.Itoa(int(game.Type()))
// 	var bts []byte
// 	var err error
// 	var request string

// 	if game.Type() == entity.TypeSeek {

// 		bts, err = json.Marshal(game.SeekRequest)
// 	} else if game.Type() == entity.TypeMatch {
// 		bts, err = json.Marshal(game.MatchRequest)
// 	}
// 	if err != nil {
// 		return err
// 	}
// 	request = string(bts)

// 	keys := []string{
// 		"soughtgame:" + game.ID(),
// 		"soughtgamebyconnid:" + game.ConnID(),
// 		"soughtgamebyuser:" + game.Seeker(),
// 	}

// 	conn := s.redisPool.Get()
// 	defer conn.Close()

// 	for _, key := range keys {
// 		_, err = conn.Do("HSET", key, "type", redistype, "request", request)
// 		if err != nil {
// 			return err
// 		}
// 		_, err = conn.Do("EXPIRE", key, SoughtGameExpiration)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	if game.Type() == entity.TypeMatch {
// 		// Store this in a hash map with keys being the different match req IDs
// 		// and the values being the JSON request.
// 		key := "matchrequestbyreceiver:" + game.MatchRequest.ReceivingUser.UserId
// 		_, err = conn.Do("HSET", key, game.ID(), request)
// 		if err != nil {
// 			return err
// 		}
// 		// Might as well expire this one too.
// 		_, err = conn.Do("EXPIRE", key, SoughtGameExpiration)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (s *RedisStore) deleteSoughtGame(conn redis.Conn, sg *entity.SoughtGame) error {
// 	var err error
// 	keys := []string{
// 		"soughtgame:" + sg.ID(),
// 		"soughtgamebyconnid:" + sg.ConnID(),
// 		"soughtgamebyuser:" + sg.Seeker(),
// 	}
// 	for _, key := range keys {
// 		_, err = conn.Do("DEL", key)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	if sg.Type() == entity.TypeMatch {
// 		err = s.deleteFromReqsByReceiver(conn, sg)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (s *RedisStore) deleteFromReqsByReceiver(conn redis.Conn, g *entity.SoughtGame) error {
// 	key := "matchrequestbyreceiver:" + g.MatchRequest.ReceivingUser.UserId
// 	_, err := conn.Do("HDEL", key, g.ID())
// 	return err
// }

// // Delete deletes the game by game ID.
// func (s *RedisStore) Delete(ctx context.Context, id string) error {

// 	sg, err := s.Get(ctx, id)
// 	if err != nil {
// 		return err
// 	}

// 	conn := s.redisPool.Get()
// 	defer conn.Close()
// 	return s.deleteSoughtGame(conn, sg)
// }

// // DeleteForUser deletes the game by seeker ID.
// func (s *RedisStore) DeleteForUser(ctx context.Context, userID string) (*entity.SoughtGame, error) {

// 	sg, err := s.getByUser(ctx, userID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	conn := s.redisPool.Get()
// 	defer conn.Close()
// 	return sg, s.deleteSoughtGame(conn, sg)
// }

// // DeleteForConnID deletes the game by connection ID
// func (s *RedisStore) DeleteForConnID(ctx context.Context, connID string) (*entity.SoughtGame, error) {
// 	sg, err := s.GetByConnID(ctx, connID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	conn := s.redisPool.Get()
// 	defer conn.Close()

// 	return sg, s.deleteSoughtGame(conn, sg)
// }

// // ListOpenSeeks lists all open seek requests
// func (s *RedisStore) ListOpenSeeks(ctx context.Context) ([]*entity.SoughtGame, error) {

// }
