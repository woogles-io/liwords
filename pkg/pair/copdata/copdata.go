package copdata

import (
	"fmt"
	"math"
	"slices"

	"golang.org/x/exp/rand"

	"strconv"
	"strings"

	pkgstnd "github.com/woogles-io/liwords/pkg/pair/standings"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

var (
	standingsHeader = []string{"Rank", "Num", "Name", "Wins", "Spr"}
)

const (
	// runawayLeadersWinPctThreshold is the combined win% for the leader and
	// 2nd place above which other players only need half as many simulated
	// wins to be considered hopeful contenders for 1st or 2nd.
	runawayLeadersWinPctThreshold = 0.80
)

// IsLastQuarter reports whether roundPairingsRemaining rounds still to be
// paired puts the tournament in its last quarter of rounds.
func IsLastQuarter(roundPairingsRemaining int, rounds int32) bool {
	return roundPairingsRemaining*4 <= int(rounds)
}

type PrecompData struct {
	Standings     *pkgstnd.Standings
	PairingCounts map[string]int

	// Indexed by player id
	RepeatCounts []int

	// The remaining following fields are indexed by player rank

	HighestRankHopefully  []int
	HighestRankAbsolutely []int
	LowestRankAbsolutely  []int
	LowestPossibleHopeNth []int
	// HopefulToCashPromotedPlayerRankIdx is the rank index of the player
	// promoted below to fix an odd hopeful-to-cash count, or -1 if no such
	// promotion fired this round. Exposed so cop.go's
	// adjustLowestPossibleHopeCasherForBye can tell whether the boundary it
	// sees already includes an artificial promotion, so it can retract it
	// instead of double-promoting when a bye also restores parity.
	HopefulToCashPromotedPlayerRankIdx int
	DestinysChild                      int
	GibsonGroups                       []int
	GibsonizedPlayers                  []bool
	CompletePairings                   int

	// BaselineFinalRanks/BaselineTotalSims are the final-rank sim results from
	// the normal (non-factor-3) factor pairing used above to build
	// HighestRankHopefully etc. Exposed so callers (e.g. Factor 3's
	// win-chance-gain check) can compare a baseline win-outright probability
	// against a factor-3 scenario without re-simulating. Both are indexed by
	// the same player-rank ordering as the standings snapshot this
	// PrecompData was built from, within a single pairing call - don't reuse
	// across calls or after re-ranking.
	BaselineFinalRanks [][]int
	BaselineTotalSims  int
}

// hopefulRankData bundles the sim-derived fields that a specific
// penultimate-round pairing structure (the baseline factor pairing, or a
// later Factor-3 expansion) determines. See computeHopefulRankData.
type hopefulRankData struct {
	HighestRankHopefully               []int
	HighestRankAbsolutely              []int
	LowestRankAbsolutely               []int
	LowestPossibleHopeNth              []int
	HopefulToCashPromotedPlayerRankIdx int
}

// computeHopefulRankData derives the hopeful/absolute rank boundaries used
// throughout the RD/PC/CC/GC/BB weight policies from a final-rank
// simulation. It's factored out of GetPrecompData so the same derivation can
// be re-run later in the same pairing call against a different simulation -
// specifically, when the Factor-3 penultimate-round expansion fires, its own
// already-run simulation (which reflects the real 1v4/2v5/3v6 structure,
// not the generic factor-2-capped structure GetPrecompData started from)
// becomes the more accurate source for these fields. See
// PrecompData.ApplyFactor3Sim, which calls this with the Factor-3 sim.
func computeHopefulRankData(
	req *pb.PairRequest,
	standings *pkgstnd.Standings,
	numCompletePairings int,
	finalRanks [][]int,
	totalSims int,
	gibsonizedPlayers []bool,
	logsb *strings.Builder,
) hopefulRankData {
	numPlayers := standings.GetNumPlayers()
	minWinsForHopeful := int(math.Round(float64(totalSims) * req.HopefulnessThreshold))
	// If the leader and 2nd place have run away with the tournament (their
	// combined probability of finishing 1st exceeds 80%), other players only
	// need half as many simulated wins to be considered hopeful contenders
	// for 1st or 2nd.
	//
	// This must be computed from the simulated probability of finishing in
	// 1st place (finalRanks[...][0]/totalSims), not from either player's
	// actual completed-game win rate: only one player can finish 1st, so the
	// simulated figure is a single probability bounded at 100%, while two
	// independent win rates can each approach 100% and sum past it.
	//
	// Only applies when 1st isn't Gibsonized: once the leader is locked into
	// 1st, there's no "runaway" race for it to speak of, so a high combined
	// leader+2nd 1st-place% is just Gibsonization already at work, not a
	// signal that everyone else's bar for 1st/2nd should be halved too.
	halfMinWinsForHopeful := int(math.Round(float64(minWinsForHopeful) / 2.0))
	runawayLeaders := false
	if numPlayers >= 2 && totalSims > 0 && !gibsonizedPlayers[0] {
		leaderFirstPct := float64(finalRanks[0][0]) / float64(totalSims)
		secondFirstPct := float64(finalRanks[1][0]) / float64(totalSims)
		if leaderFirstPct+secondFirstPct > runawayLeadersWinPctThreshold {
			runawayLeaders = true
			logsb.WriteString(fmt.Sprintf(
				"Leader+2nd combined 1st-place%% (%.1f%%) > %.0f%%: halving the hopeful-for-1st/2nd bar to %.0f%% (normally %.0f%%)\n",
				(leaderFirstPct+secondFirstPct)*100, runawayLeadersWinPctThreshold*100,
				float64(halfMinWinsForHopeful)/float64(totalSims)*100,
				float64(minWinsForHopeful)/float64(totalSims)*100))
		}
	}
	highestRankHopefully := make([]int, numPlayers)
	highestRankAbsolutely := make([]int, numPlayers)
	lowestRankAbsolutely := make([]int, numPlayers)
	for playerRankIdx := 0; playerRankIdx < numPlayers; playerRankIdx++ {
		winsSum := 0
		hopefulRank := numPlayers - 1
		absoluteRank := numPlayers - 1
		for rank := 0; rank < numPlayers; rank++ {
			rankSum := finalRanks[playerRankIdx][rank]
			if winsSum == 0 && rankSum > 0 {
				absoluteRank = rank
			}
			winsSum += rankSum
			threshold := minWinsForHopeful
			if runawayLeaders && rank <= 1 {
				threshold = halfMinWinsForHopeful
			}
			if winsSum >= threshold {
				hopefulRank = rank
				break
			}
		}
		highestRankHopefully[playerRankIdx] = hopefulRank
		highestRankAbsolutely[playerRankIdx] = absoluteRank
		lowestRank := 0
		for rank := numPlayers - 1; rank >= 0; rank-- {
			rankSum := finalRanks[playerRankIdx][rank]
			if rankSum > 0 {
				lowestRank = rank
				break
			}
		}
		lowestRankAbsolutely[playerRankIdx] = lowestRank
	}

	// See hopefulToCashExtensionRankIdx for the last-quarter hopeful-to-cash
	// parity fix applied here.
	roundPairingsRemaining := int(req.Rounds) - numCompletePairings
	hopefulToCashPromotedPlayerRankIdx := -1
	if IsLastQuarter(roundPairingsRemaining, req.Rounds) {
		numHopefulToCash, extendedRankIdx := hopefulToCashExtensionRankIdx(highestRankHopefully, gibsonizedPlayers, int(req.PlacePrizes))
		if extendedRankIdx >= 0 {
			lowestCashPlace := int(req.PlacePrizes) - 1
			logsb.WriteString(fmt.Sprintf(
				"Odd number of non-Gibsonized hopeful-to-cash players (%d) in the last quarter: extending the "+
					"window to rank %d, altering %s from hopeful-for-%d to hopeful-for-%d (lowest cash position)\n",
				numHopefulToCash, extendedRankIdx+1, req.PlayerNames[standings.GetPlayerIndex(extendedRankIdx)],
				highestRankHopefully[extendedRankIdx]+1, lowestCashPlace+1))
			highestRankHopefully[extendedRankIdx] = lowestCashPlace
			hopefulToCashPromotedPlayerRankIdx = extendedRankIdx
		}
	}

	lowestPossibleHopeNth := computeLowestPossibleHopeNth(highestRankHopefully)

	return hopefulRankData{
		HighestRankHopefully:               highestRankHopefully,
		HighestRankAbsolutely:              highestRankAbsolutely,
		LowestRankAbsolutely:               lowestRankAbsolutely,
		LowestPossibleHopeNth:              lowestPossibleHopeNth,
		HopefulToCashPromotedPlayerRankIdx: hopefulToCashPromotedPlayerRankIdx,
	}
}

// computeLowestPossibleHopeNth derives, for each place N, the rank index of
// the worst-ranked player still hopeful for Nth or better - i.e. the rank
// window CC/PC actually pair against (see hopeCasherBoundary and
// riHopeful/rjHopeful in cop.go, which just check ri <= boundary).
// computeHopefulRankData calls this twice: once on the raw hopefulRanks to
// find the natural (pre-promotion) window for hopefulToCashExtensionRankIdx,
// and again after any promotion to produce the final returned value.
func computeLowestPossibleHopeNth(hopefulRanks []int) []int {
	lowestPossibleHopeNth := make([]int, len(hopefulRanks))
	prevPlace := 0
	for playerRankIdx, place := range hopefulRanks {
		if playerRankIdx > lowestPossibleHopeNth[place] {
			lowestPossibleHopeNth[place] = playerRankIdx
		}
		for i := prevPlace + 1; i < place; i++ {
			lowestPossibleHopeNth[i] = playerRankIdx - 1
		}
		prevPlace = place
	}
	for i := prevPlace + 1; i < len(hopefulRanks); i++ {
		lowestPossibleHopeNth[i] = len(hopefulRanks) - 1
	}
	return lowestPossibleHopeNth
}

// hopefulToCashExtensionRankIdx implements the last-quarter hopeful-to-cash
// parity fix: an odd number of hopeful-to-cash players leaves one of them
// with no hopeful-to-cash opponent (the CC weight policy major-penalizes
// pairing a hopeful-to-cash player against a non-hopeful one), so extend the
// window by one rank to restore parity - the same fix
// computeDisallowedLeaderOpponent makes for the hopeful-for-1st contender
// group, but folded into the data instead of layered on as a pairing
// constraint. Returns the rank index to extend the window to (promoting
// that player to hopeful for the lowest cash position), or -1 if no
// extension is needed or possible.
//
// The relevant parity is the WINDOW's (ranks 1..naturalBoundary), not a
// headcount of individually hopeful-for-cash players: CC/PC treat every
// rank inside the window as part of the hopeful group via a plain
// ri <= boundary check, regardless of whether that specific player is
// individually hopeful for cash - so a "gap" player inside the window (not
// themselves hopeful, but ranked above another player who independently is,
// extending the window past them) is still grouped as hopeful. Checking a
// scattered headcount of only the individually hopeful players - as this
// used to do - can find an odd count and promote a gap player whose own
// status the window's parity never actually depended on, since a further
// independently-hopeful player was already extending the window past the
// gap regardless. That leaves downstream bye-adjustment logic
// (adjustLowestPossibleHopeCasherForBye in cop.go), which assumes a
// recorded promotion IS the window boundary, retracting past that further
// genuinely-hopeful player when it later undoes a promotion it thinks is
// redundant - reintroducing the very parity problem the promotion was
// meant to prevent.
//
// Gibsonized players don't count toward this parity check even though they
// may still nominally be "hopeful" for cash (e.g. a player Gibsonized for
// 1st is trivially hopeful for every lower place too): PC and CC (see
// cop.go) exclude any Gibsonized player from being treated as a pairable
// hopeful-to-cash contender regardless of their raw hopeful-for rank, so
// counting them here would under-promote and leave a genuine contender
// without an in-group opponent.
//
// This promotion still matters in the true final round even though CC's
// hopeful-vs-hopeful grouping rule is disabled there (see isFinalRound in
// cop.go): lowestPossibleHopeCasher, which this feeds via
// LowestPossibleHopeNth, is also read unconditionally (not gated by
// last-quarter/final-round) by the BB and computeForcedContenderBye
// policies in cop.go, so it must stay correct in every round.
// Returns the natural window's non-Gibsonized headcount alongside the
// extension decision so callers can log it without recomputing it.
func hopefulToCashExtensionRankIdx(highestRankHopefully []int, gibsonizedPlayers []bool, placePrizes int) (numHopefulToCash int, extendedRankIdx int) {
	naturalBoundary := computeLowestPossibleHopeNth(highestRankHopefully)[placePrizes-1]
	for playerRankIdx := 0; playerRankIdx <= naturalBoundary; playerRankIdx++ {
		if !gibsonizedPlayers[playerRankIdx] {
			numHopefulToCash++
		}
	}
	extendedRankIdx = naturalBoundary + 1
	if numHopefulToCash%2 == 1 && extendedRankIdx < len(highestRankHopefully) {
		return numHopefulToCash, extendedRankIdx
	}
	return numHopefulToCash, -1
}

// ApplyFactor3Sim overwrites the hopeful/absolute rank boundaries (and
// everything downstream of them - lowestPossibleHopeCasher,
// lowestPossibleAbsCasher, hopeToCashBoundary, etc., all read live off these
// fields at matching time) using a Factor-3 expansion simulation instead of
// the generic baseline one GetPrecompData used. Call this only when the
// Factor-3 expansion actually fires (the full 1v4/2v5/3v6 structure is
// forced) - only then does finalRanks reflect the pairing structure that's
// actually being played this round; a single ad hoc forced pair (e.g. the
// control-loss branch in computeFactor3ForcedPairings) doesn't warrant
// this, since the rest of the field still isn't paired the way finalRanks
// simulated.
//
// GibsonizedPlayers/GibsonGroups are deliberately left untouched: both are
// purely analytic functions of current win/spread standings and
// roundsRemaining/GibsonSpread (see standings.GetGibsonizedPlayers), not of
// which pairing structure gets simulated, so there's nothing to update
// there.
func (pd *PrecompData) ApplyFactor3Sim(req *pb.PairRequest, finalRanks [][]int, totalSims int, logsb *strings.Builder) {
	logsb.WriteString("Factor 3 fired: recomputing hopeful/absolute rank boundaries from the Factor-3 sim\n")
	hrd := computeHopefulRankData(req, pd.Standings, pd.CompletePairings, finalRanks, totalSims, pd.GibsonizedPlayers, logsb)
	pd.HighestRankHopefully = hrd.HighestRankHopefully
	pd.HighestRankAbsolutely = hrd.HighestRankAbsolutely
	pd.LowestRankAbsolutely = hrd.LowestRankAbsolutely
	pd.LowestPossibleHopeNth = hrd.LowestPossibleHopeNth
	pd.HopefulToCashPromotedPlayerRankIdx = hrd.HopefulToCashPromotedPlayerRankIdx
}

// ExtractPrepairedPlayers returns a map of playerIdx -> oppIdx for all prepaired players
// (where oppIdx == playerIdx means bye), the count of forced byes, and the prepaired round index.
// Players not in this map are unpaired and should be assigned by the pairing method.
func ExtractPrepairedPlayers(req *pb.PairRequest) (map[int]int, int, int) {
	prepairedPlayerIndexes := map[int]int{}
	numForcedByes := 0
	prepairedRoundIdx := -1
	numDivPairings := len(req.DivisionPairings)
	removedPlayersSet := map[int]bool{}
	for _, idx := range req.RemovedPlayers {
		removedPlayersSet[int(idx)] = true
	}
	if numDivPairings > 0 {
		if slices.Contains(req.DivisionPairings[numDivPairings-1].Pairings, -1) {
			prepairedRoundIdx = numDivPairings - 1
		}
		if prepairedRoundIdx >= 0 {
			for playerIdx, oppIdx := range req.DivisionPairings[prepairedRoundIdx].Pairings {
				if int(oppIdx) < playerIdx || removedPlayersSet[playerIdx] || removedPlayersSet[int(oppIdx)] {
					continue
				}
				prepairedPlayerIndexes[playerIdx] = int(oppIdx)
				prepairedPlayerIndexes[int(oppIdx)] = playerIdx
				if playerIdx == int(oppIdx) {
					numForcedByes++
				}
			}
		}
	}
	return prepairedPlayerIndexes, numForcedByes, prepairedRoundIdx
}

// ComputeTopDownByeRankIdx determines which player rank (if any) will
// receive this round's top-down bye: the player with the fewest previous
// byes, scanning from the top of the standings down (ties go to the
// higher-ranked player). Returns -1 if top-down byes don't apply this round
// - either req.TopDownByes is off, or the round doesn't need an
// added/unforced bye at all (an even number of players once forced-bye
// prepairs are accounted for).
//
// This mirrors cop.go's computeTopDownByePlayer (the "TB" constraint policy,
// which actually forces the bye pairing) exactly, but is computed here, up
// front, so GetPrecompData can exclude this rank from the control-loss
// search below: a player sitting out this round via the bye isn't playing
// anyone, so they can't be usefully flagged as the destiny child CL forces
// 1st to play - doing so would conflict with the bye's own forced pairing
// and leave 1st with no legal opponent at all.
func ComputeTopDownByeRankIdx(req *pb.PairRequest, standings *pkgstnd.Standings, pairingCounts map[string]int) int {
	if !req.TopDownByes {
		return -1
	}
	numPlayers := standings.GetNumPlayers()
	_, numForcedByes, _ := ExtractPrepairedPlayers(req)
	if (numPlayers-numForcedByes)%2 == 0 {
		return -1
	}
	byeRankIdx := -1
	leastByes := int(req.Rounds + 1)
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		pi := standings.GetPlayerIndex(rankIdx)
		pairingKey := GetPairingKey(pi, pkgstnd.ByePlayerIndex)
		numByes := pairingCounts[pairingKey]
		if numByes < leastByes {
			leastByes = numByes
			byeRankIdx = rankIdx
		}
	}
	return byeRankIdx
}

func GetPrecompData(req *pb.PairRequest, copRand *rand.Rand, logsb *strings.Builder) (*PrecompData, pb.PairError) {
	standings := pkgstnd.CreateInitialStandings(req)

	// Use the initial results to get a tighter bound on the maximum factor
	initialFactor := pkgstnd.GetRoundsRemaining(req)
	initialSimResults, pairErr := standings.SimFactorPairAll(req, copRand, int(req.DivisionSims), initialFactor, -1, nil, -1)
	if pairErr != pb.PairError_SUCCESS {
		return nil, pairErr
	}
	WriteFinalRankResultsToLog(fmt.Sprintf("Initial Sim Results (factor ceiling of %d)", initialFactor), initialSimResults.FinalRanks, standings, req, logsb)

	numPlayers := standings.GetNumPlayers()

	// Attempt to tighten the max factor.
	// When tightening the factor bounds, the  maximum factor should
	// be with respect to the highest non-gibsonized rank.
	highestNongibsonizedRank := 0
	for rankIdx, isGibsonized := range initialSimResults.GibsonizedPlayers {
		if !isGibsonized {
			highestNongibsonizedRank = rankIdx
			break
		}
	}

	// If the initial factor is N, then 1st played (1 + N)th, but if (1 + N)th never achieved
	// 1st, then (1 + N)th should never have played 1st at all, so we use the initial results
	// to get a tighter bound on the maximum factor.
	maxFactor := 0
	for playerRankIdx := highestNongibsonizedRank + 1; playerRankIdx < numPlayers; playerRankIdx++ {
		if initialSimResults.FinalRanks[playerRankIdx][highestNongibsonizedRank] > 0 {
			maxFactor++
		} else {
			break
		}
	}

	// Get the number of players in the highest gibson group where the factor would be applied
	numPlayersInhighestNongibsonGroup := 0
	highestNongibsonizedRankGroup := initialSimResults.GibsonGroups[highestNongibsonizedRank]
	for rankIdx := highestNongibsonizedRank; rankIdx < numPlayers; rankIdx++ {
		if initialSimResults.GibsonGroups[rankIdx] == highestNongibsonizedRankGroup {
			numPlayersInhighestNongibsonGroup++
		} else {
			break
		}
	}

	var improvedFactorSimResults *pkgstnd.SimResults

	// Only re-sim with the improved bound if it actually makes the max factor smaller
	// for the highest gibson group.
	if maxFactor*2 < numPlayersInhighestNongibsonGroup {
		improvedFactorSimResults, pairErr = standings.SimFactorPairAll(req, copRand, int(req.DivisionSims), maxFactor, -1, initialSimResults.SegmentRoundFactors, -1)
		if pairErr != pb.PairError_SUCCESS {
			return nil, pairErr
		}
	}

	if improvedFactorSimResults == nil {
		improvedFactorSimResults = initialSimResults
		logsb.WriteString("\n\nNo factor improvement made.\n\n")
	} else {
		WriteFinalRankResultsToLog(fmt.Sprintf("Improved Factor Sim Results (factor ceiling of %d)", maxFactor), improvedFactorSimResults.FinalRanks, standings, req, logsb)
	}

	// numCompletePairings is needed early to compute the leader/2nd win% below;
	// it's recomputed (identically) further down alongside pairingCounts.
	numCompletePairings := 0
completePairingsLoop:
	for _, roundPairings := range req.DivisionPairings {
		for playerIdx := range roundPairings.Pairings {
			oppIdx := int(roundPairings.Pairings[playerIdx])
			if oppIdx == -1 {
				break completePairingsLoop
			}
		}
		numCompletePairings++
	}

	hrd := computeHopefulRankData(req, standings, numCompletePairings,
		improvedFactorSimResults.FinalRanks, improvedFactorSimResults.TotalSims, improvedFactorSimResults.GibsonizedPlayers,
		logsb)
	highestRankHopefully := hrd.HighestRankHopefully
	highestRankAbsolutely := hrd.HighestRankAbsolutely
	lowestRankAbsolutely := hrd.LowestRankAbsolutely
	lowestPossibleHopeNth := hrd.LowestPossibleHopeNth
	hopefulToCashPromotedPlayerRankIdx := hrd.HopefulToCashPromotedPlayerRankIdx

	pairingCounts := make(map[string]int)
	repeatCounts := make([]int, int(req.AllPlayers))
	for roundIdx := range numCompletePairings {
		roundPairings := req.DivisionPairings[roundIdx]
		for playerIdx := range roundPairings.Pairings {
			oppIdx := int(roundPairings.Pairings[playerIdx])

			if oppIdx > playerIdx {
				continue
			}
			pairingKey := GetPairingKey(playerIdx, oppIdx)
			if pairingCounts[pairingKey] > 0 {
				repeatCounts[playerIdx]++
				if playerIdx != oppIdx {
					repeatCounts[oppIdx]++
				}
			}
			pairingCounts[pairingKey]++
		}
	}

	var controlLossSimResults *pkgstnd.SimResults
	var allControlLosses map[int]int
	var vsFirstWins map[int]int
	destinysChild := -1
	if numCompletePairings >= int(req.ControlLossActivationRound) && !improvedFactorSimResults.GibsonizedPlayers[0] && initialFactor > 1 && maxFactor > 0 {
		topDownByeRankIdx := ComputeTopDownByeRankIdx(req, standings, pairingCounts)
		if topDownByeRankIdx >= 0 {
			logsb.WriteString(fmt.Sprintf(
				"Control loss: excluding rank %d (%s) from the destiny-child search - top-down byes will assign them this round's bye, so they aren't playing anyone and can't be usefully forced to play 1st\n",
				topDownByeRankIdx+1, req.PlayerNames[standings.GetPlayerIndex(topDownByeRankIdx)],
			))
		}
		controlLossSimResults, pairErr = standings.SimFactorPairAll(req, copRand, int(req.ControlLossSims), maxFactor, lowestPossibleHopeNth[0], nil, topDownByeRankIdx)
		if pairErr != pb.PairError_SUCCESS {
			return nil, pairErr
		}
		allControlLosses = controlLossSimResults.AllControlLosses
		vsFirstWins = controlLossSimResults.VsFirstWins
		if controlLossSimResults.HighestControlLossRankIdx >= 0 {
			destinysChild = controlLossSimResults.HighestControlLossRankIdx
			if controlLossSimResults.ControlLossViaLockFallback {
				lockEndRankIdx := controlLossSimResults.ControlLossLockRunEndRankIdx
				lockedNames := make([]string, 0, lockEndRankIdx)
				for rankIdx := 1; rankIdx <= lockEndRankIdx; rankIdx++ {
					lockedNames = append(lockedNames, req.PlayerNames[standings.GetPlayerIndex(rankIdx)])
				}
				lockedRanks, be := "rank 2", "is"
				if lockEndRankIdx > 1 {
					lockedRanks, be = fmt.Sprintf("ranks 2-%d", lockEndRankIdx+1), "are"
				}
				childIdx := standings.GetPlayerIndex(destinysChild)
				logsb.WriteString(fmt.Sprintf(
					"Control loss: rank %d (%s) flagged via the 2-rounds-remaining lock-fallback rule, not the normal threshold check - %s (%s) %s already locked to win outright regardless of opponent (vsFirst=vsFactorPair=%d/%d sims), so rank %d is the last player whose destiny is still undecided; vs1st=%d > vsFactor=%d confirms playing 1st still helps them\n",
					destinysChild+1, req.PlayerNames[childIdx],
					lockedRanks, strings.Join(lockedNames, ", "), be,
					vsFirstWins[lockEndRankIdx], int(req.ControlLossSims),
					destinysChild+1,
					vsFirstWins[destinysChild], allControlLosses[destinysChild],
				))
			}
		}
	}

	writePrecompDataToLog("Precomp Data", improvedFactorSimResults, allControlLosses, vsFirstWins, highestRankHopefully, highestRankAbsolutely, standings, req, logsb)

	return &PrecompData{
		Standings:                          standings,
		PairingCounts:                      pairingCounts,
		RepeatCounts:                       repeatCounts,
		HighestRankHopefully:               highestRankHopefully,
		HighestRankAbsolutely:              highestRankAbsolutely,
		LowestRankAbsolutely:               lowestRankAbsolutely,
		LowestPossibleHopeNth:              lowestPossibleHopeNth,
		HopefulToCashPromotedPlayerRankIdx: hopefulToCashPromotedPlayerRankIdx,
		DestinysChild:                      destinysChild,
		GibsonGroups:                       improvedFactorSimResults.GibsonGroups,
		GibsonizedPlayers:                  improvedFactorSimResults.GibsonizedPlayers,
		CompletePairings:                   numCompletePairings,
		BaselineFinalRanks:                 improvedFactorSimResults.FinalRanks,
		BaselineTotalSims:                  improvedFactorSimResults.TotalSims,
	}, pb.PairError_SUCCESS
}

func GetPairingKey(playerIdx int, oppIdx int) string {
	var pairingKey string
	if playerIdx == oppIdx || oppIdx == pkgstnd.ByePlayerIndex {
		pairingKey = fmt.Sprintf("%d:BYE", playerIdx)
	} else {
		if oppIdx > playerIdx {
			playerIdx, oppIdx = oppIdx, playerIdx
		}
		pairingKey = fmt.Sprintf("%d:%d", playerIdx, oppIdx)
	}
	return pairingKey
}

func writePrecompDataToLog(title string, simResults *pkgstnd.SimResults, allControlLosses map[int]int, vsFirstWins map[int]int, highestRankHopefully []int, highestRankAbsolutely []int, standings *pkgstnd.Standings, req *pb.PairRequest, logsb *strings.Builder) {
	numPlayers := len(highestRankHopefully)
	matrix := make([][]string, numPlayers)

	useControlLoss := allControlLosses != nil
	var header []string
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		if useControlLoss {
			matrix[rankIdx] = make([]string, 6)
			header = append(standingsHeader, []string{"Gb", "Gr", "H", "A", "vs1st", "vsFactor"}...)
		} else {
			matrix[rankIdx] = make([]string, 4)
			header = append(standingsHeader, []string{"Gb", "Gr", "H", "A"}...)
		}
		matrix[rankIdx][0] = boolToYesEmpty(simResults.GibsonizedPlayers[rankIdx])
		matrix[rankIdx][1] = strconv.Itoa(simResults.GibsonGroups[rankIdx] + 1)
		matrix[rankIdx][2] = strconv.Itoa(highestRankHopefully[rankIdx] + 1)
		matrix[rankIdx][3] = strconv.Itoa(highestRankAbsolutely[rankIdx] + 1)
		if useControlLoss {
			matrix[rankIdx][4] = ""
			matrix[rankIdx][5] = ""
			vsFirstWinsCount, exists := vsFirstWins[rankIdx]
			if exists {
				if vsFirstWinsCount < 0 {
					matrix[rankIdx][4] = "-"
				} else {
					matrix[rankIdx][4] = strconv.Itoa(vsFirstWinsCount)
				}
			}
			playerControlLosses, exists := allControlLosses[rankIdx]
			if exists {
				if playerControlLosses < 0 {
					matrix[rankIdx][5] = "-"
				} else {
					matrix[rankIdx][5] = strconv.Itoa(playerControlLosses)
				}
			}
		}
	}

	WriteStringDataToLog(title, header, combineStringMatrices(standings.StringData(req), matrix), logsb)
}

func WriteFinalRankResultsToLog(title string, finalRanks [][]int, standings *pkgstnd.Standings, req *pb.PairRequest, logsb *strings.Builder) {
	header := append([]string{}, standingsHeader[:]...)
	numPlayers := standings.GetNumPlayers()
	totalSims := 0
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		header = append(header, strconv.Itoa(rankIdx+1))
		totalSims += finalRanks[rankIdx][0]
	}

	finalRanksStrPct := make([][]string, numPlayers)
	finalRanksStrRaw := make([][]string, numPlayers)
	for rankIdx := 0; rankIdx < numPlayers; rankIdx++ {
		finalRanksStrPct[rankIdx] = make([]string, numPlayers)
		finalRanksStrRaw[rankIdx] = make([]string, numPlayers)
		for colIdx, value := range finalRanks[rankIdx] {
			finalRanksStrPct[rankIdx][colIdx] = fmt.Sprintf("%.4f%%", float64(value)*100/float64(totalSims))
			finalRanksStrRaw[rankIdx][colIdx] = fmt.Sprintf("%d", value)
		}
	}

	WriteStringDataToLog(title, header, combineStringMatrices(standings.StringData(req), finalRanksStrPct), logsb)
	WriteStringDataToLog("Totals", header, combineStringMatrices(standings.StringData(req), finalRanksStrRaw), logsb)
	logsb.WriteString(fmt.Sprintf("Total Sims: %d\n\n", totalSims))
}

func formatStringData(header []string, data [][]string) string {
	numRows := len(data)
	numCols := len(header)

	for rowIdx := 0; rowIdx < numRows; rowIdx++ {
		if len(data[rowIdx]) != numCols {
			return fmt.Sprintf("row %d has %d columns, expected %d", rowIdx, len(data[rowIdx]), numCols)
		}
	}

	maxColumnWidths := make([]int, numCols)
	for colIdx := 0; colIdx < numCols; colIdx++ {
		if len(header[colIdx]) > maxColumnWidths[colIdx] {
			maxColumnWidths[colIdx] = len(header[colIdx])
		}
		for rowIdx := 0; rowIdx < numRows; rowIdx++ {
			if len(data[rowIdx][colIdx]) > maxColumnWidths[colIdx] {
				maxColumnWidths[colIdx] = len(data[rowIdx][colIdx])
			}
		}
	}

	var sb strings.Builder

	for colIdx := 0; colIdx < numCols; colIdx++ {
		sb.WriteString(fmt.Sprintf("%-*s", maxColumnWidths[colIdx], header[colIdx]))
		if colIdx < numCols-1 {
			sb.WriteString(" | ")
		}
	}

	sb.WriteString("\n" + strings.Repeat("-", sb.Len()) + "\n")

	for rowIdx := 0; rowIdx < numRows; rowIdx++ {
		for colIdx := 0; colIdx < numCols; colIdx++ {
			sb.WriteString(fmt.Sprintf("%-*s", maxColumnWidths[colIdx], data[rowIdx][colIdx]))
			if colIdx < numCols-1 {
				sb.WriteString(" | ")
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	return sb.String()
}

func WriteStringDataToLog(title string, header []string, data [][]string, logsb *strings.Builder) {
	titleLine := fmt.Sprintf("** %s **", title)
	border := strings.Repeat("*", len(titleLine))
	logsb.WriteString(fmt.Sprintf("%s\n%s\n%s\n\n", border, titleLine, border))
	logsb.WriteString(formatStringData(header, data))
}

func combineStringMatrices(m1, m2 [][]string) [][]string {
	if len(m1) != len(m2) {
		return [][]string{}
	}

	// Create a new matrix to hold the combined rows
	rowCount := len(m1)
	combined := make([][]string, rowCount)

	// Combine rows from m1 and m2
	for i := 0; i < rowCount; i++ {
		combined[i] = append(m1[i], m2[i]...)
	}

	return combined
}

func boolToYesEmpty(value bool) string {
	if value {
		return "Yes"
	}
	return ""
}
