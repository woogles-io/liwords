package stores

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/redigo"
	"github.com/golang/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/rpc/api/proto/ipc"
)

const MaxExpirationSeconds = 5 * 24 * 60 * 60 // 5 days
const RedisExpirationSeconds = 15 * 60        // 15 minutes
const RedisDocPrefix = "gdoc:"
const RedisMutexPrefix = "gdocmutex:"

// MaybeLockedDocument wraps a game document but also contains a value. If the
// value is not blank then the document is locked.
type MaybeLockedDocument struct {
	*ipc.GameDocument
	LockValue string
}

type GameDocumentStore struct {
	redisPool *redis.Pool
	s3Client  *s3.Client
	redsync   *redsync.Redsync
}

func NewGameDocumentStore(r *redis.Pool, s *s3.Client) (*GameDocumentStore, error) {
	pool := redigo.NewPool(r)
	rs := redsync.New(pool)
	return &GameDocumentStore{redisPool: r, s3Client: s, redsync: rs}, nil
}

// GetDocument gets a game document from the store. It tries Redis first,
// then S3 if not found in Redis.
// The lock parameter is ignored if this item is not in Redis.
// If it is in Redis, and lock is true, the document is locked. The SetDocument
// function will try to unlock it.
// If it locked, we return the lock value for future usage.
func (gs *GameDocumentStore) GetDocument(ctx context.Context, uuid string, lock bool) (*MaybeLockedDocument, error) {
	var mutexName string
	var mutex *redsync.Mutex

	conn := gs.redisPool.Get()
	defer conn.Close()

	res, err := redis.Int(conn.Do("EXISTS", RedisDocPrefix+uuid))
	if err != nil {
		return nil, err
	}
	if res == 0 {
		// Does not exist in Redis. Try S3.
		doc, err := gs.getFromS3(ctx, uuid)
		if err != nil {
			return nil, err
		}
		return &MaybeLockedDocument{GameDocument: doc}, nil
	}

	if lock {
		mutexName = RedisMutexPrefix + uuid
		mutex = gs.redsync.NewMutex(mutexName)
		if err := mutex.Lock(); err != nil {
			log.Err(err).Msg("lock failed")
			return nil, err
		}
		log.Debug().Str("name", mutex.Name()).Str("val", mutex.Value()).Msg("locked mutx")
	}
	log.Debug().Msg("getting document")
	bts, err := redis.Bytes(conn.Do("GET", RedisDocPrefix+uuid))
	if err != nil {
		if lock {
			mutex.Unlock()
		}
		return nil, err
	}
	gdoc := &ipc.GameDocument{}
	err = proto.Unmarshal(bts, gdoc)
	if err != nil {
		if lock {
			mutex.Unlock()
		}
		return nil, err
	}
	log.Debug().Msg("returning document")

	// Don't unlock the mutex when we leave. We will unlock it after the
	// SetDocument operation. (Or it will expire if there is no such operation)
	var mv string
	if lock {
		mv = mutex.Value()
	}
	return &MaybeLockedDocument{GameDocument: gdoc, LockValue: mv}, nil
}

func (gs *GameDocumentStore) getFromS3(ctx context.Context, uuid string) (*ipc.GameDocument, error) {
	return nil, errors.New("getFromS3 not implemented")
}

// SetDocument should be called to set the initial document in redis.
func (gs *GameDocumentStore) SetDocument(ctx context.Context, gdoc *ipc.GameDocument) error {
	bts, err := proto.Marshal(gdoc)
	if err != nil {
		return err
	}
	gid := gdoc.Uid
	conn := gs.redisPool.Get()
	defer conn.Close()

	r, err := redis.String(conn.Do("SET", RedisDocPrefix+gid, bts, "EX", MaxExpirationSeconds))
	if err != nil {
		return err
	}
	if r != "OK" {
		return errors.New("wrong return for SET: " + r)
	}
	return nil
}

// UpdateDocument makes an atomic update to document in the Redis store.
// If the game is done, though, it will write it to S3 and expire it from the Redis
// store.
// UpdateDocument should never be called without a previous lock.
func (gs *GameDocumentStore) UpdateDocument(ctx context.Context, doc *MaybeLockedDocument) error {
	saveToS3 := doc.PlayState == ipc.PlayState_GAME_OVER

	bts, err := proto.Marshal(doc.GameDocument)
	if err != nil {
		return err
	}
	gid := doc.Uid
	conn := gs.redisPool.Get()
	defer conn.Close()

	defer func() {
		mutex := gs.redsync.NewMutex(RedisMutexPrefix+gid,
			redsync.WithValue(doc.LockValue))
		if ok, err := mutex.Unlock(); !ok || err != nil {
			// The unlock failed. Maybe it wasn't locked?
			log.Err(err).Str("mutexname", mutex.Name()).Str("val", mutex.Value()).Msg("redsync-unlock-failed")
		}

		if saveToS3 {
			gs.saveToS3(ctx, doc.GameDocument)
		}
	}()

	expTimer := MaxExpirationSeconds
	if saveToS3 {
		expTimer = RedisExpirationSeconds
	}

	r, err := redis.String(conn.Do("SET", RedisDocPrefix+gid, bts, "EX", expTimer))
	if err != nil {
		return err
	}
	if r != "OK" {
		return errors.New("wrong return for SET: " + r)
	}

	return nil
}

func (gs *GameDocumentStore) saveToS3(ctx context.Context, gdoc *ipc.GameDocument) error {
	// save as protojson
	return errors.New("saveToS3 not implemented")
}
