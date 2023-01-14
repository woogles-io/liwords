package dawg

// This file came directly from Macondo and modified a bit.
// Maybe we can make a utility repo.
// This file is GPL-licensed, unlike the rest of this repo, which is AGPL
// (except where noted)

import (
	"errors"
	"fmt"

	"github.com/domino14/liwords/pkg/cwgame/runemapping"
)

// zero value works. not threadsafe.
type DawgAnagrammer struct {
	ans         runemapping.MachineWord
	freq        []uint8
	blanks      uint8
	queryLength int
}

func (da *DawgAnagrammer) commonInit(dawg *SimpleDawg) {
	alph := dawg.GetRuneMapping()
	numLetters := alph.NumLetters()
	if cap(da.freq) < int(numLetters)+1 {
		da.freq = make([]uint8, numLetters+1)
	} else {
		da.freq = da.freq[:numLetters+1]
		for i := range da.freq {
			da.freq[i] = 0
		}
	}
	da.blanks = 0
	da.ans = da.ans[:0]
}

func (da *DawgAnagrammer) InitForString(dawg *SimpleDawg, tiles string) error {
	da.commonInit(dawg)
	da.queryLength = 0
	alph := dawg.GetRuneMapping()
	vals := alph.Vals()
	for _, r := range tiles {
		da.queryLength++ // count number of runes, not number of bytes
		if r == runemapping.BlankToken {
			da.blanks++
		} else if val, ok := vals[r]; ok {
			da.freq[val]++
		} else {
			return fmt.Errorf("invalid rune %v", r)
		}
	}
	return nil
}

func (da *DawgAnagrammer) InitForMachineWord(dawg *SimpleDawg, machineTiles runemapping.MachineWord) error {
	da.commonInit(dawg)
	da.queryLength = len(machineTiles)
	alph := dawg.GetRuneMapping()
	numLetters := alph.NumLetters()
	for _, v := range machineTiles {
		if v == 0 {
			da.blanks++
		} else if uint8(v) <= numLetters {
			da.freq[v]++
		} else {
			return fmt.Errorf("invalid byte %v", v)
		}
	}
	return nil
}

// f must not modify the given slice. if f returns error, abort iteration.
func (da *DawgAnagrammer) iterate(dawg *SimpleDawg, nodeIdx uint32, minLen int, minExact int, f func(runemapping.MachineWord) error) error {
	alph := dawg.GetRuneMapping()
	numLetters := alph.NumLetters()
	letterSet := dawg.GetLetterSet(nodeIdx)
	numArcs := dawg.NumArcs(nodeIdx)
	j := runemapping.MachineLetter(1)
	for i := byte(1); i <= numArcs; i++ {
		nextNodeIdx, nextLetter := dawg.ArcToIdxLetter(nodeIdx + uint32(i))
		if uint8(nextLetter) > numLetters {
			continue
		}
		for ; j <= nextLetter; j++ {
			if letterSet&(1<<(j-1)) != 0 {
				if da.freq[j] > 0 {
					da.freq[j]--
					da.ans = append(da.ans, j)
					if minLen <= 1 && minExact <= 1 {
						if err := f(da.ans); err != nil {
							return err
						}
					}
					da.ans = da.ans[:len(da.ans)-1]
					da.freq[j]++
				} else if da.blanks > 0 {
					da.blanks--
					da.ans = append(da.ans, j)
					if minLen <= 1 && minExact <= 0 {
						if err := f(da.ans); err != nil {
							return err
						}
					}
					da.ans = da.ans[:len(da.ans)-1]
					da.blanks++
				}
			}
		}
		if da.freq[nextLetter] > 0 {
			da.freq[nextLetter]--
			da.ans = append(da.ans, nextLetter)
			if err := da.iterate(dawg, nextNodeIdx, minLen-1, minExact-1, f); err != nil {
				return err
			}
			da.ans = da.ans[:len(da.ans)-1]
			da.freq[nextLetter]++
		} else if da.blanks > 0 {
			da.blanks--
			da.ans = append(da.ans, nextLetter)
			if err := da.iterate(dawg, nextNodeIdx, minLen-1, minExact, f); err != nil {
				return err
			}
			da.ans = da.ans[:len(da.ans)-1]
			da.blanks++
		}
	}
	for ; uint8(j) <= numLetters; j++ {
		if letterSet&(1<<(j-1)) != 0 {
			if da.freq[j] > 0 {
				da.freq[j]--
				da.ans = append(da.ans, j)
				if minLen <= 1 && minExact <= 1 {
					if err := f(da.ans); err != nil {
						return err
					}
				}
				da.ans = da.ans[:len(da.ans)-1]
				da.freq[j]++
			} else if da.blanks > 0 {
				da.blanks--
				da.ans = append(da.ans, j)
				if minLen <= 1 && minExact <= 0 {
					if err := f(da.ans); err != nil {
						return err
					}
				}
				da.ans = da.ans[:len(da.ans)-1]
				da.blanks++
			}
		}
	}
	return nil
}

func (da *DawgAnagrammer) Anagram(dawg *SimpleDawg, f func(runemapping.MachineWord) error) error {
	return da.iterate(dawg, dawg.GetRootNodeIndex(), da.queryLength, 0, f)
}

func (da *DawgAnagrammer) Subanagram(dawg *SimpleDawg, f func(runemapping.MachineWord) error) error {
	return da.iterate(dawg, dawg.GetRootNodeIndex(), 1, 0, f)
}

func (da *DawgAnagrammer) Superanagram(dawg *SimpleDawg, f func(runemapping.MachineWord) error) error {
	minExact := da.queryLength - int(da.blanks)
	blanks := da.blanks
	da.blanks = 255
	err := da.iterate(dawg, dawg.GetRootNodeIndex(), da.queryLength, minExact, f)
	da.blanks = blanks
	return err
}

var errHasAnagram = errors.New("has anagram")
var errHasBlanks = errors.New("has blanks")

func foundAnagram(runemapping.MachineWord) error {
	return errHasAnagram
}

// checks if a word with no blanks has any valid anagrams.
func (da *DawgAnagrammer) IsValidJumble(dawg *SimpleDawg, word runemapping.MachineWord) (bool, error) {
	if err := da.InitForMachineWord(dawg, word); err != nil {
		return false, err
	} else if da.blanks > 0 {
		return false, errHasBlanks
	}
	err := da.Anagram(dawg, foundAnagram)
	if err == nil {
		return false, nil
	} else if err == errHasAnagram {
		return true, nil
	} else {
		return false, err
	}
}
