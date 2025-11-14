package league

import (
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

// Clock provides the current time, allowing for dependency injection in tests
type Clock interface {
	Now() time.Time
}

// RealClock implements Clock using the system time
type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now()
}

// FixedClock implements Clock using a fixed time (useful for testing)
type FixedClock struct {
	time time.Time
}

func (fc FixedClock) Now() time.Time {
	return fc.time
}

// NewFixedClock creates a Clock that always returns the specified time
func NewFixedClock(t time.Time) Clock {
	return FixedClock{time: t}
}

// NewClockFromEnv creates a Clock based on the LEAGUE_NOW environment variable
// If LEAGUE_NOW is set with an ISO 8601 format time (e.g., "2025-01-15T08:00:00Z"),
// it returns a FixedClock with that time.
// Otherwise, it returns a RealClock that uses the system time.
func NewClockFromEnv() Clock {
	envNow := os.Getenv("LEAGUE_NOW")
	if envNow == "" {
		return RealClock{}
	}

	// Try parsing as RFC3339 (ISO 8601)
	t, err := time.Parse(time.RFC3339, envNow)
	if err != nil {
		log.Warn().Err(err).Str("LEAGUE_NOW", envNow).Msg("failed to parse LEAGUE_NOW, using real time")
		return RealClock{}
	}

	log.Info().Time("fixedTime", t).Msg("using fixed time from LEAGUE_NOW environment variable")
	return NewFixedClock(t)
}
