# Tile Inventory Architecture Refactoring Plan

## Problem Statement

The current tile management code is overly complex, with numerous special cases, distributed responsibilities, and opportunities for bugs. Managing tiles across bag/racks/board for a Scrabble annotator should be simpler than it currently is.

## Root Causes of Complexity

### 1. Mutable State + Distributed Responsibility
- Tiles move between bag/racks/board through many different code paths
- Multiple functions manipulate tiles: `AssignRacks`, `ValidatedPutBack`, `playTilePlacementMove`, `ApplyEventInEditorMode`, `EditOldRack`
- No single source of truth for "how do I move tiles safely?"
- Each code path must remember to validate invariants

### 2. Imperative "Put Back Then Remove" Pattern
Current approach uses many sequential steps:
1. Put player's rack back in bag
2. Remove tiles from bag for the play
3. Put opponent's rack back in bag (if needed)
4. Try removing from bag again
5. Draw replacement tiles

This is like doing accounting by physically moving coins around rather than updating ledgers. Each step can fail and needs rollback logic.

### 3. Validation After The Fact
- Operations execute first, then check if valid
- `ValidatedPutBack` checks "would this create too many?" but had to count tiles inconsistently (bag+board but not racks)
- Rollback complexity when validation fails mid-operation
- Clone-and-swap added to prevent corruption, but doesn't address root cause

### 4. Special Cases Everywhere
- Annotated vs regular games
- Amendments vs new events
- Rack inference vs explicit racks
- Through-tiles vs regular tiles
- Designated blanks (high bit set)
- Partial racks, empty racks, nil racks
- Borrowing from opponent
- Each combination needs careful handling

### 5. Reactive vs Proactive Invariant Enforcement
The code enforces tile conservation (core invariant) **reactively**:
- Do operation → check if valid → rollback if not

Instead of **proactively**:
- Validate before operation → make invalid states impossible

## Proposed Solutions

### 1. TileInventory Manager (Biggest Win)

Create ONE struct responsible for ALL tile movements. No other code touches `gdoc.Bag`, `gdoc.Racks`, or `gdoc.Board` directly.

```go
type TileInventory struct {
    gdoc *ipc.GameDocument
    dist *tilemapping.LetterDistribution
}

// Declarative operations - specify WHAT you want, not HOW to get there
func (inv *TileInventory) SetRacks(racks [][]byte) error {
    // Calculate what bag SHOULD be given board + new racks
    tilesOnBoard := inv.countBoardTiles()
    tilesInRacks := inv.countTilesInRacks(racks)

    // Validate BEFORE changing anything
    if err := inv.validateDistribution(tilesOnBoard, tilesInRacks); err != nil {
        return err
    }

    // Calculate remaining tiles for bag (deterministic)
    remainingTiles := inv.computeRemainingTiles(tilesOnBoard, tilesInRacks)

    // Atomic update - all or nothing
    inv.gdoc.Racks = racks
    inv.gdoc.Bag.Tiles = remainingTiles

    return nil
}

func (inv *TileInventory) PlayTiles(playerIdx int, tiles []Tile, position BoardPosition) error {
    // Compute new state declaratively
    newBoard := inv.board.withTilesAt(position, tiles)
    tilesFromRack := inv.filterOutThroughTiles(tiles)
    newRack := inv.removeFromRack(playerIdx, tilesFromRack).drawTiles(len(tilesFromRack))

    // Validate entire state
    if err := inv.validateWithState(newBoard, newRack); err != nil {
        return err
    }

    // Apply atomically
    inv.gdoc.Board = newBoard
    inv.gdoc.Racks[playerIdx] = newRack
    return nil
}

// ONE function that checks ALL invariants
func (inv *TileInventory) ValidateInvariants() error {
    // Count everything once, correctly
    counts := make(map[byte]int)

    // Count bag tiles
    for _, t := range inv.gdoc.Bag.Tiles {
        counts[t]++
    }

    // Count rack tiles
    for _, rack := range inv.gdoc.Racks {
        for _, t := range rack {
            counts[t]++
        }
    }

    // Count board tiles (handle designated blanks)
    for _, tile := range inv.gdoc.Board.Tiles {
        if tile == 0 {
            continue // Empty square
        }
        if tile&0x80 != 0 {
            counts[0]++ // Designated blank counts as blank
        } else {
            counts[tile]++ // Regular tile
        }
    }

    // Check against expected distribution
    for tile, count := range counts {
        expected := int(inv.dist.Distribution()[tilemapping.MachineLetter(tile)])
        if count != expected {
            return fmt.Errorf("tile %d: have %d, expected %d", tile, count, expected)
        }
    }

    // Also check for missing tiles
    for ml, expectedCount := range inv.dist.Distribution() {
        if expectedCount > 0 {
            actualCount := counts[byte(ml)]
            if actualCount != int(expectedCount) {
                return fmt.Errorf("tile %d: have %d, expected %d", ml, actualCount, expectedCount)
            }
        }
    }

    return nil
}
```

**Benefits:**
- Single place for all tile logic - easier to understand, test, and debug
- Invariants enforced automatically after every operation
- Declarative API (say WHAT you want) vs imperative (HOW to do it)
- Can't bypass validation - all tile operations go through this class
- Easy to add comprehensive logging/debugging in one place

### 2. Immutable Amendment Operations (Already Started!)

Continue the pattern of working on clones:

```go
func handleAmendment(...) error {
    // Clone at START - work on copy
    gdocClone := proto.Clone(g.GameDocument).(*ipc.GameDocument)

    // Apply ALL changes to clone
    // Every operation here can fail without corrupting original
    if err := replayEventsUpToAmendment(gdocClone, ...); err != nil {
        return err
    }
    if err := applyAmendedEvent(gdocClone, ...); err != nil {
        return err
    }
    if err := reapplySubsequentEvents(gdocClone, ...); err != nil {
        // Truncate clone at failure point - still safe
    }

    // Only if we get here, atomically swap
    g.GameDocument = gdocClone
    return nil
}
```

No rollback logic needed - just don't use the clone if anything fails.

**Benefits:**
- Eliminates rollback complexity
- Can't accidentally save partial state
- Easier to reason about - either everything succeeds or nothing changes
- Already implemented for amendments - extend pattern to other operations

### 3. Eliminate Special Cases with Polymorphism

Instead of if/else chains for different game types and scenarios:

```go
// Current approach - many nested conditions
if gdoc.Type == ipc.GameType_ANNOTATED {
    if len(rack) == 0 {
        // infer rack
    } else if !matchesInferredRack {
        if canBorrowFromOpponent {
            // borrow logic
        } else {
            // error
        }
    }
}
```

Use interfaces and polymorphism:

```go
// Single interface for "how do I get tiles for this play?"
type TileSource interface {
    GetTilesForPlay(play *Play, currentRack []byte) ([]byte, error)
}

type ExplicitRackSource struct {
    gdoc *ipc.GameDocument
}
func (s *ExplicitRackSource) GetTilesForPlay(play *Play, currentRack []byte) ([]byte, error) {
    // Must use exact rack - no inference
    if !play.UsesSubsetOf(currentRack) {
        return nil, errors.New("rack doesn't contain these tiles")
    }
    return currentRack, nil
}

type InferredRackSource struct {
    gdoc *ipc.GameDocument
    allowBorrow bool
}
func (s *InferredRackSource) GetTilesForPlay(play *Play, currentRack []byte) ([]byte, error) {
    inferredRack := play.InferRequiredTiles()

    // Try with current rack first
    if currentRack != nil && play.UsesSubsetOf(currentRack) {
        return currentRack, nil
    }

    // Infer from play
    if s.allowBorrow {
        return inferredRack, nil // TileInventory will handle borrowing
    }

    return inferredRack, nil
}

// Factory pattern for game type
func NewTileSource(gameType ipc.GameType) TileSource {
    switch gameType {
    case ipc.GameType_ANNOTATED:
        return &InferredRackSource{allowBorrow: true}
    default:
        return &ExplicitRackSource{}
    }
}

// Usage - no special case logic in main code
source := NewTileSource(gdoc.Type)
rack, err := source.GetTilesForPlay(play, gdoc.Racks[playerIdx])
```

**Benefits:**
- Special cases isolated in separate implementations
- Main code path is clean and simple
- Easy to add new game types or tile sources
- Each implementation can be tested independently

### 4. Make Invalid States Unrepresentable

Current problem: `gdoc.Bag`, `gdoc.Racks`, `gdoc.Board` can be in ANY state, including invalid ones.

Better approach using type system:

```go
type ValidatedGameState struct {
    board *ipc.Board
    racks [][]byte
    bag   *ipc.Bag
    dist  *tilemapping.LetterDistribution
    // Private fields - can only be created/modified through validated constructors
}

// Only way to create a ValidatedGameState - validation is required
func NewValidatedGameState(board *ipc.Board, racks [][]byte, bag *ipc.Bag,
                          dist *tilemapping.LetterDistribution) (*ValidatedGameState, error) {
    s := &ValidatedGameState{
        board: board,
        racks: racks,
        bag:   bag,
        dist:  dist,
    }

    if err := validateInvariants(s); err != nil {
        return nil, err
    }

    return s, nil
}

// All mutations return new validated state
func (s *ValidatedGameState) WithTilesPlayed(playerIdx int, tiles []Tile,
                                             pos Position) (*ValidatedGameState, error) {
    newBoard := s.board.withTilesAt(pos, tiles)
    newRacks := make([][]byte, len(s.racks))
    copy(newRacks, s.racks)
    newRacks[playerIdx] = removeAndDraw(s.racks[playerIdx], tiles, s.bag)

    // Validation happens in constructor
    return NewValidatedGameState(newBoard, newRacks, s.bag, s.dist)
}

// Access is read-only
func (s *ValidatedGameState) GetRack(playerIdx int) []byte {
    rack := make([]byte, len(s.racks[playerIdx]))
    copy(rack, s.racks[playerIdx])
    return rack
}
```

Now it's **impossible** to have an invalid state - the type system enforces validation.

**Benefits:**
- Invalid states can't exist - caught at compile time
- No need to remember to validate - it's automatic
- Immutability by default - no accidental modifications
- Thread-safe if needed

### 5. Separate Read and Write Paths

Make it clear which code reads vs modifies tile state:

```go
// Reading is simple - just access the proto (read-only operations)
func GetRack(gdoc *ipc.GameDocument, playerIdx int) []byte {
    return gdoc.Racks[playerIdx]
}

func GetBagSize(gdoc *ipc.GameDocument) int {
    return len(gdoc.Bag.Tiles)
}

// Writing ONLY goes through TileInventory
type TileInventory struct {
    gdoc *ipc.GameDocument
    // Make bag/racks inaccessible directly
}

func (inv *TileInventory) SetRack(playerIdx int, rack []byte) error {
    // Validation logic
    inv.gdoc.Racks[playerIdx] = rack
    return nil
}
```

**Benefits:**
- Can't accidentally corrupt state when reading
- All writes go through validation
- Clear separation of concerns
- Easier to add caching, logging, or access control

## Concrete Refactoring Phases

### Phase 1: Create TileInventory Manager (High Priority)

**Goal:** Centralize all tile manipulation logic

**Steps:**
1. Create `pkg/cwgame/tile_inventory.go` with `TileInventory` struct
2. Move `ValidatedPutBack`, `ValidatedDraw`, `ValidatedRemoveTiles` into TileInventory methods
3. Add `ValidateInvariants()` method (consolidate existing validation code)
4. Update `AssignRacks`, `playTilePlacementMove` to use TileInventory
5. Update `ApplyEventInEditorMode` to use TileInventory
6. Run existing tests to ensure behavior unchanged

**Expected benefit:**
- Single source of truth for tile operations
- Easier to find and fix bugs
- Validation consistently enforced

**Estimated effort:** 1-2 days

### Phase 2: Declarative Operations (Medium Priority)

**Goal:** Replace imperative "put back then remove" with declarative "set state to X"

**Steps:**
1. Add `TileInventory.SetRacks(racks [][]byte)` - calculates bag from racks+board
2. Replace `AssignRacks` implementation with call to `SetRacks`
3. Add `TileInventory.PlayMove(player, tiles, position)` - computes entire new state
4. Replace rack inference logic with declarative approach
5. Remove old "put back then remove then put back again" code paths

**Expected benefit:**
- Simpler logic - say what you want, not how to get there
- Fewer intermediate states to validate
- Easier to understand code flow

**Estimated effort:** 2-3 days

### Phase 3: Finish Immutable Amendments (High Priority)

**Goal:** Ensure all amendment operations work on clones

**Steps:**
1. Already done for `handleAmendment` - verify it works correctly
2. Apply same pattern to `SetRacks` if it modifies state
3. Add immutability to any other state-modifying editor operations
4. Document the pattern for future code

**Expected benefit:**
- No rollback complexity
- Can't save partial/corrupted state
- Easier to reason about correctness

**Estimated effort:** 0.5-1 day (mostly done)

### Phase 4: Single Validation Point (Medium Priority)

**Goal:** One function checks all invariants, called automatically

**Steps:**
1. Consolidate `ValidateTotalTiles`, `ValidateTileDistribution`, `countBoardTiles` into `TileInventory.ValidateInvariants()`
2. Call automatically after every TileInventory operation
3. Make it optional via flag for performance in production
4. Add detailed error messages with tile counts per location

**Expected benefit:**
- Can't forget to validate
- Consistent validation everywhere
- Better error messages for debugging

**Estimated effort:** 1 day

### Phase 5: Eliminate Special Cases (Low Priority)

**Goal:** Use polymorphism for game type differences

**Steps:**
1. Create `TileSource` interface with implementations for different game types
2. Create `RackAssignmentStrategy` interface for different assignment behaviors
3. Replace if/else chains with factory pattern + polymorphism
4. Treat nil/empty/partial racks uniformly using Option pattern

**Expected benefit:**
- Cleaner main code path
- Special cases isolated and testable
- Easier to add new game variants

**Estimated effort:** 2-3 days

## Expected Overall Benefits

After completing these phases:

1. **Fewer bugs** - invalid states become impossible or are caught immediately
2. **Easier to understand** - single place for tile logic, declarative operations
3. **Easier to modify** - changes to tile logic only affect TileInventory
4. **Better testing** - can test TileInventory in isolation with comprehensive cases
5. **Better debugging** - single place to add logging, breakpoints, assertions
6. **Less code** - eliminate redundant validation, rollback logic, special cases

## Migration Strategy

These changes can be made incrementally without breaking existing functionality:

1. Add new TileInventory alongside existing code
2. Gradually migrate operations to use TileInventory
3. Remove old code once migration complete
4. Keep comprehensive tests running throughout

The existing test suite should pass at each phase, ensuring no regressions.

## Notes

- Phase 1 (TileInventory Manager) gives the biggest immediate win
- Phase 3 (Immutable Amendments) is mostly complete
- Phases can be done in order or prioritized based on pain points
- Each phase can be done independently without requiring others
- Focus on correctness first, performance second (can optimize later if needed)

## Open Questions

1. Should TileInventory own the entire GameDocument or just bag/racks/board?
2. Should validation be always-on or optionally disabled for performance?
3. Should we use immutable data structures throughout (more GC pressure but safer)?
4. How do we handle concurrent amendments (if that's a concern)?

## References

Current problematic code locations:
- `pkg/cwgame/validated_bag.go` - tile validation functions
- `pkg/cwgame/api.go` - `AssignRacks`, `clientEventToGameEvent`, rack inference
- `pkg/cwgame/game.go` - `playTilePlacementMove`
- `pkg/omgwords/handlers.go` - `handleAmendment`
- `pkg/omgwords/service.go` - `SetRacks`
