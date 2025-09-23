package config

import (
	"context"
	"errors"
	"net/http"

	"github.com/lithammer/shortuuid/v4"
	"github.com/namsral/flag"

	macondoconfig "github.com/domino14/macondo/config"
	wglconfig "github.com/domino14/word-golib/config"
	"github.com/woogles-io/liwords/pkg/stores/common"
)

type ArgonConfig struct {
	Time    int
	Memory  int
	Threads int
	Keylen  int
}

type Config struct {
	macondoConfig *macondoconfig.Config
	ArgonConfig   ArgonConfig

	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBSSLMode        string
	DBName           string
	DBMigrationsPath string
	DBConnUri        string
	DBConnDSN        string

	ListenAddr   string
	SecretKey    string
	NatsURL      string
	MailgunKey   string
	RedisURL     string
	DiscordToken string
	// Puzzles
	PuzzleGenerationSecretKey      string
	ECSClusterName                 string
	PuzzleGenerationTaskDefinition string

	TourneyPDFLambdaFunctionName string
	COPPairLambdaFunctionName    string

	// Integrations
	PatreonClientID     string
	PatreonClientSecret string
	PatreonRedirectURI  string

	TwitchClientID     string
	TwitchClientSecret string
	TwitchRedirectURI  string

	Debug bool
}

type ctxKey string

const ctxKeyword ctxKey = ctxKey("config")

// Load loads the configs from the given arguments
func (c *Config) Load(args []string) error {
	c.macondoConfig = &macondoconfig.Config{}
	err := c.macondoConfig.Load(nil)
	if err != nil {
		return err
	}

	fs := flag.NewFlagSet("liwords", flag.ContinueOnError)

	fs.BoolVar(&c.Debug, "debug", false, "debug logging on")

	fs.StringVar(&c.DBHost, "db-host", "", "the database host")
	fs.StringVar(&c.DBPort, "db-port", "", "the database port")
	fs.StringVar(&c.DBUser, "db-user", "", "the database user")
	fs.StringVar(&c.DBPassword, "db-password", "", "the database password")
	fs.StringVar(&c.DBSSLMode, "db-ssl-mode", "", "the database SSL mode")
	fs.StringVar(&c.DBName, "db-name", "", "the database name")
	fs.StringVar(&c.ListenAddr, "listen-addr", ":8001", "listen on this address")
	fs.StringVar(&c.SecretKey, "secret-key", "", "secret key must be a random unguessable string")
	fs.StringVar(&c.NatsURL, "nats-url", "nats://localhost:4222", "the NATS server URL")
	fs.StringVar(&c.MailgunKey, "mailgun-key", "", "the Mailgun secret key")
	fs.StringVar(&c.RedisURL, "redis-url", "", "the Redis URL")
	fs.StringVar(&c.DiscordToken, "discord-token", "", "the token used for moderator action discord notifications")
	fs.StringVar(&c.DBMigrationsPath, "db-migrations-path", "", "the path where migrations are stored")
	fs.StringVar(&c.PuzzleGenerationSecretKey, "puzzle-generation-secret-key", shortuuid.New(), "a secret key used for generating puzzles")
	fs.StringVar(&c.ECSClusterName, "ecs-cluster-name", "", "the ECS cluster this runs on")
	fs.StringVar(&c.PuzzleGenerationTaskDefinition, "puzzle-generation-task-definition", "", "the task definition for the puzzle generation ECS task")
	fs.StringVar(&c.TourneyPDFLambdaFunctionName, "tourney-pdf-lambda-function-name", "", "the name of the TourneyPDF lambda function")
	fs.StringVar(&c.COPPairLambdaFunctionName, "cop-pair-lambda-function-name", "", "the name of the COPPair lambda function")
	fs.StringVar(&c.PatreonClientID, "patreon-client-id", "", "The Patreon Integration Client ID")
	fs.StringVar(&c.PatreonClientSecret, "patreon-client-secret", "", "The Patreon Integration Client secret")
	fs.StringVar(&c.PatreonRedirectURI, "patreon-redirect-uri", "", "The Patreon redirect URI")
	fs.StringVar(&c.TwitchClientID, "twitch-client-id", "", "The Twitch Integration Client ID")
	fs.StringVar(&c.TwitchClientSecret, "twitch-client-secret", "", "The Twitch Integration Client secret")
	fs.StringVar(&c.TwitchRedirectURI, "twitch-redirect-uri", "", "The Twitch redirect URI")

	// For password hashing:
	fs.IntVar(&c.ArgonConfig.Keylen, "argon-key-len", 32, "the Argon key length")
	fs.IntVar(&c.ArgonConfig.Time, "argon-time", 1, "the Argon time")
	fs.IntVar(&c.ArgonConfig.Memory, "argon-memory", 64*1024, "the Argon memory (KB)")
	fs.IntVar(&c.ArgonConfig.Threads, "argon-threads", 4, "the Argon threads")
	err = fs.Parse(args)
	// build the DB conn string from the passed-in DB arguments
	c.DBConnUri = common.PostgresConnUri(c.DBHost, c.DBPort, c.DBName, c.DBUser, c.DBPassword, c.DBSSLMode)
	c.DBConnDSN = common.PostgresConnDSN(c.DBHost, c.DBPort, c.DBName, c.DBUser, c.DBPassword, c.DBSSLMode)

	return err
}

// WithContext stores the config in the passed-in context, returning a new context. Context.
func (c *Config) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKeyword, c)
}

func (c *Config) MacondoConfig() *macondoconfig.Config {
	return c.macondoConfig
}

func (c *Config) WGLConfig() *wglconfig.Config {
	return c.macondoConfig.WGLConfig()
}

// ctx gets the config from the context, or an error if no config is found.
func Ctx(ctx context.Context) (*Config, error) {
	ctxConfig, ok := ctx.Value(ctxKeyword).(*Config)
	if !ok {
		return nil, errors.New("config in context is not ok")
	}
	if ctxConfig == nil {
		return nil, errors.New("config in context is nil")
	}
	return ctxConfig, nil
}

var defaultConfig = &Config{
	macondoConfig: macondoconfig.DefaultConfig(),
}

func DefaultConfig() *Config {
	return defaultConfig
}

func CtxMiddlewareGenerator(config *Config) (mw func(http.Handler) http.Handler) {
	mw = func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := config.WithContext(r.Context())
			r = r.WithContext(ctx)
			h.ServeHTTP(w, r)
		})
	}
	return
}
