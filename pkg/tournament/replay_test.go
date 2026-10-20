//go:build replay

package tournament

// This replay harness checks that a change to ClassicDivision leaves
// divisions without pairing lag behaving exactly as before. It plays real
// divisions (the Woogles divisions in testdata/wtf5.json and the TSH
// events in pkg/broadcasts/testdata) under several unlagged configurations,
// and writes the division state after every result to a transcript. Run it
// in two checkouts and diff the transcripts:
//
//	REPLAY_OUT=/tmp/new.txt go test -tags replay ./pkg/tournament/ -run TestReplayHarness
//	(in a checkout of the old revision, with this file copied in)
//	REPLAY_OUT=/tmp/old.txt go test -tags replay ./pkg/tournament/ -run TestReplayHarness
//	diff /tmp/old.txt /tmp/new.txt
//
// It only uses ClassicDivision methods that predate pairing lag, so it can
// be copied into an older checkout as is. Which player goes first can be
// random and isn't seeded, so the transcript ignores it.

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/woogles-io/liwords/pkg/broadcasts"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type replayPlayer struct {
	id     string
	rating int32
	// The opponent and the player's own and opponent's score in each
	// original round, where known.
	opp    map[int]string
	scores map[int][2]int
}

type replayDivision struct {
	name     string
	players  []*replayPlayer
	rounds   int
	controls []*pb.RoundControl // the original round controls, if known
	divCtrls *pb.DivisionControls
}

type replayConfig struct {
	name      string
	methods   func(rounds int) []pb.PairingMethod // nil for the original controls
	autoStart bool
	gibsonize bool
	addAfter  int // add a latecomer once this round is complete, -1 for none
	remAfter  int // remove a player once this round is complete, -1 for none
}

func replayRoundControls(methods []pb.PairingMethod) []*pb.RoundControl {
	rcs := []*pb.RoundControl{}
	for i, m := range methods {
		rcs = append(rcs, &pb.RoundControl{FirstMethod: pb.FirstMethod_AUTOMATIC_FIRST,
			PairingMethod: m, GamesPerRound: 1, Round: int32(i), Factor: 1,
			MaxRepeats: 1, AllowOverMaxRepeats: true, RepeatRelativeWeight: 1,
			WinDifferenceRelativeWeight: 1})
	}
	return rcs
}

func repeatMethod(m pb.PairingMethod, n int) []pb.PairingMethod {
	out := make([]pb.PairingMethod, n)
	for i := range out {
		out[i] = m
	}
	return out
}

var replayConfigs = []replayConfig{
	{name: "koth-manualstart", methods: func(n int) []pb.PairingMethod { return repeatMethod(pb.PairingMethod_KING_OF_THE_HILL, n) }, addAfter: -1, remAfter: -1},
	{name: "koth-autostart-gibson", methods: func(n int) []pb.PairingMethod { return repeatMethod(pb.PairingMethod_KING_OF_THE_HILL, n) }, autoStart: true, gibsonize: true, addAfter: -1, remAfter: -1},
	{name: "swiss-koth-final-autostart", methods: func(n int) []pb.PairingMethod {
		m := repeatMethod(pb.PairingMethod_SWISS, n)
		m[n-1] = pb.PairingMethod_KING_OF_THE_HILL
		return m
	}, autoStart: true, addAfter: 1, remAfter: 2},
	{name: "fontes-factor-gibson", methods: func(n int) []pb.PairingMethod {
		m := repeatMethod(pb.PairingMethod_FACTOR, n)
		for i := 0; i < 3 && i < n; i++ {
			m[i] = pb.PairingMethod_INITIAL_FONTES
		}
		return m
	}, gibsonize: true, addAfter: -1, remAfter: 3},
	{name: "random-then-swiss-autostart", methods: func(n int) []pb.PairingMethod {
		m := repeatMethod(pb.PairingMethod_SWISS, n)
		m[0] = pb.PairingMethod_RANDOM
		return m
	}, autoStart: true, addAfter: 2, remAfter: 1},
	{name: "australian-draw-autostart", methods: func(n int) []pb.PairingMethod { return repeatMethod(pb.PairingMethod_AUSTRALIAN_DRAW, n) }, autoStart: true, addAfter: -1, remAfter: 2},
}

func loadReplayDivisions(t *testing.T) []*replayDivision {
	divs := []*replayDivision{}

	// Real Woogles divisions.
	bts, err := os.ReadFile("./testdata/wtf5.json")
	if err != nil {
		t.Fatal(err)
	}
	var wtf map[string]struct {
		JSON json.RawMessage `json:"json"`
	}
	if err := json.Unmarshal(bts, &wtf); err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for name := range wtf {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cd := &ClassicDivision{}
		if err := json.Unmarshal(wtf[name].JSON, cd); err != nil {
			t.Fatal(err)
		}
		rd := &replayDivision{name: "wtf5-" + name, rounds: len(cd.RoundControls),
			controls: cd.RoundControls, divCtrls: cd.DivisionControls}
		for _, p := range cd.Players.Persons {
			rd.players = append(rd.players, &replayPlayer{id: p.Id, rating: p.Rating,
				opp: map[int]string{}, scores: map[int][2]int{}})
		}
		for r := range cd.Matrix {
			for pidx, key := range cd.Matrix[r] {
				pairing := cd.PairingMap[key]
				if pairing == nil || len(pairing.Players) != 2 || len(pairing.Games) == 0 {
					continue
				}
				me := 0
				if int(pairing.Players[1]) == pidx {
					me = 1
				}
				rd.players[pidx].opp[r] = cd.Players.Persons[pairing.Players[1-me]].Id
				rd.players[pidx].scores[r] = [2]int{int(pairing.Games[0].Scores[me]), int(pairing.Games[0].Scores[1-me])}
			}
		}
		divs = append(divs, rd)
	}

	// Real TSH events.
	parser := &broadcasts.TSHNewtParser{}
	for _, file := range []string{"2024-spc-a-tourney.js", "sample-tourney.js", "test22-tourney.js"} {
		data, err := os.ReadFile("../broadcasts/testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		divNames, err := parser.ListDivisions(data)
		if err != nil {
			t.Fatal(err)
		}
		for _, dn := range divNames {
			fd, err := parser.ParseDivision(data, dn)
			if err != nil {
				t.Fatal(err)
			}
			if len(fd.Players) < 6 || fd.TotalRounds < 4 {
				continue
			}
			rd := &replayDivision{name: file + "-" + dn, rounds: min(fd.TotalRounds, 16)}
			byID := map[int]*broadcasts.FeedPlayer{}
			for i := range fd.Players {
				byID[fd.Players[i].ID] = &fd.Players[i]
			}
			idOf := func(p *broadcasts.FeedPlayer) string {
				return fmt.Sprintf("tsh%d:%s", p.ID, strings.ReplaceAll(p.Name, " ", "_"))
			}
			for i := range fd.Players {
				fp := &fd.Players[i]
				rp := &replayPlayer{id: idOf(fp), rating: int32(fp.Rating), opp: map[int]string{}, scores: map[int][2]int{}}
				for r := 0; r < len(fp.Pairings) && r < len(fp.Scores); r++ {
					opp := byID[fp.Pairings[r]]
					if opp == nil || r >= len(opp.Scores) {
						continue
					}
					rp.opp[r] = idOf(opp)
					rp.scores[r] = [2]int{fp.Scores[r], opp.Scores[r]}
				}
				rd.players = append(rd.players, rp)
			}
			divs = append(divs, rd)
		}
	}
	return divs
}

// replayScore returns the scores for a game: the real ones if the players
// met in that round of the original event, otherwise deterministic ones.
func replayScore(rd *replayDivision, round int, a, b string) (int, int) {
	for _, p := range rd.players {
		if p.id == a && p.opp[round] == b {
			return p.scores[round][0], p.scores[round][1]
		}
	}
	h := fnv.New64a()
	fmt.Fprintf(h, "%d|%s|%s", round, a, b)
	v := h.Sum64()
	return 300 + int(v%250), 300 + int((v>>16)%250)
}

func replaySnapshot(tc *ClassicDivision) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "current=%d\n", tc.CurrentRound)
	for r := range tc.Matrix {
		fmt.Fprintf(&sb, " r%d:", r)
		for pidx, key := range tc.Matrix[r] {
			pairing := tc.PairingMap[key]
			if key == "" || pairing == nil || len(pairing.Players) != 2 {
				sb.WriteString(" -")
				continue
			}
			me := 0
			if int(pairing.Players[1]) == pidx {
				me = 1
			}
			fmt.Fprintf(&sb, " %d/%d", pairing.Players[1-me], pairing.Outcomes[me])
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func replayResponse(resp *pb.DivisionPairingsResponse) string {
	if resp == nil {
		return "resp=nil\n"
	}
	ps := []string{}
	for _, p := range resp.DivisionPairings {
		players, outcomes := p.Players, p.Outcomes
		if len(players) == 2 && players[0] > players[1] {
			players = []int32{players[1], players[0]}
			outcomes = []pb.TournamentGameResult{outcomes[1], outcomes[0]}
		}
		ps = append(ps, fmt.Sprintf("%d:%v:%v", p.Round, players, outcomes))
	}
	sort.Strings(ps)
	rounds := []int{}
	for r := range resp.DivisionStandings {
		rounds = append(rounds, int(r))
	}
	sort.Ints(rounds)
	st := []string{}
	for _, r := range rounds {
		ids := []string{}
		for _, s := range resp.DivisionStandings[int32(r)].Standings {
			ids = append(ids, fmt.Sprintf("%s:%d-%d-%d:%d:%v", s.PlayerId, s.Wins, s.Losses, s.Draws, s.Spread, s.Gibsonized))
		}
		st = append(st, fmt.Sprintf("%d=[%s]", r, strings.Join(ids, " ")))
	}
	return fmt.Sprintf("resp pairings=%v standings=%v\n", ps, st)
}

func TestReplayHarness(t *testing.T) {
	out := os.Getenv("REPLAY_OUT")
	if out == "" {
		t.Skip("set REPLAY_OUT to the transcript file to write")
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	numRuns, numResults := 0, 0

	for _, rd := range loadReplayDivisions(t) {
		cfgs := replayConfigs
		if rd.controls != nil {
			cfgs = append([]replayConfig{{name: "original-controls", addAfter: -1, remAfter: -1}}, replayConfigs...)
		}
		for _, cfg := range cfgs {
			numRuns++
			fmt.Fprintf(f, "=== %s / %s\n", rd.name, cfg.name)
			tc := NewClassicDivision("replay", rd.name)
			tc.Seed = 12345
			latecomer := rd.players[len(rd.players)-1]
			persons := &pb.TournamentPersons{}
			for i, p := range rd.players {
				if cfg.addAfter >= 0 && i == len(rd.players)-1 {
					continue
				}
				persons.Persons = append(persons.Persons, &pb.TournamentPerson{Id: p.id, Rating: p.rating})
			}
			if _, err := tc.AddPlayers(persons); err != nil {
				fmt.Fprintf(f, "addplayers err=%v\n", err)
				continue
			}
			rcs := rd.controls
			if cfg.methods != nil {
				rcs = replayRoundControls(cfg.methods(rd.rounds))
			}
			if _, _, err := tc.SetRoundControls(rcs); err != nil {
				fmt.Fprintf(f, "setroundcontrols err=%v\n", err)
				continue
			}
			dc := &pb.DivisionControls{SuspendedResult: pb.TournamentGameResult_FORFEIT_LOSS, SuspendedSpread: -50,
				AutoStart: cfg.autoStart, Gibsonize: cfg.gibsonize, GibsonSpread: 200, MinimumPlacement: 2}
			if cfg.methods == nil && rd.divCtrls != nil {
				dc = rd.divCtrls
				dc.GameRequest = nil
			}
			if _, _, err := tc.SetDivisionControls(dc); err != nil {
				fmt.Fprintf(f, "setdivisioncontrols err=%v\n", err)
				continue
			}
			fmt.Fprint(f, replaySnapshot(tc))
			rng := rand.New(rand.NewSource(int64(numRuns)))
			added, removed := false, false

			for step := 0; step < 400; step++ {
				finished, err := tc.IsFinished()
				if err != nil || finished {
					fmt.Fprintf(f, "finished=%v err=%v\n", finished, err)
					break
				}
				// Add a latecomer and remove a player once the configured
				// rounds are complete, whatever round is open by then.
				if cfg.addAfter >= 0 && !added {
					if done, _ := tc.IsRoundComplete(cfg.addAfter); done {
						added = true
						resp, err := tc.AddPlayers(&pb.TournamentPersons{Persons: []*pb.TournamentPerson{{Id: latecomer.id, Rating: latecomer.rating}}})
						fmt.Fprintf(f, "add %s err=%v\n", latecomer.id, err)
						fmt.Fprint(f, replayResponse(resp), replaySnapshot(tc))
					}
				}
				if cfg.remAfter >= 0 && !removed {
					if done, _ := tc.IsRoundComplete(cfg.remAfter); done {
						removed = true
						victim := tc.Players.Persons[len(tc.Players.Persons)/2].Id
						resp, err := tc.RemovePlayers(&pb.TournamentPersons{Persons: []*pb.TournamentPerson{{Id: victim}}})
						fmt.Fprintf(f, "remove %s err=%v\n", victim, err)
						fmt.Fprint(f, replayResponse(resp), replaySnapshot(tc))
					}
				}
				round := int(tc.CurrentRound)
				complete := round >= 0
				if round >= 0 {
					complete, _ = tc.IsRoundComplete(round)
				}
				if round < 0 || complete {
					err := tc.StartRound(true)
					fmt.Fprintf(f, "start err=%v\n", err)
					fmt.Fprint(f, replaySnapshot(tc))
					if err != nil {
						break
					}
					continue
				}
				// Submit every remaining game in the round, in a shuffled
				// order.
				games := [][2]string{}
				seen := map[string]bool{}
				for _, key := range tc.Matrix[round] {
					pairing := tc.PairingMap[key]
					if pairing == nil || len(pairing.Players) != 2 || pairing.Players[0] == pairing.Players[1] ||
						seen[key] || pairing.Outcomes[0] != pb.TournamentGameResult_NO_RESULT {
						continue
					}
					seen[key] = true
					a, b := tc.Players.Persons[pairing.Players[0]].Id, tc.Players.Persons[pairing.Players[1]].Id
					if b < a {
						a, b = b, a
					}
					games = append(games, [2]string{a, b})
				}
				if len(games) == 0 {
					fmt.Fprintf(f, "stuck round=%d\n", round)
					break
				}
				sort.Slice(games, func(i, j int) bool { return games[i][0] < games[j][0] })
				rng.Shuffle(len(games), func(i, j int) { games[i], games[j] = games[j], games[i] })
				for _, g := range games {
					s1, s2 := replayScore(rd, round, g[0], g[1])
					r1, r2 := pb.TournamentGameResult_DRAW, pb.TournamentGameResult_DRAW
					if s1 > s2 {
						r1, r2 = pb.TournamentGameResult_WIN, pb.TournamentGameResult_LOSS
					} else if s2 > s1 {
						r1, r2 = pb.TournamentGameResult_LOSS, pb.TournamentGameResult_WIN
					}
					resp, err := tc.SubmitResult(round, g[0], g[1], s1, s2, r1, r2, pb.GameEndReason_STANDARD, false, 0, "")
					numResults++
					fmt.Fprintf(f, "result r%d %s %d %s %d err=%v\n", round, g[0], s1, g[1], s2, err)
					fmt.Fprint(f, replayResponse(resp), replaySnapshot(tc))
				}
			}
		}
	}
	fmt.Fprintf(f, "runs=%d results=%d\n", numRuns, numResults)
	t.Logf("runs=%d results=%d", numRuns, numResults)
}
