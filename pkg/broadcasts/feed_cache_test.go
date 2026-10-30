package broadcasts

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestGetCachedEntryRemembersFailures(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.NotFound(w, r)
	}))
	defer srv.Close()

	bs := &BroadcastService{feedCache: make(map[string]*feedCacheEntry)}
	for range 3 {
		if entry := bs.getCachedEntry("dead", srv.URL, "tsh_newt_json"); entry != nil {
			t.Fatalf("expected nil entry for a failing feed, got %+v", entry)
		}
	}
	if n := hits.Load(); n != 1 {
		t.Fatalf("expected the failing feed to be fetched once, got %d", n)
	}
	if entry := bs.peekCachedEntry("dead"); entry != nil {
		t.Fatalf("peek returned a failed entry: %+v", entry)
	}
}

func TestPeekCachedEntryDoesNotFetch(t *testing.T) {
	bs := &BroadcastService{feedCache: make(map[string]*feedCacheEntry)}
	if entry := bs.peekCachedEntry("missing"); entry != nil {
		t.Fatalf("expected nil for an uncached slug, got %+v", entry)
	}

	divs := map[string]*FeedData{"A": {CurrentRound: 3}}
	bs.setCachedFeed("live", divs)
	entry := bs.peekCachedEntry("live")
	if entry == nil || entry.divisions["A"].CurrentRound != 3 {
		t.Fatalf("expected the cached entry, got %+v", entry)
	}
}
