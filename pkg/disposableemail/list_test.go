package disposableemail

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestListed(t *testing.T) {
	is := is.New(t)
	domains, err := parse(strings.NewReader("mailinator.com\n\nTrashMail.de\n"))
	is.NoErr(err)

	is.True(Listed(domains, "someone@mailinator.com"))
	is.True(Listed(domains, "Someone@MAILINATOR.COM"))
	is.True(Listed(domains, "someone@trashmail.de"))
	// A subdomain of a listed domain is listed too.
	is.True(Listed(domains, "someone@inbox.mailinator.com"))
	// The domain is what comes after the last @.
	is.True(Listed(domains, `"a@b"@mailinator.com`))

	is.True(!Listed(domains, "someone@gmail.com"))
	is.True(!Listed(domains, "mailinator.com@gmail.com"))
	is.True(!Listed(domains, "someone@notmailinator.com"))
	is.True(!Listed(domains, "mailinator.com"))
	is.True(!Listed(domains, ""))
}

func TestListedNeverMatchesABareTopLevelDomain(t *testing.T) {
	is := is.New(t)
	domains, err := parse(strings.NewReader("com\n"))
	is.NoErr(err)
	is.True(!Listed(domains, "someone@example.com"))
}

func TestParseRefusesAnEmptyList(t *testing.T) {
	is := is.New(t)
	_, err := parse(strings.NewReader("\n  \n"))
	is.True(err != nil)
}

// fakeServer serves body with status, counting the requests it gets.
type fakeServer struct {
	status atomic.Int32
	hits   atomic.Int32
	srv    *httptest.Server
}

func newFakeServer(body string) *fakeServer {
	f := &fakeServer{}
	f.status.Store(http.StatusOK)
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		w.WriteHeader(int(f.status.Load()))
		w.Write([]byte(body))
	}))
	return f
}

func newTestList(url string, clock *time.Time) *List {
	l := NewList(url)
	l.now = func() time.Time { return *clock }
	return l
}

func TestDomainsDownloadsOnceADay(t *testing.T) {
	is := is.New(t)
	f := newFakeServer("mailinator.com\n")
	defer f.srv.Close()
	clock := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	l := newTestList(f.srv.URL, &clock)
	ctx := context.Background()

	domains, err := l.Domains(ctx)
	is.NoErr(err)
	is.True(Listed(domains, "x@mailinator.com"))
	is.Equal(f.hits.Load(), int32(1))

	clock = clock.Add(23 * time.Hour)
	_, err = l.Domains(ctx)
	is.NoErr(err)
	is.Equal(f.hits.Load(), int32(1)) // still fresh

	clock = clock.Add(2 * time.Hour)
	_, err = l.Domains(ctx)
	is.NoErr(err)
	is.Equal(f.hits.Load(), int32(2)) // more than a day old
}

func TestDomainsKeepsTheLastGoodCopyWhenADownloadFails(t *testing.T) {
	is := is.New(t)
	f := newFakeServer("mailinator.com\n")
	defer f.srv.Close()
	clock := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	l := newTestList(f.srv.URL, &clock)
	ctx := context.Background()

	_, err := l.Domains(ctx)
	is.NoErr(err)

	f.status.Store(http.StatusInternalServerError)
	clock = clock.Add(25 * time.Hour)
	domains, err := l.Domains(ctx)
	is.NoErr(err)
	is.True(Listed(domains, "x@mailinator.com"))
	is.Equal(f.hits.Load(), int32(2))

	// No new attempt right after a failure.
	clock = clock.Add(time.Minute)
	_, err = l.Domains(ctx)
	is.NoErr(err)
	is.Equal(f.hits.Load(), int32(2))

	// Tried again later, and a good download replaces the old copy.
	f.status.Store(http.StatusOK)
	clock = clock.Add(retryAfter)
	_, err = l.Domains(ctx)
	is.NoErr(err)
	is.Equal(f.hits.Load(), int32(3))
	clock = clock.Add(23 * time.Hour)
	_, err = l.Domains(ctx)
	is.NoErr(err)
	is.Equal(f.hits.Load(), int32(3))
}

func TestDomainsWithoutAnyCopyReportsTheListUnavailable(t *testing.T) {
	is := is.New(t)
	f := newFakeServer("")
	f.status.Store(http.StatusNotFound)
	defer f.srv.Close()
	clock := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	l := newTestList(f.srv.URL, &clock)
	ctx := context.Background()

	domains, err := l.Domains(ctx)
	is.True(err != nil)
	is.True(domains == nil)
	is.Equal(f.hits.Load(), int32(1))

	// Within the retry wait, the same error comes back without a request.
	clock = clock.Add(time.Minute)
	_, err = l.Domains(ctx)
	is.True(err != nil)
	is.Equal(f.hits.Load(), int32(1))

	f.status.Store(http.StatusOK)
	clock = clock.Add(retryAfter)
	_, err = l.Domains(ctx)
	// The server's body is empty, so the list is still unavailable.
	is.True(err != nil)
	is.Equal(f.hits.Load(), int32(2))
}

func TestDomainsIgnoresACanceledRequest(t *testing.T) {
	is := is.New(t)
	f := newFakeServer("mailinator.com\n")
	defer f.srv.Close()
	clock := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	l := newTestList(f.srv.URL, &clock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	domains, err := l.Domains(ctx)
	is.NoErr(err)
	is.True(Listed(domains, "x@mailinator.com"))
}
