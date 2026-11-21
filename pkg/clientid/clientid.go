// Package clientid is the long-lived cookie that identifies a browser across
// sessions. It is its own package, importing nothing of liwords, so the
// socket server can read the cookie without depending on the API server.
package clientid

import (
	"crypto/rand"
	"encoding/base64"
)

// Cookie is the name of the cookie.
const Cookie = "client_id"

const idBytes = 16

// Valid reports whether s looks like an ID made by New.
func Valid(s string) bool {
	if len(s) != base64.RawURLEncoding.EncodedLen(idBytes) {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(s)
	return err == nil
}

// New returns a new random ID, or "" if randomness is unavailable.
func New() string {
	b := make([]byte, idBytes)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
