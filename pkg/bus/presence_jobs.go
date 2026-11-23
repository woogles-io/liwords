package bus

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// endSpan records err on span and ends it.
func endSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

const (
	sweepBatch      = 500
	sweepMaxBatches = 20
)

// SweepPresence clears the presence of connections that went away without
// clearing it (a socket server that crashed or was redeployed, a network
// that dropped), and tells the rooms they were in, and their followers,
// that they left. Run every 30 seconds; such an entry expires 3 minutes
// after its connection's last renewal.
func (b *Bus) SweepPresence(ctx context.Context) (err error) {
	ctx, span := otel.Tracer("presence-jobs").Start(ctx, "presence.sweep")
	swept, users, left, batches := 0, 0, 0, 0
	defer func() {
		span.SetAttributes(
			attribute.Int("presence.swept", swept),
			attribute.Int("presence.users", users),
			attribute.Int("presence.left_notices", left),
			attribute.Int("presence.batches", batches))
		endSpan(span, err)
	}()

	for batches < sweepMaxBatches {
		batches++
		n, gone, err := b.stores.PresenceStore.SweepPresence(ctx, sweepBatch)
		if err != nil {
			return err
		}
		swept += n
		users += len(gone)
		for _, u := range gone {
			if len(u.Removed) > 0 {
				left += len(u.Removed)
				if err := b.broadcastPresence(u.Username, u.UUID, u.Anon, u.Removed, true); err != nil {
					return err
				}
			}
			if err := b.broadcastChannelChanges(ctx, u.Before, u.After, u.UUID, u.Username); err != nil {
				return err
			}
		}
		if n < sweepBatch {
			break
		}
	}
	if swept > 0 {
		log.Info().Int("swept", swept).Int("users", users).Int("left-notices", left).Msg("presence-swept")
	}
	return nil
}

// ReconcileActiveGames makes the active-game presence entries (who is
// playing which game, shown to followers) match the started real-time
// games in Postgres: it adds those missing, as after a Redis restart, and
// removes those of games that ended without removing theirs.
func (b *Bus) ReconcileActiveGames(ctx context.Context) (err error) {
	ctx, span := otel.Tracer("presence-jobs").Start(ctx, "presence.reconcile_active_games")
	var games, entries, added, removed int
	defer func() {
		span.SetAttributes(
			attribute.Int("presence.games", games),
			attribute.Int("presence.entries", entries),
			attribute.Int("presence.added", added),
			attribute.Int("presence.removed", removed))
		endSpan(span, err)
	}()

	// Entries written after this were written by games starting since; the
	// rows below may not show those games.
	snapshot := time.Now()
	rows, err := b.stores.Queries.ListStartedRealtimeGamePlayers(ctx)
	if err != nil {
		return err
	}
	games = len(rows)
	have, err := b.stores.PresenceStore.ListActiveGames(ctx)
	if err != nil {
		return err
	}
	entries = len(have)

	type key struct{ user, game string }
	want := make(map[key]string, 2*len(rows)) // to the username
	for _, r := range rows {
		want[key{r.Player0Uuid, r.GameUuid}] = r.Player0Username
		want[key{r.Player1Uuid, r.GameUuid}] = r.Player1Username
	}
	present := make(map[key]bool, len(have))
	for _, e := range have {
		present[key{e.UserID, e.GameID}] = true
	}

	ttl := int64(entity.ActiveGameTTL / time.Second)
	for k, username := range want {
		if present[k] {
			continue
		}
		if err := b.setActiveGame(ctx, k.game, ttl, k.user, username); err != nil {
			return err
		}
		added++
	}
	// A minute's margin for the clocks of Redis and this server.
	writtenBefore := snapshot.Add(entity.ActiveGameTTL - time.Minute)
	for _, e := range have {
		if _, ok := want[key{e.UserID, e.GameID}]; ok || !e.Expires.Before(writtenBefore) {
			continue
		}
		username, err := b.stores.UserStore.Username(ctx, e.UserID)
		if err != nil {
			log.Err(err).Str("userID", e.UserID).Msg("reconcile-active-games-username")
		}
		if err := b.setActiveGame(ctx, e.GameID, 0, e.UserID, username); err != nil {
			return err
		}
		removed++
	}
	if added > 0 || removed > 0 {
		log.Info().Int("games", games).Int("added", added).Int("removed", removed).
			Msg("active-games-reconciled")
	}
	return nil
}

// setActiveGame records that userID is playing gameID (for ttl seconds) or,
// with a ttl of 0, isn't, and tells their followers if that changes their
// channels.
func (b *Bus) setActiveGame(ctx context.Context, gameID string, ttl int64, userID, username string) error {
	ret, err := b.stores.PresenceStore.UpdateActiveGame(ctx, &pb.ActiveGameEntry{
		Id:     gameID,
		Ttl:    ttl,
		Player: []*pb.ActiveGamePlayer{{UserId: userID, Username: username}},
	})
	if err != nil {
		return err
	}
	if username == "" {
		return nil
	}
	return b.broadcastChannelChanges(ctx, ret[0][0], ret[0][1], userID, username)
}
