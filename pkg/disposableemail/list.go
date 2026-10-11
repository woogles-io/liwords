// Package disposableemail recognizes email addresses on known disposable
// (throwaway) email domains, using the community-maintained list at
// https://github.com/disposable-email-domains/disposable-email-domains.
//
// The list is downloaded when it is first needed and kept in memory, never
// stored, so a refreshed list applies to every account at once.
package disposableemail

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// ListURL is where the list is downloaded from.
const ListURL = "https://raw.githubusercontent.com/disposable-email-domains/disposable-email-domains/refs/heads/main/disposable_email_blocklist.conf"

const (
	// The list changes about once a day.
	maxAge = 24 * time.Hour
	// After a failed download, lookups use what they have (or report the
	// list unavailable) for this long instead of each waiting on a server
	// that just failed.
	retryAfter = 10 * time.Minute
	// The list is about 130 KB; anything far larger is not the list.
	maxListBytes = 10 << 20
)

// List holds the downloaded list of disposable-email domains.
type List struct {
	url    string
	client *http.Client
	now    func() time.Time

	mu        sync.Mutex
	domains   map[string]struct{}
	fetchedAt time.Time
	failedAt  time.Time
	lastErr   error
}

func NewList(url string) *List {
	return &List{
		url:    url,
		client: &http.Client{Timeout: 10 * time.Second},
		now:    time.Now,
	}
}

// Domains returns the listed domains, downloading the list the first time
// and again once the copy is more than a day old. A failed download keeps
// the copy it has; with no copy at all, it returns the error. The returned
// map is shared and must not be modified.
func (l *List) Domains(ctx context.Context) (map[string]struct{}, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if l.domains != nil && now.Sub(l.fetchedAt) < maxAge {
		return l.domains, nil
	}
	if !l.failedAt.IsZero() && now.Sub(l.failedAt) < retryAfter {
		if l.domains != nil {
			return l.domains, nil
		}
		return nil, l.lastErr
	}

	// A lookup whose request is canceled should not count as a failed
	// download; the client's own timeout still bounds it.
	domains, err := l.download(context.WithoutCancel(ctx))
	if err != nil {
		log.Err(err).Str("url", l.url).Msg("disposable-email-list-download-failed")
		l.failedAt = now
		l.lastErr = err
		if l.domains != nil {
			return l.domains, nil
		}
		return nil, err
	}
	l.domains = domains
	l.fetchedAt = now
	l.failedAt = time.Time{}
	l.lastErr = nil
	return domains, nil
}

func (l *List) download(ctx context.Context) (map[string]struct{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("disposable email list returned status %d", resp.StatusCode)
	}
	return parse(io.LimitReader(resp.Body, maxListBytes))
}

// parse reads one domain per line.
func parse(r io.Reader) (map[string]struct{}, error) {
	domains := map[string]struct{}{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		d := strings.ToLower(strings.TrimSpace(sc.Text()))
		if d == "" || strings.HasPrefix(d, "#") {
			continue
		}
		domains[d] = struct{}{}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(domains) == 0 {
		return nil, errors.New("disposable email list is empty")
	}
	return domains, nil
}

// Listed reports whether the domain of email, or a parent domain of it, is
// in domains. As in the list's own examples, a bare top-level domain never
// matches.
func Listed(domains map[string]struct{}, email string) bool {
	at := strings.LastIndexByte(email, '@')
	if at < 0 {
		return false
	}
	labels := strings.Split(strings.ToLower(strings.TrimSpace(email[at+1:])), ".")
	for i := 0; i < len(labels)-1; i++ {
		if _, ok := domains[strings.Join(labels[i:], ".")]; ok {
			return true
		}
	}
	return false
}
