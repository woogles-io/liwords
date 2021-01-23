package user

import (
	"bytes"
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/lithammer/shortuuid"
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
