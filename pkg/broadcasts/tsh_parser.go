package broadcasts

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FeedParser is the interface for parsing external tournament feed data.
// Currently only tsh_newt_json is implemented.
type FeedParser interface {
	Parse(data []byte) (*FeedData, error)
}

// FeedData is the normalized, format-agnostic result of parsing a tournament feed.
type FeedData struct {
	Players      []FeedPlayer
	TotalRounds  int
	CurrentRound int    // 1-indexed; 0 if tournament hasn't started
	DivisionName string // empty string if single-division feed
}

// FeedPlayer is one player's data from the feed.
type FeedPlayer struct {
	ID     int
	Name   string
	Rating int
	// Per-round data, 0-indexed (round 0 = first round of tournament)
	Scores   []int
	Pairings []int // opponent player ID; 0 = bye/forfeit/unplayed
	Boards   []int // table number per round
	// Going first (1) or second (2) per round; 0 if not available
	P12 []int
}

// RoundPairing describes a single game in a round, derived from feed data.
type RoundPairing struct {
	Round         int
	TableNumber   int
	Player1Name   string
	Player2Name   string
	Player1Score  int
	Player2Score  int
	Player1GoesFirst bool
	Finalized     bool // true if scores are entered
}

// NewFeedParser returns the appropriate parser for the given format string.
func NewFeedParser(format string) (FeedParser, error) {
	switch format {
	case "tsh_newt_json":
		return &TSHNewtParser{}, nil
	default:
		return nil, fmt.Errorf("unknown broadcast_url_format: %q", format)
	}
}

// GetRoundPairings returns all pairings for the given 1-indexed round.
// The slice is deduplicated (each game appears once) and sorted by table number.
func GetRoundPairings(fd *FeedData, round int) []RoundPairing {
	if round < 1 || round > fd.TotalRounds {
		return nil
	}
	r := round - 1 // convert to 0-indexed

	seen := make(map[int]bool) // table number → already added
	var pairings []RoundPairing

	for _, p := range fd.Players {
		if r >= len(p.Pairings) {
			continue
		}
		oppID := p.Pairings[r]
		if oppID == 0 {
			// bye or forfeit — skip for annotation purposes
			continue
		}
		var table int
		if r < len(p.Boards) {
			table = p.Boards[r]
		}
		if table == 0 || seen[table] {
			continue
		}
		seen[table] = true

		// Find opponent
		opp := findPlayer(fd.Players, oppID)
		if opp == nil {
			continue
		}

		var p1Score, p2Score int
		var finalized bool
		if r < len(p.Scores) {
			p1Score = p.Scores[r]
		}
		if r < len(opp.Scores) {
			p2Score = opp.Scores[r]
		}
		// Scores are finalized if both are non-zero (or one was a forfeit)
		finalized = p1Score != 0 || p2Score != 0

		var goesFirst bool
		if r < len(p.P12) {
			goesFirst = p.P12[r] == 1
		}

		pairings = append(pairings, RoundPairing{
			Round:            round,
			TableNumber:      table,
			Player1Name:      p.Name,
			Player2Name:      opp.Name,
			Player1Score:     p1Score,
			Player2Score:     p2Score,
			Player1GoesFirst: goesFirst,
			Finalized:        finalized,
		})
	}

	// Sort by table number
	sortRoundPairingsByTable(pairings)
	return pairings
}

func findPlayer(players []FeedPlayer, id int) *FeedPlayer {
	for i := range players {
		if players[i].ID == id {
			return &players[i]
		}
	}
	return nil
}

func sortRoundPairingsByTable(pairings []RoundPairing) {
	for i := 1; i < len(pairings); i++ {
		for j := i; j > 0 && pairings[j].TableNumber < pairings[j-1].TableNumber; j-- {
			pairings[j], pairings[j-1] = pairings[j-1], pairings[j]
		}
	}
}

// detectCurrentRound returns the 1-indexed current round.
// It finds the latest round that has at least one score entered.
// Returns 0 if no games have been played yet.
func detectCurrentRound(players []FeedPlayer) int {
	maxRound := 0
	for _, p := range players {
		for r := len(p.Scores) - 1; r >= 0; r-- {
			if r >= len(p.Pairings) {
				continue
			}
			if p.Pairings[r] != 0 && p.Scores[r] != 0 {
				if r+1 > maxRound {
					maxRound = r + 1
				}
				break
			}
		}
	}
	return maxRound
}

// ---- TSH newt={...}; parser ----

// TSHNewtParser parses the tsh tourney.js format.
// It handles both the minified JSON form (newt={...};) and the pretty-printed
// JavaScript object literal form (newt = { key: value, ... };).
type TSHNewtParser struct{}

// tshNewt is the top-level structure of the newt variable.
type tshNewt struct {
	Config    json.RawMessage `json:"config"`
	Divisions []tshDivision   `json:"divisions"`
}

type tshDivision struct {
	Name string `json:"name"`
	MaxR int    `json:"maxr"`
	// Players is 1-indexed; index 0 is always null.
	Players []json.RawMessage `json:"players"`
}

// tshPlayer is the per-player structure inside a division.
type tshPlayer struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Rating   int    `json:"rating"`
	Scores   []int  `json:"scores"`
	Pairings []int  `json:"pairings"`
	Etc      tshEtc `json:"etc"`
}

type tshEtc struct {
	Board []int `json:"board"`
	P12   []int `json:"p12"`
}

// parseNewt extracts and unmarshals the tshNewt structure from raw feed bytes.
func (p *TSHNewtParser) parseNewt(data []byte) (*tshNewt, error) {
	jsonBytes, err := extractNewtJSON(data)
	if err != nil {
		return nil, err
	}
	var newt tshNewt
	if err := json.Unmarshal(jsonBytes, &newt); err != nil {
		return nil, fmt.Errorf("tsh parser: unmarshal failed: %w", err)
	}
	if len(newt.Divisions) == 0 {
		return nil, fmt.Errorf("tsh parser: no divisions found")
	}
	return &newt, nil
}

// ListDivisions returns the names of all divisions in the feed (e.g. ["A", "B"]).
func (p *TSHNewtParser) ListDivisions(data []byte) ([]string, error) {
	newt, err := p.parseNewt(data)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(newt.Divisions))
	for i, d := range newt.Divisions {
		names[i] = d.Name
	}
	return names, nil
}

// Parse implements FeedParser for the tsh_newt_json format.
// It parses division 0 (the first/only division in single-division feeds).
// Use ParseDivision to select a specific division by name.
func (p *TSHNewtParser) Parse(data []byte) (*FeedData, error) {
	return p.ParseDivision(data, "")
}

// ParseDivision parses a specific division by name (e.g. "A", "B").
// If divisionName is empty, the first division is used.
func (p *TSHNewtParser) ParseDivision(data []byte, divisionName string) (*FeedData, error) {
	newt, err := p.parseNewt(data)
	if err != nil {
		return nil, err
	}

	var div tshDivision
	if divisionName == "" {
		div = newt.Divisions[0]
	} else {
		found := false
		for _, d := range newt.Divisions {
			if d.Name == divisionName {
				div = d
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("tsh parser: division %q not found", divisionName)
		}
	}

	var players []FeedPlayer
	for _, raw := range div.Players {
		if raw == nil || string(raw) == "null" {
			continue
		}
		var tp tshPlayer
		if err := json.Unmarshal(raw, &tp); err != nil {
			continue
		}
		if tp.ID == 0 {
			continue
		}
		players = append(players, FeedPlayer{
			ID:       tp.ID,
			Name:     tp.Name,
			Rating:   tp.Rating,
			Scores:   tp.Scores,
			Pairings: tp.Pairings,
			Boards:   tp.Etc.Board,
			P12:      tp.Etc.P12,
		})
	}

	// totalRounds: prefer maxr from the division header if set.
	totalRounds := div.MaxR
	if totalRounds == 0 {
		for _, p := range players {
			if len(p.Pairings) > totalRounds {
				totalRounds = len(p.Pairings)
			}
		}
	}

	currentRound := detectCurrentRound(players)

	return &FeedData{
		Players:      players,
		TotalRounds:  totalRounds,
		CurrentRound: currentRound,
		DivisionName: div.Name,
	}, nil
}

// extractNewtJSON strips the `newt=...;` JS wrapper and returns valid JSON bytes.
// Handles both `newt={...};` and `var newt={...};` forms.
// Also replaces JS `undefined` literals with `null`.
// If the data has no `newt=` prefix it is assumed to already be extracted JSON
// and is returned as-is (after the undefined→null replacement).
func extractNewtJSON(data []byte) ([]byte, error) {
	s := string(data)

	if idx := strings.Index(s, "newt="); idx != -1 {
		s = s[idx+len("newt="):]
		s = strings.TrimRight(s, " \t\r\n;")
	}

	// Replace JS undefined with null
	s = strings.ReplaceAll(s, "undefined", "null")

	return []byte(s), nil
}
