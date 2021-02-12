package user

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/lithammer/shortuuid"
)

const (
	MaxFilesize = 524288
)

type S3Uploader struct {
	bucket string
}

func NewS3Uploader(bucket string) *S3Uploader {
	return &S3Uploader{bucket: bucket}
}

// Upload takes in JPG bytes
func (s *S3Uploader) Upload(ctx context.Context, prefix string, data []byte) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", err
	}

	if len(data) > MaxFilesize {
		return "", fmt.Errorf("file size is too big; avatar must be < %dKB", MaxFilesize/1024)
	}

	client := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(client)

	// create a unique id
	suffix := shortuuid.New()[1:10] + ".jpg"
	id := prefix + "-" + suffix

	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(id),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return "", err
	}

	return "https://" + s.bucket + ".s3.amazonaws.com/" + id, nil
}
