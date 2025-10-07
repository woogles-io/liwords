package sockets

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/rs/zerolog/log"
)

// FollowsCache caches follow lists to reduce API calls
var FollowsCache *expirable.LRU[string, []string]

// InitFollowsCache initializes the global follows cache
func InitFollowsCache(size int, ttl time.Duration) {
	if size == 0 {
		FollowsCache = nil
		log.Info().Msg("follows-cache-disabled")
		return
	}
	FollowsCache = expirable.NewLRU[string, []string](size, onFollowsEvicted, ttl)
	log.Info().Int("cache_size", size).Dur("ttl", ttl).Msg("initialized-follows-cache")
}

// onFollowsEvicted is called when an entry is evicted from the cache
func onFollowsEvicted(userID string, follows []string) {
	log.Debug().Str("userID", userID).Int("follows_count", len(follows)).
		Msg("evicted-follows-from-cache")
}

// GetFollows retrieves follows from cache, returns nil if not found
func GetFollows(userID string) ([]string, bool) {
	if FollowsCache == nil {
		return nil, false
	}

	follows, found := FollowsCache.Get(userID)
	if found {
		log.Debug().Str("userID", userID).Int("follows_count", len(follows)).
			Msg("cache-hit-follows")
		return follows, true
	}

	log.Debug().Str("userID", userID).Msg("cache-miss-follows")
	return nil, false
}

// CacheFollows stores follows list in cache
func CacheFollows(userID string, follows []string) {
	if FollowsCache == nil {
		return
	}

	evicted := FollowsCache.Add(userID, follows)
	log.Debug().Str("userID", userID).Int("follows_count", len(follows)).
		Bool("evicted", evicted).Msg("cached-follows")
}

// GetFollowsCacheStats returns cache statistics for monitoring
func GetFollowsCacheStats() int {
	if FollowsCache == nil {
		return 0
	}
	return FollowsCache.Len()
}
