package apiserver

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
)

// statusOfStalledRequest serves h behind an http.Server with a short
// ReadTimeout, sends a Connect request whose body never finishes, and
// returns the response status.
func statusOfStalledRequest(t *testing.T, h http.Handler) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: h, ReadTimeout: 200 * time.Millisecond}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	// Promise 100 bytes of body and send 2.
	fmt.Fprintf(conn, "POST /test.Svc/Do HTTP/1.1\r\nHost: x\r\nContent-Type: application/proto\r\nContent-Length: 100\r\n\r\nab")
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode
}

func testConnectHandler(work func(context.Context) error) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/test.Svc/Do", connect.NewUnaryHandler("/test.Svc/Do",
		func(ctx context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
			if err := work(ctx); err != nil {
				return nil, err
			}
			return connect.NewResponse(&emptypb.Empty{}), nil
		}))
	return mux
}

func TestStalledBodyIs408(t *testing.T) {
	h := testConnectHandler(func(context.Context) error { return nil })

	if got := statusOfStalledRequest(t, h); got != http.StatusGatewayTimeout {
		t.Fatalf("without the middleware: got %d, want 504 (has Connect changed?)", got)
	}
	if got := statusOfStalledRequest(t, ReadTimeoutAs408(h)); got != http.StatusRequestTimeout {
		t.Fatalf("with the middleware: got %d, want 408", got)
	}
}

func TestHandlerDeadlineStays504(t *testing.T) {
	h := ReadTimeoutAs408(testConnectHandler(func(context.Context) error {
		return connect.NewError(connect.CodeDeadlineExceeded, context.DeadlineExceeded)
	}))
	srv := &http.Server{Handler: h}
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	go srv.Serve(ln)
	defer srv.Close()

	client := connect.NewClient[emptypb.Empty, emptypb.Empty](http.DefaultClient, "http://"+ln.Addr().String()+"/test.Svc/Do")
	_, err := client.CallUnary(context.Background(), connect.NewRequest(&emptypb.Empty{}))
	if connect.CodeOf(err) != connect.CodeDeadlineExceeded {
		t.Fatalf("got %v, want deadline_exceeded", err)
	}
	// And the HTTP status is still 504.
	resp, err := http.Post("http://"+ln.Addr().String()+"/test.Svc/Do", "application/proto", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("got %d, want 504", resp.StatusCode)
	}
}

// The SSE handlers (pkg/bus, pkg/broadcasts) flush and clear their write
// deadline through the wrapper.
func TestWrapperKeepsStreaming(t *testing.T) {
	h := ReadTimeoutAs408(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := w.(http.Flusher); !ok {
			t.Error("not a Flusher")
		}
		if err := http.NewResponseController(w).SetWriteDeadline(time.Time{}); err != nil {
			t.Errorf("SetWriteDeadline: %v", err)
		}
	}))
	srv := &http.Server{Handler: h, WriteTimeout: time.Second}
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	go srv.Serve(ln)
	defer srv.Close()
	resp, err := http.Get("http://" + ln.Addr().String() + "/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}
