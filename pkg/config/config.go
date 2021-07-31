package config

import (
	"github.com/domino14/macondo/config"
	"github.com/namsral/flag"
)

type ArgonConfig struct {
	Time    int
	Memory  int
	Threads int
	Keylen  int
}

type Config struct {
	MacondoConfig config.Config
	ArgonConfig   ArgonConfig

	DBConnString string
	ListenAddr   string
	SecretKey    string
	NatsURL      string
	MailgunKey   string
	RedisURL     string
	DiscordToken string
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
	fs.StringVar(&c.DiscordToken, "discord-token", "", "the token used for moderator action discord notifications")

	// For password hashing:
	fs.IntVar(&c.ArgonConfig.Keylen, "argon-key-len", 32, "the Argon key length")
	fs.IntVar(&c.ArgonConfig.Time, "argon-time", 1, "the Argon time")
	fs.IntVar(&c.ArgonConfig.Memory, "argon-memory", 64*1024, "the Argon memory (KB)")
	fs.IntVar(&c.ArgonConfig.Threads, "argon-threads", 4, "the Argon threads")
	err := fs.Parse(args)
	return err
}
