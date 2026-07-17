package broadcasts

import "testing"

// standingsFeedData returns a 3-player, 3-round FeedData with a clean,
// hand-computed win/loss/spread outcome:
//
//	Round 0: Alice beat Bob     400-350  (Carol had no game)
//	Round 1: Alice beat Carol   380-300  (Bob had no game)
//	Round 2: Bob beat Carol     360-340  (Alice had no game)
//
// Standings (wins desc, then spread desc): Alice 2-0 (+130), Bob 1-1 (-30),
// Carol 0-2 (-100).
func standingsFeedData() *FeedData {
	return &FeedData{
		DivisionName: "A",
		TotalRounds:  3,
		CurrentRound: 3,
		Players: []FeedPlayer{
			{ID: 1, Name: "Alice", Rating: 1500, Scores: []int{400, 380, 0}, Pairings: []int{2, 3, 0}},
			{ID: 2, Name: "Bob", Rating: 1400, Scores: []int{350, 0, 360}, Pairings: []int{1, 0, 3}},
			{ID: 3, Name: "Carol", Rating: 1300, Scores: []int{0, 300, 340}, Pairings: []int{0, 1, 2}},
		},
	}
}

func TestPlayerStandingFields(t *testing.T) {
	fd := standingsFeedData()

	tests := []struct {
		name       string
		wantRecord string
		wantPlace  string
		wantSpread string
		wantRating string
	}{
		{"Alice", "2-0", "1st", "+130", "1500"},
		{"Bob", "1-1", "2nd", "-30", "1400"},
		{"Carol", "0-2", "3rd", "-100", "1300"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, place, spread, rating := playerStandingFields(fd, tt.name)
			if record != tt.wantRecord {
				t.Errorf("record = %q, want %q", record, tt.wantRecord)
			}
			if place != tt.wantPlace {
				t.Errorf("place = %q, want %q", place, tt.wantPlace)
			}
			if spread != tt.wantSpread {
				t.Errorf("spread = %q, want %q", spread, tt.wantSpread)
			}
			if rating != tt.wantRating {
				t.Errorf("rating = %q, want %q", rating, tt.wantRating)
			}
		})
	}

	t.Run("unknown player name falls back to placeholder", func(t *testing.T) {
		record, place, spread, rating := playerStandingFields(fd, "Dave")
		for _, got := range []string{record, place, spread, rating} {
			if got != obsPlaceholder {
				t.Errorf("got %q, want placeholder %q", got, obsPlaceholder)
			}
		}
	})

	t.Run("nil feed falls back to placeholder", func(t *testing.T) {
		record, place, spread, rating := playerStandingFields(nil, "Alice")
		for _, got := range []string{record, place, spread, rating} {
			if got != obsPlaceholder {
				t.Errorf("got %q, want placeholder %q", got, obsPlaceholder)
			}
		}
	})
}

func TestApplyTournamentFields(t *testing.T) {
	fd := standingsFeedData()
	data := OBSData{P1Name: "Alice", P2Name: "Bob"}
	applyTournamentFields(&data, fd, "A Division", 2, 5, "Albany Open 2026", data.P1Name, data.P2Name)

	if data.Division != "A Division" {
		t.Errorf("Division = %q", data.Division)
	}
	if data.Tournament != "Albany Open 2026" {
		t.Errorf("Tournament = %q", data.Tournament)
	}
	if data.Table != "5" {
		t.Errorf("Table = %q", data.Table)
	}
	if data.Round != "2 of 3" {
		t.Errorf("Round = %q", data.Round)
	}
	if data.P1Record != "2-0" || data.P1Place != "1st" || data.P1Spread != "+130" || data.P1Rating != "1500" {
		t.Errorf("p1 fields wrong: record=%q place=%q spread=%q rating=%q",
			data.P1Record, data.P1Place, data.P1Spread, data.P1Rating)
	}
	if data.P2Record != "1-1" || data.P2Place != "2nd" || data.P2Spread != "-30" || data.P2Rating != "1400" {
		t.Errorf("p2 fields wrong: record=%q place=%q spread=%q rating=%q",
			data.P2Record, data.P2Place, data.P2Spread, data.P2Rating)
	}
}

func TestApplyTournamentFieldsNilFeed(t *testing.T) {
	data := OBSData{P1Name: "Alice", P2Name: "Bob"}
	applyTournamentFields(&data, nil, "A Division", 2, 5, "", "Alice", "Bob")

	if data.Division != "A Division" {
		t.Errorf("Division = %q, want unaffected by missing feed", data.Division)
	}
	if data.Table != "5" {
		t.Errorf("Table = %q", data.Table)
	}
	if data.Tournament != obsPlaceholder {
		t.Errorf("Tournament = %q, want placeholder for empty name", data.Tournament)
	}
	if data.Round != "2" {
		t.Errorf(`Round = %q, want "2" (no total-rounds available without a feed)`, data.Round)
	}
	for _, got := range []string{data.P1Record, data.P1Place, data.P1Spread, data.P1Rating} {
		if got != obsPlaceholder {
			t.Errorf("expected placeholder per-player fields with nil feed, got %q", got)
		}
	}
}

func TestFormatSpread(t *testing.T) {
	tests := []struct {
		spread int
		want   string
	}{
		{130, "+130"},
		{0, "0"},
		{-30, "-30"},
	}
	for _, tt := range tests {
		if got := formatSpread(tt.spread); got != tt.want {
			t.Errorf("formatSpread(%d) = %q, want %q", tt.spread, got, tt.want)
		}
	}
}

func TestFormatRecord(t *testing.T) {
	if got := formatRecord(6, 1); got != "6-1" {
		t.Errorf("formatRecord(6, 1) = %q, want %q", got, "6-1")
	}
	if got := formatRecord(5.5, 1.5); got != "5.5-1.5" {
		t.Errorf("formatRecord(5.5, 1.5) = %q, want %q", got, "5.5-1.5")
	}
}
