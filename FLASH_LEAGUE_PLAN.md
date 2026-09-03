# Flash League Implementation Plan

## Overview

Flash Leagues are on-demand, quick-start leagues where players join a pool and when the pool reaches a threshold (e.g., 11 players), an immediate round-robin league spawns with a fixed number of players (e.g., 10).

**Key Design Decision:** Reuse the existing Season model. Each flash league instance is a Season within a parent Flash League entity. This allows us to reuse ~90% of existing code (divisions, pairings, standings, game creation).

## Terminology

- **Flash League** - The parent league entity (e.g., "Flash League - CSW24 - 5+0")
- **Instance** - Each spawned season/round-robin (e.g., "Week 1 - Round 2")
- **Pool** - Players waiting for next instance (stored as registrations for next `REGISTRATION_OPEN` season)

## Naming Convention

- **Week Numbering:** ISO 8601 weeks (Monday 00:00 UTC start), continuous from league creation (Week 1, 2, 3... never resets)
- **Round Numbering:** Resets to 1 when week changes
- **Display Format:** "Week X - Round Y"
  - Example: "Week 1 - Round 3" → "Week 2 - Round 1" (new week)

## Core Mechanics

### Pool System
- Pool = registrations for the next available `REGISTRATION_OPEN` season
- When pool reaches `min_pool_size` (e.g., 11), trigger instant season start
- Take first N players (e.g., 10), start season immediately
- Remaining player(s) stay in pool for next instance
- Always maintain one `REGISTRATION_OPEN` season as the active pool

### Season Lifecycle (Modified)

**Traditional League:**
- Fixed schedule: SCHEDULED → ACTIVE (Day 1) → COMPLETED (Day 20)
- Registration opens mid-season for next season
- Promotions/relegations between seasons

**Flash League:**
- On-demand: Pool fills → SCHEDULED → ACTIVE (immediate) → COMPLETED (when done)
- Always maintain a `REGISTRATION_OPEN` season
- No promotions/relegations (each instance is independent)
- Shorter duration (e.g., 7 days instead of 20)

### Eligibility Constraint
- Players can only join pool if they have < N active league games (e.g., 7)
- Prevents players from being overwhelmed with concurrent games

## Database Schema Changes

### 1. leagues.settings (JSONB - add fields)

```json
{
  "season_length_days": 7,
  "ideal_division_size": 10,
  "is_flash_league": true,
  "min_pool_size": 11,
  "max_concurrent_league_games": 7
}
```

**New fields:**
- `is_flash_league` (bool): Flag to distinguish flash leagues from traditional
- `min_pool_size` (int32): Pool size needed to spawn instance (e.g., 11)
- `max_concurrent_league_games` (int32): Max active games to join pool (e.g., 7)

### 2. league_seasons (add columns)

```sql
ALTER TABLE league_seasons ADD COLUMN week_number INT;
ALTER TABLE league_seasons ADD COLUMN round_number INT;
ALTER TABLE league_seasons ADD COLUMN display_name TEXT;
```

**Purpose:**
- `week_number`: Continuous week count from league creation (1, 2, 3...)
- `round_number`: Round within that week (resets to 1 on new week)
- `display_name`: Precomputed "Week X - Round Y" for fast display

**Index:**
```sql
CREATE INDEX idx_league_seasons_week_round ON league_seasons(league_id, week_number, round_number);
```

## Proto Changes

### api/proto/ipc/league.proto

```protobuf
message LeagueSettings {
    int32 season_length_days = 1;
    TimeControl time_control = 3;
    string lexicon = 4;
    string variant = 5;
    int32 ideal_division_size = 6;
    ChallengeRule challenge_rule = 9;

    // Flash league settings
    bool is_flash_league = 10;
    int32 min_pool_size = 11;  // e.g., 11 players needed to spawn
    int32 max_concurrent_league_games = 12;  // e.g., 7 - constraint for joining pool
}
```

### api/proto/league_service/league_service.proto

Add new RPC methods:

```protobuf
// Flash League Pool Management
rpc JoinFlashLeaguePool(JoinFlashLeaguePoolRequest) returns (JoinFlashLeaguePoolResponse);
rpc LeaveFlashLeaguePool(LeaveFlashLeaguePoolRequest) returns (LeaveFlashLeaguePoolResponse);
rpc GetFlashLeaguePoolStatus(GetFlashLeaguePoolStatusRequest) returns (GetFlashLeaguePoolStatusResponse);

message JoinFlashLeaguePoolRequest {
    string league_id = 1;
}

message JoinFlashLeaguePoolResponse {
    bool success = 1;
    string error = 2;  // e.g., "You have 8 active league games (max 7)"
    FlashLeaguePoolStatus pool_status = 3;
}

message LeaveFlashLeaguePoolRequest {
    string league_id = 1;
}

message LeaveFlashLeaguePoolResponse {
    bool success = 1;
    FlashLeaguePoolStatus pool_status = 2;
}

message GetFlashLeaguePoolStatusRequest {
    string league_id = 1;
}

message GetFlashLeaguePoolStatusResponse {
    FlashLeaguePoolStatus pool_status = 1;
}

message FlashLeaguePoolStatus {
    int32 current_count = 1;  // e.g., 7
    int32 required_count = 2;  // e.g., 11
    repeated string player_usernames = 3;  // Players currently in pool
    bool user_in_pool = 4;  // Is requesting user in pool?
    int32 user_active_game_count = 5;  // How many active league games user has
}
```

## Backend Implementation

### 1. Helper Functions (pkg/league/flash_league.go - new file)

```go
package league

import (
    "context"
    "time"
)

// GetMondayOfWeek returns Monday 00:00 UTC of the week containing t
func GetMondayOfWeek(t time.Time) time.Time {
    // Normalize to 00:00 UTC
    t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)

    // Days to subtract to get to Monday
    weekday := int(t.Weekday())
    if weekday == 0 { // Sunday
        weekday = 7
    }
    daysToMonday := weekday - 1

    return t.AddDate(0, 0, -daysToMonday)
}

// GetFlashLeagueWeek calculates week number (continuous from league creation)
func GetFlashLeagueWeek(leagueCreatedAt time.Time, seasonStartDate time.Time) int {
    firstMonday := GetMondayOfWeek(leagueCreatedAt)
    seasonMonday := GetMondayOfWeek(seasonStartDate)

    daysDiff := seasonMonday.Sub(firstMonday).Hours() / 24
    weeksElapsed := int(daysDiff / 7)

    return weeksElapsed + 1 // Week 1, Week 2, Week 3...
}

// GetFlashLeagueRound counts existing seasons in the same week
func GetFlashLeagueRound(ctx context.Context, store LeagueStore, leagueID string, weekNumber int) (int, error) {
    count, err := store.CountSeasonsInWeek(ctx, leagueID, weekNumber)
    if err != nil {
        return 0, err
    }
    return count + 1, nil
}

// GetFlashLeagueDisplayName returns "Week X - Round Y"
func GetFlashLeagueDisplayName(weekNumber, roundNumber int) string {
    return fmt.Sprintf("Week %d - Round %d", weekNumber, roundNumber)
}
```

### 2. Database Store Methods (pkg/stores/league/db.go)

Add to `LeagueStore` interface:

```go
type LeagueStore interface {
    // ... existing methods ...

    // Flash league specific
    CountSeasonsInWeek(ctx context.Context, leagueID string, weekNumber int) (int, error)
    GetActiveLeagueGameCount(ctx context.Context, userID string) (int, error)
    GetOrCreatePoolSeason(ctx context.Context, leagueID string) (*entity.Season, error)
}
```

Implementation:

```go
func (s *DBStore) CountSeasonsInWeek(ctx context.Context, leagueID string, weekNumber int) (int, error) {
    var count int
    err := s.db.QueryRowContext(ctx, `
        SELECT COUNT(*)
        FROM league_seasons
        WHERE league_id = $1 AND week_number = $2
    `, leagueID, weekNumber).Scan(&count)
    return count, err
}

func (s *DBStore) GetActiveLeagueGameCount(ctx context.Context, userID string) (int, error) {
    var count int
    err := s.db.QueryRowContext(ctx, `
        SELECT COUNT(DISTINCT g.uuid)
        FROM games g
        JOIN league_seasons ls ON g.season_id = ls.uuid
        WHERE ls.status = $1
        AND (g.player1_id = $2 OR g.player2_id = $2)
        AND g.game_end_reason = 0  -- NONE (game still active)
    `, entity.SeasonStatusActive, userID).Scan(&count)
    return count, err
}

func (s *DBStore) GetOrCreatePoolSeason(ctx context.Context, leagueID string) (*entity.Season, error) {
    // Try to find existing REGISTRATION_OPEN season
    season, err := s.GetSeasonByStatus(ctx, leagueID, entity.SeasonStatusRegistrationOpen)
    if err == nil {
        return season, nil
    }

    // Create new pool season
    league, err := s.GetLeague(ctx, leagueID)
    if err != nil {
        return nil, err
    }

    now := time.Now()
    endDate := now.Add(time.Duration(league.Settings.SeasonLengthDays) * 24 * time.Hour)

    // Get next season number
    lastSeason, _ := s.GetLatestSeason(ctx, leagueID)
    seasonNum := 1
    if lastSeason != nil {
        seasonNum = lastSeason.SeasonNumber + 1
    }

    // Calculate week/round for display name (even though it's just the pool)
    weekNum := GetFlashLeagueWeek(league.CreatedAt, now)
    roundNum, _ := GetFlashLeagueRound(ctx, s, leagueID, weekNum)

    newSeason := &entity.Season{
        UUID: uuid.New().String(),
        LeagueID: leagueID,
        SeasonNumber: seasonNum,
        StartDate: now,
        EndDate: endDate,
        Status: entity.SeasonStatusRegistrationOpen,
        WeekNumber: weekNum,
        RoundNumber: roundNum,
        DisplayName: GetFlashLeagueDisplayName(weekNum, roundNum),
    }

    return s.CreateSeason(ctx, newSeason)
}
```

### 3. Service Methods (pkg/league/service.go)

```go
func (s *LeagueService) JoinFlashLeaguePool(ctx context.Context, leagueID, userID string) (*pb.FlashLeaguePoolStatus, error) {
    // 1. Verify league is a flash league
    league, err := s.store.GetLeague(ctx, leagueID)
    if err != nil {
        return nil, err
    }
    if !league.Settings.IsFlashLeague {
        return nil, errors.New("not a flash league")
    }

    // 2. Check user's active game count
    activeCount, err := s.store.GetActiveLeagueGameCount(ctx, userID)
    if err != nil {
        return nil, err
    }
    if activeCount >= league.Settings.MaxConcurrentLeagueGames {
        return nil, fmt.Errorf("you have %d active league games (max %d)",
            activeCount, league.Settings.MaxConcurrentLeagueGames)
    }

    // 3. Get or create pool season
    poolSeason, err := s.store.GetOrCreatePoolSeason(ctx, leagueID)
    if err != nil {
        return nil, err
    }

    // 4. Create registration
    err = s.store.CreateRegistration(ctx, &entity.Registration{
        UserID: userID,
        SeasonID: poolSeason.UUID,
        RegistrationDate: time.Now(),
        Status: "ACTIVE",
    })
    if err != nil {
        return nil, err
    }

    // 5. Check if pool is full and trigger instant start
    poolStatus, err := s.GetFlashLeaguePoolStatus(ctx, leagueID, userID)
    if err != nil {
        return nil, err
    }

    if poolStatus.CurrentCount >= league.Settings.MinPoolSize {
        // Trigger instant start asynchronously
        go s.StartFlashLeagueInstance(context.Background(), leagueID, poolSeason.UUID, league.Settings)
    }

    return poolStatus, nil
}

func (s *LeagueService) LeaveFlashLeaguePool(ctx context.Context, leagueID, userID string) (*pb.FlashLeaguePoolStatus, error) {
    poolSeason, err := s.store.GetOrCreatePoolSeason(ctx, leagueID)
    if err != nil {
        return nil, err
    }

    err = s.store.DeleteRegistration(ctx, userID, poolSeason.UUID)
    if err != nil {
        return nil, err
    }

    return s.GetFlashLeaguePoolStatus(ctx, leagueID, userID)
}

func (s *LeagueService) GetFlashLeaguePoolStatus(ctx context.Context, leagueID, userID string) (*pb.FlashLeaguePoolStatus, error) {
    league, err := s.store.GetLeague(ctx, leagueID)
    if err != nil {
        return nil, err
    }

    poolSeason, err := s.store.GetOrCreatePoolSeason(ctx, leagueID)
    if err != nil {
        return nil, err
    }

    registrations, err := s.store.GetSeasonRegistrations(ctx, poolSeason.UUID)
    if err != nil {
        return nil, err
    }

    usernames := make([]string, len(registrations))
    userInPool := false
    for i, reg := range registrations {
        usernames[i] = reg.Username
        if reg.UserID == userID {
            userInPool = true
        }
    }

    activeCount, _ := s.store.GetActiveLeagueGameCount(ctx, userID)

    return &pb.FlashLeaguePoolStatus{
        CurrentCount: int32(len(registrations)),
        RequiredCount: int32(league.Settings.MinPoolSize),
        PlayerUsernames: usernames,
        UserInPool: userInPool,
        UserActiveGameCount: int32(activeCount),
    }, nil
}
```

### 4. Instant Start Logic (pkg/league/flash_league.go)

```go
func (s *LeagueService) StartFlashLeagueInstance(ctx context.Context, leagueID, poolSeasonID string, settings *entity.LeagueSettings) error {
    // 1. Get pool registrations
    registrations, err := s.store.GetSeasonRegistrations(ctx, poolSeasonID)
    if err != nil {
        return err
    }

    // Take first N players (e.g., 10 out of 11)
    playersToStart := settings.IdealDivisionSize
    if len(registrations) < playersToStart {
        return fmt.Errorf("not enough players: %d < %d", len(registrations), playersToStart)
    }

    selectedPlayers := registrations[:playersToStart]
    remainingPlayers := registrations[playersToStart:]

    // 2. Close pool season and prepare it as the instance
    poolSeason, err := s.store.GetSeason(ctx, poolSeasonID)
    if err != nil {
        return err
    }

    // Update season status and start date
    poolSeason.Status = entity.SeasonStatusScheduled
    poolSeason.StartDate = time.Now()
    err = s.store.UpdateSeason(ctx, poolSeason)
    if err != nil {
        return err
    }

    // 3. Create single division with selected players
    division, err := s.createDivision(ctx, poolSeasonID, 1, "Main")
    if err != nil {
        return err
    }

    // Assign selected players to division
    for _, reg := range selectedPlayers {
        err = s.store.UpdateRegistrationDivision(ctx, reg.UserID, poolSeasonID, division.UUID)
        if err != nil {
            return err
        }
    }

    // 4. Remove remaining players from this season
    for _, reg := range remainingPlayers {
        err = s.store.DeleteRegistration(ctx, reg.UserID, poolSeasonID)
        if err != nil {
            return err
        }
    }

    // 5. Start the season (creates games)
    err = StartSeason(ctx, s.store, poolSeasonID)
    if err != nil {
        return err
    }

    // 6. Create new pool season for next instance
    _, err = s.store.GetOrCreatePoolSeason(ctx, leagueID)
    if err != nil {
        return err
    }

    // 7. Re-add remaining players to new pool
    newPoolSeason, _ := s.store.GetOrCreatePoolSeason(ctx, leagueID)
    for _, reg := range remainingPlayers {
        s.store.CreateRegistration(ctx, &entity.Registration{
            UserID: reg.UserID,
            SeasonID: newPoolSeason.UUID,
            RegistrationDate: time.Now(),
            Status: "ACTIVE",
        })
    }

    return nil
}
```

### 5. Modified End-of-Season Logic (pkg/league/end_of_season.go)

```go
func CloseSeason(ctx context.Context, store LeagueStore, seasonID string) error {
    season, err := store.GetSeason(ctx, seasonID)
    if err != nil {
        return err
    }

    league, err := store.GetLeague(ctx, season.LeagueID)
    if err != nil {
        return err
    }

    // Force-finish unfinished games
    err = ForceFinishGames(ctx, store, seasonID)
    if err != nil {
        return err
    }

    // Calculate final standings
    err = CalculateStandings(ctx, store, seasonID)
    if err != nil {
        return err
    }

    // Mark outcomes (promotions/relegations) - SKIP for flash leagues
    if !league.Settings.IsFlashLeague {
        err = MarkStandingOutcomes(ctx, store, seasonID, season.PromotionFormula)
        if err != nil {
            return err
        }
    }

    // Mark season as complete
    season.Status = entity.SeasonStatusCompleted
    season.ActualEndDate = time.Now()
    err = store.UpdateSeason(ctx, season)
    if err != nil {
        return err
    }

    return nil
}
```

### 6. Maintenance Task Updates (cmd/maintenance/league_tasks.go)

The existing hourly runner should handle flash leagues automatically:
- **Close season**: Works as-is (just skips promotions)
- **Start season**: Not needed (flash leagues start instantly)
- **Open registration**: Not needed (always have a pool season)

Add check to skip certain operations for flash leagues:

```go
func (r *LeagueHourlyRunner) Run(ctx context.Context) error {
    leagues, err := r.store.GetActiveLeagues(ctx)
    if err != nil {
        return err
    }

    for _, league := range leagues {
        if league.Settings.IsFlashLeague {
            // For flash leagues, only check for season closure
            currentSeason, err := r.getCurrentSeason(ctx, league.UUID)
            if err != nil {
                continue
            }

            if currentSeason != nil && time.Now().After(currentSeason.EndDate) {
                r.closeSeason(ctx, currentSeason.UUID)
            }
        } else {
            // Traditional league logic
            r.handleTraditionalLeague(ctx, league)
        }
    }

    return nil
}
```

## Frontend Implementation

### 1. New Component: FlashLeaguePoolWidget.tsx

```tsx
import React, { useState, useEffect } from 'react';
import { Button, Card, List, Space, Typography } from 'antd';
import { useMountedState } from '../utils/hooks/useDebounce';

const { Text, Title } = Typography;

interface FlashLeaguePoolWidgetProps {
  leagueId: string;
}

export const FlashLeaguePoolWidget: React.FC<FlashLeaguePoolWidgetProps> = ({ leagueId }) => {
  const [poolStatus, setPoolStatus] = useState<FlashLeaguePoolStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const isMounted = useMountedState();

  useEffect(() => {
    fetchPoolStatus();
    const interval = setInterval(fetchPoolStatus, 5000); // Poll every 5s
    return () => clearInterval(interval);
  }, [leagueId]);

  const fetchPoolStatus = async () => {
    const resp = await leagueService.getFlashLeaguePoolStatus({ league_id: leagueId });
    if (isMounted()) {
      setPoolStatus(resp.pool_status);
    }
  };

  const handleJoin = async () => {
    setLoading(true);
    try {
      const resp = await leagueService.joinFlashLeaguePool({ league_id: leagueId });
      if (resp.success) {
        setPoolStatus(resp.pool_status);
      } else {
        message.error(resp.error);
      }
    } catch (err) {
      message.error('Failed to join pool');
    } finally {
      setLoading(false);
    }
  };

  const handleLeave = async () => {
    setLoading(true);
    try {
      await leagueService.leaveFlashLeaguePool({ league_id: leagueId });
      fetchPoolStatus();
    } catch (err) {
      message.error('Failed to leave pool');
    } finally {
      setLoading(false);
    }
  };

  if (!poolStatus) return <Spin />;

  const canJoin = !poolStatus.user_in_pool &&
                  poolStatus.user_active_game_count < 7; // Or read from settings

  return (
    <Card title="Flash League Pool">
      <Space direction="vertical" style={{ width: '100%' }}>
        <Title level={4}>
          Pool: {poolStatus.current_count} / {poolStatus.required_count} players
        </Title>

        {poolStatus.user_in_pool ? (
          <Button onClick={handleLeave} loading={loading}>
            Leave Pool
          </Button>
        ) : (
          <Button
            type="primary"
            onClick={handleJoin}
            loading={loading}
            disabled={!canJoin}
          >
            Join Pool
          </Button>
        )}

        {!canJoin && !poolStatus.user_in_pool && (
          <Text type="danger">
            You have {poolStatus.user_active_game_count} active league games (max 7)
          </Text>
        )}

        <List
          header={<div>Players in Pool</div>}
          bordered
          dataSource={poolStatus.player_usernames}
          renderItem={(username) => <List.Item>{username}</List.Item>}
        />
      </Space>
    </Card>
  );
};
```

### 2. Modified LeaguePage.tsx

```tsx
export const LeaguePage: React.FC = () => {
  const { leagueId } = useParams();
  const [league, setLeague] = useState<League | null>(null);

  useEffect(() => {
    // Fetch league data
  }, [leagueId]);

  if (!league) return <Spin />;

  return (
    <div className="league-page">
      <h1>{league.name}</h1>
      <p>{league.description}</p>

      {league.settings.is_flash_league ? (
        <>
          <FlashLeaguePoolWidget leagueId={league.uuid} />
          <FlashLeagueInstanceList leagueId={league.uuid} />
        </>
      ) : (
        <>
          <SeasonSelector leagueId={league.uuid} />
          <DivisionStandings />
        </>
      )}
    </div>
  );
};
```

### 3. New Component: FlashLeagueInstanceList.tsx

```tsx
export const FlashLeagueInstanceList: React.FC<{ leagueId: string }> = ({ leagueId }) => {
  const [seasons, setSeasons] = useState<Season[]>([]);

  useEffect(() => {
    // Fetch all seasons for this league (excluding REGISTRATION_OPEN)
    fetchSeasons();
  }, [leagueId]);

  return (
    <div>
      <h2>Your Flash League Instances</h2>
      <List
        dataSource={seasons}
        renderItem={(season) => (
          <List.Item>
            <Link to={`/league/${leagueId}/season/${season.uuid}`}>
              {season.display_name} {/* e.g., "Week 1 - Round 2" */}
            </Link>
            <Text type="secondary">
              {formatDate(season.start_date)} - {season.status}
            </Text>
            {/* Show user's standing if available */}
          </List.Item>
        )}
      />
    </div>
  );
};
```

## Migration Script

```sql
-- Add flash league fields to league_seasons
ALTER TABLE league_seasons ADD COLUMN week_number INT;
ALTER TABLE league_seasons ADD COLUMN round_number INT;
ALTER TABLE league_seasons ADD COLUMN display_name TEXT;

-- Index for fast week/round lookups
CREATE INDEX idx_league_seasons_week_round ON league_seasons(league_id, week_number, round_number);

-- No need to alter leagues.settings (JSONB is schemaless)
-- Just add the new fields when creating flash leagues
```

## Testing Plan

### Unit Tests

1. **Week/Round Calculation** (`pkg/league/flash_league_test.go`)
   - Test `GetMondayOfWeek` for various dates (Mon, Sun, Wed)
   - Test `GetFlashLeagueWeek` for league creation scenarios
   - Test week transitions (Sunday night → Monday)
   - Test round incrementing within same week
   - Test round reset on new week

2. **Pool Management** (`pkg/league/service_test.go`)
   - Test joining pool with valid user
   - Test joining pool with too many active games
   - Test leaving pool
   - Test pool status calculation
   - Test concurrent joins (race conditions)

3. **Instant Start** (`pkg/league/flash_league_test.go`)
   - Test triggering start at exact threshold (11 players)
   - Test player selection (first 10 of 11)
   - Test remaining player moved to new pool
   - Test division creation
   - Test game creation

### Integration Tests

1. **Full Flash League Flow**
   - Create flash league
   - 11 players join pool
   - Verify instance spawns
   - Verify 10 players in game, 1 in new pool
   - Complete games
   - Verify season closes
   - Verify no promotions/relegations

2. **Concurrent Instances**
   - Multiple pools filling simultaneously
   - Verify correct week/round numbering
   - Verify no cross-contamination

### Manual Testing Checklist

- [ ] Create flash league via admin UI
- [ ] Join pool as multiple users
- [ ] Verify pool fills and instance spawns
- [ ] Play games in instance
- [ ] Verify standings update correctly
- [ ] Complete instance and verify no promotions
- [ ] Join multiple instances and verify week/round naming
- [ ] Test eligibility constraint (max concurrent games)
- [ ] Test leaving pool
- [ ] Test edge cases (pool fills while you're viewing it)

## Rollout Plan

### Phase 1: Database + Proto (No UI)
1. Migration script for new columns
2. Proto changes + regenerate
3. Backend helper functions (week/round calculation)
4. Database store methods

### Phase 2: Backend Service Layer
1. Pool management endpoints (join/leave/status)
2. Instant start logic
3. Modified end-of-season logic
4. Unit tests

### Phase 3: Frontend
1. Pool widget component
2. Instance list component
3. Modified league page routing
4. Integration with existing league UI

### Phase 4: Admin Tools
1. Flash league creation UI
2. Pool monitoring dashboard
3. Manual instance triggering (for testing)

### Phase 5: Launch
1. Integration testing
2. Soft launch (beta users only)
3. Monitor for issues
4. Full rollout

## Open Questions / Future Enhancements

1. **Pool timeout**: Should players be auto-removed from pool after X hours of inactivity?
2. **Notifications**: Notify pool members when instance spawns? When pool is 1 away from full?
3. **Multiple pools**: Allow multiple concurrent pools per flash league (different skill levels)?
4. **Partial pool start**: Allow starting with <10 players if pool stagnates?
5. **Rating-based matching**: Group similar-rated players into instances?
6. **Custom pool sizes**: Let admins configure min/max pool sizes per flash league?
7. **Stats tracking**: Aggregate stats across all flash league instances for a player?
8. **Leaderboards**: Overall flash league leaderboards (not just per-instance)?

## Files to Create/Modify

### New Files
- `pkg/league/flash_league.go` - Core flash league logic
- `pkg/league/flash_league_test.go` - Unit tests
- `liwords-ui/src/leagues/flash_league_pool_widget.tsx` - Pool UI
- `liwords-ui/src/leagues/flash_league_instance_list.tsx` - Instance list
- `db/migrations/YYYYMMDDXXXX_add_flash_league_support.up.sql` - Migration

### Modified Files
- `api/proto/ipc/league.proto` - Add flash league settings
- `api/proto/league_service/league_service.proto` - Add pool endpoints
- `pkg/stores/league/db.go` - Add store methods
- `pkg/league/service.go` - Add service methods
- `pkg/league/end_of_season.go` - Skip promotions for flash leagues
- `cmd/maintenance/league_tasks.go` - Flash league handling
- `liwords-ui/src/leagues/league_page.tsx` - Conditional rendering
- Generated proto files (via `go generate`)

## Estimated Effort

- **Backend**: 2-3 days
- **Frontend**: 2 days
- **Testing**: 1-2 days
- **Total**: ~1 week for core implementation
- **Polish + edge cases**: +1-2 days

## Success Metrics

- Flash league instances spawn reliably when pool fills
- No race conditions in pool management
- Players cannot exceed concurrent game limit
- Week/round naming is consistent and intuitive
- Traditional leagues unaffected by changes
- Average time in pool before instance spawns: <X hours
- Player retention in flash leagues vs traditional leagues
