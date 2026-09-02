package xwordbridge

// What it costs to load a game each way.
//
// The choice between storing a position and replaying an event log is not
// really about capability -- a sufficiently complete log can reconstruct
// anything -- so it comes down to what each one costs on the load path. These
// benchmarks measure that on a realistic game rather than arguing about it.
//
//	go test ./pkg/xwordbridge/ -run XXX -bench BenchmarkLoad -benchmem

import (
	"math/rand/v2"
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/xwordgame"
)

// playedOutGame drives a full random game through macondo and returns its
// history, so the event count is whatever a real game produces rather than a
// number chosen to flatter one side.
func playedOutGame(tb testing.TB) (*macondopb.GameHistory, *xwordgame.Rules) {
	tb.Helper()
	pc := parityConfigs[0] // classic english CSW21
	t, ok := tb.(*testing.T)
	if !ok {
		// newGamePair wants a *testing.T; benchmarks build the fixture through
		// a throwaway one.
		t = &testing.T{}
	}
	g, s, r := newGamePair(t, pc)
	maxLtr := maxTile(r.LetterDistribution)
	rng := rand.New(rand.NewPCG(1, 2))

	alph := r.LetterDistribution.TileMapping()
	for turn := 0; turn < 200 && s.PlayState != xwordgame.GameOver; turn++ {
		m := randomLegalMove(s, r, rng, maxLtr)
		mm, err := macondoMove(g, s, m, alph, r.LetterDistribution)
		if err != nil {
			tb.Fatal(err)
		}
		if _, err := s.ApplyMove(r, rng, m); err != nil {
			continue
		}
		if err := g.PlayMove(mm, true, 0); err != nil {
			tb.Fatal(err)
		}
		theirs, err := StateFromGame(g)
		if err != nil {
			tb.Fatal(err)
		}
		for p := range xwordgame.MaxPlayers {
			if err := s.AssignRack(p, theirs.Rack(p)); err != nil {
				tb.Fatal(err)
			}
		}
	}
	return g.History(), r
}

// BenchmarkLoadByReplay is the cost of rebuilding a position from the event
// log: unmarshal the events, then drive the state machine through all of them.
func BenchmarkLoadByReplay(b *testing.B) {
	hist, r := playedOutGame(b)
	raw, err := proto.Marshal(hist)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(len(hist.Events)), "events/game")
	b.ReportMetric(float64(len(raw)), "wire-bytes")

	b.ResetTimer()
	for range b.N {
		var h macondopb.GameHistory
		if err := proto.Unmarshal(raw, &h); err != nil {
			b.Fatal(err)
		}
		if _, err := ReplayHistory(&h, r, nil); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadBySnapshot is the cost of decoding a stored position.
func BenchmarkLoadBySnapshot(b *testing.B) {
	hist, r := playedOutGame(b)
	res, err := ReplayHistory(hist, r, nil)
	if err != nil {
		b.Fatal(err)
	}
	enc := res.State.Encode()
	b.ReportMetric(float64(len(enc)), "wire-bytes")

	b.ResetTimer()
	for range b.N {
		var s xwordgame.State
		if err := s.Decode(enc); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadByReplayFromTurns is the cost of the shape actually proposed:
// rebuilding from game_turns, where each event is stored as protojson in a
// jsonb column rather than as binary proto. protojson is materially slower to
// parse, so measuring binary and calling it the cost of replaying the turns
// table would flatter the option.
func BenchmarkLoadByReplayFromTurns(b *testing.B) {
	hist, r := playedOutGame(b)

	// One protojson document per event, as game_turns holds them.
	rows := make([][]byte, len(hist.Events))
	total := 0
	for i, e := range hist.Events {
		raw, err := protojson.Marshal(e)
		if err != nil {
			b.Fatal(err)
		}
		rows[i] = raw
		total += len(raw)
	}
	b.ReportMetric(float64(len(rows)), "events/game")
	b.ReportMetric(float64(total), "jsonb-bytes")

	b.ResetTimer()
	for range b.N {
		h := &macondopb.GameHistory{
			Lexicon:            hist.Lexicon,
			LetterDistribution: hist.LetterDistribution,
			BoardLayout:        hist.BoardLayout,
			Variant:            hist.Variant,
			ChallengeRule:      hist.ChallengeRule,
			Players:            hist.Players,
			LastKnownRacks:     hist.LastKnownRacks,
			FinalScores:        hist.FinalScores,
			PlayState:          hist.PlayState,
			Events:             make([]*macondopb.GameEvent, 0, len(rows)),
		}
		for _, raw := range rows {
			e := &macondopb.GameEvent{}
			if err := protojson.Unmarshal(raw, e); err != nil {
				b.Fatal(err)
			}
			h.Events = append(h.Events, e)
		}
		if _, err := ReplayHistory(h, r, nil); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalHistoryOnly separates parsing the log from replaying it,
// because the two scale differently: parsing is proportional to the bytes and
// replaying to the rules work per event.
func BenchmarkUnmarshalHistoryOnly(b *testing.B) {
	hist, _ := playedOutGame(b)
	raw, err := proto.Marshal(hist)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for range b.N {
		var h macondopb.GameHistory
		if err := proto.Unmarshal(raw, &h); err != nil {
			b.Fatal(err)
		}
	}
}

// TestLoadCostShape reports the shape of the game the benchmarks measure, so
// the per-event cost can be read off rather than guessed.
func TestLoadCostShape(t *testing.T) {
	hist, r := playedOutGame(t)
	res, err := ReplayHistory(hist, r, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("events=%d snapshot=%d bytes", len(hist.Events), len(res.State.Encode()))
}
