# Presence

Presence says who is in which chat channel (the lists beside each chat) and
which channels and games each user is in, which their followers see.

Every member of these sorted sets has its expiry time as its score. A member
is live while that time is after now. Everything that reads skips members
that have expired.

## Keys

- `channelpresence:<channel>`: who is in a chat channel. Members are
  `uuid#username#auth|anon#connID`. The key expires 3 minutes after its
  last write.
- `userpresence:<uuid>`: the channels a user's connections are in. Members
  are `connID#channel`. Same TTL.
- `userpresences`: every connection in every channel, site-wide. Members are
  `uuid#username#auth|anon#connID#channel`. This is the index the sweep reads.
  It could later give a "who's online" count or list.
- `activeusergames:<uuid>`: the real-time games a user is playing. Members are
  game IDs, each expiring 12 hours after it starts unless the game ends
  first. The key expires with its last game.
- `closedconn:<connID>`: a string set for 3 minutes when a tab closes.
  Renewals skip such connections, so a pong handled after the close (in
  another goroutine, or on another API server) can't put the tab back.

Channels are only the `chat.*` ones (lobby, tournament, game and spectator
chat). A user's channels as followers see them also hold
`activegame:<gameID>` for each game they're playing.

## Scripts

Each script runs atomically and returns the user's channels before and
after, so the API can tell followers (`presence.changed.<uuid>` on NATS)
when they change.

- `set_presence.lua`: a tab joins a channel. Called when a socket connects
  (`initRealmInfo`).
- `renew_presence.lua`: a tab is still there. Called every tenth pong, about
  every 50 s; entries last 3 minutes.
- `clear_presence.lua`: a tab closed (`leaveTab`). Also returns the channels
  the user has left altogether.
- `get_channels.lua`: the channels of any number of users, for the follows
  list.
- `update_active_game.lua`: a game starts or ends.
- `sweep_presence.lua`: removes the entries of connections that went away
  without clearing them (a socket server that crashed or was redeployed, a
  network that dropped).

## Jobs (pkg/bus/presence_jobs.go)

- **`presence_sweep`, every 30 s:** sweeps expired `userpresences` entries,
  then announces that those users left the chats no other tab of theirs is
  in, and tells their followers.
  - Span `presence.sweep`; attributes `presence.swept`, `presence.users`,
    `presence.left_notices`, `presence.batches`.
- **`active_games_reconcile`, every 10 min:** makes `activeusergames` match
  the started real-time games in Postgres (`ongoing_games`).
  - Adds missing entries, for example after Redis lost its data.
  - Removes those of games that ended without removing theirs.
  - Span `presence.reconcile_active_games`; attributes `presence.games`,
    `presence.entries`, `presence.added`, `presence.removed`.

The store's calls have spans named `presence.*` (scope `presence-store`).
Renewals, the busiest call, add their attributes to the pong's span instead
of making one.
