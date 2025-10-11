package sockets

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/rs/zerolog/log"
)

// FollowersCache caches follower lists to reduce API calls
var FollowersCache *expirable.LRU[string, []string]

// InitFollowersCache initializes the global followers cache
func InitFollowersCache(size int, ttl time.Duration) {
	if size == 0 {
		FollowersCache = nil
		log.Info().Msg("followers-cache-disabled")
		return
	}
	FollowersCache = expirable.NewLRU[string, []string](size, onEvicted, ttl)
	log.Info().Int("cache_size", size).Dur("ttl", ttl).Msg("initialized-followers-cache")
}

// onEvicted is called when an entry is evicted from the cache
func onEvicted(userID string, followers []string) {
	log.Debug().Str("userID", userID).Int("follower_count", len(followers)).
		Msg("evicted-followers-from-cache")
}

// GetFollowers retrieves followers from cache, returns nil if not found
func GetFollowers(userID string) ([]string, bool) {
	if FollowersCache == nil {
		return nil, false
	}

	followers, found := FollowersCache.Get(userID)
	if found {
		log.Debug().Str("userID", userID).Int("follower_count", len(followers)).
			Msg("cache-hit-followers")
		return followers, true
	}

	log.Debug().Str("userID", userID).Msg("cache-miss-followers")
	return nil, false
}

// CacheFollowers stores followers list in cache
func CacheFollowers(userID string, followers []string) {
	if FollowersCache == nil {
		return
	}

	evicted := FollowersCache.Add(userID, followers)
	log.Debug().Str("userID", userID).Int("follower_count", len(followers)).
		Bool("evicted", evicted).Msg("cached-followers")
}

// GetCacheStats returns cache statistics for monitoring
func GetCacheStats() int {
	if FollowersCache == nil {
		return 0
	}
	return FollowersCache.Len()
}
