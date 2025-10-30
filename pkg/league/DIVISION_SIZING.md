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

## Enforcement

**During Creation:**
- Division sizing is enforced when creating rookie divisions
- Regular divisions are created manually by administrators

**During Placement:**
- Returning players: Placed without size checks (they go back to their division)
- Rookies into regular divisions: Added without size checks
- Rookies into rookie divisions: Strict size enforcement (10-20 range)

**During Rebalancing (Future Phase):**
- Final regular division sizes should fall within 13-20 range (or whatever is possible if league has collapsed)
- Final rookie division sizes must fall within 10-20 range
- Rebalancing logic will move players between divisions to achieve proper sizes
- Priority system prevents double-relegation and protects stayed players
