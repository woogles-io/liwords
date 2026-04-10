package broadcasts

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/domino14/word-golib/tilemapping"

	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Definer looks up a single word and returns its definition and whether it is
// valid in the given lexicon. This interface is satisfied by *words.WordService.
type Definer interface {
	Define(lexicon, word string) (string, bool)
}

// obsCSWLexicon and obsNWLLexicon are the fixed reference lexicons used only
// for computing the # / $ symbol. They are independent of the game's own lexicon.
const obsCSWLexicon = "CSW24"
const obsNWLLexicon = "NWL23"

// OBSData holds all display strings for a single game state, matching
// the output files produced by WatchGCG (https://github.com/jvc56/WatchGCG).
type OBSData struct {
	Score       string `json:"score"`        // "NNN - NNN"
	P1Score     string `json:"p1_score"`     // right-justified 3 chars
	P2Score     string `json:"p2_score"`     // right-justified 3 chars
	UnseenTiles string `json:"unseen_tiles"` // "AA BB C ?"
	UnseenCount string `json:"unseen_count"` // 3-line breakdown
	LastPlay    string `json:"last_play"`    // "     LAST PLAY: ..."
	Blank1      string `json:"blank1"`       // first word played with a blank, blank letter lowercase e.g. "CoSTARS"
	Blank2      string `json:"blank2"`       // second word played with a blank
}

// ComputeOBSData renders all display strings from a live GameDocument.
// definer may be nil, in which case definitions and symbols are omitted.
func ComputeOBSData(doc *ipc.GameDocument, dist *tilemapping.LetterDistribution, definer Definer) OBSData {
	blank1, blank2 := formatBlankPlays(doc, dist)
	return OBSData{
		Score:       formatScore(doc),
		P1Score:     formatPlayerScore(doc, 0),
		P2Score:     formatPlayerScore(doc, 1),
		UnseenTiles: formatUnseenTiles(doc, dist),
		UnseenCount: formatUnseenCount(doc, dist),
		LastPlay:    formatLastPlay(doc, dist, definer),
		Blank1:      blank1,
		Blank2:      blank2,
	}
}

// formatBlankPlays scans events forward and returns the rendered main word for
// the first and second TILE_PLACEMENT_MOVE events that contain a blank tile.
// Blank-designated letters appear lowercase (e.g. "CoSTARS"). Empty string if
// fewer than two such plays have occurred.
func formatBlankPlays(doc *ipc.GameDocument, dist *tilemapping.LetterDistribution) (string, string) {
	rm := dist.TileMapping()
	var results []string
	for _, evt := range doc.GetEvents() {
		if evt.Type != ipc.GameEvent_TILE_PLACEMENT_MOVE {
			continue
		}
		word, hasBlank := renderMainWordWithBlanks(evt, rm)
		if !hasBlank {
			continue
		}
		results = append(results, word)
		if len(results) == 2 {
			break
		}
	}
	blank1, blank2 := "", ""
	if len(results) > 0 {
		blank1 = results[0]
	}
	if len(results) > 1 {
		blank2 = results[1]
	}
	return blank1, blank2
}

// renderMainWordWithBlanks returns the main word for a placement event with
// blank-designated letters lowercased. Returns ("", false) if no blank was played.
func renderMainWordWithBlanks(evt *ipc.GameEvent, rm *tilemapping.TileMapping) (string, bool) {
	hasBlank := false
	type tileInfo struct {
		rendered string
		isBlank  bool
	}
	infos := make([]tileInfo, len(evt.PlayedTiles))
	for i, b := range evt.PlayedTiles {
		ml := tilemapping.MachineLetter(b)
		uv := ml.UserVisible(rm, true)
		// A blank designation renders as a lowercase letter (not "." which is a through tile).
		isBlank := ml != 0 && uv != "." && uv != strings.ToUpper(uv)
		infos[i] = tileInfo{uv, isBlank}
		if isBlank {
			hasBlank = true
		}
	}
	if !hasBlank {
		return "", false
	}

	// Use WordsFormedFriendly[0] as base (it includes through-tile letters in uppercase)
	// and lowercase the positions where a blank was played.
	if len(evt.WordsFormedFriendly) > 0 {
		friendly := []rune(evt.WordsFormedFriendly[0])
		if len(friendly) == len(infos) {
			for i, t := range infos {
				if t.isBlank {
					friendly[i] = unicode.ToLower(friendly[i])
				}
			}
			return string(friendly), true
		}
	}

	// Fallback: join rendered tiles (may show "." for through tiles).
	var sb strings.Builder
	for _, t := range infos {
		sb.WriteString(t.rendered)
	}
	return sb.String(), true
}

// formatScore returns the combined score string, WatchGCG std-compatible.
// Example: "045 - 032"
func formatScore(doc *ipc.GameDocument) string {
	p1, p2 := scoresPair(doc)
	return fmt.Sprintf("%03d - %03d", p1, p2)
}

// formatPlayerScore returns the score for a single player, right-justified to
// 3 chars (WatchGCG au-compatible). Example: " 45"
func formatPlayerScore(doc *ipc.GameDocument, playerIdx int) string {
	scores := doc.GetCurrentScores()
	if playerIdx >= len(scores) {
		return "  0"
	}
	return fmt.Sprintf("%3d", scores[playerIdx])
}

// scoresPair returns (player0Score, player1Score) with safe defaults.
func scoresPair(doc *ipc.GameDocument) (int32, int32) {
	scores := doc.GetCurrentScores()
	var p1, p2 int32
	if len(scores) > 0 {
		p1 = scores[0]
	}
	if len(scores) > 1 {
		p2 = scores[1]
	}
	return p1, p2
}

// collectUnseen builds a count map (machine-letter → count) of tiles that are
// "unseen" from the annotator's perspective: the bag plus the opponent's rack.
// The player-on-turn's rack is excluded because the annotator can see it.
// Blank tiles are represented as MachineLetter(0).
func collectUnseen(doc *ipc.GameDocument) map[tilemapping.MachineLetter]int {
	counts := make(map[tilemapping.MachineLetter]int)
	if doc.Bag != nil {
		for _, b := range doc.Bag.Tiles {
			counts[tilemapping.MachineLetter(b)]++
		}
	}
	for i, rack := range doc.Racks {
		if uint32(i) == doc.PlayerOnTurn {
			continue // player on turn's rack is visible — exclude it
		}
		for _, b := range rack {
			counts[tilemapping.MachineLetter(b)]++
		}
	}
	return counts
}

// formatUnseenTiles renders unseen tiles in dictionary order with a space
// after each letter group — WatchGCG-compatible.
// Example: "A BB CCC ? " (blanks shown as ?)
func formatUnseenTiles(doc *ipc.GameDocument, dist *tilemapping.LetterDistribution) string {
	counts := collectUnseen(doc)
	rm := dist.TileMapping()

	// Collect and sort the machine letters that appear.
	// Blanks (0) go last, matching WatchGCG "?" ordering.
	var letters []tilemapping.MachineLetter
	for ml := range counts {
		letters = append(letters, ml)
	}
	sort.Slice(letters, func(i, j int) bool {
		li, lj := letters[i], letters[j]
		if li == 0 {
			return false // blank sorts last
		}
		if lj == 0 {
			return true
		}
		return rm.Letter(li) < rm.Letter(lj)
	})

	var sb strings.Builder
	for _, ml := range letters {
		n := counts[ml]
		var ch string
		if ml == 0 {
			ch = "?"
		} else {
			ch = rm.Letter(ml)
		}
		for i := 0; i < n; i++ {
			sb.WriteString(ch)
		}
		sb.WriteByte(' ')
	}
	return sb.String()
}

// formatUnseenCount renders the tile/vowel/consonant breakdown — WatchGCG-compatible.
// Example:
//
//	42 tiles
//	 8 vowels |  6 consonants
func formatUnseenCount(doc *ipc.GameDocument, dist *tilemapping.LetterDistribution) string {
	counts := collectUnseen(doc)

	total, vowelCount, consonantCount := 0, 0, 0
	for ml, n := range counts {
		total += n
		if ml == 0 {
			// blanks are neither vowel nor consonant
			continue
		}
		if ml.IsVowel(dist) {
			vowelCount += n
		} else {
			consonantCount += n
		}
	}

	tileWord := "tiles"
	if total == 1 {
		tileWord = "tile"
	}
	vowelWord := "vowels"
	if vowelCount == 1 {
		vowelWord = "vowel"
	}
	consonantWord := "consonants"
	if consonantCount == 1 {
		consonantWord = "consonant"
	}

	return fmt.Sprintf("%d %s\n%2d %s | %2d %s",
		total, tileWord,
		vowelCount, vowelWord,
		consonantCount, consonantWord)
}

const lastPlayPrefix = "     LAST PLAY: "

// formatLastPlay renders the most recent game event — WatchGCG-compatible
// with improved handling of challenges and blank tiles.
func formatLastPlay(doc *ipc.GameDocument, dist *tilemapping.LetterDistribution, definer Definer) string {
	events := doc.GetEvents()
	if len(events) == 0 {
		return lastPlayPrefix + "(no play yet)"
	}

	// Walk backwards to find the last "meaningful" event for display.
	// Skip end-rack scoring events which are not plays.
	for i := len(events) - 1; i >= 0; i-- {
		evt := events[i]
		switch evt.Type {
		case ipc.GameEvent_TILE_PLACEMENT_MOVE:
			return formatPlacementEvent(evt, doc, dist, definer)
		case ipc.GameEvent_EXCHANGE:
			return formatExchangeEvent(evt, doc, dist)
		case ipc.GameEvent_PASS:
			return formatPassEvent(evt, doc)
		case ipc.GameEvent_PHONY_TILES_RETURNED:
			return formatChallengeOffEvent(evt, doc, dist, events, i)
		case ipc.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS:
			// The challenged play stayed; look back for the actual play.
			continue
		case ipc.GameEvent_CHALLENGE_BONUS:
			continue
		case ipc.GameEvent_END_RACK_PTS,
			ipc.GameEvent_END_RACK_PENALTY,
			ipc.GameEvent_TIME_PENALTY,
			ipc.GameEvent_TIMED_OUT,
			ipc.GameEvent_RESIGNED:
			continue
		}
	}
	return lastPlayPrefix + "(no play yet)"
}

func playerName(doc *ipc.GameDocument, playerIdx uint32) string {
	players := doc.GetPlayers()
	if int(playerIdx) < len(players) {
		if n := players[playerIdx].GetRealName(); n != "" {
			return n
		}
		return players[playerIdx].GetNickname()
	}
	return fmt.Sprintf("Player%d", playerIdx+1)
}

// machineWordToDisplay converts PlayedTiles bytes to a human-friendly string.
// Through squares (ml==0) are shown as "(X)" using the corresponding letter
// from friendlyMainWord (WordsFormedFriendly[0]). If friendlyMainWord is empty
// or mismatches in length, through squares fall back to ".".
// Blank-designated letters are shown lowercase.
func machineWordToDisplay(mls []byte, friendlyMainWord string, dist *tilemapping.LetterDistribution) string {
	rm := dist.TileMapping()
	friendly := []rune(friendlyMainWord)
	var sb strings.Builder
	for i, b := range mls {
		ml := tilemapping.MachineLetter(b)
		if ml == 0 {
			if i < len(friendly) {
				sb.WriteByte('(')
				sb.WriteRune(friendly[i])
				sb.WriteByte(')')
			} else {
				sb.WriteByte('.')
			}
		} else {
			sb.WriteString(ml.UserVisible(rm, true))
		}
	}
	return sb.String()
}

// exchangedTilesToDisplay converts exchanged tile bytes to uppercase letters / "?" for blanks.
func exchangedTilesToDisplay(mls []byte, dist *tilemapping.LetterDistribution) string {
	rm := dist.TileMapping()
	var sb strings.Builder
	for _, b := range mls {
		ml := tilemapping.MachineLetter(b)
		if ml == 0 {
			sb.WriteByte('?')
		} else {
			sb.WriteString(strings.ToUpper(rm.Letter(ml)))
		}
	}
	return sb.String()
}

// lexiconSymbol returns '#' if word is CSW-only, '$' if NWL-only, "" otherwise.
// Errors are silently swallowed so a bad lookup never breaks the display.
func lexiconSymbol(definer Definer, word string) string {
	if definer == nil {
		return ""
	}
	word = strings.ToUpper(word)
	_, inCSW := definer.Define(obsCSWLexicon, word)
	_, inNWL := definer.Define(obsNWLLexicon, word)
	if inCSW && !inNWL {
		return "#"
	}
	if inNWL && !inCSW {
		return "$"
	}
	return ""
}

func formatPlacementEvent(evt *ipc.GameEvent, doc *ipc.GameDocument, dist *tilemapping.LetterDistribution, definer Definer) string {
	name := playerName(doc, evt.PlayerIndex)
	pos := evt.Position
	friendly := ""
	if len(evt.WordsFormedFriendly) > 0 {
		friendly = evt.WordsFormedFriendly[0]
	}
	word := machineWordToDisplay(evt.PlayedTiles, friendly, dist)
	score := evt.Score
	total := evt.Cumulative

	// Use WordsFormedFriendly[0] directly for symbol+definition lookup —
	// it's already the full main word in uppercase with no punctuation.
	mainWordUp := strings.ToUpper(friendly)

	symbol := lexiconSymbol(definer, mainWordUp)

	var def string
	if definer != nil && mainWordUp != "" {
		def, _ = definer.Define(doc.GetLexicon(), mainWordUp)
	}

	line := fmt.Sprintf("%s%s %s %s%s %d %d", lastPlayPrefix, name, pos, word, symbol, score, total)
	if def != "" {
		line += " | " + def
	}
	return line
}

func formatExchangeEvent(evt *ipc.GameEvent, doc *ipc.GameDocument, dist *tilemapping.LetterDistribution) string {
	name := playerName(doc, evt.PlayerIndex)
	tiles := exchangedTilesToDisplay(evt.Exchanged, dist)
	if tiles == "" {
		tiles = "(unknown)"
	}
	return fmt.Sprintf("%s%s exch %s %d %d", lastPlayPrefix, name, tiles, evt.Score, evt.Cumulative)
}

func formatPassEvent(evt *ipc.GameEvent, doc *ipc.GameDocument) string {
	name := playerName(doc, evt.PlayerIndex)
	return fmt.Sprintf("%s%s pass %d %d", lastPlayPrefix, name, evt.Score, evt.Cumulative)
}

func formatChallengeOffEvent(evt *ipc.GameEvent, doc *ipc.GameDocument, dist *tilemapping.LetterDistribution, events []*ipc.GameEvent, idx int) string {
	name := playerName(doc, evt.PlayerIndex)
	// Find the play that was challenged off (the event just before this one
	// with type TILE_PLACEMENT_MOVE belonging to the other player).
	for j := idx - 1; j >= 0; j-- {
		prev := events[j]
		if prev.Type == ipc.GameEvent_TILE_PLACEMENT_MOVE {
			prevFriendly := ""
			if len(prev.WordsFormedFriendly) > 0 {
				prevFriendly = prev.WordsFormedFriendly[0]
			}
			word := machineWordToDisplay(prev.PlayedTiles, prevFriendly, dist)
			return fmt.Sprintf("%s%s CHALLENGED OFF %s", lastPlayPrefix, name, word)
		}
	}
	return fmt.Sprintf("%s%s CHALLENGED OFF (unknown)", lastPlayPrefix, name)
}
