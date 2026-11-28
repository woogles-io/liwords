// Package rerate replays every rated game under a rating system, from the
// first game on, to get everyone's ratings as if that system had always been
// used. See docs/ratings_plan.md.
//
// The games are read from the database once (Extract) into a Data file, which
// later runs can bring up to date without reading everything again.
package rerate

import (
	"compress/gzip"
	"encoding/gob"
	"os"

	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Game is one finished rated game.
type Game struct {
	ID  int64
	End int64  // when the game ended (games.updated_at, else created_at), unix seconds
	Key uint16 // index into Data.Keys
	// Players are user ids in the game's own player order, which scores,
	// winner and bot flags follow. (Older games' player0_id/player1_id
	// columns don't always follow it.)
	Players [2]int64
	// QuickdataBot marks players flagged as bots in the game's own record.
	// Some early bots aren't flagged as bots in users.
	QuickdataBot [2]bool
	Reason       pb.GameEndReason
	Winner       int8 // winner_idx: 0, 1, or -1 for a tie
	Scores       [2]int32
}

// User is what a replay needs to know about a player.
type User struct {
	UUID     string
	Username string
	Bot      bool // users.internal_bot
}

// Reset is a moderator's rating reset: all of the user's ratings were cleared.
type Reset struct {
	UserID int64
	Time   int64
}

// Data is everything a replay reads.
type Data struct {
	Games []Game
	Keys  []entity.VariantKey
	Users map[int64]User
	// Resets are moderator rating resets, in time order.
	Resets []Reset
	// Current is each user's saved game ratings when the data was read, for
	// comparing (and, on the switch, for keeping puzzle ratings out of it).
	Current map[int64]map[entity.VariantKey]entity.SingleRating
	// MaxID is the highest game id read; Unfinished are games at or below it
	// that hadn't ended yet. Together they say which games a catch-up reads.
	MaxID      int64
	Unfinished []int64
	ReadAt     int64
	// Skipped counts finished rated games left out, by reason, and
	// SkippedIDs lists them.
	Skipped    map[string]int
	SkippedIDs map[string][]int64
	// Penalties are league results cancelled for cheating, by game id.
	Penalties map[int64]Penalty
}

// Penalty is a game whose result a league cancelled for cheating
// (CancelPlayerResults): the cheater loses by 100 points more than the
// opponent's score. The penalty rewrites the game's winner and end time
// (games.updated_at) but not its recorded scores, so the penalized scores
// come from game_players and the real end time from the game's timers.
type Penalty struct {
	Loser  int8     // the cheater, in the game's own player order
	Scores [2]int32 // after the penalty
	End    int64    // when the game really ended; 0 if unknown
}

func (d *Data) keyIndex(k entity.VariantKey, idx map[entity.VariantKey]uint16) uint16 {
	if i, ok := idx[k]; ok {
		return i
	}
	i := uint16(len(d.Keys))
	d.Keys = append(d.Keys, k)
	idx[k] = i
	return i
}

func (d *Data) keyIndexMap() map[entity.VariantKey]uint16 {
	m := make(map[entity.VariantKey]uint16, len(d.Keys))
	for i, k := range d.Keys {
		m[k] = uint16(i)
	}
	return m
}

// Save writes the data to a gzipped gob file.
func (d *Data) Save(path string) error {
	f, err := os.Create(path + ".tmp")
	if err != nil {
		return err
	}
	defer f.Close()
	zw := gzip.NewWriter(f)
	if err := gob.NewEncoder(zw).Encode(d); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

// Load reads data written by Save.
func Load(path string) (*Data, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	d := &Data{}
	if err := gob.NewDecoder(zr).Decode(d); err != nil {
		return nil, err
	}
	return d, nil
}
