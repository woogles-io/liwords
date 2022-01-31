module github.com/domino14/liwords

go 1.17

// XXX: get rid of github.com/golang/protobuf module, it's obsolete
require (
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/aws/aws-sdk-go-v2 v1.2.0
	github.com/aws/aws-sdk-go-v2/config v1.1.1
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.0.2
	github.com/aws/aws-sdk-go-v2/service/s3 v1.2.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/domino14/macondo v0.4.5-0.20220125183153-f32108b23887
	github.com/golang/protobuf v1.4.2
	github.com/gomodule/redigo v1.8.2
	github.com/hashicorp/golang-lru v0.5.4
	github.com/jinzhu/gorm v1.9.16
	github.com/justinas/alice v1.2.0
	github.com/lib/pq v1.10.0
	github.com/lithammer/shortuuid v3.0.0+incompatible
	github.com/mailgun/mailgun-go/v4 v4.1.4
	github.com/matryer/is v1.4.0
	github.com/namsral/flag v1.7.4-pre
	github.com/nats-io/nats.go v1.10.0
	github.com/rs/zerolog v1.19.0
	github.com/twitchtv/twirp v8.1.1+incompatible
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	google.golang.org/protobuf v1.25.0
	gorm.io/datatypes v0.0.0-20200924071644-3967db6857cf
	gorm.io/driver/postgres v1.0.2
	gorm.io/gorm v1.20.2
)

require (
	github.com/aead/chacha20 v0.0.0-20180709150244-8b13a72661da // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.1.1 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.0.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.0.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.0.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.1.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.1.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.1.1 // indirect
	github.com/aws/smithy-go v1.1.0 // indirect
	github.com/go-chi/chi v4.0.0+incompatible // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.7.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.0.5 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/pgtype v1.5.0 // indirect
	github.com/jackc/pgx/v4 v4.9.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/mailru/easyjson v0.7.0 // indirect
	github.com/nats-io/jwt v0.3.2 // indirect
	github.com/nats-io/nkeys v0.1.4 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rs/xid v1.2.1 // indirect
	golang.org/x/sys v0.0.0-20200905004654-be1d3432aa8f // indirect
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	lukechampine.com/frand v1.4.1 // indirect
)
