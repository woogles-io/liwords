package dawg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"path/filepath"
	"strings"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/cwgame/cache"
	"github.com/domino14/liwords/pkg/cwgame/runemapping"
	"github.com/rs/zerolog/log"
)

const (
	dawgCacheKeyPrefix = "dawg:"

	DawgMagicNumber = "cdwg"
)

// NumArcsBitLoc is the bit location where the number of arcs start.
// A Node has a number of arcs and a letterSet
const NumArcsBitLoc = 24
const LetterSetBitMask = (1 << NumArcsBitLoc) - 1

// LetterBitLoc is the location where the letter starts.
// An Arc has a letter and a next node.
const LetterBitLoc = 24
const NodeIdxBitMask = (1 << LetterBitLoc) - 1

// LetterSet is a bit mask of acceptable letters, with indices from 0 to
// the maximum alphabet size.
type LetterSet uint64

type SimpleDawg struct {
	// Nodes is just a slice of 32-bit elements, the node array.
	nodes []uint32
	// The bit-mask letter sets
	letterSets  []LetterSet
	runeMapping *runemapping.RuneMapping
	lexiconName string
}

// GetRuneMapping returns the runemapping for this gaddag.
func (g *SimpleDawg) GetRuneMapping() *runemapping.RuneMapping {
	return g.runeMapping
}

// LexiconName returns the name of the lexicon.
func (g *SimpleDawg) LexiconName() string {
	return g.lexiconName
}

// GetLetterSet gets the letter set of the node at nodeIdx.
func (g *SimpleDawg) GetLetterSet(nodeIdx uint32) LetterSet {
	letterSetCode := g.nodes[nodeIdx] & LetterSetBitMask
	return g.letterSets[letterSetCode]
}

// NumArcs is simply the number of arcs for the given node.
func (g *SimpleDawg) NumArcs(nodeIdx uint32) byte {
	// if g.Nodes[nodeIdx].Arcs == nil {
	// 	return 0
	// }
	// return byte(len(g.Nodes[nodeIdx].Arcs))
	return byte(g.nodes[nodeIdx] >> NumArcsBitLoc)
}

// ArcToIdxLetter finds the index of the node pointed to by this arc and
// returns it and the letter.
func (g *SimpleDawg) ArcToIdxLetter(arcIdx uint32) (uint32, runemapping.MachineLetter) {
	// MachineLetters are 1-indexed since we reserve 0 for the blank and other
	// special values:
	letterCode := runemapping.MachineLetter(g.nodes[arcIdx]>>LetterBitLoc) + 1
	return g.nodes[arcIdx] & NodeIdxBitMask, letterCode
}

// GetRootNodeIndex gets the index of the root node.
func (g *SimpleDawg) GetRootNodeIndex() uint32 {
	return 0
}

// InLetterSet returns whether the `letter` is in the node at `nodeIdx`'s
// letter set.
func (g *SimpleDawg) InLetterSet(letter runemapping.MachineLetter, nodeIdx uint32) bool {
	if letter == 0 {
		return false
	}
	ltc := letter
	if letter.IsBlanked() {
		ltc = letter.Unblank()
	}
	letterSet := g.GetLetterSet(nodeIdx)
	// 1-indexed
	return letterSet&(1<<(ltc-1)) != 0
}

// CacheLoadFunc is the function that loads an object into the global cache.
func CacheLoadFunc(cfg *config.Config, key string) (interface{}, error) {
	lexiconName := strings.TrimPrefix(key, dawgCacheKeyPrefix)
	return LoadDawg(filepath.Join(cfg.DataPath, "lexica", "dawg", lexiconName+".dawg"))
}

// LoadDawg loads a dawg from a file and returns a *SimpleDawg
func LoadDawg(filename string) (*SimpleDawg, error) {
	log.Debug().Msgf("Loading %v ...", filename)
	file, err := cache.Open(filename)
	if err != nil {
		log.Debug().Msgf("Could not load %v", filename)
		return nil, err
	}
	defer file.Close()
	return ReadDawg(file)
}

// Ensure the magic number matches.
func compareMagicDawg(bytes [4]uint8) bool {
	cast := string(bytes[:])
	return cast == DawgMagicNumber
}

func ReadDawg(data io.Reader) (*SimpleDawg, error) {
	var magicStr [4]uint8
	binary.Read(data, binary.BigEndian, &magicStr)
	if !compareMagicDawg(magicStr) {
		log.Debug().Msgf("Magic number does not match")
		return nil, errors.New("magic number does not match dawg or reverse dawg")
	}
	d := &SimpleDawg{}
	nodes, letterSets, alphabetArr, lexName := loadCommonDagStructure(data)
	d.nodes = nodes
	d.letterSets = letterSets
	d.runeMapping = runemapping.FromSlice(alphabetArr)
	d.lexiconName = string(lexName)
	return d, nil
}

func dawgCacheReadFunc(data []byte) (interface{}, error) {
	stream := bytes.NewReader(data)
	return ReadDawg(stream)
}

// Set loads a dawg from bytes and populates the cache
func Set(name string, data []byte) error {
	prefix := dawgCacheKeyPrefix
	readFunc := dawgCacheReadFunc
	key := prefix + name
	return cache.Populate(key, data, readFunc)
}

// GetDawg loads a named dawg from the cache or from a file
func GetDawg(cfg *config.Config, name string) (*SimpleDawg, error) {
	key := dawgCacheKeyPrefix + name
	obj, err := cache.Load(cfg, key, CacheLoadFunc)
	if err != nil {
		return nil, err
	}
	ret, ok := obj.(*SimpleDawg)
	if !ok {
		return nil, errors.New("could not read dawg from file")
	}
	return ret, nil
}

func loadCommonDagStructure(stream io.Reader) ([]uint32, []LetterSet,
	[]uint32, []byte) {

	var lexNameLen uint8
	binary.Read(stream, binary.BigEndian, &lexNameLen)
	lexName := make([]byte, lexNameLen)
	binary.Read(stream, binary.BigEndian, &lexName)
	log.Debug().Msgf("Read lexicon name: '%v'", string(lexName))

	var alphabetSize, lettersetSize, nodeSize uint32

	binary.Read(stream, binary.BigEndian, &alphabetSize)
	log.Debug().Msgf("Rune mapping size: %v", alphabetSize)
	alphabetArr := make([]uint32, alphabetSize)
	binary.Read(stream, binary.BigEndian, &alphabetArr)

	binary.Read(stream, binary.BigEndian, &lettersetSize)
	log.Debug().Msgf("LetterSet size: %v", lettersetSize)
	letterSets := make([]LetterSet, lettersetSize)
	binary.Read(stream, binary.BigEndian, letterSets)

	binary.Read(stream, binary.BigEndian, &nodeSize)
	log.Debug().Msgf("Nodes size: %v", nodeSize)
	nodes := make([]uint32, nodeSize)
	binary.Read(stream, binary.BigEndian, &nodes)
	return nodes, letterSets, alphabetArr, lexName
}
