package dawg

import (
	"sync"

	"github.com/domino14/liwords/pkg/cwgame/runemapping"
	"github.com/rs/zerolog/log"
)

var daPool = sync.Pool{
	New: func() interface{} {
		return &DawgAnagrammer{}
	},
}

func (dawg *SimpleDawg) HasAnagram(word runemapping.MachineWord) bool {

	da := daPool.Get().(*DawgAnagrammer)
	defer daPool.Put(da)

	v, err := da.IsValidJumble(dawg, word)
	if err != nil {
		log.Err(err).Str("word", word.UserVisible(dawg.GetRuneMapping())).Msg("has-anagram?-error")
		return false
	}

	return v
}

func (dawg *SimpleDawg) HasWord(word runemapping.MachineWord) bool {
	var found bool
	found, _ = findMachineWord(dawg, dawg.GetRootNodeIndex(), word, 0)

	return found
}

func findMachineWord(d *SimpleDawg, nodeIdx uint32, word runemapping.MachineWord, curIdx uint8) (
	bool, uint32) {

	var numArcs, i byte
	var letter runemapping.MachineLetter
	var nextNodeIdx uint32
	if curIdx == uint8(len(word)-1) {
		ml := word[curIdx]
		return d.InLetterSet(ml, nodeIdx), nodeIdx
	}

	numArcs = d.NumArcs(nodeIdx)
	found := false
	for i = byte(1); i <= numArcs; i++ {
		nextNodeIdx, letter = d.ArcToIdxLetter(nodeIdx + uint32(i))
		curml := word[curIdx]
		if letter == curml {
			found = true
			break
		}
	}

	if !found {
		return false, 0
	}
	curIdx++
	return findMachineWord(d, nextNodeIdx, word, curIdx)
}
