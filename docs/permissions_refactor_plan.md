# Frontend Permissions Refactor + Self-Serve Tournament Gating

## Context

Branch `claude/self-serve-tournament-creation-VajiL` added a self-serve
tournament creation wizard whose visibility is gated by
`loginState.perms.includes("toc") || loginState.perms.includes("adm")`. The
check is wrong in shape: the backend is already fully permission-based
(`pkg/auth/rbac` + `HasPermission` everywhere), but ships a hand-curated,
lossy `"adm"/"mod"/"toc"` short-code list through the JWT that the frontend
consumes.

Two consequences:

1. The JWT mapping is **lossy** — `TournamentCreator` and `TournamentManager`
   both collapse to `"toc"` even though only the latter bypasses the
   3-tournaments/week rate limit in `NewTournament`, so the button shows but
   the action silently limits. New permissions (`CanCreateBroadcasts`,
   `CanManageLeagues`, `CanPlayLeagues`, `CanInviteToLeagues`, …) aren't
   exposed at all.
2. The frontend has grown **three parallel channels** for gating UI:
   JWT-perm short codes (~10 call sites), `getSelfRoles` + full role-name
   string matches (~5 call sites), and `GetModList` UUID sets (chat badges).
   None of them match the backend's permission checks.

**Goal:** make the frontend's source of truth a single permission set that
mirrors the backend's `rbac.Permission` constants, delivered via the existing
`GetSelfRoles` RPC, and migrate all call sites behind one `usePermissions`
hook. Then land the self-serve branch's button/wizard on top of that
mechanism.

## Approach

Refactor permissions **first** (new master-based commits), then rebase the
self-serve branch on top and switch its gating to the new hook.

### Step 1 — Backend: expand `GetSelfRoles` to return permissions

- Add a new sqlc query in `db/queries/rbac.sql`:
  ```sql
  -- name: GetUserPermissions :many
  SELECT DISTINCT p.code
  FROM user_roles ur
  JOIN role_permissions rp ON rp.role_id = ur.role_id
  JOIN permissions p ON p.id = rp.permission_id
  WHERE ur.user_id = @user_id
  ORDER BY p.code;
  ```
  Keep `admin_all_access` as-is in the result — let the frontend helper treat
  it as a wildcard, exactly mirroring `HasPermission` (db/queries/rbac.sql:1-16).
- Run `go generate` from the repo root to regenerate `pkg/stores/models/rbac.sql.go`.
- Add `rbac.UserPermissions(ctx, q, userID uint) ([]string, error)` in
  `pkg/auth/rbac/rbac.go` as a thin wrapper (symmetric with the existing
  `UserRoles`).
- Proto change in `api/proto/user_service/user_service.proto`:
  add a `repeated string permissions = 2;` field to the existing
  `UserRolesResponse` (proto field addition is backwards-compatible on the
  wire). Populate permissions only on `GetSelfRoles`
  (saves the join on admin screens that call `GetUserRoles` for a named
  user).
- `pkg/auth/authz_service.go:243-257` `GetSelfRoles`: after
  `rbac.UserRoles(...)`, also call `rbac.UserPermissions(ctx, as.q, user.ID)`
  and put the codes on the response.
- Re-run `go generate` to regenerate Go + TS proto stubs.

### Step 2 — Backend: delete the JWT `perms` short-code mapping

- In `pkg/auth/authn_service.go:206-236`, remove the role→short-code
  translation loop and the `"perms"` JWT claim entirely. The JWT should only
  carry `uid`/`unn`/`a`/`cs`/`exp` going forward (socket server uses these;
  confirm no socketsrv code reads `perms` — grep
  `services/socketsrv/` and the separate `liwords-socket` repo before
  committing).
- Update the log line that prints `perms` (authn_service.go:249) to drop
  that field.

### Step 3 — Frontend: introduce a single source of truth

- Replace `liwords-ui/src/store/login_state.ts`:
  - Remove `perms: Array<string>` from `LoginState` and `AuthInfo`.
- Replace `liwords-ui/src/socket/socket.ts`:
  - Drop `perms` from `DecodedToken` (lines 43-49) and from the
    `SetAuthentication` dispatch (line 231).
- Use the connect-query cache — it's already in the
  codebase and avoids a new reducer.
- Create a new hook
  `liwords-ui/src/utils/hooks/usePermissions.ts` exposing:
  ```ts
  export const usePermissions = () => {
    const { loginState } = useLoginStateStoreContext();
    const { data } = useQuery(
      getSelfRoles,
      {},
      { enabled: loginState.loggedIn },
    );
    const permissions = data?.permissions ?? [];
    const roles = data?.roles ?? [];
    const isAdmin = permissions.includes("admin_all_access");
    const can = (p: string) =>
      isAdmin || permissions.includes(p);
    const hasRole = (r: string) => roles.includes(r);
    return { permissions, roles, can, hasRole, isAdmin, isLoaded: !!data };
  };
  ```
  Export string constants mirroring `rbac.Permission` (e.g.
  `CAN_CREATE_TOURNAMENTS = "can_create_tournaments"`) in the same file so
  call sites don't stringly-type permission names.

### Step 4 — Frontend: migrate all existing call sites to `usePermissions`

JWT-perm short-code call sites (replace `loginState.perms.includes("adm")`
etc.):

| File                                                      | Old check                      | New check                                                 |
| --------------------------------------------------------- | ------------------------------ | --------------------------------------------------------- |
| `lobby/active_games.tsx:42-44,218`                        | `perms.includes("adm")`        | `can(ADMIN_ALL_ACCESS)` or `isAdmin`                      |
| `lobby/gameLists.tsx:58,62-72`                            | `perms.includes("adm")`        | `isAdmin`                                                 |
| `broadcasts/BroadcastsList.tsx:120`                       | `perms.includes("adm")`        | `can(CAN_CREATE_BROADCASTS)`                              |
| `broadcasts/BroadcastRoom.tsx:135,224,335,372,425,429`    | `perms.includes("adm")`        | `can(CAN_CREATE_BROADCASTS)` or `isAdmin` (case-by-case)  |
| `broadcasts/EditBroadcast.tsx:49-52`                      | `perms.includes("adm") \|\| …` | `can(CAN_CREATE_BROADCASTS) \|\| creator === username`    |
| `broadcasts/CreateBroadcast.tsx:40-48`                    | `perms.includes("adm")`        | `can(CAN_CREATE_BROADCASTS)`                              |
| `gameroom/table.tsx:466-474`                              | `!perms.includes("adm")`       | `!isAdmin` (observer bypass)                              |
| `shared/usernameWithContext.tsx:57,199,205`               | `canMod(perms)`                | `can(CAN_MODERATE_USERS)`                                 |
| `gameroom/comments.tsx:22,103,212`                        | `canMod(perms)`                | `can(CAN_MODERATE_USERS)`                                 |
| `mod/perms.ts`                                            | helper                         | **delete** — replaced by `can()` / `isAdmin`              |

`getSelfRoles` role-name string match call sites:

| File                                                 | Old check                                                                             | New check                                                 |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------- | --------------------------------------------------------- |
| `tournament/room.tsx:71-75,162-167`                  | `roles.includes("Admin") \|\| "Tournament Manager"`                                   | `can(CAN_MANAGE_TOURNAMENTS)`                             |
| `leagues/leagues_list.tsx:27-31,33-40`               | `roles.includes("League Promoter") \|\| "Admin" \|\| "Manager"`                       | `can(CAN_INVITE_TO_LEAGUES)`                              |
| `leagues/league_page.tsx:150,414-420`                | `roles.includes("Admin") \|\| "Manager" \|\| "League Promoter"`                       | `can(CAN_MANAGE_LEAGUES)`                                 |
| `lobby/seek_form.tsx:51,199-203,640`                 | `ourRoles.roles.includes("Special Access Player")`                                    | `can(CAN_BYPASS_ELITEBOT_PAYWALL)`                        |
| `settings/roles_permissions.tsx:47,70,74-77`         | Displays roles + derived perms                                                        | Use `usePermissions().roles` + `.permissions` directly     |

### Step 5 — Rebase self-serve branch on top; swap its gates

- `tournaments_page.tsx` and `tournament_wizard.tsx`: use `can(CAN_CREATE_TOURNAMENTS)`
- Backend rate limit stays as-is (`CanManageTournaments` bypass)

### Step 6 — Chat badges (`GetModList`)

Leave untouched — it's an unauthenticated endpoint for presentation-only
chat badges, not a gating check.

## Critical Files

**Backend**
- `db/queries/rbac.sql` — add `GetUserPermissions` query
- `pkg/auth/rbac/rbac.go` — add `UserPermissions` helper
- `pkg/auth/authz_service.go:243-257` — populate permissions on `GetSelfRoles`
- `pkg/auth/authn_service.go:206-236` — delete JWT short-code mapping
- `api/proto/user_service/user_service.proto:365` — add `permissions` field

**Frontend (plumbing)**
- `liwords-ui/src/store/login_state.ts` — remove `perms`
- `liwords-ui/src/socket/socket.ts` — stop decoding `perms`
- `liwords-ui/src/utils/hooks/usePermissions.ts` — new hook + constants
- `liwords-ui/src/mod/perms.ts` — delete

**Frontend (migrations)**
- ~10 JWT-perm call sites listed in Step 4
- ~5 `getSelfRoles` call sites listed in Step 4
- 2 new self-serve branch sites

## Verification

1. Run `go generate` from repo root after proto + sqlc changes
2. Backend unit check: `GetSelfRoles` for TournamentCreator should include
   `can_create_tournaments` but NOT `can_manage_tournaments`; Admin should
   include `admin_all_access`
3. Frontend smoke: exercise each migrated gate with appropriate test users
4. Diff audit: grep confirms zero remaining references to
   `loginState.perms`, `"adm"`/`"mod"`/`"toc"`, `canMod`, and role-string
   `roles.includes(` checks
5. Socket server check: verify no consumer of JWT `perms` claim in
   `services/socketsrv/` or `liwords-socket/` before deletion
