package broadcasts

import (
	"os"
	"testing"
)

// minimalTSHFeed is a minimal but valid tsh tourney.js fixture with 3 players,
// 3 rounds, and realistic board/pairing data.
//
// Player layout:
//   1: Alice  (id=1)
//   2: Bob    (id=2)
//   3: Carol  (id=3)
//
// Round 1: Alice(1) vs Bob(2) at table 2, Carol(3) has bye
// Round 2: Alice(1) vs Carol(3) at table 1, Bob(2) has bye
// Round 3: Bob(2) vs Carol(3) at table 3 (unplayed), Alice(1) has bye
//
// Scores (0-indexed):
//   Alice:  [400, 350, 0]
//   Bob:    [380, 0,   0]
//   Carol:  [50,  370, 0]  (50 = bye score for round 1)
const minimalTSHFeed = `newt={"config":{"event_name":"Test Tournament","max_rounds":3},"divisions":[{"players":[null,` +
	`{"id":1,"name":"Alice","rating":1500,"scores":[400,350,0],"pairings":[2,3,0],"etc":{"board":[2,1,0],"p12":[1,2,0]}},` +
	`{"id":2,"name":"Bob","rating":1400,"scores":[380,0,0],"pairings":[1,0,3],"etc":{"board":[2,0,3],"p12":[2,0,2]}},` +
	`{"id":3,"name":"Carol","rating":1300,"scores":[50,370,0],"pairings":[0,1,2],"etc":{"board":[0,1,3],"p12":[0,1,1]}}` +
	`]}]};`

// undefinedTSHFeed tests that JS undefined values are handled gracefully.
const undefinedTSHFeed = `newt={"config":{},"divisions":[{"players":[null,` +
	`{"id":1,"name":"Alice","rating":undefined,"scores":[400,0],"pairings":[2,0],"etc":{"board":[1,0],"p12":[1,0]}},` +
	`{"id":2,"name":"Bob","rating":1400,"scores":[380,0],"pairings":[1,0],"etc":{"board":[1,0],"p12":[2,0]}}` +
	`]}]};`

func TestExtractNewtJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"standard format", `newt={"key":"value"};`, false},
		{"no trailing semicolon", `newt={"key":"value"}`, false},
		{"var prefix", `var newt={"key":"value"};`, false},
		{"with leading content", `// some comment\nnewt={"key":"value"};`, false},
		{"already extracted json", `{"key":"value"}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractNewtJSON([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("extractNewtJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(got) == 0 {
				t.Error("expected non-empty JSON bytes")
			}
		})
	}
}

func TestTSHNewtParser_Parse(t *testing.T) {
	p := &TSHNewtParser{}
	fd, err := p.Parse([]byte(minimalTSHFeed))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(fd.Players) != 3 {
		t.Fatalf("expected 3 players, got %d", len(fd.Players))
	}
	if fd.TotalRounds != 3 {
		t.Errorf("expected TotalRounds=3, got %d", fd.TotalRounds)
	}
	// Rounds 1 and 2 have scores, round 3 is unplayed → current round = 2
	if fd.CurrentRound != 2 {
		t.Errorf("expected CurrentRound=2, got %d", fd.CurrentRound)
	}

	alice := findPlayer(fd.Players, 1)
	if alice == nil {
		t.Fatal("Alice (id=1) not found")
	}
	if alice.Name != "Alice" {
		t.Errorf("expected name Alice, got %s", alice.Name)
	}
	if len(alice.Boards) != 3 || alice.Boards[0] != 2 || alice.Boards[1] != 1 {
		t.Errorf("unexpected board data: %v", alice.Boards)
	}
}

func TestTSHNewtParser_ParseUndefined(t *testing.T) {
	p := &TSHNewtParser{}
	fd, err := p.Parse([]byte(undefinedTSHFeed))
	if err != nil {
		t.Fatalf("Parse() with undefined values should not error: %v", err)
	}
	if len(fd.Players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(fd.Players))
	}
}

func TestGetRoundPairings_Round1(t *testing.T) {
	p := &TSHNewtParser{}
	fd, err := p.Parse([]byte(minimalTSHFeed))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	pairings := GetRoundPairings(fd, 1)
	// Round 1: Alice vs Bob at table 2. Carol has a bye (opponent=0) → skipped.
	if len(pairings) != 1 {
		t.Fatalf("expected 1 pairing in round 1, got %d: %+v", len(pairings), pairings)
	}
	g := pairings[0]
	if g.TableNumber != 2 {
		t.Errorf("expected table 2, got %d", g.TableNumber)
	}
	if g.Player1Name != "Alice" {
		t.Errorf("expected Player1=Alice, got %s", g.Player1Name)
	}
	if g.Player2Name != "Bob" {
		t.Errorf("expected Player2=Bob, got %s", g.Player2Name)
	}
	if g.Player1Score != 400 || g.Player2Score != 380 {
		t.Errorf("unexpected scores: %d vs %d", g.Player1Score, g.Player2Score)
	}
	if !g.Finalized {
		t.Error("expected game to be finalized")
	}
	if !g.Player1GoesFirst {
		t.Error("expected Player1 (Alice) to go first")
	}
}

func TestGetRoundPairings_Round2(t *testing.T) {
	p := &TSHNewtParser{}
	fd, err := p.Parse([]byte(minimalTSHFeed))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	pairings := GetRoundPairings(fd, 2)
	// Round 2: Alice vs Carol at table 1. Bob has a bye → skipped.
	if len(pairings) != 1 {
		t.Fatalf("expected 1 pairing in round 2, got %d: %+v", len(pairings), pairings)
	}
	g := pairings[0]
	if g.TableNumber != 1 {
		t.Errorf("expected table 1, got %d", g.TableNumber)
	}
	if !g.Finalized {
		t.Error("expected game to be finalized")
	}
}

func TestGetRoundPairings_Round3_Unplayed(t *testing.T) {
	p := &TSHNewtParser{}
	fd, err := p.Parse([]byte(minimalTSHFeed))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	pairings := GetRoundPairings(fd, 3)
	// Round 3: Bob vs Carol at table 3, scores = 0 (unplayed).
	if len(pairings) != 1 {
		t.Fatalf("expected 1 pairing in round 3, got %d: %+v", len(pairings), pairings)
	}
	g := pairings[0]
	if g.TableNumber != 3 {
		t.Errorf("expected table 3, got %d", g.TableNumber)
	}
	if g.Finalized {
		t.Error("expected game to NOT be finalized (unplayed)")
	}
}

func TestGetRoundPairings_OutOfRange(t *testing.T) {
	p := &TSHNewtParser{}
	fd, err := p.Parse([]byte(minimalTSHFeed))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got := GetRoundPairings(fd, 0); got != nil {
		t.Errorf("round 0 should return nil, got %v", got)
	}
	if got := GetRoundPairings(fd, 99); got != nil {
		t.Errorf("round 99 should return nil, got %v", got)
	}
}

// ---- sample-tourney.js (multi-division, real tournament data) ----

func parseSampleTourney(t *testing.T, division string) *FeedData {
	t.Helper()
	data, err := os.ReadFile("testdata/sample-tourney.js")
	if err != nil {
		t.Fatalf("could not read sample-tourney.js: %v", err)
	}
	p := &TSHNewtParser{}
	fd, err := p.ParseDivision(data, division)
	if err != nil {
		t.Fatalf("ParseDivision(%q) error: %v", division, err)
	}
	return fd
}

func TestListDivisions_SampleTourney(t *testing.T) {
	data, err := os.ReadFile("testdata/sample-tourney.js")
	if err != nil {
		t.Fatalf("could not read sample-tourney.js: %v", err)
	}
	p := &TSHNewtParser{}
	divs, err := p.ListDivisions(data)
	if err != nil {
		t.Fatalf("ListDivisions() error: %v", err)
	}
	if len(divs) < 2 {
		t.Fatalf("expected at least 2 divisions, got %d: %v", len(divs), divs)
	}
	if divs[0] != "A" || divs[1] != "B" {
		t.Errorf("expected divisions to start with [A B], got %v", divs)
	}
}

func TestSampleTourney_DivA_Structure(t *testing.T) {
	fd := parseSampleTourney(t, "A")

	if fd.DivisionName != "A" {
		t.Errorf("DivisionName = %q, want %q", fd.DivisionName, "A")
	}
	// 18 players in division A
	if len(fd.Players) != 18 {
		t.Errorf("Players = %d, want 18", len(fd.Players))
	}
	// 11-round tournament
	if fd.TotalRounds != 11 {
		t.Errorf("TotalRounds = %d, want 11", fd.TotalRounds)
	}
}

func TestSampleTourney_DivA_CurrentRound(t *testing.T) {
	fd := parseSampleTourney(t, "A")

	// No scores entered yet → tournament hasn't started
	if fd.CurrentRound != 0 {
		t.Errorf("CurrentRound = %d, want 0 (no games played yet)", fd.CurrentRound)
	}
}

func TestSampleTourney_DivA_Round1Pairings(t *testing.T) {
	fd := parseSampleTourney(t, "A")

	pairings := GetRoundPairings(fd, 1)

	// 18 players → 9 games in round 1
	if len(pairings) != 9 {
		t.Fatalf("round 1: got %d pairings, want 9", len(pairings))
	}

	// Pairings must be sorted by table number and tables must be 1–9
	for i, g := range pairings {
		wantTable := i + 1
		if g.TableNumber != wantTable {
			t.Errorf("pairings[%d].TableNumber = %d, want %d", i, g.TableNumber, wantTable)
		}
		// No scores yet
		if g.Finalized {
			t.Errorf("table %d: expected not finalized (tournament not started)", g.TableNumber)
		}
	}

	// Spot-check table 1: Daniel vs Przybyszewski
	g := pairings[0]
	if g.Player1Name != "Daniel, Robin Pollock" {
		t.Errorf("table 1 player1 = %q, want %q", g.Player1Name, "Daniel, Robin Pollock")
	}
	if g.Player2Name != "Przybyszewski, Mark" {
		t.Errorf("table 1 player2 = %q, want %q", g.Player2Name, "Przybyszewski, Mark")
	}
}

func TestSampleTourney_DivA_NotDivB(t *testing.T) {
	// Division B has a different player count — ensure division filter works.
	fdA := parseSampleTourney(t, "A")
	fdB := parseSampleTourney(t, "B")

	if len(fdA.Players) == len(fdB.Players) {
		t.Errorf("div A and div B have the same player count (%d) — division filter may be broken",
			len(fdA.Players))
	}
	if fdB.DivisionName != "B" {
		t.Errorf("fdB.DivisionName = %q, want %q", fdB.DivisionName, "B")
	}
}

func TestSampleTourney_UnknownDivision(t *testing.T) {
	data, _ := os.ReadFile("testdata/sample-tourney.js")
	p := &TSHNewtParser{}
	_, err := p.ParseDivision(data, "Z")
	if err == nil {
		t.Error("expected error for unknown division, got nil")
	}
}

// ---- test22-tourney.js (single division, rounds in progress) ----

func parseTest22(t *testing.T) *FeedData {
	t.Helper()
	data, err := os.ReadFile("testdata/test22-tourney.js")
	if err != nil {
		t.Fatalf("could not read test22-tourney.js: %v", err)
	}
	p := &TSHNewtParser{}
	fd, err := p.Parse(data) // single division, use default Parse
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	return fd
}

func TestTest22_SingleDivision(t *testing.T) {
	data, err := os.ReadFile("testdata/test22-tourney.js")
	if err != nil {
		t.Fatalf("could not read test22-tourney.js: %v", err)
	}
	p := &TSHNewtParser{}
	divs, err := p.ListDivisions(data)
	if err != nil {
		t.Fatalf("ListDivisions() error: %v", err)
	}
	if len(divs) != 1 || divs[0] != "B" {
		t.Errorf("expected [B], got %v", divs)
	}
}

func TestTest22_Structure(t *testing.T) {
	fd := parseTest22(t)

	if fd.DivisionName != "B" {
		t.Errorf("DivisionName = %q, want %q", fd.DivisionName, "B")
	}
	if len(fd.Players) != 12 {
		t.Errorf("Players = %d, want 12", len(fd.Players))
	}
	if fd.TotalRounds != 21 {
		t.Errorf("TotalRounds = %d, want 21", fd.TotalRounds)
	}
}

func TestTest22_CurrentRound(t *testing.T) {
	fd := parseTest22(t)

	// 11 rounds have scores, 10 are still upcoming.
	if fd.CurrentRound != 11 {
		t.Errorf("CurrentRound = %d, want 11", fd.CurrentRound)
	}
}

func TestTest22_Round11Pairings(t *testing.T) {
	fd := parseTest22(t)

	pairings := GetRoundPairings(fd, 11)

	// 12 players → 6 games per round.
	if len(pairings) != 6 {
		t.Fatalf("round 11: got %d pairings, want 6", len(pairings))
	}

	// All games in round 11 should be finalized.
	for _, g := range pairings {
		if !g.Finalized {
			t.Errorf("table %d: expected finalized (round 11 is complete)", g.TableNumber)
		}
	}

	// Tables must be 1–6, sorted.
	for i, g := range pairings {
		if g.TableNumber != i+1 {
			t.Errorf("pairings[%d].TableNumber = %d, want %d", i, g.TableNumber, i+1)
		}
	}

	// Spot-check table 1: Thobani vs Abbasi, Abbasi wins 417-339.
	g := pairings[0]
	if g.Player1Name != "Thobani, Shafique" {
		t.Errorf("table 1 player1 = %q, want %q", g.Player1Name, "Thobani, Shafique")
	}
	if g.Player2Name != "Abbasi, Shan" {
		t.Errorf("table 1 player2 = %q, want %q", g.Player2Name, "Abbasi, Shan")
	}
	if g.Player1Score != 339 || g.Player2Score != 417 {
		t.Errorf("table 1 score = %d-%d, want 339-417", g.Player1Score, g.Player2Score)
	}
	// Spot-check table 2: Adeyeri goes first, wins 484-392.
	g2 := pairings[1]
	if !g2.Player1GoesFirst {
		t.Errorf("table 2: expected player1 (Adeyeri) to go first")
	}
	if g2.Player1Score != 484 || g2.Player2Score != 392 {
		t.Errorf("table 2 score = %d-%d, want 484-392", g2.Player1Score, g2.Player2Score)
	}
}

func TestTest22_UpcomingRounds(t *testing.T) {
	fd := parseTest22(t)

	// TSH only publishes pairings up to the current round — future rounds return nil.
	if got := GetRoundPairings(fd, 12); got != nil {
		t.Errorf("round 12: expected nil (no pairings published yet), got %d games", len(got))
	}

	// Out-of-range also returns nil.
	if got := GetRoundPairings(fd, 22); got != nil {
		t.Errorf("round 22: expected nil (beyond TotalRounds), got %v", got)
	}
}

func TestDetectCurrentRound(t *testing.T) {
	tests := []struct {
		name    string
		players []FeedPlayer
		want    int
	}{
		{
			name:    "no games played",
			players: []FeedPlayer{{ID: 1, Scores: []int{0, 0}, Pairings: []int{2, 0}}},
			want:    0,
		},
		{
			name: "one round played",
			players: []FeedPlayer{
				{ID: 1, Scores: []int{400, 0}, Pairings: []int{2, 0}},
				{ID: 2, Scores: []int{380, 0}, Pairings: []int{1, 0}},
			},
			want: 1,
		},
		{
			name: "two rounds played",
			players: []FeedPlayer{
				{ID: 1, Scores: []int{400, 350}, Pairings: []int{2, 3}},
				{ID: 2, Scores: []int{380, 420}, Pairings: []int{1, 3}},
				{ID: 3, Scores: []int{410, 360}, Pairings: []int{2, 1}},
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectCurrentRound(tt.players)
			if got != tt.want {
				t.Errorf("detectCurrentRound() = %d, want %d", got, tt.want)
			}
		})
	}
}
