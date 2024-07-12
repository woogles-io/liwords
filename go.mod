module github.com/woogles-io/liwords

go 1.22.2

require (
	connectrpc.com/connect v1.16.2
	connectrpc.com/otelconnect v0.7.0
	github.com/TwiN/go-away v1.6.13
	github.com/aws/aws-sdk-go-v2 v1.30.3
	github.com/aws/aws-sdk-go-v2/config v1.27.26
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.7
	github.com/aws/aws-sdk-go-v2/service/ecs v1.44.3
	github.com/aws/aws-sdk-go-v2/service/lambda v1.56.3
	github.com/aws/aws-sdk-go-v2/service/s3 v1.58.2
	github.com/aws/smithy-go v1.20.3
	github.com/domino14/macondo v0.9.2
	github.com/domino14/word-golib v0.1.10
	github.com/exaring/otelpgx v0.6.2
	github.com/go-redsync/redsync/v4 v4.13.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang-migrate/migrate/v4 v4.17.1
	github.com/gomodule/redigo v1.9.2
	github.com/google/uuid v1.6.0
	github.com/hashicorp/golang-lru v1.0.2
	github.com/jackc/pgx/v5 v5.6.0
	github.com/justinas/alice v1.2.0
	github.com/lithammer/shortuuid v3.0.0+incompatible
	github.com/mailgun/mailgun-go/v4 v4.12.0
	github.com/matryer/is v1.4.1
	github.com/mmcdole/gofeed v1.3.0
	github.com/namsral/flag v1.7.4-pre
	github.com/nats-io/nats.go v1.36.0
	github.com/rs/zerolog v1.33.0
	github.com/samber/lo v1.44.0
	github.com/signalfx/splunk-otel-go/instrumentation/github.com/gomodule/redigo/splunkredigo v1.18.0
	go.akshayshah.org/connectproto v0.6.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.53.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.53.0
	go.opentelemetry.io/otel v1.28.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.28.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.28.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.28.0
	go.opentelemetry.io/otel/sdk v1.28.0
	go.opentelemetry.io/otel/sdk/metric v1.28.0
	go.opentelemetry.io/otel/trace v1.28.0
	golang.org/x/crypto v0.25.0
	google.golang.org/protobuf v1.34.2
	gorm.io/datatypes v1.2.1
	gorm.io/driver/postgres v1.5.9
	gorm.io/gorm v1.25.11
	gorm.io/plugin/opentelemetry v0.1.4
	lukechampine.com/frand v1.4.2
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/PuerkitoBio/goquery v1.9.2 // indirect
	github.com/aead/chacha20 v0.0.0-20180709150244-8b13a72661da // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.3 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.26 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.34.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.3.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.9.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.17.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.34.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.22.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.26.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.30.3 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-chi/chi/v5 v5.1.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mmcdole/goxpp v1.1.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/sagikazarmark/locafero v0.6.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/signalfx/splunk-otel-go/instrumentation/internal v1.18.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.19.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20240707233637-46b078467d37 // indirect
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	gonum.org/v1/gonum v0.15.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240711142825-46eb208f015d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240711142825-46eb208f015d // indirect
	google.golang.org/grpc v1.65.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/driver/mysql v1.5.7 // indirect
)
