package broadcasts

import (
	"testing"
	"time"
)

func TestCacheBustedURL(t *testing.T) {
	now := time.Unix(0, 1234)
	tests := map[string]string{
		"https://co15.site/reno2026/html/tourney.js": "https://co15.site/reno2026/html/tourney.js?_=1234",
		"https://example.com/feed.js?div=A&b=1":      "https://example.com/feed.js?div=A&b=1&_=1234",
		"https://example.com/feed.js?div=A#frag":     "https://example.com/feed.js?div=A&_=1234#frag",
		"https://example.com/a%20b/tourney.js?x=%2F": "https://example.com/a%20b/tourney.js?x=%2F&_=1234",
	}
	for in, want := range tests {
		if got := cacheBustedURL(in, now); got != want {
			t.Errorf("cacheBustedURL(%q) = %q, want %q", in, got, want)
		}
	}
}
