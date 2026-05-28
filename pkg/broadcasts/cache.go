package broadcasts

import (
	"context"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/singleflight"
)

// cachedFetch returns the cached value for key if present, else calls compute
// under singleflight to coalesce concurrent misses into a single backend call.
// cacheName is a short label (e.g. "stats", "slots") used in trace spans.
// sfKey is the singleflight dedup key.
func cachedFetch[K comparable, V any](
	ctx context.Context,
	cache *expirable.LRU[K, V],
	sf *singleflight.Group,
	key K,
	sfKey string,
	cacheName string,
	compute func(context.Context) (V, error),
) (V, error) {
	tracer := otel.Tracer("broadcast-rpc-cache")
	ctx, span := tracer.Start(ctx, "broadcast.rpc_cache.get")
	defer span.End()
	span.SetAttributes(attribute.String("cache.name", cacheName))

	if v, ok := cache.Get(key); ok {
		span.SetAttributes(attribute.Bool("cache.hit", true))
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

	span.SetAttributes(attribute.Bool("cache.hit", false))
	if err != nil {
		var zero V
		return zero, err
	}
	return v.(V), nil //nolint:forcetypeassert
}
