package tournament

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	tpb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

func previewPlayers(n int) []*tpb.PreviewPlayer {
	players := []*tpb.PreviewPlayer{}
	for i := range n {
		players = append(players, &tpb.PreviewPlayer{Name: placeholderName(i), Placeholder: true})
	}
	return players
}

func findScheme(resp *tpb.PreviewPairingsResponse, method pb.PairingMethod, within bool) *tpb.SchemePreview {
	for _, s := range resp.Schemes {
		if s.PairingMethod == method && s.PlayWithinTeam == within {
			return s
		}
	}
	return nil
}

func TestPreviewSuggestions(t *testing.T) {
	is := is.New(t)

	// 10 players over 5 rounds: the snaked teams fit exactly and give
	// everyone nearly the same schedule; Initial Fontes needs 12.
	resp, err := previewDivision(previewPlayers(10), 7, 5, tpb.PlannerIntent_PLANNER_FAIR_START)
	is.NoErr(err)
	is.True(resp.Schemes[0].Suggested)
	is.Equal(resp.Schemes[0].PairingMethod, pb.PairingMethod_SNAKED_ROUND_ROBIN)
	is.Equal(resp.Schemes[0].PlayWithinTeam, false)
	is.Equal(resp.Schemes[0].CycleLength, int32(5))
	fontes := findScheme(resp, pb.PairingMethod_INITIAL_FONTES, false)
	is.True(!fontes.Available)
	is.Equal(fontes.UnavailableReason, "5 rounds of Initial Fontes needs at least 12 players.")
	// A shorter Initial Fontes keeps the top seeds apart.
	is.Equal(resp.Alternatives[0].Rounds, int32(3))
	is.Equal(resp.Alternatives[0].PairingMethod, pb.PairingMethod_INITIAL_FONTES)
	// Teams of 5 within teams take 5 rounds: 4 teammates and one game
	// against the other team.
	within := findScheme(resp, pb.PairingMethod_INTERLEAVED_ROUND_ROBIN, true)
	is.Equal(within.CycleLength, int32(5))
	for _, st := range within.PlayerStats {
		is.Equal(st.OwnGroupGames, int32(4))
		is.Equal(st.OtherGroupGames, int32(1))
	}
	rr := findScheme(resp, pb.PairingMethod_ROUND_ROBIN, false)
	is.Equal(rr.CycleLength, int32(9))
	is.True(slices.Contains(rr.Notes, "Only 5 of the 9 rounds a full cycle takes, so some players face a tougher schedule than others."))

	// 16 players over 3 rounds: Initial Fontes, and the top quarter never
	// meet.
	resp, err = previewDivision(previewPlayers(16), 7, 3, tpb.PlannerIntent_PLANNER_FAIR_START)
	is.NoErr(err)
	is.Equal(resp.Schemes[0].PairingMethod, pb.PairingMethod_INITIAL_FONTES)
	is.Equal(resp.Schemes[0].TopSeedGames, int32(0))
	is.Equal(len(resp.Alternatives), 0)

	// 18 players over 18 rounds: two cycles of the snaked teams fit exactly,
	// but everyone would meet only 9 of their 17 opponents, twice each. The
	// round robin meets everyone, and 17 rounds of it then COP is offered.
	for _, intent := range []tpb.PlannerIntent{tpb.PlannerIntent_PLANNER_FAIR_START, tpb.PlannerIntent_PLANNER_WHOLE_EVENT} {
		resp, err = previewDivision(previewPlayers(18), 7, 18, intent)
		is.NoErr(err)
		is.Equal(resp.Schemes[0].PairingMethod, pb.PairingMethod_ROUND_ROBIN)
		is.Equal(resp.Schemes[0].MissedOpponents, int32(0))
		snaked := findScheme(resp, pb.PairingMethod_SNAKED_ROUND_ROBIN, false)
		is.Equal(snaked.MissedOpponents, int32(8))
		is.True(slices.Contains(snaked.Notes, "Players meet only 9 different opponents in 18 rounds, when they could meet up to 17."))
		alt := resp.Alternatives[len(resp.Alternatives)-1]
		if intent == tpb.PlannerIntent_PLANNER_WHOLE_EVENT {
			alt = resp.Alternatives[0]
		}
		is.Equal(alt.Rounds, int32(17))
		is.Equal(alt.Reason, "Pre-pair 17 rounds, a full Round Robin, and let COP pair the last round. Its final round pairs the prize contenders King of the Hill style.")
	}

	// 18 players over 7 rounds: groups of 8 can't keep seeds 1–5 apart, and
	// neither can groups of 4, so the alternative says how often they meet.
	resp, err = previewDivision(previewPlayers(18), 7, 7, tpb.PlannerIntent_PLANNER_FAIR_START)
	is.NoErr(err)
	is.Equal(resp.Schemes[0].PairingMethod, pb.PairingMethod_INITIAL_FONTES)
	is.Equal(resp.Schemes[0].TopSeedGames, int32(3))
	is.Equal(resp.Alternatives[0].Rounds, int32(3))
	is.Equal(resp.Alternatives[0].Reason, "Pre-pair 3 rounds with Initial Fontes, so seeds 1–5 meet only once, and let COP pair the rest.")

	// The whole event: a full round robin.
	resp, err = previewDivision(previewPlayers(10), 7, 9, tpb.PlannerIntent_PLANNER_WHOLE_EVENT)
	is.NoErr(err)
	is.Equal(resp.Schemes[0].PairingMethod, pb.PairingMethod_ROUND_ROBIN)

	// Two teams without teams assigned: a team split by seed, across teams.
	resp, err = previewDivision(previewPlayers(10), 7, 5, tpb.PlannerIntent_PLANNER_TWO_TEAMS)
	is.NoErr(err)
	is.True(resp.Schemes[0].PairingMethod != pb.PairingMethod_ROUND_ROBIN)
	is.True(!resp.Schemes[0].PlayWithinTeam)

	// With teams assigned, only the assigned teams round robin is possible.
	players := previewPlayers(12)
	for i, p := range players {
		p.Team = "Portland"
		if i%5 == 2 || i%5 == 4 || i == 11 {
			p.Team = "Seattle"
		}
	}
	resp, err = previewDivision(players, 7, 7, tpb.PlannerIntent_PLANNER_TWO_TEAMS)
	is.NoErr(err)
	is.Equal(resp.Schemes[0].PairingMethod, pb.PairingMethod_ASSIGNED_TEAM_ROUND_ROBIN)
	is.Equal(resp.Schemes[0].MaxByes, int32(2))
	for _, s := range resp.Schemes[1:] {
		is.True(!s.Available)
	}
}

// TestPreviewMatchesDivision checks that the preview shows exactly the
// pairings and firsts the division itself makes.
func TestPreviewMatchesDivision(t *testing.T) {
	is := is.New(t)
	names := map[string]int32{}
	for i := range 11 {
		names[fmt.Sprintf("Player %d", i)] = int32(2000 - 10*i)
	}
	tc, err := compactNewClassicDivision(makeTournamentPersons(names), assignedTeamControls(5, pb.PairingMethod_SNAKED_ROUND_ROBIN), true)
	is.NoErr(err)
	players := []*tpb.PreviewPlayer{}
	for _, p := range tc.Players.Persons {
		players = append(players, &tpb.PreviewPlayer{Name: p.Id, Rating: p.Rating})
	}
	resp, err := previewDivision(players, tc.Seed, 5, tpb.PlannerIntent_PLANNER_FAIR_START)
	is.NoErr(err)
	snake := findScheme(resp, pb.PairingMethod_SNAKED_ROUND_ROBIN, false)
	for round, r := range snake.Rounds {
		for _, g := range r.Games {
			pairing := tc.PairingMap[tc.Matrix[round][g.First]]
			is.Equal(pairing.Players, []int32{g.First, g.Second})
		}
		for _, b := range r.Byes {
			pairing := tc.PairingMap[tc.Matrix[round][b]]
			is.Equal(pairing.Players, []int32{b, b})
		}
	}
}

func TestPreviewIsQuick(t *testing.T) {
	is := is.New(t)
	start := time.Now()
	_, err := previewDivision(previewPlayers(60), 7, 7, tpb.PlannerIntent_PLANNER_FAIR_START)
	is.NoErr(err)
	t.Logf("60 players: %v", time.Since(start))
	is.True(time.Since(start) < 5*time.Second)
}

func TestPlaceholderName(t *testing.T) {
	is := is.New(t)
	is.Equal(placeholderName(0), "A")
	is.Equal(placeholderName(25), "Z")
	is.Equal(placeholderName(26), "AA")
	is.Equal(placeholderName(27), "AB")
}

// Every sentence the planner shows, over many fields, round counts and
// intents, reads as a sentence: no "1 rounds", "2–2" or stray spacing.
func TestPreviewSentences(t *testing.T) {
	singular := regexp.MustCompile(`(^|[^\d.–])1 (rounds|times|games|byes|pairings|players|opponents|cycles|groups)\b`)
	sameSpan := regexp.MustCompile(`\b(\d+)–(\d+)\b`)
	teamPlayers := func(n int) []*tpb.PreviewPlayer {
		players := previewPlayers(n)
		for i, p := range players {
			p.Team = []string{"Portland", "Seattle"}[i*3/(2*n)%2]
		}
		return players
	}
	for _, n := range []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 16, 18, 21} {
		for _, rounds := range []int{1, 2, 3, 4, 5, 7, 9, 11, 12, 17, 18, 20} {
			for _, intent := range []tpb.PlannerIntent{tpb.PlannerIntent_PLANNER_FAIR_START,
				tpb.PlannerIntent_PLANNER_WHOLE_EVENT, tpb.PlannerIntent_PLANNER_TWO_TEAMS} {
				for _, players := range [][]*tpb.PreviewPlayer{previewPlayers(n), teamPlayers(n)} {
					resp, err := previewDivision(players, 7, rounds, intent)
					if err != nil {
						t.Fatal(err)
					}
					sentences := []string{}
					for _, s := range resp.Schemes {
						sentences = append(append(sentences, s.Notes...), s.UnavailableReason)
					}
					for _, a := range resp.Alternatives {
						sentences = append(sentences, a.Reason)
					}
					for _, s := range sentences {
						if s == "" {
							continue
						}
						bad := singular.MatchString(s) || strings.Contains(s, "  ") || !strings.HasSuffix(s, ".")
						for _, m := range sameSpan.FindAllStringSubmatch(s, -1) {
							bad = bad || m[1] == m[2]
						}
						if bad {
							t.Errorf("%d players, %d rounds, intent %v: %q", n, rounds, intent, s)
						}
					}
				}
			}
		}
	}
}

// TestPreviewRoundsPerOpponent checks a Team Round Robin that plays each pair
// two rounds in a row against the division itself.
func TestPreviewRoundsPerOpponent(t *testing.T) {
	is := is.New(t)
	persons := &pb.TournamentPersons{}
	for i, team := range []string{"Portland", "Seattle", "Portland", "Portland", "Seattle", "Seattle"} {
		persons.Persons = append(persons.Persons, &pb.TournamentPerson{
			Id: fmt.Sprintf("u%d:player%d", i, i), Rating: int32(2000 - 10*i), Team: team})
	}
	controls := assignedTeamControls(6, pb.PairingMethod_ASSIGNED_TEAM_ROUND_ROBIN)
	for _, rc := range controls {
		rc.GamesPerRound = 2
	}
	tc, err := compactNewClassicDivision(persons, controls, true)
	is.NoErr(err)

	players := []*tpb.PreviewPlayer{}
	for _, p := range tc.Players.Persons {
		players = append(players, &tpb.PreviewPlayer{Name: p.Id, Rating: p.Rating, Team: p.Team})
	}
	resp, err := previewDivisionPer(players, tc.Seed, 6, tpb.PlannerIntent_PLANNER_TWO_TEAMS, 2)
	is.NoErr(err)
	teams := findScheme(resp, pb.PairingMethod_ASSIGNED_TEAM_ROUND_ROBIN, false)
	is.True(teams.Suggested)
	is.Equal(teams.CycleLength, int32(6))
	is.Equal(teams.MissedOpponents, int32(0))
	is.True(slices.Contains(teams.Notes, "Each pair plays 2 rounds in a row, going first once each."))
	for round, r := range teams.Rounds {
		for _, g := range r.Games {
			pairing := tc.PairingMap[tc.Matrix[round][g.First]]
			is.Equal(pairing.Players, []int32{g.First, g.Second})
		}
	}
	// Rounds 1 and 2 pair the same players, with firsts swapped.
	is.Equal(len(teams.Rounds[0].Games), 3)
	for _, g := range teams.Rounds[0].Games {
		swapped := false
		for _, h := range teams.Rounds[1].Games {
			swapped = swapped || (h.First == g.Second && h.Second == g.First)
		}
		is.True(swapped)
	}
}
