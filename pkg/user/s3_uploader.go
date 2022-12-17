package user

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/domino14/liwords/pkg/utilities"
	"github.com/lithammer/shortuuid"
)

const (
	MaxFilesize = 1024 * 1024
)

type S3Uploader struct {
	bucket string
}

func NewS3Uploader(bucket string) *S3Uploader {
	return &S3Uploader{bucket: bucket}
}

func (s *S3Uploader) urlprefix() string {
	if os.Getenv("USE_LOCALSTACK_S3") == "1" {
		return "http://localhost:4566/" + s.bucket + "/"
	}
	return "https://" + s.bucket + ".s3.amazonaws.com/"
}

// Upload takes in JPG bytes
func (s *S3Uploader) Upload(ctx context.Context, prefix string, data []byte) (string, error) {
	cfg, err := config.LoadDefaultConfig(
		ctx, config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(utilities.CustomResolver)))
	if err != nil {
		return "", err
	}

	if len(data) > MaxFilesize {
		return "", fmt.Errorf("file size is too big; avatar must be a square JPG < %dKB", MaxFilesize/1024)
	}

	client := s3.NewFromConfig(cfg, utilities.CustomClientOptions)
	uploader := manager.NewUploader(client)

	// create a unique id
	suffix := shortuuid.New()[1:10] + ".jpg"
	id := prefix + "-" + suffix

	// cache for a long time as this is a unique id.
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(s.bucket),
		Key:          aws.String(id),
		Body:         bytes.NewReader(data),
		ContentType:  aws.String("image/jpeg"),
		Expires:      aws.Time(time.Now().Add(time.Hour * 24 * 365)),
		CacheControl: aws.String("max-age=31536000"),
	})
	if err != nil {
		return "", err
	}

	return s.urlprefix() + id, nil
}

// Delete wipes out the avatar at the given URL.
func (s *S3Uploader) Delete(ctx context.Context, url string) error {

	if !strings.HasPrefix(url, s.urlprefix()) {
		return fmt.Errorf("this is not an S3 URL: %v", url)
	}
	key := strings.TrimPrefix(url, s.urlprefix())

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	client := s3.NewFromConfig(cfg)

	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	return nil
}
