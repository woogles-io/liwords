package cache

import (
	"bytes"
	"io"
	"os"
	"sync"

	"github.com/domino14/liwords/pkg/config"
	"github.com/rs/zerolog/log"
)

// The cache is a package used for generic shared objects that we want to
// cache, for example, gaddags, alphabets, etc.

type cache struct {
	sync.Mutex
	objects map[string]interface{}
}

type loadFunc func(cfg *config.Config, key string) (interface{}, error)

type readFunc func(data []byte) (interface{}, error)

// GlobalObjectCache is our global object cache, of course.
var GlobalObjectCache *cache

func (c *cache) load(cfg *config.Config, key string, loadFunc loadFunc) error {
	log.Debug().Str("key", key).Msg("loading into cache")

	obj, err := loadFunc(cfg, key)
	if err != nil {
		return err
	}
	c.objects[key] = obj

	return nil
}

func (c *cache) get(cfg *config.Config, key string, loadFunc loadFunc, needToLock bool) (interface{}, error) {

	var ok bool
	var obj interface{}
	if needToLock {
		c.Lock()
		defer c.Unlock()
	}
	if obj, ok = c.objects[key]; !ok {
		err := c.load(cfg, key, loadFunc)
		if err != nil {
			return nil, err
		}
		return c.objects[key], nil
	}
	log.Debug().Str("key", key).Msg("getting obj from cache")

	return obj, nil
}

func (c *cache) put(key string, obj interface{}) {
	c.Lock()
	c.objects[key] = obj
	c.Unlock()
}

func init() {
	GlobalObjectCache = &cache{objects: make(map[string]interface{})}
}

func Load(cfg *config.Config, name string, loadFunc loadFunc) (interface{}, error) {
	return GlobalObjectCache.get(cfg, name, loadFunc, true)
}

func Open(filename string) (io.ReadCloser, error) {
	// Most of the time, it seems we are already holding the lock here.
	// It would deadlock to lock again, so we avoid it.
	// Hopefully it works.

	cached, err := GlobalObjectCache.get(nil, "file:"+filename,
		func(*config.Config, string) (interface{}, error) {
			return nil, os.ErrNotExist
		}, false)
	if err != nil {
		// Intentionally not caching.
		log.Debug().Str("filename", filename).Msg("not caching, opening from filesystem")
		return os.Open(filename)
	}
	log.Debug().Str("filename", filename).Msg("reading from cache")
	return io.NopCloser(bytes.NewReader(cached.([]byte))), nil
}

func Precache(filename string, rawBytes []byte) {
	log.Debug().Str("filename", filename).Msg("populating into cache")
	Load(nil, "file:"+filename,
		func(*config.Config, string) (interface{}, error) {
			return rawBytes, nil
		})
}

func Populate(name string, data []byte, readFunc readFunc) error {
	obj, err := readFunc(data)
	if err != nil {
		return err
	}
	GlobalObjectCache.put(name, obj)
	return nil
}
