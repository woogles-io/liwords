package user

import (
	"errors"
	"sync"
	"testing"

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
