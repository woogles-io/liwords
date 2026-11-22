package auth

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

// Failed logins are counted in Postgres, and an attempt that is over a limit is
// turned away before its password is hashed. Each limit stops a different
// kind of guessing:
//
//   - one address trying one account: a few tries, then a wait;
//   - one address trying many accounts;
//   - many addresses trying one account. This one could lock the owner out,
//     so it doesn't apply to a browser that has logged into the account before.
const (
	failWindowShort = 15 * time.Minute
	failWindowLong  = time.Hour

	maxFailsUserIP = 5
	maxFailsIP     = 20
	maxFailsUser   = 100

	// How long a browser that logged in is remembered for an account.
	knownClientTTL = 90 * 24 * time.Hour
)

type loginLimiter struct {
	q *models.Queries
}

type counter struct {
	key    string
	limit  int
	window time.Duration
}

func newLoginLimiter(q *models.Queries) *loginLimiter {
	if q == nil {
		return nil
	}
	return &loginLimiter{q: q}
}

// counters returns the failure counters for an attempt, the per-account one
// last. Without an address only the per-account counter applies; the others
// would lump every such request together.
func counters(username, ip string) []counter {
	u := strings.ToLower(username)
	var cs []counter
	if ip != "" {
		cs = append(cs,
			counter{"loginfail:userip:" + u + ":" + ip, maxFailsUserIP, failWindowShort},
			counter{"loginfail:ip:" + ip, maxFailsIP, failWindowShort})
	}
	return append(cs, counter{"loginfail:user:" + u, maxFailsUser, failWindowLong})
}

// allowed reports whether a login attempt may go on to the password check. It
// lets the attempt through if the counts can't be read, so an outage doesn't
// stop everyone logging in.
func (l *loginLimiter) allowed(ctx context.Context, username, ip, clientID string) bool {
	if l == nil {
		return true
	}
	cs := counters(username, ip)
	keys := make([]string, len(cs))
	for i, c := range cs {
		keys[i] = c.key
	}
	rows, err := l.q.GetLoginFailures(ctx, keys)
	if err != nil {
		log.Err(err).Msg("login-limit-db")
		return true
	}
	counts := make(map[string]int, len(rows))
	for _, r := range rows {
		counts[r.Key] = int(r.Failures)
	}
	for _, c := range cs[:len(cs)-1] {
		if counts[c.key] >= c.limit {
			return false
		}
	}
	if counts[cs[len(cs)-1].key] < maxFailsUser {
		return true
	}
	if clientID == "" {
		return false
	}
	known, err := l.q.IsKnownLoginClient(ctx, models.IsKnownLoginClientParams{
		Username: strings.ToLower(username), ClientID: clientID})
	if err != nil {
		log.Err(err).Msg("login-limit-db")
		return true
	}
	return known
}

// failed counts a failed login against the account and the address.
func (l *loginLimiter) failed(ctx context.Context, username, ip string) {
	if l == nil {
		return
	}
	for _, c := range counters(username, ip) {
		if err := l.q.AddLoginFailure(ctx, models.AddLoginFailureParams{
			Key: c.key, WindowSecs: c.window.Seconds()}); err != nil {
			log.Err(err).Msg("login-limit-db")
			return
		}
	}
}

// succeeded clears this address's failures for the account and remembers the
// browser, which then isn't held back by failures from other addresses.
func (l *loginLimiter) succeeded(ctx context.Context, username, ip, clientID string) {
	if l == nil {
		return
	}
	if ip != "" {
		if err := l.q.ClearLoginFailures(ctx, counters(username, ip)[0].key); err != nil {
			log.Err(err).Msg("login-limit-db")
		}
	}
	if clientID != "" {
		if err := l.q.AddKnownLoginClient(ctx, models.AddKnownLoginClientParams{
			Username:  strings.ToLower(username),
			ClientID:  clientID,
			ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(knownClientTTL), Valid: true},
		}); err != nil {
			log.Err(err).Msg("login-limit-db")
		}
	}
}

// CleanupLoginLimits deletes failure counts and known browsers that have
// expired. A daily job runs it.
func CleanupLoginLimits(q *models.Queries) func(context.Context) error {
	return func(ctx context.Context) error {
		f, err := q.DeleteExpiredLoginFailures(ctx)
		if err != nil {
			return err
		}
		k, err := q.DeleteExpiredKnownLoginClients(ctx)
		if err != nil {
			return err
		}
		log.Info().Int64("failures", f).Int64("known_clients", k).Msg("login-limits-cleanup")
		return nil
	}
}
