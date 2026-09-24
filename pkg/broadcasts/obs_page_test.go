package broadcasts

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// User-controlled values (player names, path segments) must not be able to
// close the page's <script> block.
func TestServeOBSPageEscapesScriptContext(t *testing.T) {
	payload := `</script><script>alert('x')</script><script>`
	rec := httptest.NewRecorder()
	serveOBSPage(rec, "p1_name", payload, "/api/annotations/obs/game/</script><script>alert(1)//events")
	body := rec.Body.String()

	if n := strings.Count(strings.ToLower(body), "</script"); n != 1 {
		t.Fatalf("expected exactly one </script> (the page's own), got %d:\n%s", n, body)
	}
	if strings.Contains(body, "alert('x')</script>") {
		t.Fatalf("payload rendered unescaped:\n%s", body)
	}
	for _, want := range []string{
		`var field      = "p1_name";`,
		`var isMarquee  =  false ;`,
		`var isBlank    =  false ;`,
		`var initVal    = "\u003c/script\u003e\u003cscript\u003ealert('x')`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("rendered page missing %q", want)
		}
	}
}
