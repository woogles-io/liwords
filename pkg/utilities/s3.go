package utilities

import (
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"
)

func CustomClientOptions(o *s3.Options) {
	if os.Getenv("USE_MINIO_S3") == "1" {
		log.Debug().Str("service", "s3").Msg("using-minio-endpoint")
		endpoint := os.Getenv("MINIO_S3_ENDPOINT")

		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
		o.EndpointOptions.DisableHTTPS = true
	}
}
