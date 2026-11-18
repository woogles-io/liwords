package utilities

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// TracedTransport is an http.Transport whose requests are client spans named
// "<METHOD> <host>", children of the span in the request's context. The span
// records the full URL: don't use it for URLs that hold a secret, such as
// Discord webhooks.
func TracedTransport() http.RoundTripper {
	return otelhttp.NewTransport(http.DefaultTransport,
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + r.URL.Host
		}))
}
