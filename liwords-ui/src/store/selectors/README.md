# Tournament Selector Pattern

This directory contains selectors for computing derived state from the Redux-like store in the application. The main goal is to separate the concerns of state storage and state derivation, making the code more maintainable and predictable.

## The Problem

In the tournament reducer, the `competitorState` was being manually updated in some reducer cases but not in others. This led to inconsistencies and made it hard to track when and where the `competitorState` needed to be recalculated.

As noted in the original code:

```typescript
// For the following two actions, it is important to recalculate
// the competitorState if it exists; this is because
// competitorState.status depends on state.activeGames.
```

This comment highlights the issue - developers had to remember to recalculate the `competitorState` in specific reducer cases, which is error-prone.

## The Solution: Selector Pattern

The selector pattern solves this by:

1. Moving the calculation of derived state (like `competitorState`) out of the reducer
2. Computing it on-demand when components need it
3. Ensuring it's always up-to-date with the latest state

### How It Works

1. We've created a `getCompetitorState` selector in `tournament_selectors.ts` that:
   - Takes the tournament state and login state as inputs
   - Computes the `competitorState` based on the current state
   - Returns a consistent, up-to-date `competitorState`

2. Components can use this selector to get the current `competitorState` without worrying about when it was last updated in the reducer.

### Benefits

- **Single Source of Truth**: The `competitorState` is always derived from the base state
- **Reduced Complexity**: The reducer only needs to manage the base state
- **Easier Maintenance**: No need to remember to update the `competitorState` in every reducer case
- **Consistency**: The `competitorState` is always calculated the same way

## How to Use

See `example_using_selector.tsx` for a complete example of how to use the selector pattern in a component.

Basic usage:

```typescript
import { getCompetitorState } from "../store/selectors/tournament_selectors";

// In your component:
const { tournamentContext } = useTournamentStoreContext();
const loginState = { /* your login state */ };
const competitorState = getCompetitorState(tournamentContext, loginState);

// Now use competitorState as before
```

## Next Steps

To fully implement this pattern:

1. Keep the `competitorState` in the reducer state for now to avoid breaking existing code
2. Gradually update components to use the selector instead of accessing state directly
3. Once all components are updated, you can remove `competitorState` from the reducer state

This approach allows for a gradual migration without breaking existing functionality.

## Performance Considerations

If performance becomes an issue (due to recalculating the state too often), consider adding memoization to the selector. Libraries like `reselect` can help with this, or you can implement a simple memoization pattern.
