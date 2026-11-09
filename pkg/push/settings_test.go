package push

import (
	"strings"
	"testing"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/config"
)

func validConfig(t *testing.T) *config.Config {
	private, public, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	return &config.Config{
		VAPIDPublicKey:  public,
		VAPIDPrivateKey: private,
		VAPIDSubject:    "mailto:push@example.com",
		PushTTL:         "48h",
	}
}

func TestSettingsAllSet(t *testing.T) {
	is := is.New(t)
	cfg := validConfig(t)
	s, problems := NewSettings(cfg)
	is.Equal(len(problems), 0)
	is.True(s != nil)
	is.Equal(s.PublicKey, cfg.VAPIDPublicKey)
	is.Equal(s.PrivateKey, cfg.VAPIDPrivateKey)
	is.Equal(s.Subject, "mailto:push@example.com")
	is.Equal(s.TTL, 48*time.Hour)
}

func TestSettingsNoneSet(t *testing.T) {
	is := is.New(t)
	s, problems := NewSettings(&config.Config{})
	is.True(s == nil)
	is.Equal(problems, []string{
		"vapid-public-key is missing",
		"vapid-private-key is missing",
		"vapid-subject is missing",
		"push-ttl is missing",
	})
}

func TestSettingsEachMissing(t *testing.T) {
	unset := map[string]func(*config.Config){
		"vapid-public-key":  func(c *config.Config) { c.VAPIDPublicKey = "" },
		"vapid-private-key": func(c *config.Config) { c.VAPIDPrivateKey = "" },
		"vapid-subject":     func(c *config.Config) { c.VAPIDSubject = "" },
		"push-ttl":          func(c *config.Config) { c.PushTTL = "" },
	}
	for name, clear := range unset {
		t.Run(name, func(t *testing.T) {
			is := is.New(t)
			cfg := validConfig(t)
			clear(cfg)
			s, problems := NewSettings(cfg)
			is.True(s == nil)
			is.Equal(problems, []string{name + " is missing"})
		})
	}
}

func TestSettingsInvalid(t *testing.T) {
	other := validConfig(t)
	cases := []struct {
		name    string
		change  func(*config.Config)
		problem string
	}{
		{"public key not base64", func(c *config.Config) { c.VAPIDPublicKey = "not a key!" },
			"vapid-public-key is invalid: not a P-256 public key in base64url"},
		{"public key too short", func(c *config.Config) { c.VAPIDPublicKey = c.VAPIDPublicKey[:40] },
			"vapid-public-key is invalid: not a P-256 public key in base64url"},
		{"private key too short", func(c *config.Config) { c.VAPIDPrivateKey = c.VAPIDPrivateKey[:20] },
			"vapid-private-key is invalid: not a P-256 private key in base64url"},
		{"keys from two pairs", func(c *config.Config) { c.VAPIDPrivateKey = other.VAPIDPrivateKey },
			"vapid-private-key is invalid: not the pair of vapid-public-key"},
		{"subject without a scheme", func(c *config.Config) { c.VAPIDSubject = "push@example.com" },
			"vapid-subject is invalid: not a mailto: or https: URL"},
		{"subject over http", func(c *config.Config) { c.VAPIDSubject = "http://example.com" },
			"vapid-subject is invalid: not a mailto: or https: URL"},
		{"subject https without a host", func(c *config.Config) { c.VAPIDSubject = "https://" },
			"vapid-subject is invalid: not a mailto: or https: URL"},
		{"subject empty mailto", func(c *config.Config) { c.VAPIDSubject = "mailto:" },
			"vapid-subject is invalid: not a mailto: or https: URL"},
		{"ttl not a duration", func(c *config.Config) { c.PushTTL = "two days" },
			"push-ttl is invalid: not a duration"},
		{"ttl negative", func(c *config.Config) { c.PushTTL = "-1s" },
			"push-ttl is invalid: negative"},
		{"ttl part of a second", func(c *config.Config) { c.PushTTL = "1500ms" },
			"push-ttl is invalid: not whole seconds"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			cfg := validConfig(t)
			tc.change(cfg)
			s, problems := NewSettings(cfg)
			is.True(s == nil)
			is.Equal(problems, []string{tc.problem})
			for _, p := range problems {
				is.True(!strings.Contains(p, cfg.VAPIDPrivateKey)) // never a value
			}
		})
	}
}

func TestSettingsZeroTTLAndHTTPSSubject(t *testing.T) {
	is := is.New(t)
	cfg := validConfig(t)
	cfg.PushTTL = "0s"
	cfg.VAPIDSubject = "https://woogles.io/about"
	s, problems := NewSettings(cfg)
	is.Equal(len(problems), 0)
	is.Equal(s.TTL, time.Duration(0))
	is.Equal(s.Subject, "https://woogles.io/about")
}
