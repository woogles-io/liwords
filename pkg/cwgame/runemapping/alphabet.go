package runemapping

import (
	"errors"
	"fmt"
	"sort"
	"unicode"

	"github.com/rs/zerolog/log"
)

// A "letter" or tile is internally represented by a byte.
// The 0 value is used to represent various things:
// - an empty space on the board
// - a blank on your rack
// - a "played-through" letter on the board, when used in the description of a play.
// The letter A is represented by 1, B by 2, ... all the way to 26, for the English
// alphabet, for example.
// A blank letter is just the negative value: -1 for a, ..., -26 for z
// Since a byte cannot be negative, we use the value that an int8 would have;
// for example, 255 for a, 254 for b ... 230 for z
const (
	// MaxAlphabetSize should be below 64 so that a letterset can be a 64-bit int.
	MaxAlphabetSize = 62
	// ASCIIPlayedThrough is a somewhat user-friendly representation of a
	// played-through letter, used mostly for debug purposes.
	// Note that in order to actually be visible on a computer screen, we
	// should use `(`and `)` around letters already on a board.
	ASCIIPlayedThrough = '.'
	// BlankToken is the user-friendly representation of a blank.
	BlankToken = '?'
)

// MachineLetter is a machine-only representation of a letter. It represents a signed
// integer; negative for blank letters, 0 for blanks, positive for regular letters.
type MachineLetter byte

type MachineWord []MachineLetter

// LetterSlice is a slice of runes. We make it a separate type for ease in
// defining sort functions on it.
type LetterSlice []rune

// A RuneMapping contains the structures needed to map a user-visible "rune",
// like the letter B, into its "MachineLetter" counterpart (for example,
// MachineLetter(2) in the english-alphabet), and vice-versa.
type RuneMapping struct {
	// vals is a map of the actual physical letter rune (like 'A') to a
	// number representing it, from 0 to MaxAlphabetSize.
	vals map[rune]MachineLetter
	// letters is a map of the 0 to MaxAlphabetSize value back to a letter.
	letters map[MachineLetter]rune

	letterSlice LetterSlice
	curIdx      MachineLetter
}

// Init initializes the alphabet data structures
func (rm *RuneMapping) Init() {
	rm.vals = make(map[rune]MachineLetter)
	rm.letters = make(map[MachineLetter]rune)
}

// update the alphabet map.
func (rm *RuneMapping) Update(word string) error {
	for _, char := range word {
		if _, ok := rm.vals[char]; !ok {
			rm.vals[char] = rm.curIdx
			rm.curIdx++
		}
	}

	if rm.curIdx == MaxAlphabetSize {
		return errors.New("exceeded max alphabet size")
	}
	return nil
}

// Letter returns the letter that this position in the alphabet corresponds to.
func (rm *RuneMapping) Letter(b MachineLetter) rune {
	if b == 0 {
		return BlankToken
	}
	if b.IsBlanked() {
		return unicode.ToLower(rm.letters[-b])
	}
	return rm.letters[b]
}

// Val returns the 'value' of this rune in the alphabet.
// Takes into account blanks (lowercase letters).
func (rm *RuneMapping) Val(r rune) (MachineLetter, error) {
	if r == BlankToken {
		return 0, nil
	}
	val, ok := rm.vals[r]
	if ok {
		return val, nil
	}
	if r == unicode.ToLower(r) {
		val, ok = rm.vals[unicode.ToUpper(r)]
		if ok {
			return -val, nil
		}
	}
	if r == ASCIIPlayedThrough {
		return 0, nil
	}
	return 0, fmt.Errorf("letter `%c` not found in alphabet", r)
}

// UserVisible turns the passed-in machine letter into a user-visible rune.
func (ml MachineLetter) UserVisible(rm *RuneMapping, zeroForPlayedThrough bool) rune {
	if ml == 0 {
		if zeroForPlayedThrough {
			return ASCIIPlayedThrough
		}
		return BlankToken
	}
	return rm.Letter(ml)
}

// Unblank turns the machine letter into its non-blank version (if it's a blanked letter)
func (ml MachineLetter) Unblank() MachineLetter {
	if int8(ml) < 0 {
		return -ml
	}
	return ml
}

// IsBlanked returns true if the machine letter is a designated blank letter.
func (ml MachineLetter) IsBlanked() bool {
	return int8(ml) < 0
}

// UserVisible turns the passed-in machine word into a user-visible string.
func (mw MachineWord) UserVisible(rm *RuneMapping) string {
	runes := make([]rune, len(mw))
	for i, l := range mw {
		runes[i] = l.UserVisible(rm, false)
	}
	return string(runes)
}

// UserVisiblePlayedTiles turns the passed-in machine word into a user-visible string.
// It assumes that the MachineWord represents played tiles and not just
// tiles on a rack, so it uses the PlayedThrough character for 0.
func (mw MachineWord) UserVisiblePlayedTiles(rm *RuneMapping) string {
	runes := make([]rune, len(mw))
	for i, l := range mw {
		runes[i] = l.UserVisible(rm, true)
	}
	return string(runes)
}

// Convert the MachineLetter array into a byte array. For now, wastefully
// allocate a byte array, but maybe we can use the unsafe package in the future.
func (mw MachineWord) ToByteArr() []byte {
	bts := make([]byte, len(mw))
	for i, l := range mw {
		bts[i] = byte(l)
	}
	return bts
}

func FromByteArr(bts []byte) MachineWord {
	mls := make([]MachineLetter, len(bts))
	for i, l := range bts {
		mls[i] = MachineLetter(l)
	}
	return mls
}

// NumLetters returns the number of letters in this alphabet.
func (rm *RuneMapping) NumLetters() uint8 {
	return uint8(len(rm.letters))
}

func (rm *RuneMapping) Vals() map[rune]MachineLetter {
	return rm.vals
}

// ToMachineLetters creates an array of MachineLetters from the given string.
func ToMachineLetters(word string, rm *RuneMapping) ([]MachineLetter, error) {
	letters := make([]MachineLetter, len([]rune(word)))
	runeIdx := 0
	for _, ch := range word {
		ml, err := rm.Val(ch)
		if err != nil {
			return nil, err
		}
		letters[runeIdx] = ml
		runeIdx++
	}
	return letters, nil
}

func (rm *RuneMapping) genLetterSlice(sortMap map[rune]int) {
	rm.letterSlice = []rune{}
	for rn := range rm.vals {
		rm.letterSlice = append(rm.letterSlice, rn)
	}
	if sortMap != nil {
		sort.Slice(rm.letterSlice, func(i, j int) bool {
			return sortMap[rm.letterSlice[i]] < sortMap[rm.letterSlice[j]]
		})
	} else {
		sort.Sort(rm.letterSlice)
	}
	log.Debug().Msgf("After sorting: %v", rm.letterSlice)
	// These maps are now deterministic. Renumber them according to
	// sort order.
	for idx, rn := range rm.letterSlice {
		rm.vals[rn] = MachineLetter(idx + 1)
		rm.letters[MachineLetter(idx+1)] = rn
	}
	log.Debug().
		Interface("letters", rm.letters).
		Interface("vals", rm.vals).
		Msg("runemapping-letters")
}

// Reconcile will take a populated alphabet, sort the glyphs, and re-index
// the numbers.
func (rm *RuneMapping) Reconcile(sortMap map[rune]int) {
	log.Debug().Msg("Reconciling alphabet")
	rm.genLetterSlice(sortMap)
}

// FromSlice creates an alphabet from a serialized array. It is the
// opposite of the Serialize function, except the length is implicitly passed
// in as the length of the slice.
func FromSlice(arr []uint32) *RuneMapping {
	rm := &RuneMapping{}
	rm.Init()
	numRunes := uint8(len(arr))

	for i := uint8(1); i < numRunes+1; i++ {
		rm.vals[rune(arr[i-1])] = MachineLetter(i)
		rm.letters[MachineLetter(i)] = rune(arr[i-1])
	}
	return rm
}

// EnglishAlphabet returns a RuneMapping that corresponds to the English
// alphabet. This function should be used for testing. In production
// we will load the alphabet from the gaddag.
func EnglishAlphabet() *RuneMapping {
	return FromSlice([]uint32{
		'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M',
		'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
	})
}

func (a LetterSlice) Len() int           { return len(a) }
func (a LetterSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a LetterSlice) Less(i, j int) bool { return a[i] < a[j] }
