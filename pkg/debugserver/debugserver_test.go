package debugserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matryer/is"
)

func TestHandlerServesDebugPages(t *testing.T) {
	is := is.New(t)
	srv := httptest.NewServer(Handler())
	defer srv.Close()
	for _, path := range []string{"/debug/pprof/", "/debug/pprof/heap", "/debug/pprof/goroutine", "/debug/vars"} {
		resp, err := http.Get(srv.URL + path)
		is.NoErr(err)
		resp.Body.Close()
		is.Equal(resp.StatusCode, http.StatusOK) // path
	}
}
