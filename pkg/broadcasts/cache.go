package broadcasts

import (
	"context"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/sync/singleflight"
)

// cachedFetch returns the cached value for key if present, else calls compute
// under singleflight to coalesce concurrent misses into a single backend call.
// sfKey is a string used as the singleflight dedup key (typically a short
// label + the cache key serialized as a string).
func cachedFetch[K comparable, V any](
	ctx context.Context,
	cache *expirable.LRU[K, V],
	sf *singleflight.Group,
	key K,
	sfKey string,
	compute func(context.Context) (V, error),
) (V, error) {
	if v, ok := cache.Get(key); ok {
		return v, nil
	}
	v, err, _ := sf.Do(sfKey, func() (any, error) {
		// Re-check: a parallel caller may have already populated the cache.
		if v, ok := cache.Get(key); ok {
			return v, nil
		}
		out, cErr := compute(ctx)
		if cErr != nil {
			return out, cErr
		}
		cache.Add(key, out)
		return out, nil
	})
	if err != nil {
		var zero V
		return zero, err
	}
	return v.(V), nil //nolint:forcetypeassert
}
