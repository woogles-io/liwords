# Division Sizing Guidelines

## Size Constraints

### Regular Divisions (1, 2, 3, ...)

- **Minimum**: 13 players (`MinRegularDivisionSize`)
  - Note: This can be relaxed if the league has collapsed and there aren't enough players
- **Target**: 15 players (`TargetRegularDivisionSize`)
- **Maximum**: 20 players (`MaxRegularDivisionSize`)

### Rookie Divisions (100, 101, 102, ...)

- **Minimum**: 10 players (`MinRookieDivisionSize`)
- **Target**: 15 players (`TargetRookieDivisionSize`)
- **Maximum**: 20 players (`MaxRookieDivisionSize`)

## Regular Divisions

Regular divisions are numbered 1, 2, 3, ... (lower numbers = higher skill level).

**Creation:**
- Created manually when setting up a new season
- Should aim for 13-15 players per division
- Can have up to 20 players if needed for balancing
- Minimum of 13 can be relaxed if the league has too few players

**Population:**
- First: Returning players are placed back into their previous divisions
- Second: Rookies (< 10 total) are split by rating into bottom 2 divisions
- Third: Rebalancing occurs to ensure proper distribution

## Rookie Divisions

Rookie divisions are numbered 100, 101, 102, ... to distinguish them from regular divisions.

**When Created:**
- Only when there are ≥ 10 new players (`MinPlayersForRookieDivision`)
- If < 10 new players, they're placed into existing regular divisions instead

**Sizing Algorithm:**
- **10-20 rookies**: Create 1 division
- **21+ rookies**: Create multiple divisions
  - Target: 15 players per division
  - Minimum: 10 players per division
  - Maximum: 20 players per division
  - Algorithm prefers larger divisions over smaller ones

**Examples:**
- 10 rookies → [10] (1 division)
- 16 rookies → [16] (1 division, within max)
- 20 rookies → [20] (1 division at max)
- 21 rookies → [11, 10] (2 divisions)
- 25 rookies → [13, 12] (2 balanced divisions)
- 30 rookies → [15, 15] (2 divisions at target)
- 40 rookies → [14, 13, 13] (3 balanced divisions)
- 60 rookies → [15, 15, 15, 15] (4 divisions at target)

## Rookie Graduation

When a season with rookie divisions ends, rookies are graduated into regular divisions for the next season.

**Graduation Formula:**
```
groupSize = ceil(numRookies / 6)
numGroups = ceil(numRookies / groupSize)
startingDivision = max(2, highestRegularDivision - numGroups + 1)
```

**Special case:** If only 1 regular division exists, all rookies go to Division 1.

**Distribution:**
- Rookies are sorted by their final standing (rank 1 = best performer)
- Divided into groups of size `groupSize`
- Top group goes to `startingDivision`, next group to `startingDivision+1`, etc.
- Division 1 is skipped unless it's the only division
- If more groups than divisions, multiple groups go to the lowest divisions

**Examples:**
- **19 rookies, 12 divisions**: groupSize=4, 5 groups → Divs [8,9,10,11,12]
- **15 rookies, 5 divisions**: groupSize=3, 5 groups → Divs [2,3,4,5,5] (skip Div 1)
- **20 rookies, 1 division**: All 20 → Div 1 (no choice)
- **12 rookies, 2 divisions**: groupSize=2, 6 groups → All to Div 2
- **100 rookies, 3 divisions**: groupSize=17, 6 groups → Divs [2,2,2,3,3,3] (overflow)

**Key points:**
- ALL rookies graduate (no one repeats rookie division)
- Top performers go to higher divisions (lower numbers)
- Division 1 is protected from rookie influx (unless it's the only division)
- Rebalancing phase handles overflow cases

## Enforcement

**During Creation:**
- Division sizing is enforced when creating rookie divisions
- Regular divisions are created manually by administrators

**During Placement:**
- Returning players: Placed without size checks (they go back to their division)
- Rookies into regular divisions: Added without size checks
- Rookies into rookie divisions: Strict size enforcement (10-20 range)

**During Graduation:**
- All rookies from rookie divisions (100+) are graduated
- Placed into regular divisions based on final standing
- May cause divisions to exceed max size (rebalancing fixes this)

**During Rebalancing:**
- Final regular division sizes should fall within 13-20 range (or whatever is possible if league has collapsed)
- Rebalancing logic moves players between divisions to achieve proper sizes
- Priority system prevents double-relegation and protects stayed players

## Division Rebalancing

After initial placement and graduation, divisions may be unbalanced (too many or too few players). The rebalancing phase adjusts division sizes to fit within the 13-20 player range while respecting player placement priorities.

### Size Targets
- **Minimum**: 13 players per division
- **Maximum**: 20 players per division
- Divisions outside this range trigger rebalancing

### Priority System

Players have different priorities for movement based on how they were placed. Lower priority = harder to move.

**Priority Levels (1 = hardest to move, 6 = easiest):**
1. **STAYED** - Players who stayed in their division
   - Never relegated (cannot move down)
   - Can be promoted if needed to fill undersized division above
2. **PROMOTED** - Players promoted from lower division
   - Cannot be promoted again (no double-promotion)
   - Can be relegated if needed
3. **RELEGATED** - Players relegated from higher division
   - Cannot be relegated again (no double-relegation)
   - Can be promoted if needed
4. **GRADUATED** - Rookies graduated from rookie divisions (100+)
   - Flexible movement in any direction
5. **SHORT_HIATUS_RETURNING** - Players returning after 1-3 seasons away
   - Flexible movement in any direction
5. **NEW** - First-time players placed directly in regular divisions
   - Only happens when <10 total rookies (no rookie division created)
   - Flexible movement in any direction
   - Same priority as SHORT_HIATUS_RETURNING
6. **LONG_HIATUS_RETURNING** - Players returning after 4+ seasons away
   - Most flexible, easiest to move

**Note on NEW players:**
- If ≥10 rookies → They go to rookie divisions, then later GRADUATE to regular divisions
- If <10 rookies → They're placed directly in bottom 2 regular divisions with status NEW
- NEW status only applies to the direct placement scenario

### Rebalancing Algorithm

**Iterative Greedy Approach** (max 10 iterations):

1. **Build division states** - Get current player counts and statuses
2. **Check if balanced** - All divisions in 13-20 range?
   - If yes: Done ✓
   - If no: Continue to step 3
3. **Fix oversized divisions** (>20 players)
   - For each division with >20 players:
     - Move lowest-priority players to division below
     - Stop when at 20 players or no more moveable players
     - Constraint: Cannot move RELEGATED or STAYED players down
4. **Fix undersized divisions** (<13 players)
   - For each division with <13 players:
     - Try to pull lowest-priority players from division above
     - If still undersized, promote top STAYED players from division below
     - Stop when at 13 players or no more moveable players
     - Constraint: Cannot move PROMOTED players up (from above)
5. **Check for convergence**
   - If no moves were made: Stop (converged)
   - If moves were made: Return to step 1
6. **Maximum iterations** - Stop after 10 iterations even if not balanced

### Movement Constraints

**Cannot move down (relegate):**
- STAYED players (priority 1) - never relegate
- RELEGATED players (priority 3) - no double-relegation

**Cannot move up (promote):**
- PROMOTED players (priority 2) - no double-promotion

**Can move in either direction:**
- GRADUATED (priority 4)
- SHORT_HIATUS_RETURNING (priority 5)
- LONG_HIATUS_RETURNING (priority 6)
- NEW (priority 5)

### Rank-Based Selection

When choosing which players to move within the same priority:
- **Moving down**: Select players with worse previous_division_rank (higher rank numbers)
- **Moving up**: Select players with better previous_division_rank (lower rank numbers)
- Top STAYED players can be promoted to fill undersized divisions above

### Edge Cases

**Oversized lowest division:**
- Cannot move players down (no division below)
- Warning logged, division remains oversized
- Constraint: RELEGATED/STAYED players block movement

**Undersized highest division:**
- Cannot pull from above (no division above)
- Try to promote top STAYED players from below
- If still undersized, warning logged

**Collapsed league (only Division 1):**
- Cannot move players anywhere
- Division may remain oversized or undersized
- Warning logged

**All players have hard constraints:**
- Example: Oversized division with only STAYED/RELEGATED players
- Cannot move anyone down
- Division remains oversized, warning logged

**Non-convergence:**
- After 10 iterations, stop even if not balanced
- Return best-effort result with warnings

### Examples

**Example 1: Oversized Division 2**
- Div 1: 15 players (STAYED)
- Div 2: 25 players (10 STAYED, 5 PROMOTED, 10 GRADUATED)
- Div 3: Empty

Result:
- Move 5 GRADUATED players from Div 2 → Div 3
- Final: Div 1=15, Div 2=20, Div 3=5 (Div 3 still undersized, but best effort)

**Example 2: Undersized Division 1**
- Div 1: 10 players (STAYED)
- Div 2: 20 players (GRADUATED)

Result:
- Move 3 GRADUATED players from Div 2 → Div 1
- Final: Div 1=13, Div 2=17

**Example 3: Double-Relegation Prevention**
- Div 1: 25 players (all RELEGATED)
- Div 2: 15 players

Result:
- Cannot move RELEGATED players down
- Div 1 remains at 25 players (warning logged)

**Example 4: Promote Top STAYED**
- Div 1: 5 players (STAYED)
- Div 2: 20 players (15 PROMOTED, 5 STAYED with ranks 1-5)

Result:
- Cannot promote PROMOTED players
- Promote 5 top STAYED players from Div 2 → Div 1
- Final: Div 1=10 (still undersized, warning), Div 2=15

### Implementation Notes

- Rebalancing is **best-effort** - never fails, only warns
- Database updates happen immediately during moves
- Each move updates the player's division_id
- placement_status is preserved (not changed by rebalancing)
- Warnings are collected and returned in RebalanceResult
- All moves are logged in MovedPlayers list for audit trail
