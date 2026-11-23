package mod

import (
	"testing"

	"github.com/matryer/is"
)

func TestPagePath(t *testing.T) {
	is := is.New(t)
	for in, want := range map[string]string{
		"https://woogles.io/club/philly/?x=1":  "/club/philly",
		"woogles.io/leagues/smoke#standings":   "/leagues/smoke",
		"/game/abc":                            "/game/abc",
		"/":                                    "/",
		"philly":                               "philly",
		"Philly Mondays":                       "Philly Mondays",
		"8f2c1b3e-0000-4000-8000-000000000000": "8f2c1b3e-0000-4000-8000-000000000000",
	} {
		is.Equal(pagePath(in), want)
	}
}
