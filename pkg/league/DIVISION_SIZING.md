# Division Sizing Guidelines

## Size Constraints

### Divisions (1, 2, 3, ...)

- **Minimum**: 13 players (`MinRegularDivisionSize`)
  - Note: This can be relaxed if the league has collapsed and there aren't enough players
- **Target**: 15 players (`TargetRegularDivisionSize`)
- **Maximum**: 20 players (`MaxRegularDivisionSize`)

## Divisions

Divisions are numbered 1, 2, 3, ... (lower numbers = higher skill level).

**Creation:**
- Created during season preparation based on total player count
- Should aim for 13-15 players per division
- Can have up to 20 players if needed for balancing
- Minimum of 13 can be relaxed if the league has too few players

**Population:**
- All players (returning and new) are mixed together and assigned to divisions through the rebalancing algorithm
- New players receive the lowest priority (-50,000) ensuring they naturally settle into lower divisions
- Returning players are assigned virtual divisions based on their previous season outcomes (promoted/relegated/stayed)
- The rebalancing algorithm distributes players across divisions respecting priority levels (see Priority System section below)

## Enforcement

**During Season Preparation:**
- Divisions are created based on total registered player count
- Target: round(playerCount / idealDivisionSize) divisions
- All players assigned through rebalancing algorithm

**During Rebalancing:**
- Final regular division sizes should fall within 13-20 range (or whatever is possible if league has collapsed)
- Rebalancing logic moves players between divisions to achieve proper sizes
- Priority system prevents double-relegation and protects stayed players

## Division Rebalancing

After initial placement, divisions may be unbalanced (too many or too few players). The rebalancing phase adjusts division sizes to fit within the 13-20 player range while respecting player placement priorities.

### Size Targets
- **Minimum**: 13 players per division
- **Maximum**: 20 players per division
- Divisions outside this range trigger rebalancing

### Priority System

Players have different priorities for movement based on how they were placed. Lower priority = harder to move.

**Priority Levels (highest priority score = hardest to move):**
1. **STAYED** (500,000) - Players who stayed in their division
   - Never relegated (cannot move down)
   - Can be promoted if needed to fill undersized division above
2. **PROMOTED** (400,000) - Players promoted from lower division
   - Cannot be promoted again (no double-promotion)
   - Can be relegated if needed
3. **RELEGATED** (300,000) - Players relegated from higher division
   - Cannot be relegated again (no double-relegation)
   - Can be promoted if needed
4. **SHORT_HIATUS_RETURNING** (5,000) - Players returning after 1-3 seasons away
   - Flexible movement in any direction
5. **LONG_HIATUS_RETURNING** (5,000) - Players returning after 4+ seasons away
   - Flexible movement in any direction
6. **NEW** (-50,000) - Brand new players
   - Lowest priority ensures they settle into lower divisions
   - Flexible movement in any direction

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
- SHORT_HIATUS_RETURNING (priority 4)
- LONG_HIATUS_RETURNING (priority 5)
- NEW (priority 6 - lowest)

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
- Div 2: 25 players (10 STAYED, 5 PROMOTED, 10 NEW)
- Div 3: Empty

Result:
- Move 5 NEW players from Div 2 → Div 3
- Final: Div 1=15, Div 2=20, Div 3=5 (Div 3 still undersized, but best effort)

**Example 2: Undersized Division 1**
- Div 1: 10 players (STAYED)
- Div 2: 20 players (NEW, HIATUS_RETURNING)

Result:
- Move 3 flexible players from Div 2 → Div 1
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
