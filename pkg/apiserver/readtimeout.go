package apiserver

import (
	"errors"
	"io"
	"net/http"
	"os"
	"sync/atomic"
)

// ReadTimeoutAs408 makes a request whose body didn't arrive before the
// server's ReadTimeout answer 408 Request Timeout instead of 504.
//
// When a client sends its headers and then stalls (a phone that sleeps or
// changes networks mid-request), reading the body fails with a deadline
// error, and Connect reports that as deadline_exceeded, which is HTTP 504.
// That is the client's timeout, not ours, so it shouldn't count as a server
// error on the load balancer. Errors from a handler's own deadlines are
// left alone: only a 504 after a body read timed out becomes a 408.
func ReadTimeoutAs408(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var timedOut atomic.Bool
		if r.Body != nil && r.Body != http.NoBody {
			r.Body = &deadlineBody{ReadCloser: r.Body, timedOut: &timedOut}
		}
		h.ServeHTTP(&readTimeoutWriter{ResponseWriter: w, timedOut: &timedOut}, r)
	})
}

type deadlineBody struct {
	io.ReadCloser
	timedOut *atomic.Bool
}

func (b *deadlineBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err != nil && errors.Is(err, os.ErrDeadlineExceeded) {
		b.timedOut.Store(true)
	}
	return n, err
}

type readTimeoutWriter struct {
	http.ResponseWriter
	timedOut *atomic.Bool
}

func (w *readTimeoutWriter) WriteHeader(code int) {
	if code == http.StatusGatewayTimeout && w.timedOut.Load() {
		code = http.StatusRequestTimeout
	}
	w.ResponseWriter.WriteHeader(code)
}

// Flush and Unwrap keep streaming responses and http.ResponseController
// working through the wrapper.
func (w *readTimeoutWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *readTimeoutWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
