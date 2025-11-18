# Frontend Migration Plan: Liwords UI Refactoring

**Status:** In Progress - Phase 1
**Started:** November 2025
**Estimated Completion:** March-April 2026 (16-20 weeks)

## Table of Contents
- [Executive Summary](#executive-summary)
- [Current State Analysis](#current-state-analysis)
- [Migration Goals](#migration-goals)
- [Core Strategy](#core-strategy)
- [Phase 1: Foundation & POC](#phase-1-foundation--poc-week-1-2)
- [Phase 2: Core Infrastructure](#phase-2-core-infrastructure-week-3-5)
- [Phase 3: View Migration](#phase-3-view-migration-week-6-12)
- [Phase 4: Game Room](#phase-4-game-room-week-13-16)
- [Phase 5: Cleanup](#phase-5-cleanup--optimization-week-17-18)
- [State Migration Patterns](#state-migration-patterns)
- [Risk Mitigation](#risk-mitigation-strategies)
- [Success Metrics](#success-metrics)

---

## Executive Summary

This document outlines a comprehensive plan to refactor the Liwords frontend application to address three major pain points:

1. **State Management:** Replace massive context-based store (1,349 lines, 20+ contexts) with TanStack Query + minimal Zustand store
2. **Socket Architecture:** Implement persistent WebSocket connection with channel subscriptions (no more disconnect/reconnect on navigation)
3. **Styling System:** Migrate from Ant Design + extensive SCSS overrides to Mantine with CSS Modules and theme tokens

**Scope:** 187+ React components, 36 SCSS files, 50+ socket message types
**Strategy:** Incremental migration with parallel component trees (Ant Design + Mantine coexistence)
**Timeline:** 16-20 weeks with 1-2 full-time developers

---

## Current State Analysis

### State Management (Complex - 20+ Contexts)

**File:** `src/store/store.tsx` (1,349 lines)

**Architecture:** Multiple React Context providers with custom reducers

**Contexts Managed:**
- LobbyContext - Sought games, active games, correspondence games
- LoginStateContext - Authentication, user info, permissions
- GameContextContext - Board state, turns, players, game events
- ExaminableGameContextContext - Analyzer/examine mode state
- TournamentContext - Tournament metadata, pairings, standings
- ChatContext - Chat messages, channels, presence
- PresenceContext - User presence data
- TimerContext - Game timers, clock control
- TentativePlayContext - Temporary tile placements
- GameMetaEventContext - Meta game events
- GameEndMessageContext - End game messages
- RematchRequestContext - Rematch request state
- PoolFormatContext - Tile pool display format
- ExcludedPlayersContext - Blocked players
- FriendsContext - Friends list
- ModeratorsContext - Moderator/admin status
- LagContext - Network lag measurements
- ContextMatchContext - Context menu handlers
- ExamineContext - Game examination/analysis state

**Key Reducers:**
- `lobby_reducer.ts` (414 lines) - Manages seeks, active games, correspondence
- `game_reducer.ts` (737 lines) - Board state, turns, game events
- `tournament_reducer.ts` (808 lines) - Tournament state

**Pain Points:**
- Hard to debug (context tree depth)
- Performance issues (unnecessary re-renders)
- Tight coupling between concerns
- Difficult to test in isolation

### Socket Connection Handling

**File:** `src/socket/socket.ts` (297 lines)

**Library:** `react-use-websocket` v4.13.0

**Current Flow:**
1. Socket initialized when App component mounts
2. Navigation triggers route unmount
3. Socket resets via `resetSocket()` key increment
4. New route mounts, socket reconnects
5. User experiences brief disconnect

**Message Protocol:** Binary format with 50+ message types
- First 2 bytes: message length
- Third byte: message type (enum MessageType)
- Rest: protobuf payload

**Socket Handlers:** `src/store/socket_handlers.ts` (1,108 lines)
- Handles all incoming messages
- Dispatches to context reducers
- Triggers navigation directly

**Pain Point:**
- Every navigation causes disconnect/reconnect
- Not a true Single Page App experience
- Lobby updates pause during game transitions

### Styling Architecture

**Current System:** Ant Design v5 + extensive SCSS overrides

**SCSS Files:** 36 files (~3,000+ lines total)
- `base.scss` (360 lines) - Variables, mixins, breakpoints
- `color_modes.scss` (260 lines) - Color theme system
- `board_modes.scss` (259 lines) - Board visual modes
- `tile_modes.scss` (239 lines) - Tile visual modes
- `App.scss` (279 lines) - Global app styles
- 31+ component-specific SCSS files

**Theming System:**
- SCSS mixins for multi-mode theming (light/dark)
- Body classes (`mode--dark`, `board--classic`, `tile--purple`)
- Ant Design theme token overrides
- Custom color variables (50+ semantic tokens)

**Pain Points:**
- Extensive SCSS needed for Ant Design overrides
- Theme system split between SCSS and Ant Design tokens
- Hard to maintain consistency
- Large CSS bundle size

### Technology Stack

**Current Dependencies:**
```json
{
  "react": "^19.1.0",              // ⚠️ Very new!
  "react-dom": "^19.1.0",
  "react-router": "^7.6.1",        // ⚠️ Major version jump!
  "antd": "^5.25.3",
  "@ant-design/icons": "^6.0.0",
  "@ant-design/v5-patch-for-react-19": "^1.0.3",
  "@reduxjs/toolkit": "^2.8.2",
  "react-redux": "^9.2.0",
  "@tanstack/react-query": "^5.79.0",  // ✅ Already installed!
  "react-use-websocket": "^4.13.0",
  "react-dnd": "^16.0.1"
}
```

**Build Tool:** Rsbuild (Rspack-based, fast bundler)

---

## Migration Goals

### Primary Goals

1. **Simplify State Management**
   - Replace 20+ contexts with TanStack Query (server state) + Zustand (client state)
   - Reduce store code by 70%+
   - Improve debuggability with React Query DevTools

2. **Persistent Socket Connection**
   - Eliminate disconnect/reconnect on navigation
   - Implement channel subscription pattern
   - True SPA experience

3. **Modern Styling System**
   - Migrate to Mantine UI library
   - Use CSS Modules for custom styles
   - Consolidate theme system with tokens
   - Reduce CSS bundle size by 30%+

### Secondary Goals

- Improve TypeScript type safety
- Better component testability
- Faster development velocity
- Improved performance (fewer re-renders)
- Better dark mode support

---

## Core Strategy

### Parallel Component Trees

Run two parallel UI trees during transition:

```
App (Root)
├── QueryClientProvider (TanStack Query)
│   ├── MantineProvider (theme)
│   │   └── NewComponents (Mantine + TanStack Query + Zustand)
│   └── AntConfigProvider (theme)
│       └── LegacyStore (context)
│           └── OldComponents (Ant Design)
```

**Why This Works:**
- Mantine uses CSS modules (scoped styles)
- Ant Design uses prefixed classes (`.ant-*`)
- Both can coexist without conflicts
- Gradual migration, one view at a time
- Feature flags for safe rollout

### Three-Tier State Architecture

```
┌─────────────────────────────────────────┐
│  TanStack Query (Server State)          │
│  - Game data, lobby seeks, tournaments  │
│  - Invalidation via socket messages     │
│  - Automatic caching & refetching       │
└─────────────────────────────────────────┘
           ↓
┌─────────────────────────────────────────┐
│  Zustand Store (Client State - Minimal) │
│  - UI state (modals, selected items)    │
│  - User preferences (theme, settings)   │
│  - Transient state (tentative plays)    │
└─────────────────────────────────────────┘
           ↓
┌─────────────────────────────────────────┐
│  Component State (Local UI)             │
│  - Form inputs, hover states, etc.      │
└─────────────────────────────────────────┘
```

**Principle:** Keep state as close to where it's used as possible

---

## Phase 1: Foundation & POC (Week 1-2)

**Goal:** Set up infrastructure and prove the migration strategy works

### 1.1 Install Mantine & Dependencies

```bash
npm install @mantine/core @mantine/hooks @mantine/notifications
npm install zustand
npm install --save-dev postcss postcss-preset-mantine postcss-simple-vars
```

### 1.2 Create Shared Theme Tokens

**File:** `src/theme/tokens.ts`

Create TypeScript-based theme tokens that both Ant Design and Mantine can reference:

```typescript
export const tokens = {
  colors: {
    background: { light: '#ffffff', dark: '#1a1a1a' },
    cardBackground: { light: '#f5f5f5', dark: '#2d2d2d' },
    primary: { light: '#1890ff', dark: '#40a9ff' },
    text: { light: '#282828', dark: '#e8e8e8' },
    textSecondary: { light: '#666666', dark: '#a0a0a0' },
    border: { light: '#d9d9d9', dark: '#424242' },
    error: { light: '#ff4d4f', dark: '#ff7875' },
    success: { light: '#52c41a', dark: '#95de64' },
    warning: { light: '#faad14', dark: '#ffc53d' },
  },
  spacing: {
    xs: 4,
    sm: 8,
    md: 16,
    lg: 24,
    xl: 32,
    xxl: 48,
  },
  fontSizes: {
    xs: 12,
    sm: 14,
    md: 16,
    lg: 18,
    xl: 20,
  },
  // Board-specific tokens
  board: {
    dls: { light: '#b3d9ff', dark: '#004085' }, // Double letter score
    tls: { light: '#ff9999', dark: '#660000' }, // Triple letter score
    dws: { light: '#ffcccc', dark: '#4d0000' }, // Double word score
    tws: { light: '#ff6666', dark: '#330000' }, // Triple word score
    background: { light: '#e8d4a0', dark: '#2d2416' },
    gridLines: { light: '#000000', dark: '#888888' },
  },
  // Tile-specific tokens
  tile: {
    background: { light: '#f5e6d3', dark: '#3d3428' },
    text: { light: '#000000', dark: '#ffffff' },
    border: { light: '#8b7355', dark: '#6b5335' },
    blank: { light: '#ffe4b5', dark: '#4a3f2f' },
  },
};

export type ThemeMode = 'light' | 'dark';

export const getColorValue = (token: string, mode: ThemeMode): string => {
  // Helper to get color value by path
  // e.g., getColorValue('colors.background', 'dark') => '#1a1a1a'
  const parts = token.split('.');
  let value: any = tokens;
  for (const part of parts) {
    value = value[part];
  }
  return typeof value === 'object' ? value[mode] : value;
};
```

### 1.3 Set Up Mantine Provider

**File:** `src/theme/mantine-theme.ts`

```typescript
import { createTheme, MantineColorsTuple } from '@mantine/core';
import { tokens } from './tokens';

const brandColors: MantineColorsTuple = [
  '#e6f7ff', // 0
  '#bae7ff', // 1
  '#91d5ff', // 2
  '#69c0ff', // 3
  '#40a9ff', // 4 - primary
  '#1890ff', // 5
  '#096dd9', // 6
  '#0050b3', // 7
  '#003a8c', // 8
  '#002766', // 9
];

export const mantineTheme = createTheme({
  primaryColor: 'brand',
  colors: {
    brand: brandColors,
  },
  fontFamily: 'Mulish, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, Helvetica, Arial, sans-serif',
  fontSizes: {
    xs: `${tokens.fontSizes.xs}px`,
    sm: `${tokens.fontSizes.sm}px`,
    md: `${tokens.fontSizes.md}px`,
    lg: `${tokens.fontSizes.lg}px`,
    xl: `${tokens.fontSizes.xl}px`,
  },
  spacing: {
    xs: `${tokens.spacing.xs}px`,
    sm: `${tokens.spacing.sm}px`,
    md: `${tokens.spacing.md}px`,
    lg: `${tokens.spacing.lg}px`,
    xl: `${tokens.spacing.xl}px`,
  },
  // Configure z-indexes to not conflict with Ant Design
  other: {
    zIndexModal: 2000,    // Higher than Ant's 1000
    zIndexPopover: 1500,
    zIndexNotification: 2500,
  },
});
```

### 1.4 Install Zustand & Create First Store

**Install:**
```bash
npm install zustand
```

**File:** `src/stores/ui-store.ts`

```typescript
import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';

export type ThemeMode = 'light' | 'dark';

interface UIState {
  // Theme
  themeMode: ThemeMode;
  setThemeMode: (mode: ThemeMode) => void;
  toggleTheme: () => void;

  // Modals
  activeModal: string | null;
  openModal: (modalId: string) => void;
  closeModal: () => void;

  // UI preferences
  showNotations: boolean;
  setShowNotations: (show: boolean) => void;
}

export const useUIStore = create<UIState>()(
  devtools(
    persist(
      (set, get) => ({
        // Theme
        themeMode: 'light',
        setThemeMode: (mode) => set({ themeMode: mode }),
        toggleTheme: () =>
          set((state) => ({
            themeMode: state.themeMode === 'light' ? 'dark' : 'light',
          })),

        // Modals
        activeModal: null,
        openModal: (modalId) => set({ activeModal: modalId }),
        closeModal: () => set({ activeModal: null }),

        // UI preferences
        showNotations: true,
        setShowNotations: (show) => set({ showNotations: show }),
      }),
      {
        name: 'liwords-ui-store', // localStorage key
        partialize: (state) => ({
          // Only persist these fields
          themeMode: state.themeMode,
          showNotations: state.showNotations,
        }),
      }
    ),
    { name: 'UIStore' }
  )
);
```

### 1.5 Set Up Dual Provider System

**File:** `src/providers/AppProviders.tsx` (new file)

```typescript
import React from 'react';
import { MantineProvider, ColorSchemeScript } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { ConfigProvider, App as AntDApp } from 'antd';
import { Provider as ReduxProvider } from 'react-redux';
import { mantineTheme } from '../theme/mantine-theme';
import { liwordsDefaultTheme, liwordsDarkTheme } from '../themes';
import { store as reduxStore } from '../store/redux_store';
import { LegacyStore } from '../store/store';
import { useUIStore } from '../stores/ui-store';

// Import Mantine styles
import '@mantine/core/styles.css';
import '@mantine/notifications/styles.css';

// TanStack Query client configuration
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      gcTime: 1000 * 60 * 30,   // 30 minutes (formerly cacheTime)
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

interface AppProvidersProps {
  children: React.ReactNode;
}

export const AppProviders: React.FC<AppProvidersProps> = ({ children }) => {
  const themeMode = useUIStore((state) => state.themeMode);

  // Ant Design theme based on mode
  const antdTheme = themeMode === 'dark' ? liwordsDarkTheme : liwordsDefaultTheme;

  return (
    <>
      <ColorSchemeScript defaultColorScheme="light" />
      <QueryClientProvider client={queryClient}>
        <MantineProvider theme={mantineTheme} defaultColorScheme={themeMode}>
          <Notifications position="top-right" zIndex={2500} />

          {/* Mantine-based components will be rendered here in the future */}

          {/* Legacy Ant Design components */}
          <ReduxProvider store={reduxStore}>
            <ConfigProvider theme={antdTheme}>
              <AntDApp>
                <LegacyStore>
                  {children}
                </LegacyStore>
              </AntDApp>
            </ConfigProvider>
          </ReduxProvider>
        </MantineProvider>

        {/* Dev tools */}
        <ReactQueryDevtools initialIsOpen={false} />
      </QueryClientProvider>
    </>
  );
};
```

### 1.6 Update App.tsx to Use New Providers

Update `src/App.tsx` to use the new provider wrapper instead of having providers directly in the file.

### 1.7 Create POC: Settings Page v2

**Goal:** Prove the migration strategy works with a real component

**File:** `src/settings-v2/SettingsV2.tsx` (new file)

Create a new version of the Settings page using:
- Mantine components (Tabs, TextInput, Switch, Button)
- CSS Modules for custom styling
- Zustand for local state
- TanStack Query for fetching user data (if applicable)

**Route:** Initially at `/settings-v2` for testing, then swap to `/settings`

**Components to create:**
- `SettingsV2.tsx` - Main container
- `PersonalInfoSection.tsx` - Personal info form (Mantine)
- `AppearanceSection.tsx` - Theme controls (Mantine)
- `GamePreferencesSection.tsx` - Game settings (Mantine)
- `SettingsV2.module.css` - Custom styles

---

## Phase 2: Core Infrastructure (Week 3-5)

**Goal:** Build foundation for persistent socket and query integration

### 2.1 Backend: Socket Subscription Support

**Backend Changes Required:**

Add new message types to protobuf:
```protobuf
message SubscribeChannel {
  string channel = 1;  // e.g., "lobby", "game.123", "tournament.abc"
}

message UnsubscribeChannel {
  string channel = 1;
}
```

Update socket handler to maintain subscription set per connection:
- Keep map of `connectionId -> Set<channel>`
- When broadcasting messages, check subscription set
- Clean up subscriptions on disconnect

### 2.2 Frontend: Persistent Socket Architecture

**File:** `src/socket/socket-subscription-manager.ts` (new file)

```typescript
import { MessageType } from '../gen/api/proto/realtime/realtime_pb';

export class SocketSubscriptionManager {
  private currentChannels = new Set<string>();
  private sendMessage: (msg: any) => void;

  constructor(sendMessage: (msg: any) => void) {
    this.sendMessage = sendMessage;
  }

  subscribe(channel: string): void {
    if (!this.currentChannels.has(channel)) {
      console.log(`[Socket] Subscribing to channel: ${channel}`);
      this.sendMessage({
        type: MessageType.SUBSCRIBE_CHANNEL,
        channel,
      });
      this.currentChannels.add(channel);
    }
  }

  unsubscribe(channel: string): void {
    if (this.currentChannels.has(channel)) {
      console.log(`[Socket] Unsubscribing from channel: ${channel}`);
      this.sendMessage({
        type: MessageType.UNSUBSCRIBE_CHANNEL,
        channel,
      });
      this.currentChannels.delete(channel);
    }
  }

  reset(): void {
    this.currentChannels.clear();
  }
}
```

**File:** `src/hooks/useSocketSubscription.ts` (new file)

```typescript
import { useEffect, useRef } from 'react';
import { useSocketContext } from '../socket/socket-context';

export const useSocketSubscription = (channel: string) => {
  const { subscriptionManager } = useSocketContext();
  const channelRef = useRef(channel);

  useEffect(() => {
    const currentChannel = channelRef.current;
    subscriptionManager.subscribe(currentChannel);

    return () => {
      subscriptionManager.unsubscribe(currentChannel);
    };
  }, [channel, subscriptionManager]);
};
```

**Update:** Move socket initialization above router in `App.tsx` so it's never unmounted

### 2.3 TanStack Query Integration with Socket

**File:** `src/hooks/useSocketMessage.ts` (new file)

```typescript
import { useEffect } from 'react';
import { MessageType } from '../gen/api/proto/realtime/realtime_pb';
import { useSocketContext } from '../socket/socket-context';

export const useSocketMessage = <T = any>(
  messageType: MessageType,
  handler: (message: T) => void
) => {
  const { addMessageHandler, removeMessageHandler } = useSocketContext();

  useEffect(() => {
    const handlerId = addMessageHandler(messageType, handler);

    return () => {
      removeMessageHandler(handlerId);
    };
  }, [messageType, handler, addMessageHandler, removeMessageHandler]);
};
```

**Pattern for query invalidation:**

```typescript
// queries/lobby-queries.ts
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useSocketMessage } from '../hooks/useSocketMessage';
import { MessageType } from '../gen/api/proto/realtime/realtime_pb';

export const useActiveGames = () => {
  const queryClient = useQueryClient();

  // Invalidate query when socket sends update
  useSocketMessage(MessageType.ACTIVE_GAMES, () => {
    queryClient.invalidateQueries({ queryKey: ['lobby', 'active-games'] });
  });

  // Or directly set query data
  useSocketMessage(MessageType.ACTIVE_GAMES, (msg) => {
    queryClient.setQueryData(['lobby', 'active-games'], msg.games);
  });

  return useQuery({
    queryKey: ['lobby', 'active-games'],
    queryFn: async () => {
      // Initial fetch (could be empty, socket will populate)
      return [];
    },
  });
};
```

### 2.4 Theme System Unification

**Update body classes to sync with Zustand:**

```typescript
// hooks/useThemeSync.ts
import { useEffect } from 'react';
import { useUIStore } from '../stores/ui-store';

export const useThemeSync = () => {
  const themeMode = useUIStore((state) => state.themeMode);

  useEffect(() => {
    // Keep body class in sync for legacy SCSS
    document.body.classList.remove('mode--default', 'mode--dark');
    document.body.classList.add(`mode--${themeMode === 'dark' ? 'dark' : 'default'}`);
  }, [themeMode]);
};
```

Call this hook in `App.tsx` to ensure theme stays in sync.

---

## Phase 3: View Migration (Week 6-12)

**Strategy:** Migrate views one at a time, easiest to hardest

### Migration Order

1. **Settings** ✅ (Done in Phase 1.7)
2. **Profile Page** (Week 6)
3. **Lobby** (Week 7-8) - Most critical
4. **Tournament Room** (Week 9)
5. **Chat Widget** (Week 10)
6. **Puzzle View** (Week 11)
7. **Other minor views** (Week 12)

### Per-View Checklist

For each view migration:

- [ ] **Analyze current implementation**
  - List all context hooks used
  - Identify socket messages consumed
  - Note SCSS dependencies
  - Check component tree depth

- [ ] **Design new architecture**
  - Map contexts to TanStack Query or Zustand
  - Design component structure
  - Plan CSS Modules

- [ ] **Implement new version**
  - Create `[view]-v2/` directory
  - Build components with Mantine
  - Use TanStack Query for server state
  - Use Zustand for UI state
  - Create CSS Modules for custom styling

- [ ] **Add feature flag**
  - Route to `/[view]-v2` initially
  - Add flag in Zustand or config
  - Allow opt-in testing

- [ ] **Test thoroughly**
  - Unit tests for new components
  - Integration tests for socket interaction
  - Visual regression tests
  - Manual QA

- [ ] **Gradual rollout**
  - Release to beta users
  - Monitor errors and performance
  - Iterate based on feedback

- [ ] **Swap routes**
  - Make new version default
  - Keep old version at `/[view]-legacy` temporarily
  - Monitor for issues

- [ ] **Cleanup**
  - Remove old components after 1-2 weeks
  - Delete associated SCSS
  - Remove feature flag

### Example: Lobby Migration

**Current State:**
- Uses LobbyContext (seeks, active games, correspondence)
- Uses LoginStateContext (user info)
- Uses ChatContext (chat widget)
- Heavy SCSS customization

**New Architecture:**

**Queries:**
```typescript
// queries/lobby-queries.ts
export const useSeeks = () => {
  // TanStack Query for seeks list
};

export const useActiveGames = () => {
  // TanStack Query for active games
};

export const useCorrespondenceGames = () => {
  // TanStack Query for correspondence games
};
```

**UI Store:**
```typescript
// stores/lobby-store.ts
interface LobbyUIState {
  selectedSeek: string | null;
  filterMode: 'all' | 'rated' | 'unrated';
  sortBy: 'time' | 'rating';
}
```

**Components:**
```
lobby-v2/
├── LobbyV2.tsx                 // Main container
├── SeeksList.tsx               // Mantine Table or List
├── ActiveGamesList.tsx         // Mantine Table
├── CorrespondenceGamesList.tsx // Mantine Table
├── SeekForm.tsx                // Mantine Modal + Form
└── styles/
    ├── Lobby.module.css
    ├── SeeksList.module.css
    └── SeekForm.module.css
```

**Socket Integration:**
```typescript
// In LobbyV2.tsx
useSocketSubscription('lobby');

// In queries/lobby-queries.ts
useSocketMessage(MessageType.SEEK_REQUESTS, (msg) => {
  queryClient.setQueryData(['lobby', 'seeks'], msg.requests);
});
```

---

## Phase 4: Game Room (Week 13-16)

**The Big One:** Game room is the most complex view

### Strategy: Incremental Component Migration

Don't migrate the entire game room at once. Instead:

1. **Week 13: Layout + Player Cards**
   - New Mantine layout (Grid/Flex)
   - Player info cards with Mantine components
   - Keep old board/rack temporarily

2. **Week 14: Board + Rack**
   - Most complex part (drag-and-drop)
   - May need to keep react-dnd or switch to dnd-kit
   - Tentative play state in Zustand

3. **Week 15: Scorecard + Controls**
   - Scorecard with Mantine Table
   - Game controls with Mantine buttons
   - Timer display

4. **Week 16: Analyzer + Polish**
   - Analyzer panel
   - Notepad
   - Pool display
   - Final integration testing

### Game State Architecture

**Hybrid Approach:** TanStack Query + Zustand

```typescript
// stores/game-store.ts
interface GameState {
  // Real-time state (from socket)
  currentTurn: number;
  rack: Tile[];
  board: BoardState;
  players: Player[];

  // Tentative plays (before commit)
  tentativeTiles: PlacedTile[];

  // Actions
  placeTile: (tile: Tile, position: Position) => void;
  removeTile: (position: Position) => void;
  clearTentative: () => void;
  commitMove: (move: Move) => Promise<void>;
}

// queries/game-queries.ts
export const useGameHistory = (gameId: string) => {
  return useQuery({
    queryKey: ['game', gameId, 'history'],
    queryFn: () => fetchGameHistory(gameId),
  });
};

export const useGameMetadata = (gameId: string) => {
  const queryClient = useQueryClient();

  useSocketMessage(MessageType.GAME_META_EVENT, (msg) => {
    queryClient.setQueryData(['game', gameId, 'metadata'], msg);
  });

  return useQuery({
    queryKey: ['game', gameId, 'metadata'],
    queryFn: () => fetchGameMetadata(gameId),
  });
};
```

### Drag-and-Drop Considerations

**Current:** react-dnd + react-dnd-multi-backend

**Options:**
1. Keep react-dnd (less migration work)
2. Switch to dnd-kit (more modern, better touch support)

**Recommendation:** Keep react-dnd for Phase 4, consider dnd-kit in future optimization phase.

---

## Phase 5: Cleanup & Optimization (Week 17-18)

### 5.1 Remove Legacy Code (Week 17)

**Checklist:**
- [ ] Delete `store/store.tsx` and all context providers
- [ ] Remove all context reducer files
- [ ] Delete all custom context hooks
- [ ] Remove all SCSS files (except board/tile if still needed)
- [ ] Uninstall Ant Design:
  ```bash
  npm uninstall antd @ant-design/icons @ant-design/v5-patch-for-react-19
  ```
- [ ] Remove Redux if no longer needed
- [ ] Clean up unused dependencies
- [ ] Update imports across codebase

### 5.2 Performance Optimization (Week 18)

**Bundle Analysis:**
```bash
npm run build -- --analyze
```

**Optimizations:**
- [ ] Code splitting by route (lazy loading)
- [ ] Lazy load heavy components (Board, Analyzer)
- [ ] Optimize TanStack Query cache settings
- [ ] Review and reduce CSS bundle size
- [ ] Optimize image assets
- [ ] Enable React Compiler if stable by then

**TanStack Query Optimizations:**
```typescript
// Configure query defaults based on usage patterns
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5,  // Tune based on data volatility
      gcTime: 1000 * 60 * 30,
      retry: (failureCount, error) => {
        // Custom retry logic
        if (error.status === 404) return false;
        return failureCount < 2;
      },
    },
  },
});
```

### 5.3 Documentation

**Create/Update:**
- [ ] Architecture documentation
- [ ] State management patterns guide
- [ ] Component development guide
- [ ] Styling guidelines (CSS Modules + Mantine)
- [ ] Socket integration patterns
- [ ] Testing guidelines
- [ ] Migration lessons learned

**File:** `docs/ARCHITECTURE.md`

---

## State Migration Patterns

### Pattern 1: Simple Context → Zustand

**Before:**
```typescript
// LagContext
const LagContext = React.createContext<LagContextType | null>(null);

export const useLagStoreContext = () => {
  const context = useContext(LagContext);
  if (!context) throw new Error('useLagStoreContext must be used within LagProvider');
  return context;
};
```

**After:**
```typescript
// stores/lag-store.ts
import { create } from 'zustand';

interface LagStore {
  lagMs: number;
  setLagMs: (lagMs: number) => void;
}

export const useLagStore = create<LagStore>((set) => ({
  lagMs: 0,
  setLagMs: (lagMs) => set({ lagMs }),
}));
```

**Usage:**
```typescript
// Before
const { lagMs, setLagMs } = useLagStoreContext();

// After
const lagMs = useLagStore((state) => state.lagMs);
const setLagMs = useLagStore((state) => state.setLagMs);
```

### Pattern 2: Server Data Context → TanStack Query

**Before:**
```typescript
// LobbyContext manages seeks from socket
const [seeks, setSeeks] = useState<Seek[]>([]);

// In socket handler
case MessageType.SEEK_REQUESTS:
  dispatchLobbyContext({ type: 'SET_SEEKS', seeks: msg.requests });
```

**After:**
```typescript
// queries/lobby-queries.ts
export const useSeeks = () => {
  const queryClient = useQueryClient();

  // Update cache when socket sends new data
  useSocketMessage(MessageType.SEEK_REQUESTS, (msg) => {
    queryClient.setQueryData(['lobby', 'seeks'], msg.requests);
  });

  return useQuery({
    queryKey: ['lobby', 'seeks'],
    queryFn: async () => {
      // Could fetch initial data, or just return empty
      // Socket will populate the real data
      return [];
    },
  });
};
```

**Usage:**
```typescript
// In component
const { data: seeks, isLoading } = useSeeks();
```

### Pattern 3: Complex Game State → Hybrid

**Game state is special:** It has both server state (history, metadata) and client state (tentative plays, UI state).

**Architecture:**

```typescript
// stores/game-store.ts (Zustand for client state)
interface GameStore {
  // Tentative plays
  tentativeTiles: PlacedTile[];
  placeTile: (tile: Tile, pos: Position) => void;
  clearTentative: () => void;

  // UI state
  selectedTile: Tile | null;
  showAnalyzer: boolean;

  // Optimistic updates
  optimisticMoves: Move[];
  addOptimisticMove: (move: Move) => void;
  rollbackOptimisticMove: (moveId: string) => void;
}

// queries/game-queries.ts (TanStack Query for server state)
export const useGameState = (gameId: string) => {
  const queryClient = useQueryClient();

  useSocketMessage(MessageType.SERVER_GAMEPLAY_EVENT, (event) => {
    queryClient.setQueryData(['game', gameId, 'state'], (old) => ({
      ...old,
      ...event,
    }));
  });

  return useQuery({
    queryKey: ['game', gameId, 'state'],
    queryFn: () => fetchGameState(gameId),
  });
};

export const useGameHistory = (gameId: string) => {
  return useQuery({
    queryKey: ['game', gameId, 'history'],
    queryFn: () => fetchGameHistory(gameId),
  });
};
```

**In component:**
```typescript
const GameBoard = ({ gameId }) => {
  // Server state
  const { data: gameState } = useGameState(gameId);
  const { data: history } = useGameHistory(gameId);

  // Client state
  const tentativeTiles = useGameStore((state) => state.tentativeTiles);
  const placeTile = useGameStore((state) => state.placeTile);

  // Render with combined state
};
```

### Pattern 4: Derived State → Selectors

**Before:**
```typescript
// ExaminableGameContextContext - derives state from GameContext
const examinableGameContext = useMemo(() => {
  if (examining) {
    return deriveExaminableState(gameContext);
  }
  return gameContext;
}, [gameContext, examining]);
```

**After:**
```typescript
// Use Zustand selectors or useMemo
const useExaminableGameState = (gameId: string) => {
  const { data: gameState } = useGameState(gameId);
  const examining = useGameStore((state) => state.examining);

  return useMemo(() => {
    if (!gameState) return null;
    if (examining) {
      return deriveExaminableState(gameState);
    }
    return gameState;
  }, [gameState, examining]);
};
```

---

## Risk Mitigation Strategies

### 1. Feature Flags

Use feature flags for gradual rollout:

```typescript
// stores/feature-flags-store.ts
interface FeatureFlagsStore {
  flags: Record<string, boolean>;
  isEnabled: (flag: string) => boolean;
  enableFlag: (flag: string) => void;
  disableFlag: (flag: string) => void;
}

export const useFeatureFlags = create<FeatureFlagsStore>((set, get) => ({
  flags: {
    'new-lobby': false,
    'new-game-room': false,
    'new-settings': true,
  },
  isEnabled: (flag) => get().flags[flag] ?? false,
  enableFlag: (flag) => set((state) => ({
    flags: { ...state.flags, [flag]: true },
  })),
  disableFlag: (flag) => set((state) => ({
    flags: { ...state.flags, [flag]: false },
  })),
}));

// In route
const NewGameRoomEnabled = () => {
  const isEnabled = useFeatureFlags((state) => state.isEnabled('new-game-room'));
  return isEnabled ? <GameRoomV2 /> : <GameTable />;
};
```

### 2. Parallel Routes

Run old and new versions side-by-side:

```typescript
<Routes>
  {/* Old routes */}
  <Route path="/game/:gameID" element={<GameTable />} />
  <Route path="/lobby" element={<Lobby />} />

  {/* New routes (opt-in) */}
  <Route path="/game-v2/:gameID" element={<GameRoomV2 />} />
  <Route path="/lobby-v2" element={<LobbyV2 />} />
</Routes>
```

Add toggle in UI to switch between versions for testing.

### 3. Monitoring & Telemetry

**Add error tracking:**
```typescript
// utils/error-tracking.ts
export const trackError = (error: Error, context: Record<string, any>) => {
  // Send to error tracking service (Sentry, LogRocket, etc.)
  console.error('[Error]', error, context);
};

// In component
try {
  commitMove(move);
} catch (error) {
  trackError(error, { component: 'GameBoard', action: 'commitMove', gameId });
}
```

**Performance monitoring:**
```typescript
// Track render performance
const GameBoard = () => {
  useEffect(() => {
    const start = performance.now();
    return () => {
      const duration = performance.now() - start;
      if (duration > 100) {
        console.warn('[Performance] GameBoard render took', duration, 'ms');
      }
    };
  });
};
```

### 4. Testing Strategy

**Unit Tests:**
```typescript
// Use vitest + @testing-library/react
describe('SeeksList', () => {
  it('renders seeks from query', () => {
    const { getByText } = render(<SeeksList />);
    // assertions
  });

  it('handles seek selection', () => {
    // test Zustand store updates
  });
});
```

**Integration Tests:**
```typescript
// Test socket + query interaction
describe('Lobby socket integration', () => {
  it('updates seeks when socket message received', async () => {
    // Mock socket
    // Trigger SEEK_REQUESTS message
    // Assert query data updated
  });
});
```

**E2E Tests:**
```typescript
// Use Playwright or Cypress
test('user can create and accept a seek', async ({ page }) => {
  await page.goto('/lobby-v2');
  await page.click('[data-testid="create-seek"]');
  // ... rest of test
});
```

---

## Success Metrics

### Technical Metrics

**State Management:**
- [ ] Zero uses of legacy context store
- [ ] Less than 5 Zustand stores (minimal client state)
- [ ] All server state managed by TanStack Query
- [ ] >80% reduction in state management code

**Styling:**
- [ ] Zero SCSS files (except board/tile if needed)
- [ ] All custom styles in CSS Modules
- [ ] Ant Design dependency removed
- [ ] >30% reduction in CSS bundle size

**Socket:**
- [ ] Persistent connection (no disconnect on navigation)
- [ ] <100ms latency for channel subscriptions
- [ ] Zero socket reconnection errors in logs

**Performance:**
- [ ] <2s initial page load (LCP)
- [ ] <100ms route transitions
- [ ] <50ms component re-render time
- [ ] >90 Lighthouse performance score

**Bundle Size:**
- [ ] Main bundle <500KB (gzipped)
- [ ] Route chunks <200KB each
- [ ] Total bundle size reduced by >20%

### User Experience Metrics

**Functionality:**
- [ ] Zero regressions in existing features
- [ ] All views migrated and functional
- [ ] Feature parity with legacy version
- [ ] No increase in bug reports

**Performance:**
- [ ] Faster perceived load time
- [ ] Smoother animations
- [ ] No lag during gameplay
- [ ] Instant route transitions

**Usability:**
- [ ] Improved dark mode consistency
- [ ] Better mobile responsiveness
- [ ] Clearer visual hierarchy
- [ ] Positive user feedback

### Developer Experience Metrics

**Code Quality:**
- [ ] >80% test coverage on new code
- [ ] Zero TypeScript errors
- [ ] <5 ESLint warnings
- [ ] All components documented

**Maintainability:**
- [ ] Clear component structure
- [ ] Consistent patterns across codebase
- [ ] Easy to add new features
- [ ] Fast onboarding for new contributors

**Debugging:**
- [ ] React Query DevTools integrated
- [ ] Zustand DevTools integrated
- [ ] Clear error messages
- [ ] Source maps in production

---

## Open Questions & Decisions

### 1. State Management: Zustand vs Redux Toolkit?

**Zustand Pros:**
- Simpler API, less boilerplate
- Better TypeScript inference
- Smaller bundle size
- Easier to learn

**Redux Toolkit Pros:**
- Already installed
- More established ecosystem
- Better DevTools
- Team might already know it

**Decision:** Recommend Zustand for simpler API and smaller footprint. Redux Toolkit is good but adds complexity we don't need.

### 2. CSS: CSS Modules vs Tailwind?

**CSS Modules Pros:**
- Full control over styles
- Familiar to SCSS users
- No new learning curve
- Scoped by default

**Tailwind Pros:**
- Faster development
- Smaller CSS bundle (purge unused)
- Consistent design system
- Popular in community

**Decision:** Recommend CSS Modules initially. Team is familiar with SCSS, so CSS Modules will be easier transition. Can consider Tailwind in future.

### 3. Drag-and-Drop: react-dnd vs dnd-kit?

**react-dnd Pros:**
- Already in use
- Less migration work
- Mature library

**dnd-kit Pros:**
- More modern
- Better performance
- Better touch support
- Better accessibility

**Decision:** Keep react-dnd for Phase 4 to reduce scope. Evaluate dnd-kit in Phase 5 if performance issues arise.

### 4. Server-Side Rendering?

React Router 7 supports SSR out of the box, but it adds complexity.

**Benefits:**
- Better SEO
- Faster initial load
- Better social sharing

**Costs:**
- More complex deployment
- Need server infrastructure
- Harder to debug
- More code complexity

**Decision:** Skip SSR for now. Liwords is an authenticated app (not SEO-critical). Focus on client-side performance first.

### 5. Backend Coordination

Socket subscription feature requires backend changes.

**Questions:**
- Who owns backend changes?
- Can backend changes be deployed independently?
- Timeline for backend work?

**Recommendation:** Backend changes can be deployed first (backwards compatible), then frontend can adopt when ready.

---

## Appendix: File Structure

### Proposed New Structure

```
src/
├── components/           # Shared Mantine components
│   ├── layout/
│   ├── forms/
│   └── data-display/
├── stores/              # Zustand stores
│   ├── ui-store.ts
│   ├── game-store.ts
│   ├── lag-store.ts
│   └── feature-flags-store.ts
├── queries/             # TanStack Query hooks
│   ├── lobby-queries.ts
│   ├── game-queries.ts
│   ├── tournament-queries.ts
│   └── user-queries.ts
├── hooks/               # Custom hooks
│   ├── useSocketSubscription.ts
│   ├── useSocketMessage.ts
│   └── useThemeSync.ts
├── theme/               # Theme configuration
│   ├── tokens.ts
│   ├── mantine-theme.ts
│   └── ant-theme.ts (temporary)
├── views/               # Main views (new Mantine versions)
│   ├── lobby/
│   ├── game/
│   ├── tournament/
│   ├── settings/
│   └── profile/
├── socket/              # Socket management
│   ├── socket.ts
│   ├── socket-context.tsx
│   ├── socket-subscription-manager.ts
│   └── socket-handlers.ts
├── providers/           # React providers
│   └── AppProviders.tsx
├── utils/               # Utilities
│   ├── error-tracking.ts
│   └── helpers.ts
└── legacy/              # Old code (to be deleted)
    ├── store/
    ├── lobby/
    └── gameroom/
```

---

## Timeline Summary

| Phase | Duration | Description | Key Deliverables |
|-------|----------|-------------|------------------|
| **Phase 1** | 2 weeks | Foundation & POC | Mantine setup, theme tokens, Settings v2 |
| **Phase 2** | 3 weeks | Core Infrastructure | Persistent socket, query integration |
| **Phase 3** | 6 weeks | View Migration | Lobby, Profile, Tournament, Chat |
| **Phase 4** | 4 weeks | Game Room | Board, rack, controls, analyzer |
| **Phase 5** | 2 weeks | Cleanup & Optimization | Remove legacy, optimize bundle |
| **Buffer** | 3 weeks | Testing & Polish | Bug fixes, performance tuning |
| **Total** | **20 weeks** | **~5 months** | Fully migrated frontend |

---

## Contact & Support

For questions about this migration plan:
- Create issue in GitHub repo
- Tag `@frontend-team` in discussions
- Refer to this doc in PRs: `docs/FRONTEND_MIGRATION_PLAN.md`

**Last Updated:** November 2025
**Document Version:** 1.0
**Status:** Phase 1 In Progress
