package tiles

import (
	"encoding/csv"
	"io"
	"strconv"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/cwgame/runemapping"
)

// LetterDistribution encodes the tile distribution for the relevant game.
type LetterDistribution struct {
	runemapping      *runemapping.RuneMapping
	Distribution     map[rune]uint8
	PointValues      map[rune]uint8
	SortOrder        map[rune]int
	Vowels           []rune
	numUniqueLetters int
	numLetters       int
	scores           []int
	Name             string
}

func ScanLetterDistribution(data io.Reader) (*LetterDistribution, error) {
	r := csv.NewReader(data)
	dist := map[rune]uint8{}
	ptValues := map[rune]uint8{}
	sortOrder := []rune{}
	vowels := []rune{}
	alph := &runemapping.RuneMapping{}
	alph.Init()
	// letter,quantity,value,vowel
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		letter := []rune(record[0])[0]
		sortOrder = append(sortOrder, letter)
		n, err := strconv.Atoi(record[1])
		if err != nil {
			return nil, err
		}
		p, err := strconv.Atoi(record[2])
		if err != nil {
			return nil, err
		}
		v, err := strconv.Atoi(record[3])
		if err != nil {
			return nil, err
		}
		if v == 1 {
			vowels = append(vowels, letter)
		}
		dist[letter] = uint8(n)
		ptValues[letter] = uint8(p)
		if letter != runemapping.BlankToken {
			// The Blank should not be part of the alphabet, only the letter dist.
			alph.Update(string(letter))
		}
	}
	alph.Reconcile()
	return newLetterDistribution(alph, dist, ptValues, makeSortMap(sortOrder), vowels), nil
}

func newLetterDistribution(alph *runemapping.RuneMapping, dist map[rune]uint8,
	ptValues map[rune]uint8, sortOrder map[rune]int, vowels []rune) *LetterDistribution {

	numTotalLetters := 0
	numUniqueLetters := len(dist)
	for _, v := range dist {
		numTotalLetters += int(v)
	}
	// Make an array of scores in alphabet order. This will allow for
	// fast lookups in move generators, etc, vs looking up a map.
	// Note: numUniqueLetters/numTotalLetters includes the blank.
	scores := make([]int, numUniqueLetters)
	for rn, ptVal := range ptValues {
		ml, err := alph.Val(rn)
		if err != nil {
			panic("Wrongly initialized")
		}
		scores[ml] = int(ptVal)
	}

	return &LetterDistribution{
		runemapping:      alph,
		Distribution:     dist,
		PointValues:      ptValues,
		SortOrder:        sortOrder,
		Vowels:           vowels,
		numUniqueLetters: numUniqueLetters,
		numLetters:       numTotalLetters,
		scores:           scores,
	}

}

func makeSortMap(order []rune) map[rune]int {
	sortMap := make(map[rune]int)
	for idx, letter := range order {
		sortMap[letter] = idx
	}
	return sortMap
}

// Score gives the score of the given machine letter. This is used by the
// move generator to score plays more rapidly than looking up a map.
func (ld *LetterDistribution) Score(ml runemapping.MachineLetter) int {
	if int8(ml) <= 0 {
		return ld.scores[0] // the blank
	}
	return ld.scores[ml]
}

func (d *LetterDistribution) RuneMapping() *runemapping.RuneMapping {
	return d.runemapping
}

// Score returns the score of this word given the ld.
func (ld *LetterDistribution) WordScore(mw runemapping.MachineWord) int {
	score := 0
	for _, c := range mw {
		score += ld.Score(c)
	}
	return score
}

// EnglishLetterDistribution returns the English letter distribution.
func EnglishLetterDistribution(cfg *config.Config) (*LetterDistribution, error) {
	return NamedLetterDistribution(cfg, "english")
}
