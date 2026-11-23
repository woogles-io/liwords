package gameplay

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/rs/zerolog/log"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/game_service"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// wordCheckAllowed returns why userID may not check words in g, or nil. The
// terms let a VOID game's players check the validity of words while they
// play it.
func wordCheckAllowed(g *entity.Game, userID string) error {
	if g.ChallengeRule() != macondopb.ChallengeRule_VOID {
		return apiserver.PermissionDenied("words can be checked in VOID games only")
	}
	if g.Playing() == macondopb.PlayState_GAME_OVER || g.GameEndReason != ipc.GameEndReason_NONE {
		return apiserver.PermissionDenied("the game is over")
	}
	if h := g.History(); h != nil {
		for _, p := range h.Players {
			if p.UserId == userID {
				return nil
			}
		}
	}
	return apiserver.PermissionDenied("only the game's players can check words")
}

// tilesByLabel reads word as tiles of tm by their labels: each tile's own
// ("[CH]"), or the one it takes on a blank, its own in lowercase as a blank
// is written ("[ch]"), read as the tile itself, a word being only letters;
// the longest label that fits first. Nothing else is read, case is never
// changed: "[Ch]" names no tile, nor does a blank ("?") or a gap.
func tilesByLabel(word string, tm *tilemapping.TileMapping) (tilemapping.MachineWord, bool) {
	labels := map[string]tilemapping.MachineLetter{}
	longest := 0
	for label, ml := range tm.Vals() {
		if ml != 0 {
			labels[label] = ml
			longest = max(longest, utf8.RuneCountInString(label))
		}
	}
	for label, ml := range tm.Vals() {
		if onBlank := strings.ToLower(label); ml != 0 && labels[onBlank] == 0 {
			labels[onBlank] = ml
		}
	}
	runes := []rune(word)
	var mw tilemapping.MachineWord
	for i := 0; i < len(runes); {
		read := false
		for j := min(len(runes), i+longest); j > i; j-- {
			if ml, ok := labels[string(runes[i:j])]; ok {
				mw = append(mw, ml)
				i = j
				read = true
				break
			}
		}
		if !read {
			return nil, false
		}
	}
	return mw, len(mw) > 0
}

// checkWord returns whether word is valid in g: in its lexicon, as its
// variant reads words, the check a VOID play goes through. The word is read
// into tiles by their labels (tilesByLabel: "[CH]" or "[ch]" is the CH tile,
// "CH" a C and an H) and returned as checked, in the tiles' labels. One that
// is not in the game's letters, or is longer than its board, is refused.
func checkWord(g *entity.Game, word string) (string, bool, error) {
	word = strings.TrimSpace(word)
	mw, ok := tilesByLabel(word, g.Alphabet())
	if !ok || len(mw) > g.Board().Dim() {
		return word, false, apiserver.InvalidArg("not a word in this game's letters")
	}
	return mw.UserVisible(g.Alphabet()), g.ValidateWords(g.Lexicon(), []tilemapping.MachineWord{mw}) == nil, nil
}

const (
	// a player may check this many words a minute
	wordChecksPerWindow = 60
	wordCheckWindow     = time.Minute
)

// checkLimiter counts each user's word checks of the last window.
type checkLimiter struct {
	mu     sync.Mutex
	recent map[string][]time.Time
	swept  time.Time
}

var wordCheckLimiter = &checkLimiter{}

// allow tells whether userID may check another word at now, and counts it if
// so. A refused check is not counted. Once a window, users with no check in
// it are let go.
func (l *checkLimiter) allow(userID string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.recent == nil {
		l.recent = map[string][]time.Time{}
	}
	fresh := func(ts []time.Time) []time.Time {
		kept := ts[:0]
		for _, t := range ts {
			if now.Sub(t) < wordCheckWindow {
				kept = append(kept, t)
			}
		}
		return kept
	}
	if now.Sub(l.swept) >= wordCheckWindow {
		for u, ts := range l.recent {
			if kept := fresh(ts); len(kept) == 0 {
				delete(l.recent, u)
			} else {
				l.recent[u] = kept
			}
		}
		l.swept = now
	}
	ts := fresh(l.recent[userID])
	if len(ts) >= wordChecksPerWindow {
		l.recent[userID] = ts
		return false
	}
	l.recent[userID] = append(ts, now)
	return true
}

// CheckWord tells one of a VOID game's players, while the game is in play,
// whether a word is valid in it, and the word as it was read: each tile by its
// own label ("naiad" is NAIAD). Every check is logged, as an invalid word
// played is (invalid-word-play).
func (gs *GameService) CheckWord(ctx context.Context, req *connect.Request[pb.CheckWordRequest],
) (*connect.Response[pb.CheckWordResponse], error) {
	user, err := apiserver.AuthUser(ctx, gs.userStore)
	if err != nil {
		return nil, err
	}
	if !wordCheckLimiter.allow(user.UUID, time.Now()) {
		return nil, connect.NewError(connect.CodeResourceExhausted,
			errors.New("too many words checked: wait a minute"))
	}
	g, err := gs.gameStore.Get(ctx, req.Msg.GameId)
	if err != nil {
		return nil, apiserver.NotFound("game not found")
	}
	if err := wordCheckAllowed(g, user.UUID); err != nil {
		return nil, err
	}
	word, valid, err := checkWord(g, req.Msg.Word)
	if err != nil {
		return nil, err
	}
	log.Info().
		Str("gameID", g.GameID()).
		Str("userID", user.UUID).
		Strs("words", []string{word}).
		Bool("valid", valid).
		Msg("void-word-check")
	return connect.NewResponse(&pb.CheckWordResponse{Valid: valid, Word: word}), nil
}
