package config

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/matryer/is"
)

// The api logs its whole config when it starts.
func TestVAPIDPrivateKeyStaysOutOfTheLog(t *testing.T) {
	is := is.New(t)
	cfg := &Config{VAPIDPublicKey: "public-part", VAPIDPrivateKey: "secret-part"}
	b, err := json.Marshal(cfg)
	is.NoErr(err)
	is.True(strings.Contains(string(b), "public-part"))
	is.True(!strings.Contains(string(b), "secret-part"))
}
