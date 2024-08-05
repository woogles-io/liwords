package stores

import (
	"github.com/domino14/word-golib/tilemapping"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const CurrentGameDocumentVersion = 2

var norwayMap map[byte]byte

func init() {
	// map of old tile index to new tile index
	norwayMap = map[byte]byte{
		0:  0,  //?
		1:  1,  // A
		2:  29, // why have Ä??
		3:  2,  // B
		4:  3,  // C
		5:  4,  // D
		6:  5,  // E
		7:  6,  // F
		8:  7,  // G
		9:  8,  // H
		10: 9,  // I
		11: 10, // J
		12: 11, // K
		13: 12, // L
		14: 13, // M
		15: 14, // N
		16: 15, // O
		17: 31, // Ö
		18: 16, // P
		19: 17, // Q
		20: 18, // R
		21: 19, // S
		22: 20, // T
		23: 21, // U
		24: 26, // Ü
		25: 22, // V
		26: 23, // W
		27: 24, // X
		28: 25, // Y
		29: 27, // Z
		30: 28, // Æ
		31: 30, // Ø
		32: 32, // Å
	}
	norwayBlankMap := map[byte]byte{}
	for k, v := range norwayMap {
		if k != 0 {
			norwayBlankMap[k|0x80] = v | 0x80
		}
	}
	for k, v := range norwayBlankMap {
		norwayMap[k] = v
	}
}

// MigrateGameDocument performs an in-place migration of a GameDocument.
// It does not save it back to any store. This is meant to be used as
// a temporary function; we should migrate the database permanently after having
// this in production for a bit while people are on old versions.
func MigrateGameDocument(cfg *config.Config, gdoc *ipc.GameDocument) error {
	if gdoc.Version == CurrentGameDocumentVersion {
		return nil
	}
	if gdoc.Version == 1 {
		log.Info().Str("gameID", gdoc.Uid).Msg("migrating-to-v2")
		err := migrateToV2(cfg, gdoc)
		if err != nil {
			return err
		}
		log.Info().Str("gameID", gdoc.Uid).Msg("migration-done")
	}
	// if gdoc.Version == 2  .. etc, we can keep migrating here if we're behind.
	return nil
}

func fromTwosComplement(t byte) byte {
	l := 256 - int(t)
	return byte(l | 0x80)
}

func migrateToV2(cfg *config.Config, gdoc *ipc.GameDocument) error {
	// Migrate a game document from version 1 to version 2.
	// Version 2 has the following changes:
	// - Internal representation of blanked tiles uses 0x80 | tile value as opposed
	// to 2's complement of a tile value. (i.e. 255 was a, 254 was b etc)
	// - Norwegian games use alphabetical tile ordering (i.e Ü goes between Y and Z)
	// German was already alphabetical.
	// - There is a new "words_formed_friendly" field in GameEvent. This will make it
	// easier to do full-text searches for played words in the future at the cost of
	// adding some more bytes.
	isNorwegian := false
	if gdoc.LetterDistribution == "norwegian" {
		isNorwegian = true
	}

	dist, err := tilemapping.GetDistribution(cfg.WGLConfig(), gdoc.LetterDistribution)
	if err != nil {
		return err
	}
	for idx, t := range gdoc.Board.Tiles {
		if t&0x80 > 1 {
			// The high bit is set. Convert from 2s complement.
			gdoc.Board.Tiles[idx] = fromTwosComplement(t)
		}
		if isNorwegian {
			gdoc.Board.Tiles[idx] = convertNorway(t)
		}
	}

	for idx, t := range gdoc.Bag.Tiles {
		if isNorwegian {
			gdoc.Bag.Tiles[idx] = convertNorway(t)
		}
	}

	for _, evt := range gdoc.Events {
		if isNorwegian {
			for idx := range evt.Rack {
				evt.Rack[idx] = convertNorway(evt.Rack[idx])
			}
		}

		for idx, t := range evt.PlayedTiles {
			if t&0x80 > 1 {
				evt.PlayedTiles[idx] = fromTwosComplement(t)
			}
			if isNorwegian {
				evt.PlayedTiles[idx] = convertNorway(evt.PlayedTiles[idx])
			}
		}
		for idx := range evt.Exchanged {
			if isNorwegian {
				evt.Exchanged[idx] = convertNorway(evt.Exchanged[idx])
			}
		}
		for _, w := range evt.WordsFormed {
			for idx, t := range w {
				if isNorwegian {
					w[idx] = convertNorway(t)
				}
			}
			evt.WordsFormedFriendly = append(
				evt.WordsFormedFriendly,
				tilemapping.FromByteArr(w).UserVisible(dist.TileMapping()))
		}

	}

	if isNorwegian {
		for _, rack := range gdoc.Racks {
			for idx := range rack {
				rack[idx] = convertNorway(rack[idx])
			}
		}
	}
	gdoc.Version = 2
	return nil
}

func convertNorway(t byte) byte {
	// convert the byte from old norwegian tile distribution value to
	// new tile distribution value. if t is blanked it must use new way
	// (| with 0x80)
	return norwayMap[t]
}
