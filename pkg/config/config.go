package config

import (
	"github.com/domino14/macondo/config"
	"github.com/namsral/flag"
)

type Config struct {
	MacondoConfig config.Config

	DBConnString string
	ListenAddr   string
	SecretKey    string
	NatsURL      string
	MailgunKey   string
	RedisURL     string
	OriginURLs   string
	CookieDomain string
}

// Load loads the configs from the given arguments
func (c *Config) Load(args []string) error {
	fs := flag.NewFlagSet("macondo", flag.ContinueOnError)

	fs.BoolVar(&c.MacondoConfig.Debug, "debug", false, "debug logging on")

	fs.StringVar(&c.MacondoConfig.LetterDistributionPath, "letter-distribution-path", "../macondo/data/letterdistributions", "directory holding letter distribution files")
	fs.StringVar(&c.MacondoConfig.StrategyParamsPath, "strategy-params-path", "../macondo/data/strategy", "directory holding strategy files")
	fs.StringVar(&c.MacondoConfig.LexiconPath, "lexicon-path", "../macondo/data/lexica", "directory holding lexicon files")
	fs.StringVar(&c.MacondoConfig.DefaultLexicon, "default-lexicon", "NWL18", "the default lexicon to use")
	fs.StringVar(&c.MacondoConfig.DefaultLetterDistribution, "default-letter-distribution", "English", "the default letter distribution to use. English, EnglishSuper, Spanish, Polish, etc.")
	fs.StringVar(&c.DBConnString, "db-conn-string", "", "the database connection string")
	fs.StringVar(&c.ListenAddr, "listen-addr", ":8001", "listen on this address")
	fs.StringVar(&c.SecretKey, "secret-key", "", "secret key must be a random unguessable string")
	fs.StringVar(&c.NatsURL, "nats-url", "nats://localhost:4222", "the NATS server URL")
	fs.StringVar(&c.MailgunKey, "mailgun-key", "", "the Mailgun secret key")
	fs.StringVar(&c.RedisURL, "redis-url", "", "the Redis URL")
	fs.StringVar(&c.OriginURLs, "origin-urls", "http://liwords.localhost", "a comma-separated list of domains for CORS; use protocols")
	fs.StringVar(&c.CookieDomain, "cookie-domain", "liwords.localhost", "the cookie domain -- should be the domain part of one of the origin URLs")
	err := fs.Parse(args)
	return err
}
