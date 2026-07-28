package broadcasts

import (
	"encoding/json"
	"testing"
)

// TestParseUserOBSPath pins the single URL shape. The two-segment form is the
// retired alias — it must be rejected rather than silently reinterpreted, so
// whoever still has one in OBS gets legacyAliasMessage naming its replacement.
func TestParseUserOBSPath(t *testing.T) {
	tests := []struct {
		tail       string
		wantUser   string
		wantSlot   string
		wantSuffix string
		wantOK     bool
	}{
		// Stream slot — the only accepted shape.
		{"cesar/stream1/score", "cesar", "stream1", "score", true},
		{"cesar/stream1/score.txt", "cesar", "stream1", "score.txt", true},
		{"cesar/stream1/events", "cesar", "stream1", "events", true},
		{"letsplayscrabble/main/p1_record", "letsplayscrabble", "main", "p1_record", true},

		// A slot named like a suffix is unambiguous: arity decides.
		{"cesar/score/score", "cesar", "score", "score", true},

		// Retired two-segment alias.
		{"cesar/score", "", "", "", false},
		{"cesar/score.txt", "", "", "", false},
		{"cesar/events", "", "", "", false},

		// Malformed.
		{"", "", "", "", false},
		{"cesar", "", "", "", false},
		{"cesar/", "", "", "", false},
		{"/score", "", "", "", false},
		{"cesar//score", "", "", "", false},
		{"cesar/stream1/", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.tail, func(t *testing.T) {
			user, slot, suffix, ok := parseUserOBSPath(tt.tail)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if user != tt.wantUser || slot != tt.wantSlot || suffix != tt.wantSuffix {
				t.Errorf("got (%q, %q, %q), want (%q, %q, %q)",
					user, slot, suffix, tt.wantUser, tt.wantSlot, tt.wantSuffix)
			}
		})
	}
}

// TestPlaceholderOBSDataCoversEveryField walks OBSData's JSON representation so
// a newly added field can't silently ship as "" — which in OBS renders as a
// blank overlay rather than the visible "(no game assigned)" marker.
func TestPlaceholderOBSDataCoversEveryField(t *testing.T) {
	bts, err := json.Marshal(placeholderOBSData())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var fields map[string]string
	if err := json.Unmarshal(bts, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(fields) == 0 {
		t.Fatal("no fields found in OBSData")
	}
	for name, val := range fields {
		if val != obsPlaceholder {
			t.Errorf("field %q = %q, want %q — add it to placeholderOBSData", name, val, obsPlaceholder)
		}
	}
}

// TestClearStreamPushesPlaceholder covers the frozen-overlay bug: when a slot
// stops resolving to a game, live subscribers must be told. An SSE page only
// repaints on a message, so silence leaves the previous game on screen
// indefinitely — and a fresh load of the same URL would show the placeholder,
// so two browser sources on one URL would disagree.
func TestClearStreamPushesPlaceholder(t *testing.T) {
	h := &OBSHandler{streams: make(map[string]*obsStream)}
	const key = "albany-open/featured"

	ch := make(chan OBSData, 8)
	stream := h.addSubscriber(key, ch)
	defer h.removeSubscriber(key, ch)

	// The stream is showing a game; its slot then moves to an unclaimed table.
	stream.mu.Lock()
	stream.currentGameID = "game-abc"
	stream.mu.Unlock()

	h.clearStream(stream, key)

	select {
	case got := <-ch:
		if got.Score != obsPlaceholder || got.P1Name != obsPlaceholder {
			t.Errorf("cleared payload = %+v, want every field %q", got, obsPlaceholder)
		}
	default:
		t.Fatal("no data pushed to subscriber — the overlay would stay frozen on the old game")
	}

	stream.mu.Lock()
	gameID := stream.currentGameID
	stream.mu.Unlock()
	if gameID != "" {
		t.Errorf("currentGameID = %q, want empty", gameID)
	}

	// Already empty: clearing again must stay silent, or every unrelated
	// broadcast event would re-push the placeholder to idle streams.
	h.clearStream(stream, key)
	select {
	case got := <-ch:
		t.Errorf("second clear pushed %+v, want nothing", got)
	default:
	}
}

// TestStreamSlotNameRe guards the names that end up as a path segment in a
// permanent OBS URL.
func TestStreamSlotNameRe(t *testing.T) {
	valid := []string{"a", "1", "main", "stream1", "stream-1", "lps-main-board",
		"abcdefghijabcdefghijabcdefghijab"} // 32 chars
	for _, s := range valid {
		if !streamSlotNameRe.MatchString(s) {
			t.Errorf("%q should be valid", s)
		}
	}

	invalid := []string{
		"",                                  // empty
		"Main",                              // uppercase would 404 against the lowercase path segment
		"-main",                             // leading dash
		"my slot",                           // space
		"main/sub",                          // path separator
		"main.txt",                          // would collide with the .txt variant
		"abcdefghijabcdefghijabcdefghijabc", // 33 chars
	}
	for _, s := range invalid {
		if streamSlotNameRe.MatchString(s) {
			t.Errorf("%q should be invalid", s)
		}
	}
}
