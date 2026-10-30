package tournament

import (
	"fmt"
	"slices"
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
