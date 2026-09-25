package embed

import (
	"strings"
	"testing"

	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// User-controlled strings in the game document must not be able to close the
// page's <script> block.
func TestGenerateEmbedHTMLEscapesScriptContext(t *testing.T) {
	doc := &ipc.GameDocument{
		Type: ipc.GameType_ANNOTATED,
		Players: []*ipc.GameDocument_MinimalPlayerInfo{
			{Nickname: "p1", RealName: "</script><script>alert(1)</script>"},
		},
	}
	page, err := (&EmbedService{}).generateEmbedHTML(doc, "abc", parseEmbedOptions(nil))
	if err != nil {
		t.Fatal(err)
	}
	// The page has two script tags of its own.
	if n := strings.Count(strings.ToLower(page), "</script"); n != 2 {
		t.Fatalf("expected 2 </script> tags, got %d:\n%s", n, page)
	}
	if !strings.Contains(page, `\u003c/script\u003e\u003cscript\u003ealert(1)`) {
		t.Errorf("player name not escaped as expected:\n%s", page)
	}
	if !strings.Contains(page, `gameId: "abc"`) || !strings.Contains(page, `id="woogles-embed-abc"`) {
		t.Errorf("game ID not rendered as expected:\n%s", page)
	}
}
