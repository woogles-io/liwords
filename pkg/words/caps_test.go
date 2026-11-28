package words

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"

	pb "github.com/woogles-io/liwords/rpc/api/proto/word_service"
)

func TestDefineWordsCaps(t *testing.T) {
	ws := &WordService{}
	define := func(words []string) error {
		_, err := ws.DefineWords(context.Background(), connect.NewRequest(&pb.DefineWordsRequest{
			Lexicon: "NWL23", Words: words, Anagrams: true,
		}))
		return err
	}

	tooMany := make([]string, maxDefineWords+1)
	for i := range tooMany {
		tooMany[i] = "CAT"
	}
	if err := define(tooMany); connect.CodeOf(err) != connect.CodeInvalidArgument ||
		!strings.Contains(err.Error(), "words per request") {
		t.Fatalf("too many words: got %v", err)
	}

	if err := define([]string{strings.Repeat("A", maxDefineWordLen+1)}); connect.CodeOf(err) != connect.CodeInvalidArgument ||
		!strings.Contains(err.Error(), "letters") {
		t.Fatalf("long word: got %v", err)
	}

	// At the limits, the request gets past the caps (and fails on the
	// lexicon, which this bare service doesn't have).
	atLimit := tooMany[:maxDefineWords]
	atLimit[0] = strings.Repeat("Ä", maxDefineWordLen) // letters, not bytes
	if err := define(atLimit); err == nil || !strings.Contains(err.Error(), "no such lexicon") {
		t.Fatalf("at the limits: got %v", err)
	}
}
