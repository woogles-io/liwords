package sessions

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

// Revoker logs a user out everywhere: it deletes their sessions and tells the
// socket servers to close their open connections.
type Revoker struct {
	store    SessionStore
	natsconn *nats.Conn
}

func NewRevoker(store SessionStore, natsconn *nats.Conn) *Revoker {
	return &Revoker{store: store, natsconn: natsconn}
}

func (r *Revoker) RevokeUserSessions(ctx context.Context, userUUID string) error {
	n, err := r.store.DeleteForUser(ctx, userUUID)
	if err != nil {
		return err
	}
	zerolog.Ctx(ctx).Info().Str("userID", userUUID).Int64("sessions", n).Msg("revoked-user-sessions")
	if r.natsconn == nil {
		return nil
	}
	return r.natsconn.Publish("sessionend."+userUUID, []byte{})
}
