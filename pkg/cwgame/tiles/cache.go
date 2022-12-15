package tiles

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/cwgame/cache"
)

var CacheKeyPrefix = "letterdist:"

// CacheLoadFunc is the function that loads an object into the global cache.
func CacheLoadFunc(cfg *config.Config, key string) (interface{}, error) {
	dist := strings.TrimPrefix(key, CacheKeyPrefix)
	return NamedLetterDistribution(cfg, dist)
}

// NamedLetterDistribution loads a letter distribution by name.
func NamedLetterDistribution(cfg *config.Config, name string) (*LetterDistribution, error) {
	name = strings.ToLower(name)
	filename := filepath.Join(cfg.DataPath, "letterdistributions", name+".csv")

	file, err := cache.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	d, err := ScanLetterDistribution(file)
	if err != nil {
		return nil, err
	}
	d.Name = name
	return d, nil
}

// CacheReadFunc converts raw data when populating the global cache
func CacheReadFunc(data []byte) (interface{}, error) {
	stream := bytes.NewReader(data)
	return ScanLetterDistribution(stream)
}

// Set loads an alphabet from bytes and populates the cache
func Set(name string, data []byte) error {
	key := CacheKeyPrefix + name
	return cache.Populate(key, data, CacheReadFunc)
}

// Get loads a named alphabet from the cache or from a file
func GetDistribution(cfg *config.Config, name string) (*LetterDistribution, error) {
	key := CacheKeyPrefix + name
	obj, err := cache.Load(cfg, key, CacheLoadFunc)
	if err != nil {
		return nil, err
	}
	ret, ok := obj.(*LetterDistribution)
	if !ok {
		return nil, errors.New("could not read letter distribution from file")
	}
	ret.Name = name
	return ret, nil
}
