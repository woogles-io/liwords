package xwordgame

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/domino14/word-golib/config"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/rs/zerolog"
)

// testConfig points word-golib's loaders at the letter distributions checked
// into this repo, so these tests need no environment setup. MACONDO_DATA_PATH
// still wins if it is set, matching the rest of the test suite.
var testConfig = config.Config{DataPath: testDataPath()}

func testDataPath() string {
	if p := os.Getenv("MACONDO_DATA_PATH"); p != "" {
		return p
	}
	return filepath.Join("..", "..", "data")
}

func TestMain(m *testing.M) {
	// word-golib and macondo log at debug on every distribution load, which
	// buries test failures under tens of thousands of lines.
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	os.Exit(m.Run())
}

var (
	ldMu    sync.Mutex
	ldCache = map[string]*tilemapping.LetterDistribution{}
)

// testLetterDistribution caches by name. word-golib's loader re-reads and
// re-parses the file on every call, which dominates the runtime of the
// randomized parity tests.
func testLetterDistribution(tb testing.TB, name string) *tilemapping.LetterDistribution {
	tb.Helper()
	ldMu.Lock()
	defer ldMu.Unlock()
	if ld, ok := ldCache[name]; ok {
		return ld
	}
	ld, err := tilemapping.NamedLetterDistribution(&testConfig, name)
	if err != nil {
		tb.Fatal(err)
	}
	ldCache[name] = ld
	return ld
}
