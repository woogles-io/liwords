package cop

import (
	"fmt"
	"hash/fnv"
	"math"
	"slices"
	"strconv"

	"golang.org/x/exp/rand"
	"google.golang.org/protobuf/encoding/protojson"

	"strings"
	"time"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/matching"
	"github.com/woogles-io/liwords/pkg/pair"
	copdatapkg "github.com/woogles-io/liwords/pkg/pair/copdata"
	pkgstnd "github.com/woogles-io/liwords/pkg/pair/standings"
	"github.com/woogles-io/liwords/pkg/pair/verifyreq"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	timeFormat   = "2006-01-02T15:04:05.000Z"
	majorPenalty = 1e9
	minorPenalty = majorPenalty / 1e3
	// hopefulCasherByeWeight is the cost of giving a hopeful-to-cash
	// contender the bye. It's set well above 2*majorPenalty so that when the
	// matching must choose between that and a combination of two other
	// major-penalty violations elsewhere (e.g. two repeat byes), it picks the
	// latter instead. Top-down byes bypass this entirely (TB force-assigns
	// the bye), so this only matters when byes aren't top-down.
	hopefulCasherByeWeight               = 3e9
	byePlayerName                        = "BYE"
	controlLossLowestContenderOnlyRounds = 4
	// f3MinWinChanceGain is the minimum win-outright-probability gain
	// (finishing 1st overall, compared to the baseline non-factor-3 factor
	// pairing) that at least one of 4th/5th/6th must get from factor-3
	// pairings for F3 to fire. Without this, F3 can trigger purely because it
	// helps the leader (and strips destiny control from the field below)
	// while giving none of 4th/5th/6th a meaningful shot at actually winning
	// outright - e.g. a real case where F3 only moved a trailing player's win
	// chance from 0.05% to 0.1%.
	f3MinWinChanceGain = 0.02
)

var pairingMethodMap = map[pb.PairMethod]pb.PairingMethod{
	pb.PairMethod_PAIR_RANDOM:                  pb.PairingMethod_RANDOM,
	pb.PairMethod_PAIR_ROUND_ROBIN:             pb.PairingMethod_ROUND_ROBIN,
	pb.PairMethod_PAIR_KING_OF_THE_HILL:        pb.PairingMethod_KING_OF_THE_HILL,
	pb.PairMethod_PAIR_FACTOR:                  pb.PairingMethod_FACTOR,
	pb.PairMethod_PAIR_INITIAL_FONTES:          pb.PairingMethod_INITIAL_FONTES,
	pb.PairMethod_PAIR_TEAM_ROUND_ROBIN:        pb.PairingMethod_TEAM_ROUND_ROBIN,
	pb.PairMethod_PAIR_INTERLEAVED_ROUND_ROBIN: pb.PairingMethod_INTERLEAVED_ROUND_ROBIN,
}

type policyArgs struct {
	req                      *pb.PairRequest
	copdata                  *copdatapkg.PrecompData
	playerNodes              []int
	lowestPossibleAbsCasher  int
	lowestPossibleHopeCasher int
	roundsRemaining          int
	roundPairingsRemaining   int
	gibsonGetsBye            bool
	prepairedRoundIdx        int
	prepairedPlayerIndexes   map[int]int
	lowestHopeOverride       map[int]int
	factor3ForcedPairings    [][2]int
	// topDownByePlayer is the player index chosen to receive the top-down bye,
	// or -1 if top-down byes don't apply. It is computed once (see
	// computeTopDownByePlayer) so that KH can avoid forcing a KOTH pairing
	// that would conflict with it; top-down byes take precedence over KOTH.
	topDownByePlayer int
	// disallowedLeaderOpponent is the player index added to an odd hopeful
	// contender group (for 1st place) who is barred from playing the leader,
	// or -1 if no such player exists this round. It is computed once (see
	// computeDisallowedLeaderOpponent) so that KH can avoid forcing a cash
	// prize KOTH pairing that would conflict with it.
	disallowedLeaderOpponent int
	// forcedLeaderVsThird is the player index of "3rd" (rank index 2) when
	// rank 2 is taking this round's bye while the leader's 1st-place
	// contention has narrowed to at most rank 3 - leaving the leader
	// without their strongest hopeful opponent this round. See
	// computeForcedLeaderVsThird. -1 when this special case doesn't apply.
	forcedLeaderVsThird int
	// forcedContenderByePlayer is the player index forced to take the bye
	// because PC (no bye for contenders) and BR (no repeat byes) would
	// otherwise be in direct conflict this round, or -1 if there's no such
	// clash. It is computed once (see computeForcedContenderBye).
	forcedContenderByePlayer int
	// top4LockActive reports whether the top-4-must-play-each-other policy
	// (see computeTop4LockActive and the "T4" constraint policy) is in
	// effect this round: exactly 4 players are still hopeful contenders for
	// 1st in the 2nd-to-last round, and Factor 3 didn't fire.
	top4LockActive bool
}

// computeForcedLeaderVsThird handles rank 2 receiving this round's bye while
// the "hopeful for 1st" contender group is small - the leader alone, the
// leader and rank 2, or the leader, rank 2, and rank 3 - ahead of, and
// instead of, the general odd-group-parity rule in
// computeDisallowedLeaderOpponent. In every one of those cases, rank 2
// sitting out this round leaves the leader without the strongest hopeful
// opponent actually available this round. Rather than let the general rule
// promote rank 3 into the group and then bar the leader from playing them -
// handing the leader an arbitrary weaker opponent instead - force the
// leader onto rank 3 directly (see the "L3" constraint policy and the CC
// weight policy's matching early-out).
//
// Like computeDisallowedLeaderOpponent, this defers entirely to Factor 3,
// and must run after computeTopDownByePlayer and computeForcedContenderBye
// so the definite bye recipient, if any, is known.
func computeForcedLeaderVsThird(pargs *policyArgs) int {
	if len(pargs.factor3ForcedPairings) > 0 {
		return -1
	}
	// The hopeful-for-1st group must be small enough that rank 2 is either
	// the group's last member or already outside it (i.e. the lowest-ranked
	// hopeful player is rank 1, 2, or 3) - LowestPossibleHopeNth[0] is
	// 0-indexed, so that's a boundary of at most 2.
	if pargs.copdata.LowestPossibleHopeNth[0] > 2 {
		return -1
	}
	// A gibsonized leader has already settled the race (see the doc comment
	// on computeDisallowedLeaderOpponent), so rank 1 sitting out doesn't
	// leave the leader without a hopeful opponent in the same sense - fall
	// through to the general rule instead.
	if pargs.copdata.GibsonizedPlayers[0] {
		return -1
	}
	numPlayers := len(pargs.playerNodes)
	// Do not consider the bye as a player in this case
	if pargs.playerNodes[numPlayers-1] == pkgstnd.ByePlayerIndex {
		numPlayers--
	}
	if numPlayers < 3 {
		return -1
	}
	byeRecipient := -1
	switch {
	case pargs.topDownByePlayer >= 0:
		byeRecipient = pargs.topDownByePlayer
	case pargs.forcedContenderByePlayer >= 0:
		byeRecipient = pargs.forcedContenderByePlayer
	}
	if byeRecipient < 0 || pargs.playerNodes[1] != byeRecipient {
		return -1
	}
	return pargs.playerNodes[2]
}

// computeDisallowedLeaderOpponent implements the odd-hopeful-contender-group
// rule: the group of players still hopeful to reach 1st is ranks
// [0..LowestPossibleHopeNth[0]]. If that group's size is odd, the next player
// down is pulled in to even it out, but is barred from playing the leader.
//
// Exception: if the raw group has exactly one player (only the leader
// itself) and the leader isn't gibsonized, no player is added or barred -
// 1st and 2nd are left to play each other via the usual weight policies.
//
// The leader (rank 0) doesn't count toward the group's pairable parity if
// they're Gibsonized: GibsonizedPlayers[0] uniquely means "guaranteed to
// finish 1st" (nothing beats 1st), so a Gibsonized leader has already
// settled the race - unlike the hopeful-to-cash boundary's blanket
// exclusion (any Gibsonized player, at whatever rank, is guaranteed to
// cash), a Gibson lock at any other rank in this group only guarantees that
// player won't fall below their current spot and says nothing about
// whether they can still win outright, so only rank 0's own Gibson status
// is excluded here, never anyone else's.
//
// This must run after computeTopDownByePlayer and computeForcedContenderBye
// (both already computed by this point - see the call order in
// copMinWeightMatching), so it can account for the definite bye recipient,
// if any: unlike LowestPossibleHopeNth[PlacePrizes-1] (the hopeful-to-cash
// boundary), GetPrecompData never pre-promotes LowestPossibleHopeNth[0] for
// raw parity, so there's no earlier promotion to retract here - just check
// parity against the bye-adjusted (and Gibson-adjusted) group size directly.
func computeDisallowedLeaderOpponent(pargs *policyArgs) int {
	// Factor 3 expansion already constrains the top-6 pairings (including the
	// leader's); applying this rule on top of it would overconstrain the matching.
	if len(pargs.factor3ForcedPairings) > 0 {
		return -1
	}
	// computeForcedLeaderVsThird handles the narrower rank-2-bye case of
	// this same group on its own terms - forcing the leader onto 3rd
	// instead of barring 3rd from the leader - so defer to it entirely
	// rather than competing over the same edge.
	if pargs.forcedLeaderVsThird >= 0 {
		return -1
	}
	numPlayers := len(pargs.playerNodes)
	// Do not consider the bye as a player in this case
	if pargs.playerNodes[numPlayers-1] == pkgstnd.ByePlayerIndex {
		numPlayers--
	}
	if numPlayers < 1 {
		return -1
	}
	contenderBoundary := pargs.copdata.LowestPossibleHopeNth[0]
	rawGroupSize := contenderBoundary + 1
	extraRankIdx := contenderBoundary + 1
	// The size-1 case is handled entirely separately from the general
	// parity check below, on purpose: even when the leader is Gibsonized, a
	// lone raw contender (just the leader) still gets the next player
	// pulled in and barred, to protect a genuine 2nd/3rd-place race even
	// though 1st is already settled - it must not fall through to the
	// general check, where subtracting the Gibsonized leader would turn
	// this into an (incorrect) even/no-op case.
	if rawGroupSize == 1 {
		if !pargs.copdata.GibsonizedPlayers[0] {
			return -1
		}
		if extraRankIdx >= numPlayers {
			return -1
		}
		return pargs.playerNodes[extraRankIdx]
	}

	// The leader doesn't count toward the group's pairable parity if
	// they're Gibsonized (see doc comment above).
	groupSize := rawGroupSize
	if pargs.copdata.GibsonizedPlayers[0] {
		groupSize--
	}

	byeRecipient := -1
	switch {
	case pargs.topDownByePlayer >= 0:
		byeRecipient = pargs.topDownByePlayer
	case pargs.forcedContenderByePlayer >= 0:
		byeRecipient = pargs.forcedContenderByePlayer
	}
	if byeRecipient >= 0 {
		for pri := 0; pri <= contenderBoundary; pri++ {
			if pargs.playerNodes[pri] == byeRecipient {
				// The bye recipient won't need a pairing partner this round,
				// so they don't count toward the group's pairable parity.
				groupSize--
				break
			}
		}
	}

	if groupSize%2 == 0 {
		return -1
	}
	if extraRankIdx >= numPlayers {
		return -1
	}
	return pargs.playerNodes[extraRankIdx]
}

// computeTop4LockActive implements the top-4-must-play-each-other policy: in
// the 2nd-to-last round, if exactly 4 players are still hopeful contenders
// for 1st (LowestPossibleHopeNth[0] == 3) and Factor 3 didn't fire, ranks
// 0-3 are barred from playing anyone outside that group of four (see the
// "T4" constraint policy below), forcing all four to play each other this
// round. Which specific matchups result - 1v2 & 3v4, 1v3 & 2v4, or 1v4 &
// 2v3 - is left entirely to the normal weight policies (RD, repeats, etc.)
// to decide; this only restricts the pool they choose from.
//
// Like computeDisallowedLeaderOpponent, this defers entirely to Factor 3
// when it fires: F3 already constrains the top-6 pairings (which include
// all of the top 4), so layering this rule on top would overconstrain the
// matching.
//
// This must run after computeTopDownByePlayer and computeForcedContenderBye
// (both already computed by this point - see the call order in
// copMinWeightMatching), so that a bye landing inside the top 4 - leaving
// only 3 of them to pair off - can be detected and the policy skipped
// rather than forcing an impossible 3-player group.
func computeTop4LockActive(pargs *policyArgs) bool {
	if len(pargs.factor3ForcedPairings) > 0 {
		return false
	}
	if pargs.roundsRemaining != 2 {
		return false
	}
	if pargs.copdata.LowestPossibleHopeNth[0] != 3 {
		return false
	}
	numPlayers := len(pargs.playerNodes)
	// Do not consider the bye as a player in this case
	if numPlayers > 0 && pargs.playerNodes[numPlayers-1] == pkgstnd.ByePlayerIndex {
		numPlayers--
	}
	if numPlayers < 4 {
		return false
	}
	byeRecipient := -1
	switch {
	case pargs.topDownByePlayer >= 0:
		byeRecipient = pargs.topDownByePlayer
	case pargs.forcedContenderByePlayer >= 0:
		byeRecipient = pargs.forcedContenderByePlayer
	}
	if byeRecipient >= 0 {
		for pri := 0; pri <= 3; pri++ {
			if pargs.playerNodes[pri] == byeRecipient {
				// One of the top 4 is sitting out this round, leaving only 3
				// to pair off - locking them together is impossible, so skip.
				return false
			}
		}
	}
	return true
}

// computeTopDownByePlayer determines which player (if any) should receive the
// top-down bye: the player with the fewest previous byes, scanning from the
// top of the standings down (ties go to the higher-ranked player). Returns -1
// if top-down byes don't apply in this round. Top-down byes take precedence
// over the Gibson bye (see the "GB" policy below), so this does not defer to
// pargs.gibsonGetsBye.
//
// This delegates to copdatapkg.ComputeTopDownByeRankIdx, which GetPrecompData
// also calls (before this function ever runs) so the control-loss search can
// exclude this same rank from being flagged as the destiny child - see that
// function's doc comment. Both call sites must agree on who gets the bye, so
// the computation lives in one place.
func computeTopDownByePlayer(pargs *policyArgs) int {
	byeRankIdx := copdatapkg.ComputeTopDownByeRankIdx(pargs.req, pargs.copdata.Standings, pargs.copdata.PairingCounts)
	if byeRankIdx < 0 {
		return -1
	}
	return pargs.copdata.Standings.GetPlayerIndex(byeRankIdx)
}

// computeForcedContenderBye detects when PC (which major-penalizes giving a
// hopeful-to-cash contender the bye) and BR (which major-penalizes giving
// anyone a repeat bye, when AllowRepeatByes is false) are in direct conflict:
// every player who hasn't yet had a bye is a contender, so the bye must
// either go to a contender (a PC violation) or repeat on someone (a BR
// violation). When that's the case, this forces the bye onto the
// lowest-ranked such contender with the fewest prior byes - the mirror image
// of computeTopDownByePlayer's top-down scan. Returns -1 if there's no clash
// this round (a bye-repeat-free non-contender is available, so PC and BR
// don't conflict), or if repeats can't be avoided regardless of who gets the
// bye (nobody, contender or not, is bye-repeat-free).
func computeForcedContenderBye(pargs *policyArgs) int {
	numPlayers := len(pargs.playerNodes)
	if pargs.req.AllowRepeatByes || pargs.playerNodes[numPlayers-1] != pkgstnd.ByePlayerIndex ||
		pargs.gibsonGetsBye || pargs.topDownByePlayer >= 0 {
		return -1
	}
	byePlayer := -1
	leastByes := int(pargs.req.Rounds + 1)
	// Scan bottom-up, excluding the bye - the mirror of computeTopDownByePlayer's
	// top-down scan, so ties favor the lower-ranked (bottom) contender.
	for playerRankIdx := numPlayers - 2; playerRankIdx >= 0; playerRankIdx-- {
		pi := pargs.playerNodes[playerRankIdx]
		numByes := pargs.copdata.PairingCounts[copdatapkg.GetPairingKey(pi, pkgstnd.ByePlayerIndex)]
		isContender := playerRankIdx <= pargs.lowestPossibleHopeCasher && !pargs.copdata.GibsonizedPlayers[playerRankIdx]
		if numByes == 0 && !isContender {
			// A non-contender is still bye-repeat-free; PC and BR don't conflict.
			return -1
		}
		if isContender && numByes < leastByes {
			leastByes = numByes
			byePlayer = pi
		}
	}
	// Only force the bye onto a contender if doing so actually avoids a
	// repeat - if even the least-byed contender already has one, forcing it
	// on them wouldn't help, so leave it to the weight policies.
	if byePlayer < 0 || leastByes > 0 {
		return -1
	}
	return byePlayer
}

// adjustLowestPossibleHopeCasherForBye corrects the hopeful-to-cash
// contender-group boundary once a specific player has definitively been
// assigned the bye this round (via a top-down bye or a forced-contender
// bye - both computed before this point). GetPrecompData's odd/even parity
// promotion guarantees the contender group's size is even *before* any bye
// is known, so that everyone in it can pair off within the group. If the
// bye recipient turns out to be inside that group, removing them from the
// pairing pool (they're paired with BYE, not a contender) makes the
// group's *remaining* pairable size odd again.
//
// How that's corrected depends on whether the bye recipient is the very
// player GetPrecompData promoted to fix a raw odd count
// (copdata.HopefulToCashPromotedPlayerRankIdx), or a genuine contender:
//   - If the bye recipient is a genuine contender (ranked above the
//     promoted player, or no promotion fired at all), extend the boundary
//     by one more rank, promoting the next player down into the group -
//     mirroring the promotion GetPrecompData already does for the raw
//     odd-count case.
//   - If the bye recipient is itself the artificially-promoted player, that
//     promotion no longer serves its purpose (the promoted player isn't
//     pairing with anyone this round either way), so extend past them too.
//   - If a promotion fired AND the bye recipient is a genuine contender
//     ranked *above* the promoted player, the promotion and the bye removal
//     cancel out: the raw (pre-promotion) contender count was odd, the
//     promotion made it even, and the bye now removes one, making it odd
//     again - back to exactly the raw count minus one, which is even. No
//     promotion is needed at all, so retract GetPrecompData's promotion
//     instead of extending further; extending on top of it would promote
//     two extra players for a group that needed zero.
//
// Pure Gibson byes are not handled here because a Gibsonized player is
// never counted as a hopeful-to-cash contender in the first place (see PC,
// CC, computeForcedContenderBye), so their removal never affects this
// parity.
func adjustLowestPossibleHopeCasherForBye(pargs *policyArgs, playerNodes []int, numPlayers int, logsb *strings.Builder) int {
	byeRecipient := -1
	switch {
	case pargs.topDownByePlayer >= 0:
		byeRecipient = pargs.topDownByePlayer
	case pargs.forcedContenderByePlayer >= 0:
		byeRecipient = pargs.forcedContenderByePlayer
	default:
		return pargs.lowestPossibleHopeCasher
	}
	byeRank := -1
	for pri := 0; pri < numPlayers; pri++ {
		if playerNodes[pri] == byeRecipient {
			byeRank = pri
			break
		}
	}
	if byeRank < 0 || byeRank > pargs.lowestPossibleHopeCasher {
		// Bye recipient isn't in the hopeful-to-cash contender group; no adjustment needed.
		return pargs.lowestPossibleHopeCasher
	}
	if pargs.copdata.GibsonizedPlayers[byeRank] {
		// The bye recipient is Gibsonized, so they were never counted toward
		// the parity-relevant contender count in the first place (see the
		// Gibson exclusion in GetPrecompData and PC/CC) - removing them from
		// the pairing pool via a bye doesn't change that count, so no
		// adjustment is needed even though their rank falls within the
		// boundary.
		return pargs.lowestPossibleHopeCasher
	}

	promotedRankIdx := pargs.copdata.HopefulToCashPromotedPlayerRankIdx
	if promotedRankIdx >= 0 && byeRank < promotedRankIdx {
		// The bye landed on a genuine contender, not the artificially
		// promoted one - GetPrecompData's promotion is now redundant, so
		// retract it instead of extending further (which would promote two
		// extra players for a group that needed none).
		newBoundary := promotedRankIdx - 1
		logsb.WriteString(fmt.Sprintf(
			"Contender group parity: %s (rank %d) is inside the hopeful-to-cash contender group "+
				"(boundary rank %d) and is receiving the bye this round; this already restores parity "+
				"for the raw (pre-promotion) contender count, so retracting the earlier promotion of %s, "+
				"restoring the boundary to rank %d\n",
			pargs.req.PlayerNames[byeRecipient], byeRank+1, pargs.lowestPossibleHopeCasher+1,
			pargs.req.PlayerNames[playerNodes[promotedRankIdx]], newBoundary+1,
		))
		return newBoundary
	}

	newBoundary := pargs.lowestPossibleHopeCasher + 1
	if newBoundary >= numPlayers {
		// No player left to promote; leave the boundary as-is rather than
		// running off the end of the standings.
		return pargs.lowestPossibleHopeCasher
	}
	logsb.WriteString(fmt.Sprintf(
		"Contender group parity: %s (rank %d) is inside the hopeful-to-cash contender group "+
			"(boundary rank %d) and is receiving the bye this round, leaving an odd number of "+
			"contenders needing an opponent; extending the boundary to rank %d (promoting %s) to restore parity\n",
		pargs.req.PlayerNames[byeRecipient], byeRank+1, pargs.lowestPossibleHopeCasher+1,
		newBoundary+1, pargs.req.PlayerNames[playerNodes[newBoundary]],
	))
	return newBoundary
}

type constraintPolicy struct {
	name    string
	handler func(*policyArgs) ([][2]int, [][2]int)
}

type weightPolicy struct {
	name    string
	handler func(*policyArgs, int, int) int64
}

var constraintPolicies = []constraintPolicy{
	{
		// Prepaired players
		name: "PP",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			if pargs.prepairedRoundIdx == -1 {
				return [][2]int{}, [][2]int{}
			}
			numPlayers := len(pargs.playerNodes)
			disallowedPairings := [][2]int{}
			for playerIdx := range pargs.prepairedPlayerIndexes {
				for i := 0; i < numPlayers; i++ {
					disallowedPairings = append(disallowedPairings, [2]int{playerIdx, pargs.playerNodes[i]})
				}
			}
			return [][2]int{}, disallowedPairings
		},
	},
	{
		// KOTH
		name: "KH",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			if pargs.roundsRemaining != 1 {
				return [][2]int{}, [][2]int{}
			}
			numPlayers := len(pargs.playerNodes)
			forcedPairings := [][2]int{}

			// First, compute the cash prize KOTH players

			// Ranks are scanned as adjacent pairs below, and a rank that's
			// skipped (rather than force-paired) is implicitly left for the
			// *next* window to retry against its neighbor - which is correct
			// for e.g. a Gibsonized player, who still plays a real game this
			// round, just not a forced one. The top-down bye recipient is
			// different: they have no opponent at all this round, so they
			// must be removed from the scan entirely up front. Leaving them
			// in would silently drop their neighbor from KOTH consideration
			// too - e.g. rank 9 skipped against rank 10 (the bye recipient),
			// then rank 10 skipped again against rank 11, forcing rank 11
			// against rank 12 and leaving rank 9 unpaired by KOTH altogether,
			// so they fall through to the general weighted matching and can
			// end up paired with someone far outside contention.
			rankIdxs := make([]int, 0, numPlayers)
			for ri := 0; ri < numPlayers; ri++ {
				if pargs.topDownByePlayer >= 0 && pargs.playerNodes[ri] == pargs.topDownByePlayer {
					continue
				}
				rankIdxs = append(rankIdxs, ri)
			}

			var highestNoncontender int
			for k := 0; k < len(rankIdxs)-1; k++ {
				playerRankIdx := rankIdxs[k]
				if pargs.lowestPossibleAbsCasher < playerRankIdx {
					highestNoncontender = playerRankIdx
					break
				}
				nextRankIdx := rankIdxs[k+1]
				pi := pargs.playerNodes[playerRankIdx]
				pj := pargs.playerNodes[nextRankIdx]
				if pi == pkgstnd.ByePlayerIndex || pj == pkgstnd.ByePlayerIndex {
					continue
				}
				if pargs.copdata.GibsonizedPlayers[playerRankIdx] || pargs.copdata.GibsonizedPlayers[nextRankIdx] {
					continue
				}
				// The odd-hopeful-contender-group rule also takes precedence over
				// cash prize KOTH: don't force the leader to play the player barred
				// from playing them. In practice this only matters if
				// computeDisallowedLeaderOpponent's single-contender exception
				// doesn't apply (i.e. the leader is gibsonized), in which case the
				// gibsonized check above already skips this pairing - this guard is
				// kept for defense in depth.
				if playerRankIdx == 0 && (pi == pargs.disallowedLeaderOpponent || pj == pargs.disallowedLeaderOpponent) {
					continue
				}
				forcedPairings = append(forcedPairings, [2]int{pi, pj})
				// A pairing with pi and pj was forced, so we need to
				// skip evaluation for player pj in the next iteration
				// by incrementing k, which combined with this for loop
				// effectively performs a k += 2
				k++
			}

			// Then, compute the class prize KOTH players

			// Do not consider the bye as a player in this case
			if pargs.playerNodes[numPlayers-1] == pkgstnd.ByePlayerIndex {
				numPlayers--
			}
			for classPrizesIdx, classPrizes := range pargs.req.ClassPrizes {
				classIdx := classPrizesIdx + 1
				availableClassPrizes := int(classPrizes)
				for pRankIdx := 0; pRankIdx < highestNoncontender; pRankIdx++ {
					pIdx := pargs.copdata.Standings.GetPlayerIndex(pRankIdx)
					if int(pargs.req.PlayerClasses[pIdx]) == classIdx && pargs.copdata.LowestRankAbsolutely[pRankIdx] >= int(pargs.req.PlacePrizes) {
						availableClassPrizes--
					}
				}
				if availableClassPrizes < 1 {
					continue
				}
				ri := highestNoncontender
				numPlayersAhead := 0
				playerToCatch := -1
				KOTHCumeGibsonSpread := int(pargs.req.GibsonSpread * 2)
				for {
					// Top-down byes take precedence over class prize KOTH: the bye
					// recipient (if in this class) has no opponent at all this
					// round, so they're skipped from the scan entirely - just like
					// the cash-prize KOTH scan above - and the algorithm works
					// around them, forcing the next two available class contenders
					// together instead of dropping the forced pairing altogether.
					for ri < numPlayers && (int(pargs.req.PlayerClasses[pargs.playerNodes[ri]]) != classIdx ||
						pargs.playerNodes[ri] == pargs.topDownByePlayer) {
						ri++
					}
					rj := ri + 1
					for rj < numPlayers && (int(pargs.req.PlayerClasses[pargs.playerNodes[rj]]) != classIdx ||
						pargs.playerNodes[rj] == pargs.topDownByePlayer) {
						rj++
					}
					if rj >= numPlayers {
						break
					}
					if !pargs.copdata.Standings.CanCatch(1, KOTHCumeGibsonSpread, ri, rj) {
						ri = rj
						numPlayersAhead++
						if numPlayersAhead == availableClassPrizes {
							break
						}
						continue
					}
					placesRemaining := availableClassPrizes - numPlayersAhead
					if placesRemaining == 2 {
						playerToCatch = rj
					} else if placesRemaining == 1 {
						playerToCatch = ri
					} else if playerToCatch >= 0 && !pargs.copdata.Standings.CanCatch(1, KOTHCumeGibsonSpread, playerToCatch, rj) {
						break
					}
					forcedPairings = append(forcedPairings, [2]int{pargs.playerNodes[ri], pargs.playerNodes[rj]})
					numPlayersAhead += 2
					ri = rj + 1
				}
			}
			return forcedPairings, [][2]int{}
		},
	},
	{
		// Control loss
		name: "CL",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			if pargs.copdata.DestinysChild < 0 {
				return [][2]int{}, [][2]int{}
			}
			// Factor 3 expansion already constrains the top-6 pairings; applying
			// control loss on top of it would overconstraint the matching.
			if len(pargs.factor3ForcedPairings) > 0 {
				return [][2]int{}, [][2]int{}
			}
			disallowedPairings := [][2]int{}
			numPlayers := len(pargs.playerNodes)
			for playerRankIdx := 1; playerRankIdx < numPlayers; playerRankIdx++ {
				// This if statement implements the following logic:
				//
				// Force pair the player in first with the bottom contender if:
				//  - There are controlLossLowestContenderOnlyRounds or fewer rounds remaining, OR
				//  - This is the first round where control loss is active
				// Force pair the player in first with the bottom 2 contenders:
				//  - otherwise
				if playerRankIdx == pargs.copdata.DestinysChild ||
					(pargs.roundsRemaining > controlLossLowestContenderOnlyRounds && int(pargs.req.ControlLossActivationRound) != pargs.copdata.CompletePairings &&
						playerRankIdx == pargs.copdata.DestinysChild-1) {
					continue
				}
				disallowedPairings = append(disallowedPairings, [2]int{pargs.playerNodes[0], pargs.playerNodes[playerRankIdx]})
			}
			return [][2]int{}, disallowedPairings
		},
	},
	{
		// Gibson groups
		name: "GG",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			numPlayers := len(pargs.playerNodes)
			// Do not consider the bye as a player in this case
			if pargs.playerNodes[numPlayers-1] == pkgstnd.ByePlayerIndex {
				numPlayers--
			}
			disallowedPairings := [][2]int{}
			for pri := 0; pri < numPlayers; pri++ {
				for prj := pri + 1; prj < numPlayers; prj++ {
					if pargs.copdata.GibsonGroups[pri] != pargs.copdata.GibsonGroups[prj] {
						disallowedPairings = append(disallowedPairings, [2]int{pargs.playerNodes[pri], pargs.playerNodes[prj]})
					}
				}
			}
			return [][2]int{}, disallowedPairings
		},
	},
	{
		// Gibson Bye
		name: "GB",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			// Top-down byes take precedence over the Gibson bye: if TB has
			// already claimed the bye for a specific player this round, don't
			// also disallow it from everyone else here.
			if !pargs.gibsonGetsBye || pargs.topDownByePlayer >= 0 {
				return [][2]int{}, [][2]int{}
			}
			disallowedPairings := [][2]int{}
			numPlayers := len(pargs.playerNodes)
			// Do not consider the bye as a player in this case
			if pargs.playerNodes[numPlayers-1] == pkgstnd.ByePlayerIndex {
				numPlayers--
			}
			for pri := 0; pri < numPlayers; pri++ {
				if pargs.copdata.GibsonizedPlayers[pri] {
					continue
				}
				disallowedPairings = append(disallowedPairings, [2]int{pargs.playerNodes[pri], pkgstnd.ByePlayerIndex})
			}
			return [][2]int{}, disallowedPairings
		},
	},
	{
		// Top Down Byes
		name: "TB",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			if pargs.topDownByePlayer < 0 {
				return [][2]int{}, [][2]int{}
			}
			return [][2]int{{pargs.topDownByePlayer, pkgstnd.ByePlayerIndex}}, [][2]int{}
		},
	},
	{
		// Forced Contender Bye: PC (no bye for contenders) and BR (no repeat
		// byes) are in direct conflict this round, so force the bye onto the
		// contender it picks rather than letting a repeat bye happen instead.
		name: "CB",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			if pargs.forcedContenderByePlayer < 0 {
				return [][2]int{}, [][2]int{}
			}
			return [][2]int{{pargs.forcedContenderByePlayer, pkgstnd.ByePlayerIndex}}, [][2]int{}
		},
	},
	{
		// Factor 3 expansion for the 2nd-to-last round
		name: "F3",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			if len(pargs.factor3ForcedPairings) == 0 {
				return [][2]int{}, [][2]int{}
			}
			return pargs.factor3ForcedPairings, [][2]int{}
		},
	},
	{
		// Top 4 lock: in the 2nd-to-last round, when exactly 4 players are
		// still hopeful contenders for 1st and Factor 3 didn't fire, bar all
		// of them from playing anyone outside that group - see
		// computeTop4LockActive.
		name: "T4",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			if !pargs.top4LockActive {
				return [][2]int{}, [][2]int{}
			}
			disallowedPairings := [][2]int{}
			numPlayers := len(pargs.playerNodes)
			for pri := 0; pri <= 3; pri++ {
				for prj := 4; prj < numPlayers; prj++ {
					disallowedPairings = append(disallowedPairings, [2]int{pargs.playerNodes[pri], pargs.playerNodes[prj]})
				}
			}
			return [][2]int{}, disallowedPairings
		},
	},
	{
		// Leader vs 3rd: when only the leader and one other player are
		// still hopeful for 1st, and that other player is receiving this
		// round's bye, force the leader to play 3rd instead of leaving it
		// to the general odd-hopeful-contender-group bar - see
		// computeForcedLeaderVsThird.
		name: "L3",
		handler: func(pargs *policyArgs) ([][2]int, [][2]int) {
			if pargs.forcedLeaderVsThird < 0 {
				return [][2]int{}, [][2]int{}
			}
			return [][2]int{{pargs.playerNodes[0], pargs.forcedLeaderVsThird}}, [][2]int{}
		},
	},
}

// isLastQuarter reports whether the tournament is in its last quarter of
// rounds - the shared boundary the RD, PC, and CC weight policies all key off.
func isLastQuarter(pargs *policyArgs) bool {
	return copdatapkg.IsLastQuarter(pargs.roundPairingsRemaining, pargs.req.Rounds)
}

// isFinalRound reports whether this is the tournament's true final round -
// distinct from isLastQuarter, which covers the whole last quarter of rounds.
// In the final round, the fourth-quarter-only cash-contention weight logic in
// PC/CC/RD is disabled, so those policies fall back to plain rank-diff KOTH
// weighting for everyone (including Gibsonized players and hopeful cashers)
// in whatever pool is left after KH's forced KOTH pairings.
func isFinalRound(pargs *policyArgs) bool {
	return pargs.roundsRemaining == 1
}

// hopeToCashBoundary returns the hopeful-to-cash rank boundary used by the CC
// weight policy. In divisions of 12+ players it's clamped so the bottom 6
// are never counted as hopeful to cash, even if the normal
// lowestPossibleHopeCasher computation would reach that far down. Without
// this clamp, a bottom-6 player who is still (barely) hopeful to cash would
// have no non-major-penalty pairing available: CC's hopeful-vs-hopeful rule
// would major-penalize pairing them with a non-hopeful player, and its
// hopeful-vs-bottom-6 rule would major-penalize pairing them with a hopeful
// one.
func hopeToCashBoundary(pargs *policyArgs) int {
	boundary := pargs.lowestPossibleHopeCasher
	numPlayers := len(pargs.playerNodes)
	// Do not consider the bye as a player in this case
	if pargs.playerNodes[numPlayers-1] == pkgstnd.ByePlayerIndex {
		numPlayers--
	}
	if numPlayers < 12 {
		return boundary
	}
	maxBoundary := numPlayers - 6 - 1
	if boundary > maxBoundary {
		boundary = maxBoundary
	}
	return boundary
}

var weightPolicies = []weightPolicy{
	{
		// Rank diff
		name: "RD",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			diff := int64(rj - ri)
			// rj might be the Bye, which is out of range for this array
			rjGibsonized := false
			if rj < len(pargs.copdata.GibsonizedPlayers) {
				rjGibsonized = pargs.copdata.GibsonizedPlayers[rj]
			}
			// In the fourth quarter (but not the final round), cashers use PC
			// weight exclusively; zero out RD. In the final round, PC is
			// disabled entirely (see below), so RD stays active for everyone.
			if isLastQuarter(pargs) && !isFinalRound(pargs) &&
				!pargs.copdata.GibsonizedPlayers[ri] &&
				ri <= pargs.lowestPossibleHopeCasher {
				return 0
			}
			// If
			//
			// - either play is gibsonized, or
			// - neither player cashed even once in the simulation, then
			//
			// the rank difference should squared
			// so it doesn't overwhelm the repeat penalty.
			if pargs.copdata.GibsonizedPlayers[ri] || rjGibsonized ||
				ri >= int(pargs.req.PlacePrizes) {
				return diff * diff
			}
			return diff * diff * diff
		},
	},
	{
		// Pair with Casher
		name: "PC",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			// In the final round, cash-contention weighting is disabled
			// entirely (Gibsonized players get no special exemption from it
			// either) - everyone falls back to plain RD/RE weighting for
			// whatever pool KH's forced KOTH pairings leave behind.
			if isFinalRound(pargs) {
				return 0
			}
			// rj might be the Bye, which is out of range for this array
			rjGibsonized := false
			if rj < len(pargs.copdata.GibsonizedPlayers) {
				rjGibsonized = pargs.copdata.GibsonizedPlayers[rj]
			}
			if pargs.copdata.GibsonizedPlayers[ri] || rjGibsonized || ri > pargs.lowestPossibleHopeCasher {
				return 0
			}
			if pargs.playerNodes[rj] == pkgstnd.ByePlayerIndex {
				// The TB constraint policy already forces this exact
				// pairing (it disallows every other pairing for the
				// top-down bye recipient), so this edge is the only one
				// the matcher can pick regardless of its weight - major-
				// penalizing it here would just trigger cop.go's Retry
				// fallback for nothing.
				if pargs.playerNodes[ri] == pargs.topDownByePlayer {
					return 0
				}
				return hopefulCasherByeWeight
			}
			lowestContender := pargs.copdata.LowestPossibleHopeNth[ri]
			if override, ok := pargs.lowestHopeOverride[ri]; ok {
				lowestContender = override
			}
			// Check if we should apply an inverse distance penalty
			if rj <= lowestContender || (lowestContender == ri && ri == rj-1) {
				// Only apply PC weight in the fourth quarter.
				if !isLastQuarter(pargs) {
					return 0
				}
				// Calculate the inverse distance penalty
				casherDiff := lowestContender - rj
				if casherDiff < 0 {
					casherDiff *= -1
				}
				return int64(math.Pow(float64(casherDiff), 3) * 2)
			}
			// Apply a major penalty if the lower ranked player cannot catch
			// the higher ranked player - except for the first rank just
			// outside the contention window, which gets a halved major
			// penalty instead. If COP is ever forced into a major-penalty
			// pairing anyway (e.g. because every in-window opponent is
			// otherwise taken), the halved weight makes it prefer reaching
			// only one rank outside the window over reaching further.
			if rj == lowestContender+1 {
				return majorPenalty / 2
			}
			return majorPenalty
		},
	},
	{
		// Cash contention. This folds together three rules that all turn on
		// who's still "hopeful" for something, so that they're expressed as
		// one set of major-penalty weights instead of separate rules that
		// could stack into a hard, unsatisfiable constraint for a player who
		// straddles more than one boundary at once:
		//
		//   - Hopeful-to-cash vs hopeful-to-cash (4th quarter only): players
		//     who are hopeful to cash should play other hopeful-to-cash
		//     players, and players who aren't hopeful to cash should play
		//     other players who aren't hopeful to cash.
		//   - Hopeful-to-cash vs bottom 6 (divisions of 12+ players, 4th
		//     quarter only): discourage hopeful-to-cash players from playing
		//     players in the bottom 6 of the standings.
		//   - Odd hopeful-contender-group for 1st (any round): when the
		//     group of players still hopeful to reach 1st is odd-sized, the
		//     extra player pulled in to even it out is barred from playing
		//     the leader. This used to be a hard-disallowed pairing; folding
		//     it in here as a major penalty instead means it can never
		//     combine with the other two rules to leave a player with no
		//     legal pairing at all.
		//   - Leader vs 3rd (any round): the rank-2-bye exception to the
		//     rule above - see computeForcedLeaderVsThird - is the mirror
		//     image: that edge is forced, not barred, so it must be
		//     exempted from every major penalty below (not just the
		//     leader-opponent one) or the L3 constraint policy would force
		//     an edge this policy then major-penalizes anyway.
		name: "CC",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			// Leader vs 3rd: this edge is forced by the L3 constraint
			// policy, so it must never be penalized here.
			if ri == 0 && pargs.forcedLeaderVsThird >= 0 &&
				pargs.playerNodes[rj] == pargs.forcedLeaderVsThird {
				return 0
			}
			// Odd hopeful-contender-group for 1st: bar the added player from
			// playing the leader (rank 0). This applies in every round, not
			// just the fourth quarter.
			if ri == 0 && pargs.disallowedLeaderOpponent >= 0 &&
				pargs.playerNodes[rj] == pargs.disallowedLeaderOpponent {
				return majorPenalty
			}
			// The hopeful-vs-hopeful and hopeful-vs-bottom-6 rules below are
			// fourth-quarter-only, and are further disabled in the final
			// round specifically (see isFinalRound), where cash-contention
			// grouping gives way to plain RD/RE weighting for the pool left
			// over after KH's forced KOTH pairings.
			if !isLastQuarter(pargs) || isFinalRound(pargs) {
				return 0
			}
			// The bye is neither hopeful nor unhopeful to cash; leave it out
			// of this policy so it doesn't get major-penalized against everyone.
			if pargs.playerNodes[rj] == pkgstnd.ByePlayerIndex {
				return 0
			}
			hopeCasherBoundary := hopeToCashBoundary(pargs)
			riHopeful := ri <= hopeCasherBoundary && !pargs.copdata.GibsonizedPlayers[ri]
			rjHopeful := rj <= hopeCasherBoundary && !pargs.copdata.GibsonizedPlayers[rj]
			if riHopeful != rjHopeful {
				// The first rank just outside the hopeful-to-cash boundary
				// (ri < rj always, so this can only be rj) gets a halved
				// major penalty instead of the full one - same rationale as
				// PC's halved penalty for the first rank outside its
				// contention window.
				if rj == hopeCasherBoundary+1 {
					return majorPenalty / 2
				}
				return majorPenalty
			}
			numPlayers := len(pargs.playerNodes)
			// Do not consider the bye as a player in this case
			if pargs.playerNodes[numPlayers-1] == pkgstnd.ByePlayerIndex {
				numPlayers--
			}
			if numPlayers >= 12 {
				bottomSixBoundary := numPlayers - 6
				riBottomSix := ri >= bottomSixBoundary
				rjBottomSix := rj >= bottomSixBoundary
				if (riHopeful && rjBottomSix) || (rjHopeful && riBottomSix) {
					// Same halving for the first rank of the bottom six -
					// the closest to the rest of the field.
					if ri == bottomSixBoundary || rj == bottomSixBoundary {
						return majorPenalty / 2
					}
					return majorPenalty
				}
			}
			return 0
		},
	},
	{
		// Gibson cashers
		name: "GC",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			// rj might be the Bye, which is out of range for these arrays
			rjGibsonGroup := 0
			if rj < len(pargs.copdata.GibsonGroups) {
				rjGibsonGroup = pargs.copdata.GibsonGroups[rj]
			}
			rjGibsonized := false
			if rj < len(pargs.copdata.GibsonizedPlayers) {
				rjGibsonized = pargs.copdata.GibsonizedPlayers[rj]
			}
			if pargs.copdata.GibsonGroups[ri] != 0 || rjGibsonGroup != 0 ||
				(pargs.copdata.GibsonizedPlayers[ri] && rjGibsonized) {
				return 0
			}
			if pargs.copdata.GibsonizedPlayers[ri] && rj <= pargs.lowestPossibleAbsCasher ||
				rjGibsonized && ri <= pargs.lowestPossibleAbsCasher {
				return majorPenalty
			}
			return 0
		},
	},
	{
		// Repeats
		name: "RE",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			pi := pargs.playerNodes[ri]
			pj := pargs.playerNodes[rj]
			pairingKey := copdatapkg.GetPairingKey(pi, pj)
			timesPlayed := pargs.copdata.PairingCounts[pairingKey]
			unitWeight := int64(4 * int(math.Pow(float64(pargs.copdata.Standings.GetNumPlayers())/3.0, 3)))
			// We would like the following to always be true:
			//
			// n-peat weight > 2 * (n-1)-peat weight
			//
			// The minimal recursive formula satisfying this is:
			//
			// RE(1) = 1
			// RE(n) = 2 * RE(n-1) + 1
			//
			// which results in the following values for repeats:
			// RE(1) = 1
			// RE(2) = 3
			// RE(3) = 7
			// RE(4) = 15
			// RE(5) = 31
			// ...
			multiplier := 0
			if timesPlayed > 0 {
				multiplier = (1 << timesPlayed) - 1
			}
			weight := int64(multiplier) * unitWeight
			// 1st vs 2nd gets an unconditional extra tenth of a repeat's
			// weight (regardless of timesPlayed), nudging the matching away
			// from pairing them together unless doing so is otherwise
			// favorable - without outright forbidding it the way a major
			// penalty would.
			if ri == 0 && rj == 1 {
				weight += unitWeight / 10
			}
			return weight
		},
	},
	{
		// Back-to-back repeats for non-cashers
		name: "BB",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			if ri <= pargs.lowestPossibleHopeCasher {
				return 0
			}
			mostRecentCompletedRound := pargs.copdata.CompletePairings - 1
			if pargs.prepairedRoundIdx >= 0 {
				mostRecentCompletedRound = pargs.prepairedRoundIdx - 1
			}
			if mostRecentCompletedRound < 0 {
				return 0
			}
			pi := pargs.playerNodes[ri]
			pj := pargs.playerNodes[rj]
			if int(pargs.req.DivisionPairings[mostRecentCompletedRound].Pairings[pi]) != pj {
				return 0
			}
			return minorPenalty
		},
	},
	{
		// Bye Repeats
		name: "BR",
		handler: func(pargs *policyArgs, ri int, rj int) int64 {
			if pargs.req.AllowRepeatByes {
				return 0
			}
			pj := pargs.playerNodes[rj]
			if pj != pkgstnd.ByePlayerIndex {
				return 0
			}
			pi := pargs.playerNodes[ri]
			numTimesWithBye := pargs.copdata.PairingCounts[copdatapkg.GetPairingKey(pi, pj)]
			if numTimesWithBye == 0 {
				return 0
			}
			return majorPenalty * int64(numTimesWithBye)
		},
	},
}

func addPairRequestAsJSONToLog(req *pb.PairRequest, logsb *strings.Builder, includeResultsAndPairings bool) {
	divisionPairings := req.DivisionPairings
	divisionResults := req.DivisionResults
	playerClasses := req.PlayerClasses
	playerNames := req.PlayerNames
	if !includeResultsAndPairings {
		req.DivisionPairings = nil
		req.DivisionResults = nil
		req.PlayerClasses = nil
		req.PlayerNames = nil
	}
	marshaler := protojson.MarshalOptions{
		Multiline:    true, // Enables pretty printing
		Indent:       "  ", // Sets the indentation level
		AllowPartial: true,
	}
	jsonData, err := marshaler.Marshal(req)
	if err != nil {
		logsb.WriteString("error writing pair request to log: " + err.Error() + "\n\n")
		return
	}
	if !includeResultsAndPairings {
		logsb.WriteString("Abridged pair request:\n\n")
	} else {
		logsb.WriteString("\n\nPair request:\n\n")
	}
	logsb.Write(jsonData)
	logsb.WriteString("\n\n")
	req.DivisionPairings = divisionPairings
	req.DivisionResults = divisionResults
	req.PlayerClasses = playerClasses
	req.PlayerNames = playerNames
}

func COPPair(req *pb.PairRequest) *pb.PairResponse {
	logsb := &strings.Builder{}
	starttime := time.Now()
	if req.Seed == 0 {
		// Create a seed from the player names and the length of the pairings
		hash := fnv.New64()
		for _, name := range req.PlayerNames {
			_, _ = hash.Write([]byte(name))
		}
		req.Seed = int64(hash.Sum64()) + int64(len(req.DivisionPairings))
	}
	addPairRequestAsJSONToLog(req, logsb, false)
	resp := copPairWithLog(req, logsb)
	endtime := time.Now()
	duration := endtime.Sub(starttime)
	if resp.ErrorCode != pb.PairError_SUCCESS {
		logsb.WriteString("\nCOP finished with error:\n\n" + resp.ErrorMessage + "\n\n")
	} else {
		logsb.WriteString("\nCOP finished successfully.\n\n")
	}
	logsb.WriteString(fmt.Sprintf("Started:  %s\nFinished: %s\nDuration: %s",
		starttime.Format(timeFormat), endtime.Format(timeFormat), duration))
	addPairRequestAsJSONToLog(req, logsb, true)
	resp.Log = logsb.String()
	return resp
}

func copPairWithLog(req *pb.PairRequest, logsb *strings.Builder) *pb.PairResponse {
	resp := verifyreq.Verify(req)
	if resp != nil {
		return resp
	}

	switch req.PairMethod {
	case pb.PairMethod_PAIR_SWISS:
		// Swiss is just COP: COP's weight policies (repeats, win/rank diff, etc.)
		// subsume Swiss's simpler weighting, so there's no separate implementation.
		return copMethodPair(req, logsb)
	case pb.PairMethod_COP:
		return copMethodPair(req, logsb)
	case pb.PairMethod_PAIR_AUTO:
		return autoPair(req, logsb)
	default:
		return simplePair(req, logsb)
	}
}

func copMethodPair(req *pb.PairRequest, logsb *strings.Builder) *pb.PairResponse {
	copRand := rand.New(rand.NewSource(uint64(req.Seed)))
	copdata, pairErr := copdatapkg.GetPrecompData(req, copRand, logsb)

	if pairErr != pb.PairError_SUCCESS {
		return &pb.PairResponse{
			ErrorCode:    pairErr,
			ErrorMessage: fmt.Sprintf("error computing required inputs: %s", pb.PairError_name[int32(pairErr)]),
		}
	}

	factor3ForcedPairings, factor3FinalRanks, factor3TotalSims := computeFactor3ForcedPairings(req, copdata, copRand, logsb)
	if factor3FinalRanks != nil {
		copdata.ApplyFactor3Sim(req, factor3FinalRanks, factor3TotalSims, logsb)
	}
	pairings, resp := copMinWeightMatching(req, copdata, factor3ForcedPairings, logsb)

	if resp != nil {
		return resp
	}

	return &pb.PairResponse{
		ErrorCode:         pb.PairError_SUCCESS,
		Pairings:          pairings,
		GibsonizedPlayers: copdata.GibsonizedPlayers,
	}
}

// computeFactor3ForcedPairings checks whether the 2nd-to-last round should use
// factor-3 pairings (1v4, 2v5, 3v6). It first checks whether 2nd or 3rd place
// would lose control of their destiny under factor-3 (compared to playing 1st
// directly); if so, only the affected player is paired against 1st. Otherwise it
// checks whether 4th/5th/6th can each reach 1st/2nd/3rd within the hopefulness
// threshold, and if so returns the three factor-3 forced pairs. Returns nil
// forced pairings when none are needed.
//
// The second and third return values are the Factor-3 simulation's
// FinalRanks/TotalSims, non-nil only when the full 1v4/2v5/3v6 expansion
// actually fires (the last return statement below) - the caller should feed
// these into copdata.ApplyFactor3Sim so the rest of the round's pairing
// weights are computed against the pairing structure that's actually being
// played, not the generic baseline one GetPrecompData started from. The
// control-loss branch's single forced pair doesn't get this treatment: it
// doesn't lock in the full factor-3 structure the simulation assumed, so
// that simulation isn't an accurate stand-in for the rest of the field.
func computeFactor3ForcedPairings(req *pb.PairRequest, copdata *copdatapkg.PrecompData, copRand *rand.Rand, logsb *strings.Builder) ([][2]int, [][]int, int) {
	if pkgstnd.GetRoundsRemaining(req) != 2 {
		logsb.WriteString("Factor 3 skipped: not 2 rounds remaining\n")
		return nil, nil, 0
	}
	numPlayers := copdata.Standings.GetNumPlayers()
	if numPlayers < 6 {
		logsb.WriteString(fmt.Sprintf("Factor 3 skipped: fewer than 6 players (%d)\n", numPlayers))
		return nil, nil, 0
	}

	// Top-down byes take precedence over Factor 3: if this round's bye
	// recipient is one of the top 6 (the group F3 forces into 0v3/1v4/2v5
	// pairings), F3 can't fire at all - the bye recipient can't be forced
	// into one of those pairings (they're not playing anyone this round),
	// and reshaping the structure around them would no longer be the
	// 1v4/2v5/3v6 expansion the control-loss/hopefulness checks below
	// actually simulated, so cancel F3 outright rather than patch around it.
	if topDownByeRankIdx := copdatapkg.ComputeTopDownByeRankIdx(req, copdata.Standings, copdata.PairingCounts); topDownByeRankIdx >= 0 && topDownByeRankIdx < 6 {
		logsb.WriteString(fmt.Sprintf(
			"Factor 3 skipped: rank %d (%s) is in the top 6 and would receive this round's top-down bye, which takes precedence over Factor 3\n",
			topDownByeRankIdx+1, req.PlayerNames[copdata.Standings.GetPlayerIndex(topDownByeRankIdx)],
		))
		return nil, nil, 0
	}

	// Build factor-3 pairings for the penultimate round (ranks 0v3, 1v4, 2v5,
	// then factor M/2 for the remaining players) plus KOTH for the final round.
	// pairingsLen is rounded up to even so simRound can always iterate in pairs
	// (RunParallelSimForceWinner adds a dummy bye player for odd player counts).
	pairingsLen := numPlayers
	if pairingsLen%2 == 1 {
		pairingsLen++
	}
	f3Pairings := make([][]int, 2)
	f3Pairings[0] = make([]int, pairingsLen) // penultimate: factor-3
	f3Pairings[1] = make([]int, pairingsLen) // final: KOTH
	// Top 6: factor-3 pairs (0,3), (1,4), (2,5).
	for i := 0; i < 3; i++ {
		f3Pairings[0][2*i] = i
		f3Pairings[0][2*i+1] = i + 3
	}
	// Remaining slots (ranks 6 to pairingsLen-1): factor (pairingsLen-6)/2.
	remCount := pairingsLen - 6
	remFactor := remCount / 2
	for i := 0; i < remFactor; i++ {
		f3Pairings[0][6+2*i] = 6 + i
		f3Pairings[0][6+2*i+1] = 6 + i + remFactor
	}
	// Final round: KOTH (consecutive pairs).
	for i := 0; i < pairingsLen/2; i++ {
		f3Pairings[1][2*i] = 2 * i
		f3Pairings[1][2*i+1] = 2*i + 1
	}

	// Simulate factor-3 pairings using the pre-built f3Pairings so the sim uses the
	// correct factor-3 structure (roundsRemaining=2 < maxFactor=3, so SimFactorPairAll
	// would cap at factor-2 internally).
	factor3FinalRanks, totalSims, err := copdata.Standings.RunSimsWithPairings(copRand, int(req.DivisionSims), 2, f3Pairings)
	if err != pb.PairError_SUCCESS {
		logsb.WriteString(fmt.Sprintf("Factor 3 skipped: sim error %v\n", err))
		return nil, nil, 0
	}
	if totalSims == 0 {
		logsb.WriteString("Factor 3 skipped: zero sims completed\n")
		return nil, nil, 0
	}
	copdatapkg.WriteFinalRankResultsToLog("Factor 3 Sim Results", factor3FinalRanks, copdata.Standings, req, logsb)

	// Check whether 2nd or 3rd place loses control under factor-3 pairings.
	// Each player is assumed to win all their games in both scenarios; the only
	// difference is whether they play 1st (vsFirst) or their factor-3 opponent
	// (vsFactor3) in the penultimate round.
	roundsRemaining := pkgstnd.GetRoundsRemaining(req)
	controlSims := int(req.ControlLossSims)
	threshold := req.ControlLossThreshold * float64(controlSims)
	stopTime := time.Now().Add(6 * time.Second).UnixNano()

	p0 := copdata.Standings.GetPlayerIndex(0)
	p1 := copdata.Standings.GetPlayerIndex(1)
	p2 := copdata.Standings.GetPlayerIndex(2)

	logsb.WriteString(fmt.Sprintf(
		"Factor 3 control loss standings: %s (%.1f/%d) %s (%.1f/%d) %s (%.1f/%d) %s (%.1f/%d) %s (%.1f/%d) %s (%.1f/%d)\n",
		req.PlayerNames[p0], float64(copdata.Standings.GetPlayerWinsIntTimesTwo(0))/2, copdata.Standings.GetPlayerSpread(0),
		req.PlayerNames[p1], float64(copdata.Standings.GetPlayerWinsIntTimesTwo(1))/2, copdata.Standings.GetPlayerSpread(1),
		req.PlayerNames[p2], float64(copdata.Standings.GetPlayerWinsIntTimesTwo(2))/2, copdata.Standings.GetPlayerSpread(2),
		req.PlayerNames[copdata.Standings.GetPlayerIndex(3)], float64(copdata.Standings.GetPlayerWinsIntTimesTwo(3))/2, copdata.Standings.GetPlayerSpread(3),
		req.PlayerNames[copdata.Standings.GetPlayerIndex(4)], float64(copdata.Standings.GetPlayerWinsIntTimesTwo(4))/2, copdata.Standings.GetPlayerSpread(4),
		req.PlayerNames[copdata.Standings.GetPlayerIndex(5)], float64(copdata.Standings.GetPlayerWinsIntTimesTwo(5))/2, copdata.Standings.GetPlayerSpread(5),
	))

	for _, rankIdx := range []int{1, 2} {
		pIdx := copdata.Standings.GetPlayerIndex(rankIdx)
		vsFirst, pairErr := copdata.Standings.RunParallelSimForceWinner(copRand, controlSims, roundsRemaining, 3, f3Pairings, pIdx, true, stopTime)
		if pairErr != pb.PairError_SUCCESS {
			continue
		}
		vsFactor3, pairErr := copdata.Standings.RunParallelSimForceWinner(copRand, controlSims, roundsRemaining, 3, f3Pairings, pIdx, false, stopTime)
		if pairErr != pb.PairError_SUCCESS {
			continue
		}
		logsb.WriteString(fmt.Sprintf(
			"Factor 3 control loss check rank %d (%s): vsFirst=%d/%d vsFactor3=%d/%d threshold=%.0f\n",
			rankIdx+1, req.PlayerNames[pIdx], vsFirst, controlSims, vsFactor3, controlSims, threshold,
		))
		if float64(vsFirst-vsFactor3) >= threshold {
			logsb.WriteString(fmt.Sprintf(
				"Factor 3 control loss: rank %d %s loses control, forcing %s vs %s\n",
				rankIdx+1, req.PlayerNames[pIdx], req.PlayerNames[p0], req.PlayerNames[pIdx],
			))
			return [][2]int{{p0, pIdx}}, nil, 0
		}
	}

	// Neither 2nd nor 3rd loses control under factor-3; check whether 4th/5th/6th
	// can each reach 1st/2nd/3rd respectively within the hopefulness threshold.
	minWins := int(math.Round(float64(totalSims) * float64(req.HopefulnessThreshold)))

	// 4th can reach 1st-or-better, 5th can reach 2nd-or-better, 6th can reach
	// 3rd-or-better. Use cumulative ("at least") sims rather than the exact-rank
	// cell, since a sim where e.g. 6th finishes 1st or 2nd is a strictly better
	// outcome than 3rd and must also count toward being hopeful for 3rd.
	p3 := copdata.Standings.GetPlayerIndex(3)
	p4 := copdata.Standings.GetPlayerIndex(4)
	p5 := copdata.Standings.GetPlayerIndex(5)
	atLeast := func(finalRanks []int, targetRank int) int {
		sum := 0
		for rank := 0; rank <= targetRank; rank++ {
			sum += finalRanks[rank]
		}
		return sum
	}
	p3AtLeast1st := atLeast(factor3FinalRanks[3], 0)
	p4AtLeast2nd := atLeast(factor3FinalRanks[4], 1)
	p5AtLeast3rd := atLeast(factor3FinalRanks[5], 2)
	if p3AtLeast1st < minWins ||
		p4AtLeast2nd < minWins ||
		p5AtLeast3rd < minWins {
		logsb.WriteString(fmt.Sprintf(
			"Factor 3 skipped: hopefulness threshold not met (minWins=%d 4th=%s->1st-or-better:%d 5th=%s->2nd-or-better:%d 6th=%s->3rd-or-better:%d)\n",
			minWins,
			req.PlayerNames[p3], p3AtLeast1st,
			req.PlayerNames[p4], p4AtLeast2nd,
			req.PlayerNames[p5], p5AtLeast3rd,
		))
		return nil, nil, 0
	}

	// Even when the hopefulness threshold is met, F3 shouldn't fire if it
	// only benefits the leader while taking destiny control away from the
	// field: require that at least one of 4th/5th/6th gains a meaningful
	// (>= f3MinWinChanceGain) shot at actually winning the tournament
	// outright, compared to the baseline (non-F3) factor pairing.
	// BaselineFinalRanks/factor3FinalRanks are both indexed by current player
	// rank from the same standings snapshot within this pairing call, so
	// ranks 3/4/5 line up between the two.
	if copdata.BaselineTotalSims > 0 {
		baseline3 := float64(copdata.BaselineFinalRanks[3][0]) / float64(copdata.BaselineTotalSims)
		baseline4 := float64(copdata.BaselineFinalRanks[4][0]) / float64(copdata.BaselineTotalSims)
		baseline5 := float64(copdata.BaselineFinalRanks[5][0]) / float64(copdata.BaselineTotalSims)
		f3Win3 := float64(factor3FinalRanks[3][0]) / float64(totalSims)
		f3Win4 := float64(factor3FinalRanks[4][0]) / float64(totalSims)
		f3Win5 := float64(factor3FinalRanks[5][0]) / float64(totalSims)
		gain3 := f3Win3 - baseline3
		gain4 := f3Win4 - baseline4
		gain5 := f3Win5 - baseline5
		maxGain := math.Max(gain3, math.Max(gain4, gain5))
		logsb.WriteString(fmt.Sprintf(
			"Factor 3 win-chance gain check: %s %.2f%%->%.2f%% (%+.2fpp), %s %.2f%%->%.2f%% (%+.2fpp), %s %.2f%%->%.2f%% (%+.2fpp), min required=%.2fpp\n",
			req.PlayerNames[p3], baseline3*100, f3Win3*100, gain3*100,
			req.PlayerNames[p4], baseline4*100, f3Win4*100, gain4*100,
			req.PlayerNames[p5], baseline5*100, f3Win5*100, gain5*100,
			f3MinWinChanceGain*100,
		))
		if maxGain < f3MinWinChanceGain {
			logsb.WriteString(fmt.Sprintf(
				"Factor 3 skipped: no meaningful win-chance gain (max gain %.2fpp < required %.2fpp) - would primarily help %s while reducing the field's destiny control\n",
				maxGain*100, f3MinWinChanceGain*100, req.PlayerNames[p0],
			))
			return nil, nil, 0
		}
	}

	logsb.WriteString(fmt.Sprintf(
		"Factor 3 expansion: forcing %s vs %s, %s vs %s, %s vs %s\n",
		req.PlayerNames[p0], req.PlayerNames[p3],
		req.PlayerNames[p1], req.PlayerNames[p4],
		req.PlayerNames[p2], req.PlayerNames[p5],
	))
	return [][2]int{{p0, p3}, {p1, p4}, {p2, p5}}, factor3FinalRanks, totalSims
}

// currentRoundIndex returns the index of the round being paired.
// If the last round in DivisionPairings is incomplete (has a -1 entry), that round is being
// completed, so its index is returned. Otherwise the next new round index is returned.
func currentRoundIndex(req *pb.PairRequest) int32 {
	n := len(req.DivisionPairings)
	if n > 0 && slices.Contains(req.DivisionPairings[n-1].Pairings, -1) {
		return int32(n - 1)
	}
	return int32(n)
}

// simplePairOnce runs a single round of a non-COP pairing method and returns
// the full allPlayers-length pairings slice (bye represented as self-pairing).
func simplePairOnce(req *pb.PairRequest, pairingMethod pb.PairingMethod, poolMembers []*entity.PoolMember, playerOrder []int, roundIdx int32, seed uint64) ([]int32, error) {
	roundControls := &pb.RoundControl{
		PairingMethod:               pairingMethod,
		Round:                       roundIdx,
		Factor:                      req.Factor,
		InitialFontes:               req.InitialNonperfRounds,
		GamesPerRound:               1,
		MaxRepeats:                  1,
		AllowOverMaxRepeats:         true,
		RepeatRelativeWeight:        1,
		WinDifferenceRelativeWeight: 1,
	}
	m := &entity.UnpairedPoolMembers{
		PoolMembers:   poolMembers,
		RoundControls: roundControls,
		Repeats:       map[string]int{},
		Seed:          seed,
	}
	poolPairings, err := pair.Pair(m)
	if err != nil {
		return nil, err
	}
	roundPairings := make([]int32, req.AllPlayers)
	for i := range roundPairings {
		roundPairings[i] = -1
	}
	for poolIdx, poolOppIdx := range poolPairings {
		pi := playerOrder[poolIdx]
		if poolOppIdx == -1 {
			roundPairings[pi] = int32(pi)
		} else {
			roundPairings[pi] = int32(playerOrder[poolOppIdx])
		}
	}
	return roundPairings, nil
}

// simplePair handles pairing methods that delegate entirely to pkg/pair/pair.go:
// Random, Round Robin, King of the Hill, Factor, Initial Fontes, Team Round Robin,
// Interleaved Round Robin.
func simplePair(req *pb.PairRequest, logsb *strings.Builder) *pb.PairResponse {
	removedPlayersSet := map[int]bool{}
	for _, idx := range req.RemovedPlayers {
		removedPlayersSet[int(idx)] = true
	}

	// For rank-dependent methods we need standings to get the correct player order.
	needsRankOrder := req.PairMethod == pb.PairMethod_PAIR_KING_OF_THE_HILL ||
		req.PairMethod == pb.PairMethod_PAIR_FACTOR

	standings := pkgstnd.CreateInitialStandings(req)
	standings.Sort()

	// Build the ordered list of valid players.
	playerOrder := []int{}
	if needsRankOrder {
		numPlayers := standings.GetNumPlayers()
		for rankIdx := range numPlayers {
			pi := standings.GetPlayerIndex(rankIdx)
			if !removedPlayersSet[pi] {
				playerOrder = append(playerOrder, pi)
			}
		}
	} else {
		for pi := 0; pi < int(req.AllPlayers); pi++ {
			if !removedPlayersSet[pi] {
				playerOrder = append(playerOrder, pi)
			}
		}
	}

	pairingMethod, ok := pairingMethodMap[req.PairMethod]
	if !ok {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_UNSUPPORTED_PAIR_METHOD,
			ErrorMessage: fmt.Sprintf("unsupported pair method: %v", req.PairMethod),
		}
	}

	poolMembers := make([]*entity.PoolMember, len(playerOrder))
	for i, pi := range playerOrder {
		poolMembers[i] = &entity.PoolMember{
			Id: req.PlayerNames[pi],
		}
	}

	currentRound := currentRoundIndex(req)
	allPlayerPairings, err := simplePairOnce(req, pairingMethod, poolMembers, playerOrder, currentRound, uint64(req.Seed))
	if err != nil {
		return &pb.PairResponse{
			ErrorCode:    pb.PairError_SIMPLE_PAIRING_FAILED,
			ErrorMessage: err.Error(),
		}
	}

	standingsHeader := []string{"Rank", "Num", "Name", "Wins", "Spr"}
	copdatapkg.WriteStringDataToLog("Initial Standings", standingsHeader, standings.StringData(req), logsb)

	logsb.WriteString(fmt.Sprintf("Method: %s\n", req.PairMethod))
	logsb.WriteString(fmt.Sprintf("Seed: %d\n", req.Seed))
	logsb.WriteString(fmt.Sprintf("Round control: PairingMethod=%s Round=%d Factor=%d InitialNonperf=%d\n",
		pairingMethod,
		currentRound,
		req.Factor,
		req.InitialNonperfRounds,
	))
	logsb.WriteString(fmt.Sprintf("Pool members (%d):\n", len(poolMembers)))
	for i, pm := range poolMembers {
		logsb.WriteString(fmt.Sprintf("  [%d] %s\n", i, pm.Id))
	}

	// Log the pairings array (player i plays allPlayerPairings[i]).
	logsb.WriteString("Pairings: [")
	for i, opp := range allPlayerPairings {
		if i > 0 {
			logsb.WriteString(" ")
		}
		logsb.WriteString(fmt.Sprintf("%d", opp))
	}
	logsb.WriteString("]\n")

	// Log pairings by standings rank with player records.
	numStandingsPlayers := standings.GetNumPlayers()
	playerIndexToRankIdx := make(map[int]int, numStandingsPlayers)
	divisionPlayerData := make([][]string, numStandingsPlayers)
	for rankIdx := range numStandingsPlayers {
		pi := standings.GetPlayerIndex(rankIdx)
		playerIndexToRankIdx[pi] = rankIdx
		divisionPlayerData[rankIdx] = standings.StringDataForPlayer(req, rankIdx)
	}
	matchupHeader := []string{"Player", "W", "S", "Player", "W", "S"}
	pairingsLogMx := [][]string{}
	for rankIdx := range numStandingsPlayers {
		pi := standings.GetPlayerIndex(rankIdx)
		opp := int(allPlayerPairings[pi])
		// A bye is represented as either a negative opponent index or the
		// player paired with themselves (the convention simplePairOnce uses
		// above, matching how self-pairing is interpreted as a bye
		// elsewhere, e.g. standings.go). Skip both so a byed player doesn't
		// fall through and get displayed playing themselves.
		if opp < 0 || opp == pi {
			continue
		}
		oppRankIdx, oppInStandings := playerIndexToRankIdx[opp]
		if !oppInStandings || oppRankIdx < rankIdx {
			continue
		}
		pairingsLogMx = append(pairingsLogMx, getMatchupStrArray(divisionPlayerData, rankIdx, oppRankIdx, req.PlayerClasses))
	}
	copdatapkg.WriteStringDataToLog("Final Pairings", matchupHeader, pairingsLogMx, logsb)

	// Compute multiround pairings.
	// If the tournament has existing pairings, multiround is just the current round's pairings.
	// If no pairings exist yet and this is a non-rank-order method, generate N rounds ahead.
	var multiroundPairings []int32
	if len(req.DivisionPairings) > 0 {
		multiroundPairings = append(multiroundPairings, allPlayerPairings...)
	} else if !needsRankOrder {
		N := max(int(req.InitialNonperfRounds), 1)
		multiroundPairings = append(multiroundPairings, allPlayerPairings...)
		for roundIdx := 1; roundIdx < N; roundIdx++ {
			rp, err := simplePairOnce(req, pairingMethod, poolMembers, playerOrder, currentRound+int32(roundIdx), uint64(req.Seed)+uint64(roundIdx))
			if err != nil {
				return &pb.PairResponse{
					ErrorCode:    pb.PairError_SIMPLE_PAIRING_FAILED,
					ErrorMessage: err.Error(),
				}
			}
			multiroundPairings = append(multiroundPairings, rp...)
		}
	}

	return &pb.PairResponse{
		ErrorCode:          pb.PairError_SUCCESS,
		Pairings:           allPlayerPairings,
		MultiroundPairings: multiroundPairings,
	}
}

// autoPair selects the most appropriate pairing method based on the tournament state.
//
// Compute rrRoundsTotal = floor(numRounds / (numValidPlayers-1)) * (numValidPlayers-1),
// the largest number of rounds that fits one or more complete RR cycles.
// If rrRoundsTotal > 0:
//   - Use Round Robin for rounds 0 through rrRoundsTotal-1.
//   - Use COP for any remaining rounds.
//
// Otherwise (numRounds < numValidPlayers-1):
//   - Rounds 0–2: Initial Fontes
//   - Round 3 onward: COP
func autoPair(req *pb.PairRequest, logsb *strings.Builder) *pb.PairResponse {
	origMethod := req.PairMethod
	origInitialNonperfRounds := req.InitialNonperfRounds
	resp := autoPairInner(req, logsb)
	restoreAutoPairReqFields(req, origMethod, origInitialNonperfRounds)
	return resp
}

func autoPairInner(req *pb.PairRequest, logsb *strings.Builder) *pb.PairResponse {
	numValidPlayers := int(req.ValidPlayers)
	numRounds := int(req.Rounds)
	currentRound := int(currentRoundIndex(req))
	rrRounds := numValidPlayers - 1
	rrRoundsTotal := (numRounds / rrRounds) * rrRounds
	if rrRoundsTotal > 0 {
		if currentRound < rrRoundsTotal {
			req.PairMethod = pb.PairMethod_PAIR_ROUND_ROBIN
			if currentRound == 0 {
				req.InitialNonperfRounds = int32(rrRoundsTotal)
			}
			fmt.Fprintf(logsb, "Auto: fitting %d complete RR cycle(s) (%d rounds), using Round Robin for round %d\n", rrRoundsTotal/rrRounds, rrRoundsTotal, currentRound)
			return simplePair(req, logsb)
		}
		req.PairMethod = pb.PairMethod_COP
		fmt.Fprintf(logsb, "Auto: round %d past RR rounds (%d), using COP\n", currentRound, rrRoundsTotal)
		return copMethodPair(req, logsb)
	}

	if currentRound < 3 {
		req.PairMethod = pb.PairMethod_PAIR_INITIAL_FONTES
		if currentRound == 0 {
			req.InitialNonperfRounds = int32(min(3, numRounds))
		}
		fmt.Fprintf(logsb, "Auto: round %d < 3, using Initial Fontes\n", currentRound)
		return simplePair(req, logsb)
	}

	req.PairMethod = pb.PairMethod_COP
	fmt.Fprintf(logsb, "Auto: round %d >= 3, using COP\n", currentRound)
	return copMethodPair(req, logsb)
}

func restoreAutoPairReqFields(req *pb.PairRequest, origMethod pb.PairMethod, origInitialNonperfRounds int32) {
	req.PairMethod = origMethod
	req.InitialNonperfRounds = origInitialNonperfRounds
}

func copMinWeightMatching(req *pb.PairRequest, copdata *copdatapkg.PrecompData, factor3ForcedPairings [][2]int, logsb *strings.Builder) ([]int32, *pb.PairResponse) {
	prepairedPlayerIndexes, numForcedByes, prepairedRoundIdx := copdatapkg.ExtractPrepairedPlayers(req)

	for playerIdx, oppIdx := range prepairedPlayerIndexes {
		if oppIdx < playerIdx {
			continue
		}
		prepairedPlayersStr := fmt.Sprintf("Forcing (#%d) %s vs ", playerIdx+1, req.PlayerNames[playerIdx])
		if playerIdx == oppIdx {
			prepairedPlayersStr += "BYE\n"
		} else {
			prepairedPlayersStr += fmt.Sprintf("(#%d) %s\n", oppIdx+1, req.PlayerNames[oppIdx])
		}
		logsb.WriteString(prepairedPlayersStr)
	}

	logsb.WriteString(fmt.Sprintf("\nForcing %d bye(s)\n\n", numForcedByes))

	playerNodes := []int{}
	divisionPlayerData := [][]string{}
	numPlayers := copdata.Standings.GetNumPlayers()
	for playerRankIdx := 0; playerRankIdx < numPlayers; playerRankIdx++ {
		playerIdx := copdata.Standings.GetPlayerIndex(playerRankIdx)
		playerNodes = append(playerNodes, playerIdx)
		divisionPlayerData = append(divisionPlayerData, copdata.Standings.StringDataForPlayer(req, playerRankIdx))
	}

	addBye := (numPlayers-numForcedByes)%2 == 1
	if addBye {
		playerNodes = append(playerNodes, pkgstnd.ByePlayerIndex)
		divisionPlayerData = append(divisionPlayerData, []string{"", "", "BYE", "", ""})
	}

	lowestPossibleAbsCasher := 0
	for playerRankIdx, place := range copdata.HighestRankAbsolutely {
		if place < int(req.PlacePrizes) {
			lowestPossibleAbsCasher = playerRankIdx
		}
	}

	lowestPossibleHopeCasher := copdata.LowestPossibleHopeNth[int(req.PlacePrizes)-1]

	gibsonGetsBye := false
	if addBye {
		for i := 0; i < numPlayers; i++ {
			if playerNodes[i] == pkgstnd.ByePlayerIndex {
				break
			}
			if copdata.GibsonizedPlayers[i] && copdata.GibsonGroups[i] == 0 {
				gibsonGetsBye = true
				break
			}
		}
	}

	pargs := &policyArgs{
		req:                      req,
		copdata:                  copdata,
		playerNodes:              playerNodes,
		lowestPossibleAbsCasher:  lowestPossibleAbsCasher,
		lowestPossibleHopeCasher: lowestPossibleHopeCasher,
		roundsRemaining:          pkgstnd.GetRoundsRemaining(req),
		roundPairingsRemaining:   int(req.Rounds) - copdata.CompletePairings,
		gibsonGetsBye:            gibsonGetsBye,
		prepairedRoundIdx:        prepairedRoundIdx,
		prepairedPlayerIndexes:   prepairedPlayerIndexes,
		factor3ForcedPairings:    factor3ForcedPairings,
	}
	pargs.topDownByePlayer = computeTopDownByePlayer(pargs)
	pargs.forcedContenderByePlayer = computeForcedContenderBye(pargs)
	pargs.forcedLeaderVsThird = computeForcedLeaderVsThird(pargs)
	pargs.disallowedLeaderOpponent = computeDisallowedLeaderOpponent(pargs)
	pargs.top4LockActive = computeTop4LockActive(pargs)

	// Now that a specific bye recipient (if any) is known, correct the
	// contender-group boundary for their removal from the pairing pool - see
	// adjustLowestPossibleHopeCasherForBye. This must run after both
	// computeTopDownByePlayer and computeForcedContenderBye, since they
	// themselves rely on the pre-adjustment boundary to pick a bye recipient.
	pargs.lowestPossibleHopeCasher = adjustLowestPossibleHopeCasherForBye(pargs, playerNodes, numPlayers, logsb)

	logsb.WriteString(fmt.Sprintf("Control Loss Sims: %d\n", req.ControlLossSims))
	logsb.WriteString(fmt.Sprintf("Lowest Hopeful Casher: %s\n", req.PlayerNames[playerNodes[pargs.lowestPossibleHopeCasher]]))
	logsb.WriteString(fmt.Sprintf("Lowest Absolute Casher: %s\n", req.PlayerNames[playerNodes[lowestPossibleAbsCasher]]))
	logsb.WriteString(fmt.Sprintf("Number of Pairings (including prepaired): %d\n", len(req.DivisionPairings)))
	logsb.WriteString(fmt.Sprintf("Number of Results: %d\n", len(req.DivisionResults)))
	logsb.WriteString(fmt.Sprintf("Rounds Remaining: %d\n", pargs.roundsRemaining))
	logsb.WriteString(fmt.Sprintf("Round Pairings Remaining: %d\n", pargs.roundPairingsRemaining))
	logsb.WriteString(fmt.Sprintf("Using Unforced Bye: %t\n", addBye))
	logsb.WriteString(fmt.Sprintf("Gibson Gets Bye: %t\n", pargs.gibsonGetsBye))
	if pargs.forcedContenderByePlayer >= 0 {
		logsb.WriteString(fmt.Sprintf(
			"Forced Contender Bye: %s - every bye-repeat-free player is a hopeful-to-cash contender this round, "+
				"so the bye is forced onto the lowest-ranked contender with the fewest prior byes instead of repeating "+
				"a bye on someone else\n",
			req.PlayerNames[pargs.forcedContenderByePlayer]))
	}
	if pargs.top4LockActive {
		logsb.WriteString(fmt.Sprintf(
			"Top 4 Lock: exactly 4 players are hopeful contenders for 1st (%s, %s, %s, %s) with 2 rounds remaining "+
				"and Factor 3 did not fire - all 4 are barred from playing outside this group this round\n",
			req.PlayerNames[playerNodes[0]], req.PlayerNames[playerNodes[1]],
			req.PlayerNames[playerNodes[2]], req.PlayerNames[playerNodes[3]]))
	}
	if pargs.forcedLeaderVsThird >= 0 {
		logsb.WriteString(fmt.Sprintf(
			"Forced Leader vs 3rd: at most 3 players are hopeful for 1st, and %s is receiving this round's bye, "+
				"leaving %s without their strongest hopeful opponent - forcing %s to play %s instead of barring "+
				"the pairing\n",
			req.PlayerNames[playerNodes[1]], req.PlayerNames[playerNodes[0]], req.PlayerNames[playerNodes[0]],
			req.PlayerNames[pargs.forcedLeaderVsThird]))
	}
	logsb.WriteString(fmt.Sprintf("Prepaired Round (0 for none): %d\n", pargs.prepairedRoundIdx+1))
	logsb.WriteString("Destinys Child: ")
	if copdata.DestinysChild >= 0 {
		logsb.WriteString(req.PlayerNames[playerNodes[copdata.DestinysChild]])
	} else {
		logsb.WriteString("(none)")
	}
	logsb.WriteString("\n\n")

	numPlayerNodes := len(playerNodes)

	disallowedPairs := map[string]string{}
	for _, cPol := range constraintPolicies {
		forced, disallowed := cPol.handler(pargs)
		for _, dp := range disallowed {
			setDisallowPairs(disallowedPairs, dp[0], dp[1], cPol.name)
		}
		for _, fp := range forced {
			for pri := 0; pri < numPlayerNodes; pri++ {
				for prj := pri + 1; prj < numPlayerNodes; prj++ {
					pi := pargs.playerNodes[pri]
					pj := pargs.playerNodes[prj]
					if (fp[0] == pi && fp[1] == pj) || (fp[1] == pi && fp[0] == pj) {
						continue
					}
					if fp[0] == pi || fp[1] == pi || fp[0] == pj || fp[1] == pj {
						setDisallowPairs(disallowedPairs, pi, pj, cPol.name)
					}
				}
			}
		}
	}

	matchupHeader := []string{"Player", "W", "S", "Player", "W", "S"}
	pairingDetailsheader := append(matchupHeader, []string{"S", "C", "PTP", "Total"}...)
	for _, weightPolicy := range weightPolicies {
		pairingDetailsheader = append(pairingDetailsheader, weightPolicy.name)
	}
	numColums := len(pairingDetailsheader)

	var pairings []int
	var totalWeight int64
	var pairingDetails [][]string
	var pairingsToDetailsIndex map[string]int

	retried := false
	for {
		edges := []*matching.Edge{}
		edgeWeights := map[string]int64{}
		pcEdgeWeights := map[string]int64{}
		pairingDetails = [][]string{}
		pairingsToDetailsIndex = map[string]int{}

		for rankIdxI := 0; rankIdxI < numPlayerNodes; rankIdxI++ {
			for rankIdxJ := rankIdxI + 1; rankIdxJ < numPlayerNodes; rankIdxJ++ {
				pairingDataRow := getMatchupStrArray(divisionPlayerData, rankIdxI, rankIdxJ, req.PlayerClasses)
				pairKey := copdatapkg.GetPairingKey(playerNodes[rankIdxI], playerNodes[rankIdxJ])
				disallowReason, disallowPair := disallowedPairs[pairKey]
				// Pairing selected bool placeholder
				pairingDataRow = append(pairingDataRow, "")
				if disallowPair {
					pairingDataRow = append(pairingDataRow, disallowReason)
					emptyColsToAdd := numColums - len(pairingDataRow)
					for i := 0; i < emptyColsToAdd; i++ {
						pairingDataRow = append(pairingDataRow, "")
					}
				} else {
					// No disallow reason
					pairingDataRow = append(pairingDataRow, "")
					// Add the number of repeats for convenience
					pairingDataRow = append(pairingDataRow, fmt.Sprintf("%d", copdata.PairingCounts[pairKey]))
					// Placeholder for total weight
					pairingDataRow = append(pairingDataRow, "")
					weightSum := int64(0)
					for _, weightPolicy := range weightPolicies {
						weight := weightPolicy.handler(pargs, rankIdxI, rankIdxJ)
						weightSum += weight
						pairingDataRow = append(pairingDataRow, fmt.Sprintf("%d", weight))
						if weightPolicy.name == "PC" {
							rankKey := getRankPairingKey(rankIdxI, rankIdxJ)
							pcEdgeWeights[rankKey] = weight
						}
					}
					pairingDataRow[len(matchupHeader)+3] = fmt.Sprintf("%d", weightSum)
					rankKey := getRankPairingKey(rankIdxI, rankIdxJ)
					edgeWeights[rankKey] = weightSum
					edges = append(edges, matching.NewEdge(rankIdxI, rankIdxJ, weightSum))
				}
				pairingsToDetailsIndex[getRankPairingKey(rankIdxI, rankIdxJ)] = len(pairingDetails)
				pairingDetails = append(pairingDetails, pairingDataRow)
			}
			if rankIdxI < numPlayerNodes-2 {
				spacingRow := make([]string, numColums)
				pairingDetails = append(pairingDetails, spacingRow)
			}
		}

		var err error
		pairings, totalWeight, err = matching.MinWeightMatching(edges, true)
		if err != nil {
			return nil, &pb.PairResponse{
				ErrorCode:    pb.PairError_MIN_WEIGHT_MATCHING,
				ErrorMessage: fmt.Sprintf("min weight matching error: %s\n", err.Error()),
			}
		}

		if addBye {
			pairings = pairings[:len(pairings)-1]
		}

		if len(pairings) > numPlayers {
			return nil, &pb.PairResponse{
				ErrorCode:    pb.PairError_INVALID_PAIRINGS_LENGTH,
				ErrorMessage: fmt.Sprintf("invalid pairings length %d for %d players", len(pairings), numPlayers),
			}
		} else if len(pairings) < numPlayers {
			numUnpairedAtBottom := numPlayers - len(pairings)
			unpairedIndexes := make([]int, numUnpairedAtBottom)
			for i := range unpairedIndexes {
				unpairedIndexes[i] = -1
			}
			pairings = append(pairings, unpairedIndexes...)
		}

		// For each selected edge with weight >= majorPenalty, expand the PC contender
		// group for both players by incrementing LowestPossibleHopeNth, then retry once.
		// lowestPossibleHopeCasher (the global gate) is intentionally not updated.
		if retried {
			break
		}
		expanded := false
		numStandingsPlayers := copdata.Standings.GetNumPlayers()
		lowestHopeOverride := map[int]int{}
		for playerRankIdx, oppRankIdx := range pairings {
			if oppRankIdx <= playerRankIdx {
				continue
			}
			rankKey := getRankPairingKey(playerRankIdx, oppRankIdx)
			if pcEdgeWeights[rankKey] >= majorPenalty {
				nameForNode := func(rankIdx int) string {
					pi := playerNodes[rankIdx]
					if pi == pkgstnd.ByePlayerIndex {
						return "BYE"
					}
					return req.PlayerNames[pi]
				}
				logsb.WriteString(fmt.Sprintf("Retry: majorPenalty edge %s vs %s (weight %d)\n",
					nameForNode(playerRankIdx),
					nameForNode(oppRankIdx),
					edgeWeights[rankKey]))
				if playerRankIdx < len(copdata.LowestPossibleHopeNth) {
					current := copdata.LowestPossibleHopeNth[playerRankIdx]
					if override, ok := lowestHopeOverride[playerRankIdx]; ok {
						current = override
					}
					if current+1 < numStandingsPlayers {
						lowestHopeOverride[playerRankIdx] = current + 1
						expanded = true
					}
				}
			}
		}
		if !expanded {
			break
		}
		pargs.lowestHopeOverride = lowestHopeOverride
		retried = true
	}

	for playerRankIdx, oppRankIdx := range pairings {
		if oppRankIdx < playerRankIdx {
			continue
		}
		pairingDetails[pairingsToDetailsIndex[getRankPairingKey(playerRankIdx, oppRankIdx)]][len(matchupHeader)] = "*"
	}

	copdatapkg.WriteStringDataToLog("Pairing Weights", pairingDetailsheader, pairingDetails, logsb)

	pairingsLogMx := [][]string{}
	for playerRankIdx := 0; playerRankIdx < len(pairings); playerRankIdx++ {
		oppRankIdx := pairings[playerRankIdx]
		if oppRankIdx < playerRankIdx {
			continue
		}
		pairingsLogMxRow := getMatchupStrArray(divisionPlayerData, playerRankIdx, oppRankIdx, req.PlayerClasses)
		playerIdx := playerNodes[playerRankIdx]
		oppIdx := playerNodes[oppRankIdx]
		pairingsLogMxRow = append(pairingsLogMxRow, fmt.Sprintf("%d", copdata.PairingCounts[copdatapkg.GetPairingKey(playerIdx, oppIdx)]))
		pairingsLogMx = append(pairingsLogMx, pairingsLogMxRow)
	}

	copdatapkg.WriteStringDataToLog("Final COP Pairings", append(matchupHeader, []string{"Previous Times Played"}...), pairingsLogMx, logsb)

	logsb.WriteString(fmt.Sprintf("Total Weight: %d\n", totalWeight))

	allPlayerPairings := make([]int32, req.AllPlayers)
	for i := 0; i < int(req.AllPlayers); i++ {
		allPlayerPairings[i] = -1
	}
	unpairedPlayerIndexes := []int{}
	prepairedPlayersStr := ""
	// Convert rank indexes to player indexes and convert the bye format from ByePlayerIndex to player index
	for playerRankIdx := 0; playerRankIdx < len(pairings); playerRankIdx++ {
		oppRankIdx := pairings[playerRankIdx]
		playerIdx := playerNodes[playerRankIdx]
		prepairedOppIdx, playerIsPrepaired := prepairedPlayerIndexes[playerIdx]
		if oppRankIdx < 0 {
			if !playerIsPrepaired {
				unpairedPlayerIndexes = append(unpairedPlayerIndexes, playerIdx)
				continue
			}
			allPlayerPairings[playerIdx] = int32(prepairedOppIdx)
			if playerIdx <= prepairedOppIdx {
				prepairedPlayersStr += fmt.Sprintf("(#%d) %s vs ", playerIdx+1, req.PlayerNames[playerIdx])
				if playerIdx == prepairedOppIdx {
					prepairedPlayersStr += "BYE\n"
				} else {
					prepairedPlayersStr += fmt.Sprintf("(#%d) %s\n", prepairedOppIdx+1, req.PlayerNames[prepairedOppIdx])
				}
			}
		} else if playerIsPrepaired {
			return nil, &pb.PairResponse{
				ErrorCode:    pb.PairError_OVERCONSTRAINED,
				ErrorMessage: fmt.Sprintf("player %s is prepaired but was still paired by COP", req.PlayerNames[playerIdx]),
			}
		} else {
			oppIdx := playerNodes[oppRankIdx]
			if oppIdx == pkgstnd.ByePlayerIndex {
				oppIdx = playerIdx
			}
			allPlayerPairings[playerIdx] = int32(oppIdx)
		}
	}

	if prepairedPlayersStr != "" {
		logsb.WriteString(fmt.Sprintf("\nPrepaired players:\n\n%s", prepairedPlayersStr))
	}

	removedPlayersStr := ""
	for _, removedPlayerIdx := range req.RemovedPlayers {
		if allPlayerPairings[removedPlayerIdx] != -1 {
			return nil, &pb.PairResponse{
				ErrorCode:    pb.PairError_OVERCONSTRAINED,
				ErrorMessage: fmt.Sprintf("player %s was removed but was still paired by COP", req.PlayerNames[removedPlayerIdx]),
			}
		}
		removedPlayersStr += fmt.Sprintf("(#%d) %s\n", removedPlayerIdx+1, req.PlayerNames[removedPlayerIdx])
	}

	if removedPlayersStr != "" {
		logsb.WriteString(fmt.Sprintf("\nRemoved players:\n\n%s\n", removedPlayersStr))
	}

	numUnpairedPlayers := len(unpairedPlayerIndexes)
	if numUnpairedPlayers > 0 {
		msg := "COP pairings could not be completed because there were too many constraints. The unpaired players are:\n\n"
		for _, unpairedPlayerIdx := range unpairedPlayerIndexes {
			msg += fmt.Sprintf("%s\n", req.PlayerNames[unpairedPlayerIdx])
		}
		return nil, &pb.PairResponse{
			ErrorCode:    pb.PairError_OVERCONSTRAINED,
			ErrorMessage: msg,
		}
	}

	return allPlayerPairings, nil
}

// classLetter renders a player's class (0-indexed: 0 is the top, unprized
// class) as a letter - 0 is "A", 1 is "B", and so on - or "" if playerIdx is
// out of range (e.g. the request has no PlayerClasses at all).
func classLetter(playerClasses []int32, playerIdx int) string {
	if playerIdx < 0 || playerIdx >= len(playerClasses) {
		return ""
	}
	return string(rune('A') + rune(playerClasses[playerIdx]))
}

func getPlayerRecordStrArray(playerData []string, playerClasses []int32) []string {
	if playerData[2] == byePlayerName {
		return []string{byePlayerName, "", ""}
	}
	// playerData[1] is the player's 1-indexed number (see
	// standings.StringDataForPlayer); reuse it to look up their class
	// rather than threading a separate player index through every caller.
	classSuffix := ""
	if playerNum, err := strconv.Atoi(playerData[1]); err == nil {
		if letter := classLetter(playerClasses, playerNum-1); letter != "" {
			classSuffix = "/" + letter
		}
	}
	return []string{fmt.Sprintf("%s (#%s%s) %s", playerData[0], playerData[1], classSuffix, playerData[2]), playerData[3], playerData[4]}
}

func getMatchupStrArray(divisionPlayerData [][]string, i int, j int, playerClasses []int32) []string {
	return append(getPlayerRecordStrArray(divisionPlayerData[i], playerClasses), getPlayerRecordStrArray(divisionPlayerData[j], playerClasses)...)
}

func setDisallowPairs(disallowedPairs map[string]string, playerIdx int, oppIdx int, policyName string) {
	key := copdatapkg.GetPairingKey(playerIdx, oppIdx)
	if _, exists := disallowedPairs[key]; !exists {
		disallowedPairs[key] = policyName
	}
}

func getRankPairingKey(playerRankIdx int, oppRankIdx int) string {
	return fmt.Sprintf("%d:%d", playerRankIdx, oppRankIdx)
}
