package push

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"syscall"
)

var errBadEndpoint = errors.New("a push endpoint must be an https URL with a host")

// checkEndpoint accepts only the kind of URL a push service hands a browser.
func checkEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return errBadEndpoint
	}
	return nil
}

// newClient returns the client the sender posts with. Endpoints come from
// browsers, so it connects only to public addresses. The check runs on the
// address actually dialed, after the name is resolved, so a name that
// resolves to an inside address is refused too. It never uses a proxy.
func newClient() *http.Client {
	dialer := &net.Dialer{Control: publicOnly}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = dialer.DialContext
	return &http.Client{Transport: transport}
}

func publicOnly(network, address string, _ syscall.RawConn) error {
	ap, err := netip.ParseAddrPort(address)
	if err != nil {
		return err
	}
	if !isPublic(ap.Addr()) {
		return fmt.Errorf("push: %s is not a public address", ap.Addr())
	}
	return nil
}

func isPublic(a netip.Addr) bool {
	a = a.Unmap()
	// IsGlobalUnicast is false for loopback, link-local, multicast, and
	// unspecified addresses, but true for private ones.
	if !a.IsGlobalUnicast() || a.IsPrivate() {
		return false
	}
	// Every IPv6 address outside the global unicast block is special,
	// deprecated, or local, such as IPv4-compatible and site-local ones.
	if a.Is6() && !globalUnicast6.Contains(a) {
		return false
	}
	for _, p := range nonPublic {
		if p.Contains(a) {
			return false
		}
	}
	return true
}

var globalUnicast6 = netip.MustParsePrefix("2000::/3")

// nonPublic are the special-purpose blocks in IANA's registries that the
// checks above let through: in IPv4, and inside IPv6's global unicast block,
// where 6to4 can carry any IPv4 address.
var nonPublic = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),       // this network
	netip.MustParsePrefix("100.64.0.0/10"),   // shared address space
	netip.MustParsePrefix("192.0.0.0/24"),    // protocol assignments
	netip.MustParsePrefix("192.0.2.0/24"),    // documentation
	netip.MustParsePrefix("198.18.0.0/15"),   // benchmarking
	netip.MustParsePrefix("198.51.100.0/24"), // documentation
	netip.MustParsePrefix("203.0.113.0/24"),  // documentation
	netip.MustParsePrefix("240.0.0.0/4"),     // reserved, and broadcast
	netip.MustParsePrefix("2001::/23"),       // protocol assignments
	netip.MustParsePrefix("2001:db8::/32"),   // documentation
	netip.MustParsePrefix("2002::/16"),       // 6to4
	netip.MustParsePrefix("3fff::/20"),       // documentation
}
