package user

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/models"
	pkguser "github.com/woogles-io/liwords/pkg/user"
)

const (
	cesarUUID = "mozEwaVMvTfUA2oxZfYN8k"
	minaUUID  = "iW7AaqNJDuaxgcYnrFfcJF"
)

func TestChangeEmail(t *testing.T) {
	is := is.New(t)
	ustore, pool, ctx := recreateDB()
	defer ustore.Disconnect()
	q := models.New(pool)

	emailIs := func(uuid, want string) {
		u, err := ustore.GetByUUID(ctx, uuid)
		is.NoErr(err)
		is.Equal(u.Email, want)
	}
	changesAre := func(uuid string, want ...[3]string) {
		rows, err := q.ListEmailChanges(ctx, uuid)
		is.NoErr(err)
		is.Equal(len(rows), len(want))
		for i, r := range rows {
			is.Equal([3]string{r.OldEmail, r.NewEmail, r.ChangedByUuid}, want[i])
		}
	}

	// A change by an admin sets the email and records it.
	is.NoErr(ustore.ChangeEmail(ctx, cesarUUID, "cesar@wolges.io", minaUUID))
	emailIs(cesarUUID, "cesar@wolges.io")
	changesAre(cesarUUID, [3]string{"cesar@woogles.io", "cesar@wolges.io", minaUUID})

	// Setting the same email records nothing; a change of case is a change.
	is.NoErr(ustore.ChangeEmail(ctx, cesarUUID, "cesar@wolges.io", cesarUUID))
	is.NoErr(ustore.ChangeEmail(ctx, cesarUUID, "Cesar@wolges.io", cesarUUID))
	emailIs(cesarUUID, "Cesar@wolges.io")
	changesAre(cesarUUID,
		[3]string{"cesar@wolges.io", "Cesar@wolges.io", cesarUUID},
		[3]string{"cesar@woogles.io", "cesar@wolges.io", minaUUID})

	// Another account's address, in any case, is refused with ErrEmailTaken
	// and changes nothing.
	err := ustore.ChangeEmail(ctx, cesarUUID, "MINA@gmail.com", cesarUUID)
	is.True(errors.Is(err, pkguser.ErrEmailTaken))
	emailIs(cesarUUID, "Cesar@wolges.io")
	changesAre(cesarUUID,
		[3]string{"cesar@wolges.io", "Cesar@wolges.io", cesarUUID},
		[3]string{"cesar@woogles.io", "cesar@wolges.io", minaUUID})

	// No such user.
	err = ustore.ChangeEmail(ctx, "no_such_uuid", "nobody@woogles.io", cesarUUID)
	is.True(errors.Is(err, pgx.ErrNoRows))
}

// When the record cannot be written, the email is not changed either.
func TestChangeEmailRollsBack(t *testing.T) {
	is := is.New(t)
	ustore, pool, ctx := recreateDB()
	defer ustore.Disconnect()

	// No user has the UUID given as the changer, so the record's insert
	// finds no row to write, after the email was already updated.
	err := ustore.ChangeEmail(ctx, minaUUID, "mina@wolges.io", "no_such_uuid")
	is.True(err != nil)
	u, err := ustore.GetByUUID(ctx, minaUUID)
	is.NoErr(err)
	is.Equal(u.Email, "mina@gmail.com")
	rows, err := models.New(pool).ListEmailChanges(ctx, minaUUID)
	is.NoErr(err)
	is.Equal(len(rows), 0)
}

// Changes at the same time each record the email the other left, so the
// records chain from the first address to the last.
func TestChangeEmailConcurrent(t *testing.T) {
	is := is.New(t)
	ustore, pool, ctx := recreateDB()
	defer ustore.Disconnect()

	news := []string{"one@wolges.io", "two@wolges.io", "three@wolges.io", "four@wolges.io"}
	var wg sync.WaitGroup
	errs := make([]error, len(news))
	for i, e := range news {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = ustore.ChangeEmail(ctx, minaUUID, e, minaUUID)
		}()
	}
	wg.Wait()
	for _, err := range errs {
		is.NoErr(err)
	}

	rows, err := models.New(pool).ListEmailChanges(ctx, minaUUID)
	is.NoErr(err)
	is.Equal(len(rows), len(news))
	// Newest first: each change's old email is the one before's new email.
	u, err := ustore.GetByUUID(ctx, minaUUID)
	is.NoErr(err)
	is.Equal(rows[0].NewEmail, u.Email)
	for i := 0; i+1 < len(rows); i++ {
		is.Equal(rows[i].OldEmail, rows[i+1].NewEmail)
	}
	is.Equal(rows[len(rows)-1].OldEmail, "mina@gmail.com")
}

// An address counts as a suspended account's if the account has or had it,
// ignoring case and a +tag, while a suspension someone else applied is in
// force.
func TestIsEmailOfSuspendedAccount(t *testing.T) {
	is := is.New(t)
	ustore, pool, ctx := recreateDB()
	defer ustore.Disconnect()

	suspended := func(email string) bool {
		got, err := ustore.IsEmailOfSuspendedAccount(ctx, email)
		is.NoErr(err)
		return got
	}
	exec := func(sql string, args ...any) {
		_, err := pool.Exec(ctx, sql, args...)
		is.NoErr(err)
	}
	suspend := func(username, applier string, endTime string, removed bool) {
		exec(`INSERT INTO user_actions (user_id, action_type, start_time, end_time, removed_time,
				applier_id, email_type)
			SELECT u.id, 1, NOW() - interval '1 hour', `+endTime+`,
				CASE WHEN $3 THEN NOW() END, a.id, 0
			FROM users u, users a WHERE u.username = $1 AND a.username = $2`,
			username, applier, removed)
	}

	// No one is suspended yet. (The shared fixture's suspension of "mod"
	// has already ended.)
	is.True(!suspended("cesar@woogles.io"))
	is.True(!suspended("mod@woogles.io"))

	// mina changes her email, then a moderator suspends her for good: both
	// her current and her old address count, in any case and with any
	// +tag, but not a different address at the same domain.
	is.NoErr(ustore.ChangeEmail(ctx, minaUUID, "mina+old@wolges.io", minaUUID))
	is.NoErr(ustore.ChangeEmail(ctx, minaUUID, "asdfgh@gmail.com", minaUUID))
	suspend("mina", "cesar", "NULL", false)
	is.True(suspended("asdfgh@gmail.com"))
	is.True(suspended("ASDFGH+x@Gmail.com"))
	is.True(suspended("mina@gmail.com"))
	is.True(suspended("mina+again@gmail.com"))
	is.True(suspended("mina@wolges.io"))
	is.True(!suspended("mina2@gmail.com"))

	// Closing one's own account suspends it too, but that does not count.
	suspend("jesse", "jesse", "NULL", false)
	is.True(!suspended("jesse@woogles.io"))

	// Nor does a suspension that was removed, or one that has ended.
	suspend("winter", "cesar", "NULL", true)
	is.True(!suspended("winter@woogles.io"))
	suspend("smith", "cesar", "NOW() - interval '1 minute'", false)
	is.True(!suspended("smith@woogles.io"))
	// One still running does.
	suspend("adult", "cesar", "NOW() + interval '1 day'", false)
	is.True(suspended("adult@woogles.io"))
}

// An email change waits for its token, works once, and is recorded as the
// user's own change.
func TestConfirmEmailChange(t *testing.T) {
	is := is.New(t)
	ustore, pool, ctx := recreateDB()
	defer ustore.Disconnect()
	allow := func(string) error { return nil }

	emailIs := func(uuid, want string) {
		u, err := ustore.GetByUUID(ctx, uuid)
		is.NoErr(err)
		is.Equal(u.Email, want)
	}
	later := time.Now().Add(time.Hour)

	// Asking does not change the email; asking again replaces the request.
	is.NoErr(ustore.RequestEmailChange(ctx, cesarUUID, "first@wolges.io", "token-1", later))
	is.NoErr(ustore.RequestEmailChange(ctx, cesarUUID, "second@wolges.io", "token-2", later))
	emailIs(cesarUUID, "cesar@woogles.io")
	_, err := ustore.ConfirmEmailChange(ctx, "token-1", allow)
	is.True(errors.Is(err, pkguser.ErrEmailChangeNotFound))

	// A refused address leaves the change pending and the email as it was.
	refuse := func(string) error { return pkguser.ErrEmailNotAllowed }
	_, err = ustore.ConfirmEmailChange(ctx, "token-2", refuse)
	is.True(errors.Is(err, pkguser.ErrEmailNotAllowed))
	emailIs(cesarUUID, "cesar@woogles.io")

	// The right token makes the change, once.
	change, err := ustore.ConfirmEmailChange(ctx, "token-2", allow)
	is.NoErr(err)
	is.Equal(change, pkguser.EmailChange{
		UserUUID: cesarUUID, OldEmail: "cesar@woogles.io", NewEmail: "second@wolges.io"})
	emailIs(cesarUUID, "second@wolges.io")
	rows, err := models.New(pool).ListEmailChanges(ctx, cesarUUID)
	is.NoErr(err)
	is.Equal(len(rows), 1)
	is.Equal([3]string{rows[0].OldEmail, rows[0].NewEmail, rows[0].ChangedByUuid},
		[3]string{"cesar@woogles.io", "second@wolges.io", cesarUUID})
	_, err = ustore.ConfirmEmailChange(ctx, "token-2", allow)
	is.True(errors.Is(err, pkguser.ErrEmailChangeNotFound))

	// An expired link does nothing.
	is.NoErr(ustore.RequestEmailChange(ctx, minaUUID, "mina@wolges.io", "token-3",
		time.Now().Add(-time.Minute)))
	_, err = ustore.ConfirmEmailChange(ctx, "token-3", allow)
	is.True(errors.Is(err, pkguser.ErrEmailChangeNotFound))
	emailIs(minaUUID, "mina@gmail.com")

	// An address another account took in the meantime is refused.
	is.NoErr(ustore.RequestEmailChange(ctx, minaUUID, "JESSE@woogles.io", "token-4", later))
	_, err = ustore.ConfirmEmailChange(ctx, "token-4", allow)
	is.True(errors.Is(err, pkguser.ErrEmailTaken))
	emailIs(minaUUID, "mina@gmail.com")

	// No such user.
	err = ustore.RequestEmailChange(ctx, "no_such_uuid", "x@wolges.io", "token-5", later)
	is.True(err != nil)
}
