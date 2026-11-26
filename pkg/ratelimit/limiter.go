// Package ratelimit limits API requests per user, or per IP for anonymous
// callers, with token buckets kept in Valkey.
package ratelimit

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
)

//go:embed gcra.lua
var gcraScript string

// Limit is a token bucket: Rate requests per Per, refilling evenly, with up
// to Burst at once.
type Limit struct {
	Rate  int
	Per   time.Duration
	Burst int
}

func (l Limit) intervalMs() int64 {
	ms := l.Per.Milliseconds() / int64(l.Rate)
	if ms < 1 {
		ms = 1
	}
	return ms
}

// Check is one bucket to take a token from.
type Check struct {
	Key   string
	Limit Limit
}

// Result says whether a request may go ahead, and if not, when to retry.
type Result struct {
	Allowed    bool
	RetryAfter time.Duration
}

// Checker checks a request against its buckets.
type Checker interface {
	Allow(ctx context.Context, checks []Check) (Result, error)
}

// Limiter is a Checker backed by Valkey.
type Limiter struct {
	pool    *redis.Pool
	script  *redis.Script
	timeout time.Duration
}

// NewLimiter connects to the Valkey server at url, with its own small pool and
// short timeouts, so a slow server can't hold up requests for long.
func NewLimiter(url string) *Limiter {
	const timeout = 100 * time.Millisecond
	return &Limiter{
		pool: &redis.Pool{
			MaxIdle:     16,
			IdleTimeout: 240 * time.Second,
			DialContext: func(ctx context.Context) (redis.Conn, error) {
				return redis.DialURLContext(ctx, url,
					redis.DialConnectTimeout(timeout),
					redis.DialReadTimeout(timeout),
					redis.DialWriteTimeout(timeout))
			},
		},
		script:  redis.NewScript(-1, gcraScript),
		timeout: timeout,
	}
}

// Allow takes a token from every bucket if all have one.
func (l *Limiter) Allow(ctx context.Context, checks []Check) (Result, error) {
	if len(checks) == 0 {
		return Result{Allowed: true}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()
	conn, err := l.pool.GetContext(ctx)
	if err != nil {
		return Result{}, err
	}
	defer conn.Close()

	args := make([]interface{}, 0, 1+3*len(checks))
	args = append(args, len(checks))
	for _, c := range checks {
		args = append(args, c.Key)
	}
	for _, c := range checks {
		if c.Limit.Rate <= 0 || c.Limit.Burst <= 0 {
			return Result{}, fmt.Errorf("bad limit for %s", c.Key)
		}
		args = append(args, c.Limit.intervalMs(), c.Limit.Burst)
	}
	reply, err := redis.Int64s(l.script.DoContext(ctx, conn, args...))
	if err != nil {
		return Result{}, err
	}
	if len(reply) != 2 {
		return Result{}, errors.New("unexpected rate limit reply")
	}
	return Result{Allowed: reply[0] == 1, RetryAfter: time.Duration(reply[1]) * time.Millisecond}, nil
}
