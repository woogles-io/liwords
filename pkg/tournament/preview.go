package tournament

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/pair"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	tpb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

// plannerScheme is one pre-paired scheme the pairing planner compares.
type plannerScheme struct {
	method pb.PairingMethod
	within bool
	name   string
}

var plannerSchemes = []plannerScheme{
	{pb.PairingMethod_INITIAL_FONTES, false, "Initial Fontes"},
	{pb.PairingMethod_SNAKED_ROUND_ROBIN, false, "Snaked Round Robin"},
	{pb.PairingMethod_INTERLEAVED_ROUND_ROBIN, false, "Shirts and Skins Round Robin"},
	{pb.PairingMethod_SNAKED_ROUND_ROBIN, true, "Snaked Round Robin, within teams"},
	{pb.PairingMethod_INTERLEAVED_ROUND_ROBIN, true, "Shirts and Skins Round Robin, within teams"},
	{pb.PairingMethod_ROUND_ROBIN, false, "Round Robin"},
	{pb.PairingMethod_ASSIGNED_TEAM_ROUND_ROBIN, false, "Team Round Robin"},
}

func (s plannerScheme) teamSplit() bool {
	return s.method == pb.PairingMethod_INTERLEAVED_ROUND_ROBIN ||
		s.method == pb.PairingMethod_SNAKED_ROUND_ROBIN || s.method == pb.PairingMethod_ASSIGNED_TEAM_ROUND_ROBIN
}

// PreviewPairings shows how each pre-paired scheme would pair rounds rounds
// of the division, ranked for intent. If expectedPlayers is more than the
// players registered, placeholders named A, B, C and so on make up the rest,
// seeded last. Nothing in the division changes.
func PreviewPairings(ctx context.Context, ts TournamentStore, id, division string, rounds, expectedPlayers int,
	intent tpb.PlannerIntent) (*tpb.PreviewPairingsResponse, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	t.RLock()
	defer t.RUnlock()
	divisionObject, ok := t.Divisions[division]
	if !ok {
		return nil, entity.NewWooglesError(pb.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
	}
	cd, ok := divisionObject.DivisionManager.(*ClassicDivision)
	if !ok {
		return nil, entity.NewWooglesError(pb.WooglesError_TOURNAMENT_NIL_DIVISION_MANAGER, t.Name, division)
	}
	players := []*tpb.PreviewPlayer{}
	for _, p := range cd.Players.Persons {
		if p.Suspended {
			continue
		}
		// IDs are uuid:username online and token:name in IRL mode.
		_, name, _ := strings.Cut(p.Id, ":")
		players = append(players, &tpb.PreviewPlayer{Name: name, Rating: p.Rating, Team: p.Team})
	}
	for i := 0; len(players) < expectedPlayers; i++ {
		players = append(players, &tpb.PreviewPlayer{Name: placeholderName(i), Placeholder: true})
	}
	return previewDivision(players, cd.Seed, rounds, intent)
}

// placeholderName is A, B, ..., Z, AA, AB, ...
func placeholderName(i int) string {
	name := ""
	for i++; i > 0; i = (i - 1) / 26 {
		name = string(rune('A'+(i-1)%26)) + name
	}
	return name
}

// previewDivision pairs a throwaway copy of the players with every scheme.
// players must be in seed order; their pairings depend only on that order and
// the seed, so with the division's own players and seed the preview is
// exactly what the division will get.
func previewDivision(players []*tpb.PreviewPlayer, seed uint64, rounds int, intent tpb.PlannerIntent) (*tpb.PreviewPairingsResponse, error) {
	n := len(players)
	if n < 2 {
		return nil, fmt.Errorf("the planner needs at least two players")
	}
	if rounds < 1 {
		return nil, fmt.Errorf("pre-pair at least one round")
	}
	topSeeds := min(max(2, (n+3)/4), 8)
	resp := &tpb.PreviewPairingsResponse{Players: players, TopSeeds: int32(topSeeds)}
	teams := make([]string, n)
	hasTeams := false
	for i, p := range players {
		teams[i] = p.Team
		hasTeams = hasTeams || p.Team != ""
	}

	for _, s := range plannerSchemes {
		resp.Schemes = append(resp.Schemes, previewScheme(s, players, teams, hasTeams, seed, rounds, topSeeds))
	}
	resp.Schemes = rankSchemes(resp.Schemes, intent, rounds)
	fontes := func(r int) *tpb.SchemePreview {
		return previewScheme(plannerSchemes[0], players, teams, hasTeams, seed, r, topSeeds)
	}
	resp.Alternatives = roundsAlternatives(resp.Schemes, n, rounds, intent, topSeeds, hasTeams, fontes)
	return resp, nil
}

func schemeName(method pb.PairingMethod, within bool) string {
	for _, s := range plannerSchemes {
		if s.method == method && s.within == within {
			return s.name
		}
	}
	return method.String()
}

// pairSchedule pairs rounds rounds of s for the players through a throwaway
// division, and returns each round's opponents (-1 for a bye) and firsts.
func pairSchedule(s plannerScheme, players []*tpb.PreviewPlayer, seed uint64, rounds int) ([][]int, [][]bool, error) {
	n := len(players)
	persons := &pb.TournamentPersons{}
	for i, p := range players {
		// Distinct ratings keep the seed order exactly as given: sorting
		// isn't stable, so tied ratings could otherwise swap seeds.
		persons.Persons = append(persons.Persons, &pb.TournamentPerson{
			Id: fmt.Sprintf("%d:%s", i, p.Name), Rating: int32(n - i), Team: p.Team})
	}
	d := NewClassicDivision("planner", "planner")
	d.Seed = seed
	if _, err := d.AddPlayers(persons); err != nil {
		return nil, nil, err
	}
	rcs := []*pb.RoundControl{}
	for round := range rounds {
		rcs = append(rcs, &pb.RoundControl{
			PairingMethod:               s.method,
			PlayWithinTeam:              s.within,
			FirstMethod:                 pb.FirstMethod_AUTOMATIC_FIRST,
			GamesPerRound:               1,
			Round:                       int32(round),
			Factor:                      1,
			MaxRepeats:                  1,
			AllowOverMaxRepeats:         true,
			RepeatRelativeWeight:        1,
			WinDifferenceRelativeWeight: 1,
		})
	}
	if _, _, err := d.SetRoundControls(rcs); err != nil {
		return nil, nil, err
	}
	opps := make([][]int, rounds)
	firsts := make([][]bool, rounds)
	for round := range rounds {
		opps[round] = make([]int, n)
		firsts[round] = make([]bool, n)
		for p := range n {
			pairing := d.PairingMap[d.Matrix[round][p]]
			if pairing == nil {
				return nil, nil, fmt.Errorf("round %d was not paired", round+1)
			}
			a, b := int(pairing.Players[0]), int(pairing.Players[1])
			if a == b {
				opps[round][p] = -1
				continue
			}
			if a == p {
				opps[round][p] = b
				firsts[round][p] = true
			} else {
				opps[round][p] = a
			}
		}
	}
	return opps, firsts, nil
}

// unavailableReason says why s can't pair this field, if it can't, in terms
// a director can act on.
func unavailableReason(s plannerScheme, n, rounds int, hasTeams bool, teams []string) string {
	if hasTeams && s.method != pb.PairingMethod_ASSIGNED_TEAM_ROUND_ROBIN {
		return "Some players have teams, so only the Team Round Robin keeps teammates apart."
	}
	switch s.method {
	case pb.PairingMethod_ASSIGNED_TEAM_ROUND_ROBIN:
		if !hasTeams {
			return "No players have teams yet. Give each player a team to use this."
		}
		names := map[string]bool{}
		missing := 0
		for _, t := range teams {
			if t == "" {
				missing++
			}
			names[t] = true
		}
		if missing > 0 {
			return fmt.Sprintf("%d players have no team yet.", missing)
		}
		if len(names) != 2 {
			return fmt.Sprintf("This needs exactly two teams; there are %d.", len(names))
		}
	case pb.PairingMethod_INITIAL_FONTES:
		if rounds%2 == 0 {
			return "Initial Fontes needs an odd number of rounds."
		}
		if n < 2*(rounds+1) {
			return fmt.Sprintf("%d rounds of Initial Fontes needs at least %d players.", rounds, 2*(rounds+1))
		}
	}
	return ""
}

func previewScheme(s plannerScheme, players []*tpb.PreviewPlayer, teams []string, hasTeams bool, seed uint64,
	rounds, topSeeds int) *tpb.SchemePreview {
	n := len(players)
	sp := &tpb.SchemePreview{PairingMethod: s.method, PlayWithinTeam: s.within, Name: s.name}
	if reason := unavailableReason(s, n, rounds, hasTeams, teams); reason != "" {
		sp.UnavailableReason = reason
		return sp
	}

	// Pair far enough ahead to see the whole cycle, which is never more than
	// a round per player. Initial Fontes only ever runs its own rounds.
	horizon := rounds
	if s.method != pb.PairingMethod_INITIAL_FONTES {
		horizon = max(rounds, n+2)
	}
	opps, firsts, err := pairSchedule(s, players, seed, horizon)
	if err != nil {
		if s.method == pb.PairingMethod_INITIAL_FONTES {
			sp.UnavailableReason = fmt.Sprintf("%d players don't split into even groups of %d.", n, rounds+1)
		} else {
			sp.UnavailableReason = fmt.Sprintf("This can't pair %d players: %v", n, err)
		}
		return sp
	}
	sp.Available = true

	groups := make([]int, n)
	for i := range groups {
		groups[i] = -1
	}
	if s.teamSplit() {
		teamA, teamB, err := pair.Teams(n, s.method, teams)
		if err == nil {
			for gi, team := range [][]int{teamA, teamB} {
				for _, p := range team {
					groups[p] = gi
				}
			}
		}
	} else if s.method == pb.PairingMethod_INITIAL_FONTES {
		if gs, err := pair.InitialFontesGroups(n, rounds+1, seed); err == nil {
			for gi, g := range gs {
				for _, p := range g {
					groups[p] = gi
				}
			}
		}
	}
	for _, g := range groups {
		sp.Groups = append(sp.Groups, int32(g))
	}

	// The cycle: the rounds it takes for everyone to have played everyone
	// they play in the whole horizon.
	universe := make([]map[int]bool, n)
	for p := range n {
		universe[p] = map[int]bool{}
		for r := range horizon {
			if o := opps[r][p]; o >= 0 {
				universe[p][o] = true
			}
		}
	}
	if s.method == pb.PairingMethod_INITIAL_FONTES {
		sp.CycleLength = int32(rounds)
	} else {
		seen := make([]map[int]bool, n)
		for p := range n {
			seen[p] = map[int]bool{}
		}
		for r := range horizon {
			done := true
			for p := range n {
				if o := opps[r][p]; o >= 0 {
					seen[p][o] = true
				}
				done = done && len(seen[p]) == len(universe[p])
			}
			if done {
				sp.CycleLength = int32(r + 1)
				break
			}
		}
	}

	met := map[[2]int]bool{}
	for r := range rounds {
		round := &tpb.PreviewRound{}
		for p := range n {
			o := opps[r][p]
			if o < 0 {
				round.Byes = append(round.Byes, int32(p))
				continue
			}
			if firsts[r][p] {
				round.Games = append(round.Games, &tpb.PreviewGame{First: int32(p), Second: int32(o)})
			}
			if p < o {
				if met[[2]int{p, o}] {
					sp.Repeats++
				}
				met[[2]int{p, o}] = true
				if o < topSeeds {
					sp.TopSeedGames++
				}
			}
		}
		sp.Rounds = append(sp.Rounds, round)
	}

	sp.MinAverageOpponentSeed = math.Inf(1)
	for p := range n {
		st := &tpb.PreviewPlayerStats{}
		seedSum, games, run := 0, 0, 0
		var last *bool
		opponents := map[int]bool{}
		for r := range rounds {
			o := opps[r][p]
			if o < 0 {
				st.Byes++
				continue
			}
			games++
			seedSum += o + 1
			opponents[o] = true
			f := firsts[r][p]
			if f {
				st.Firsts++
			} else {
				st.Seconds++
			}
			if last != nil && *last == f {
				run++
			} else {
				run = 1
			}
			last = &f
			st.LongestRun = max(st.LongestRun, int32(run))
			if groups[p] >= 0 && groups[p] == groups[o] {
				st.OwnGroupGames++
			} else {
				st.OtherGroupGames++
			}
		}
		if games > 0 {
			st.AverageOpponentSeed = float64(seedSum) / float64(games)
			sp.MinAverageOpponentSeed = min(sp.MinAverageOpponentSeed, st.AverageOpponentSeed)
			sp.MaxAverageOpponentSeed = max(sp.MaxAverageOpponentSeed, st.AverageOpponentSeed)
		}
		st.DistinctOpponents = int32(len(opponents))
		sp.MissedOpponents = max(sp.MissedOpponents, int32(min(games, n-1)-len(opponents)))
		sp.MaxByes = max(sp.MaxByes, st.Byes)
		sp.MaxFirstImbalance = max(sp.MaxFirstImbalance, int32(math.Abs(float64(st.Firsts-st.Seconds))))
		sp.LongestRun = max(sp.LongestRun, st.LongestRun)
		sp.PlayerStats = append(sp.PlayerStats, st)
	}
	if math.IsInf(sp.MinAverageOpponentSeed, 1) {
		sp.MinAverageOpponentSeed = 0
	}
	sp.Notes = schemeNotes(s, sp, n, rounds, topSeeds)
	return sp
}

// fits says whether rounds is a whole number of the scheme's cycles.
func fits(sp *tpb.SchemePreview, rounds int) bool {
	return sp.CycleLength > 0 && rounds%int(sp.CycleLength) == 0
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func spanString(lo, hi int32) string {
	if lo == hi {
		return fmt.Sprint(lo)
	}
	return fmt.Sprintf("%d–%d", lo, hi)
}

func schemeNotes(s plannerScheme, sp *tpb.SchemePreview, n, rounds, topSeeds int) []string {
	notes := []string{}
	cycle := int(sp.CycleLength)

	// What each player's rounds are made of.
	ownLo, ownHi, otherLo, otherHi := int32(math.MaxInt32), int32(0), int32(math.MaxInt32), int32(0)
	for _, st := range sp.PlayerStats {
		ownLo, ownHi = min(ownLo, st.OwnGroupGames), max(ownHi, st.OwnGroupGames)
		otherLo, otherHi = min(otherLo, st.OtherGroupGames), max(otherHi, st.OtherGroupGames)
	}
	switch {
	case s.method == pb.PairingMethod_INITIAL_FONTES:
		notes = append(notes, fmt.Sprintf("Players are split by seed into groups of %d, one from each band of the field, and play everyone in their group once.", rounds+1))
	case s.teamSplit() && s.within:
		if otherHi == 0 {
			notes = append(notes, fmt.Sprintf("Each player plays %s games against their own team and none against the other team.",
				spanString(ownLo, ownHi)))
		} else {
			notes = append(notes, fmt.Sprintf("Each player plays %s games against their own team and %s against the other team.",
				spanString(ownLo, ownHi), spanString(otherLo, otherHi)))
		}
		if otherHi > 0 {
			notes = append(notes, "Both teams have an odd number of players, so the player who would sit out each round plays the other team's instead.")
		}
	case s.teamSplit():
		notes = append(notes, fmt.Sprintf("Each player plays %s games against the other team and none against their own.", spanString(otherLo, otherHi)))
	}

	// Repeats while other opponents go unplayed.
	if sp.MissedOpponents > 0 {
		lo, hi := int32(math.MaxInt32), int32(0)
		for _, st := range sp.PlayerStats {
			lo, hi = min(lo, st.DistinctOpponents), max(hi, st.DistinctOpponents)
		}
		notes = append(notes, fmt.Sprintf("Players meet only %s different opponents in %d rounds, when they could meet up to %d.",
			spanString(lo, hi), rounds, min(rounds, n-1)))
	}

	// How the rounds fit the scheme's cycle.
	switch {
	case cycle == 0:
	case rounds == cycle:
		notes = append(notes, fmt.Sprintf("Fits exactly: %s is one complete cycle.", plural(rounds, "round", "rounds")))
	case rounds%cycle == 0:
		notes = append(notes, fmt.Sprintf("Runs the full %s-round cycle %d times, so players meet again.", fmt.Sprint(cycle), rounds/cycle))
	case rounds < cycle:
		notes = append(notes, fmt.Sprintf("Only %d of the %d rounds a full cycle takes, so some players face a tougher schedule than others.", rounds, cycle))
	default:
		notes = append(notes, fmt.Sprintf("One full %d-round cycle and %s of the next, so some pairs meet twice and others once.", cycle, plural(rounds%cycle, "round", "rounds")))
	}

	notes = append(notes, fmt.Sprintf("Average opponent seed ranges from %.1f to %.1f.", sp.MinAverageOpponentSeed, sp.MaxAverageOpponentSeed))
	if sp.TopSeedGames == 0 {
		notes = append(notes, fmt.Sprintf("Seeds 1–%d never meet.", topSeeds))
	} else {
		notes = append(notes, fmt.Sprintf("Seeds 1–%d play each other %s.", topSeeds, plural(int(sp.TopSeedGames), "time", "times")))
	}
	if sp.MaxByes > 0 {
		notes = append(notes, fmt.Sprintf("Some players sit out: up to %s each.", plural(int(sp.MaxByes), "bye", "byes")))
	}
	if sp.Repeats > 0 && (cycle == 0 || rounds%cycle != 0) {
		notes = append(notes, fmt.Sprintf("%s repeated.", plural(int(sp.Repeats), "pairing is", "pairings are")))
	}
	return notes
}

// schemeScore ranks schemes for the intent; lower is better.
func schemeScore(sp *tpb.SchemePreview, intent tpb.PlannerIntent, rounds int) float64 {
	spread := sp.MaxAverageOpponentSeed - sp.MinAverageOpponentSeed
	partial := 0.0
	if !fits(sp, rounds) {
		partial = 1
	}
	byes := float64(sp.MaxByes)
	// Meeting the same opponents again while others go unplayed is worse
	// than a cycle that doesn't fit the rounds. Two teams is the exception:
	// there, playing only the other team is the point.
	missed := float64(sp.MissedOpponents) * 150
	switch intent {
	case tpb.PlannerIntent_PLANNER_WHOLE_EVENT:
		return missed + partial*100 + spread*3 + byes*5
	case tpb.PlannerIntent_PLANNER_TWO_TEAMS:
		score := partial*100 + spread*3 + byes*5
		switch {
		case sp.PairingMethod == pb.PairingMethod_ASSIGNED_TEAM_ROUND_ROBIN:
			score -= 1000
		case sp.PlayWithinTeam || sp.PairingMethod == pb.PairingMethod_ROUND_ROBIN ||
			sp.PairingMethod == pb.PairingMethod_INITIAL_FONTES:
			score += 1000
		}
		return score
	default:
		// A fair start keeps the top seeds apart for later, but not by
		// giving them all the weakest opponents, so a similar schedule for
		// everyone counts for nearly as much.
		return missed + partial*100 + float64(sp.Repeats)*50 + float64(sp.TopSeedGames)*10 + spread*6 + byes*5
	}
}

func rankSchemes(schemes []*tpb.SchemePreview, intent tpb.PlannerIntent, rounds int) []*tpb.SchemePreview {
	slices.SortStableFunc(schemes, func(a, b *tpb.SchemePreview) int {
		if a.Available != b.Available {
			if a.Available {
				return -1
			}
			return 1
		}
		if !a.Available {
			return 0
		}
		return cmp.Compare(schemeScore(a, intent, rounds), schemeScore(b, intent, rounds))
	})
	// Two schemes can come out with exactly the same games (12 players in
	// groups of 6 are the same as shirts and skins within teams, say); keep
	// the better ranked one and say so.
	kept := []*tpb.SchemePreview{}
	for _, s := range schemes {
		if s.Available {
			if i := slices.IndexFunc(kept, func(k *tpb.SchemePreview) bool { return k.Available && sameGames(k, s) }); i >= 0 {
				kept[i].Notes = append(kept[i].Notes, fmt.Sprintf("Same games as the %s.", schemeName(s.PairingMethod, s.PlayWithinTeam)))
				continue
			}
		}
		kept = append(kept, s)
	}
	if len(kept) > 0 && kept[0].Available {
		kept[0].Suggested = true
	}
	return kept
}

// sameGames says whether two schedules pair the same players in each round.
func sameGames(a, b *tpb.SchemePreview) bool {
	if len(a.Rounds) != len(b.Rounds) {
		return false
	}
	key := func(r *tpb.PreviewRound) []string {
		ks := []string{}
		for _, g := range r.Games {
			ks = append(ks, fmt.Sprint(min(g.First, g.Second), "-", max(g.First, g.Second)))
		}
		slices.Sort(ks)
		return ks
	}
	for i := range a.Rounds {
		if strings.Join(key(a.Rounds[i]), ",") != strings.Join(key(b.Rounds[i]), ",") {
			return false
		}
	}
	return true
}

// roundsAlternatives suggests other numbers of rounds that suit the intent
// better than the ones asked for.
func roundsAlternatives(schemes []*tpb.SchemePreview, n, rounds int, intent tpb.PlannerIntent, topSeeds int,
	hasTeams bool, fontes func(rounds int) *tpb.SchemePreview) []*tpb.RoundsAlternative {
	alts := []*tpb.RoundsAlternative{}
	best := schemes[0]
	if !best.Available {
		best = nil
	}
	if intent == tpb.PlannerIntent_PLANNER_FAIR_START && !hasTeams &&
		(best == nil || !fits(best, rounds) || best.TopSeedGames > 0) {
		// Initial Fontes keeps the top seeds apart when there are enough
		// groups for them. Suggest the round count, short of what was asked,
		// whose preview has the fewest games among the top seeds (the most
		// rounds, of those), if it beats the best scheme at this count.
		var alt *tpb.SchemePreview
		altRounds := 0
		for r := min(rounds, n/2-1); r >= 1; r-- {
			if r%2 == 0 || r == rounds {
				continue
			}
			if sp := fontes(r); sp.Available && (alt == nil || sp.TopSeedGames < alt.TopSeedGames) {
				alt, altRounds = sp, r
			}
		}
		if alt != nil && (best == nil || alt.TopSeedGames < best.TopSeedGames) {
			meet := fmt.Sprintf("seeds 1–%d never meet", topSeeds)
			if alt.TopSeedGames == 1 {
				meet = fmt.Sprintf("seeds 1–%d meet only once", topSeeds)
			} else if alt.TopSeedGames > 1 {
				meet = fmt.Sprintf("seeds 1–%d meet only %d times", topSeeds, alt.TopSeedGames)
			}
			alts = append(alts, &tpb.RoundsAlternative{Rounds: int32(altRounds), PairingMethod: pb.PairingMethod_INITIAL_FONTES,
				Reason: fmt.Sprintf("Pre-pair %d rounds with Initial Fontes, so %s, and let COP pair the rest.", altRounds, meet)})
		}
	}
	offered := map[int]bool{}
	if intent != tpb.PlannerIntent_PLANNER_TWO_TEAMS && best != nil && best.CycleLength > 0 {
		// A few rounds past full cycles of the best scheme: pre-pair the
		// cycles and let COP pair the last rounds, which it ends King of the
		// Hill style.
		cycle := int(best.CycleLength)
		if extra := rounds % cycle; rounds > cycle && extra >= 1 && extra <= 3 {
			full := rounds - extra
			what := "a full " + schemeName(best.PairingMethod, best.PlayWithinTeam)
			if full > cycle {
				what = fmt.Sprintf("%d full cycles of the %s", full/cycle, schemeName(best.PairingMethod, best.PlayWithinTeam))
			}
			last := "the last round"
			if extra > 1 {
				last = fmt.Sprintf("the last %d rounds", extra)
			}
			offered[full] = true
			alts = append(alts, &tpb.RoundsAlternative{Rounds: int32(full), PairingMethod: best.PairingMethod, PlayWithinTeam: best.PlayWithinTeam,
				Reason: fmt.Sprintf("Pre-pair %d rounds, %s, and let COP pair %s. Its final round pairs the prize contenders King of the Hill style.", full, what, last)})
		}
	}
	if best == nil || !fits(best, rounds) {
		// Nearby round counts that a scheme covers exactly, naming the best
		// ranked scheme for each.
		for _, s := range schemes {
			c := int(s.CycleLength)
			if !s.Available || c == 0 || c == rounds || s.PairingMethod == pb.PairingMethod_INITIAL_FONTES ||
				c < rounds-3 || c > rounds+3 || offered[c] {
				continue
			}
			if intent == tpb.PlannerIntent_PLANNER_TWO_TEAMS && (!(s.PairingMethod == pb.PairingMethod_ASSIGNED_TEAM_ROUND_ROBIN ||
				s.PairingMethod == pb.PairingMethod_INTERLEAVED_ROUND_ROBIN || s.PairingMethod == pb.PairingMethod_SNAKED_ROUND_ROBIN) ||
				s.PlayWithinTeam) {
				continue
			}
			offered[c] = true
			alts = append(alts, &tpb.RoundsAlternative{Rounds: int32(c), PairingMethod: s.PairingMethod, PlayWithinTeam: s.PlayWithinTeam,
				Reason: fmt.Sprintf("Pre-pair %d rounds: the %s fits that exactly.", c, schemeName(s.PairingMethod, s.PlayWithinTeam))})
		}
	}
	return alts
}
