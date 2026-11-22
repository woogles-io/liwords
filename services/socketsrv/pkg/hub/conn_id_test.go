package sockets

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/matryer/is"
)

func TestValidConnID(t *testing.T) {
	is := is.New(t)
	for _, ok := range []string{"QGJL6HsE5", "abc-123_x", strings.Repeat("a", 64)} {
		is.True(validConnID(ok))
	}
	for _, bad := range []string{"", "a#b", "a.b", "a:b", "a b", "a*", "a>", "é", strings.Repeat("a", 65)} {
		is.True(!validConnID(bad))
	}
}

func signed(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// The connection ID is the token's, never the URL's; a token without one is
// refused.
func TestSocketLoginTakesConnIDFromToken(t *testing.T) {
	is := is.New(t)
	t.Setenv("SECRET_KEY", "test-secret")
	h := &Hub{}
	base := jwt.MapClaims{"exp": time.Now().Add(time.Minute).Unix(), "uid": "u1", "unn": "cesar", "a": true}

	withCid := jwt.MapClaims{"cid": "fromToken1"}
	for k, v := range base {
		withCid[k] = v
	}
	c := &Client{connToken: signed(t, withCid)}
	is.NoErr(h.socketLogin(c))
	is.Equal(c.connID, "fromToken1")

	c = &Client{connToken: signed(t, base)}
	is.True(h.socketLogin(c) != nil)
}
