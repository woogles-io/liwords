package broadcasts

import "fmt"

// applyTournamentFields fills the tournament-standings OBSData fields for a
// broadcast slot. division/round/tableNumber and tournamentName come from the
// slot + broadcast DB rows and are always available. fd (the cached feed) may
// be nil if the feed hasn't been fetched yet or the fetch failed, in which
// case per-player fields fall back to the placeholder.
//
// p1Name/p2Name are the *game document* display names (already computed into
// data.P1Name/P2Name by ComputeOBSData) — they must string-match the feed's
// player names for the per-player fields to resolve.
func applyTournamentFields(data *OBSData, fd *FeedData, division string, round, tableNumber int32, tournamentName, p1Name, p2Name string) {
	data.Division = orPlaceholder(division)
	data.Tournament = orPlaceholder(tournamentName)
	data.Table = fmt.Sprintf("%d", tableNumber)
	data.Round = formatRound(round, fd)

	data.P1Record, data.P1Place, data.P1Spread, data.P1Rating = playerStandingFields(fd, p1Name)
	data.P2Record, data.P2Place, data.P2Spread, data.P2Rating = playerStandingFields(fd, p2Name)
}

// orPlaceholder returns obsPlaceholder when s is empty, else s unchanged.
func orPlaceholder(s string) string {
	if s == "" {
		return obsPlaceholder
	}
	return s
}

// formatRound renders "<slot round> of <total rounds>", falling back to just
// the slot round when the feed (and therefore the total-round count) isn't
// available.
func formatRound(slotRound int32, fd *FeedData) string {
	if fd != nil && fd.TotalRounds > 0 {
		return fmt.Sprintf("%d of %d", slotRound, fd.TotalRounds)
	}
	return fmt.Sprintf("%d", slotRound)
}

// playerStandingFields resolves record/place/spread/rating for a player by
// name match against the feed. Returns the placeholder for all four when the
// feed is unavailable or the name isn't found (e.g. the game-doc realName
// doesn't string-match the feed's player name).
func playerStandingFields(fd *FeedData, name string) (record, place, spread, rating string) {
	if fd == nil || name == "" {
		return obsPlaceholder, obsPlaceholder, obsPlaceholder, obsPlaceholder
	}
	p := findFeedPlayerByName(fd, name)
	if p == nil {
		return obsPlaceholder, obsPlaceholder, obsPlaceholder, obsPlaceholder
	}
	stats := computePlayerStats(*p, fd.Players)
	placeStr := obsPlaceholder
	if n := placeInDivision(fd, name); n > 0 {
		placeStr = ordinal(n)
	}
	return formatRecord(stats.wins, stats.losses), placeStr, formatSpread(stats.spread), fmt.Sprintf("%d", p.Rating)
}

func findFeedPlayerByName(fd *FeedData, name string) *FeedPlayer {
	if fd == nil {
		return nil
	}
	for i := range fd.Players {
		if fd.Players[i].Name == name {
			return &fd.Players[i]
		}
	}
	return nil
}

// placeInDivision returns the 1-based standing (sorted wins desc, then spread
// desc — matching feedDataToProto) of the named player within fd, or 0 if the
// player isn't found.
func placeInDivision(fd *FeedData, name string) int {
	if fd == nil {
		return 0
	}
	sorted, _ := feedDataToProto(fd)
	for i, p := range sorted {
		if p.Name == name {
			return i + 1
		}
	}
	return 0
}

// formatRecord renders a win-loss tally, showing ".5" for a half win/loss
// from a tied game, e.g. "6-1", "5.5-2.5".
func formatRecord(wins, losses float64) string {
	return fmt.Sprintf("%s-%s", formatTally(wins), formatTally(losses))
}

func formatTally(n float64) string {
	if n == float64(int64(n)) {
		return fmt.Sprintf("%d", int64(n))
	}
	return fmt.Sprintf("%.1f", n)
}

// formatSpread renders a signed cumulative spread, e.g. "+245", "-30", "0".
func formatSpread(spread int) string {
	if spread > 0 {
		return fmt.Sprintf("+%d", spread)
	}
	return fmt.Sprintf("%d", spread)
}

// ordinal renders a 1-based rank as an ordinal, e.g. 1 -> "1st", 2 -> "2nd",
// 3 -> "3rd", 4 -> "4th", 11 -> "11th", 21 -> "21st".
func ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}
