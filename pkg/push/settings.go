// Package push sends web push notifications (VAPID) to the browsers a user
// has subscribed, so a correspondence turn reaches them with Woogles closed.
package push

import (
	"bytes"
	"crypto/ecdh"
	"encoding/base64"
	"net/url"
	"strings"
	"time"

	"github.com/woogles-io/liwords/pkg/config"
)

// Settings are what web push needs from the api's config. Push is on only
// when every one of them is set and valid.
type Settings struct {
	PublicKey  string
	PrivateKey string
	// Subject is the contact push services see: a mailto: or https: URL.
	Subject string
	// TTL is how long a push service keeps a notification for a device that
	// is offline.
	TTL time.Duration
}

// NewSettings reads the push settings from cfg. If any is missing or invalid
// it returns nil, push stays off, and the problems name the settings, never
// their values.
func NewSettings(cfg *config.Config) (*Settings, []string) {
	var problems []string
	missing := func(name, value string) bool {
		if value == "" {
			problems = append(problems, name+" is missing")
			return true
		}
		return false
	}
	invalid := func(name, why string) {
		problems = append(problems, name+" is invalid: "+why)
	}
	s := &Settings{
		PublicKey:  cfg.VAPIDPublicKey,
		PrivateKey: cfg.VAPIDPrivateKey,
		Subject:    cfg.VAPIDSubject,
	}

	var publicKey []byte
	if !missing("vapid-public-key", cfg.VAPIDPublicKey) {
		k, err := decodeKey(cfg.VAPIDPublicKey)
		if err == nil {
			_, err = ecdh.P256().NewPublicKey(k)
		}
		if err != nil {
			invalid("vapid-public-key", "not a P-256 public key in base64url")
		} else {
			publicKey = k
		}
	}
	if !missing("vapid-private-key", cfg.VAPIDPrivateKey) {
		k, err := decodeKey(cfg.VAPIDPrivateKey)
		var private *ecdh.PrivateKey
		if err == nil {
			private, err = ecdh.P256().NewPrivateKey(k)
		}
		if err != nil {
			invalid("vapid-private-key", "not a P-256 private key in base64url")
		} else if publicKey != nil && !bytes.Equal(private.PublicKey().Bytes(), publicKey) {
			invalid("vapid-private-key", "not the pair of vapid-public-key")
		}
	}
	if !missing("vapid-subject", cfg.VAPIDSubject) && !validSubject(cfg.VAPIDSubject) {
		invalid("vapid-subject", "not a mailto: or https: URL")
	}
	if !missing("push-ttl", cfg.PushTTL) {
		d, err := time.ParseDuration(cfg.PushTTL)
		switch {
		case err != nil:
			invalid("push-ttl", "not a duration")
		case d < 0:
			invalid("push-ttl", "negative")
		case d%time.Second != 0:
			invalid("push-ttl", "not whole seconds")
		default:
			s.TTL = d
		}
	}

	if len(problems) > 0 {
		return nil, problems
	}
	return s, nil
}

// decodeKey accepts a key in base64url with or without padding, as the push
// library does.
func decodeKey(key string) ([]byte, error) {
	if b, err := base64.RawURLEncoding.DecodeString(key); err == nil {
		return b, nil
	}
	return base64.URLEncoding.DecodeString(key)
}

func validSubject(subject string) bool {
	if address, ok := strings.CutPrefix(subject, "mailto:"); ok {
		return address != ""
	}
	u, err := url.Parse(subject)
	return err == nil && u.Scheme == "https" && u.Host != ""
}
