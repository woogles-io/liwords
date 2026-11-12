// Command decode-turn prints a game_turns row as readable JSON.
//
// event_bin is binary proto, which psql shows as hex. Paste that hex (with or
// without psql's leading \x), or a protojson `event`, as an argument or on
// stdin, one row per line:
//
//	psql -Atc "SELECT event_bin FROM game_turns WHERE game_uuid = '...' ORDER BY turn_idx" | go run ./cmd/decode-turn
package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/woogles-io/liwords/pkg/xwordbridge"
)

func main() {
	failed := false
	decode := func(line string) {
		line = strings.TrimSpace(line)
		if line == "" {
			return
		}
		if err := printTurn(line); err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed = true
		}
	}

	if len(os.Args) > 1 {
		for _, a := range os.Args[1:] {
			decode(a)
		}
	} else {
		sc := bufio.NewScanner(os.Stdin)
		sc.Buffer(make([]byte, 1<<20), 1<<24)
		for sc.Scan() {
			decode(sc.Text())
		}
		if err := sc.Err(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func printTurn(s string) error {
	raw := []byte(s)
	if !strings.HasPrefix(s, "{") {
		b, err := hex.DecodeString(strings.TrimPrefix(s, `\x`))
		if err != nil {
			return fmt.Errorf("neither protojson nor hex: %w", err)
		}
		raw = b
	}
	evt, err := xwordbridge.DecodeTurn(raw)
	if err != nil {
		return err
	}
	out, err := protojson.MarshalOptions{Multiline: true}.Marshal(evt)
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
