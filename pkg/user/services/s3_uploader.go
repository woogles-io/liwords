package services

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/lithammer/shortuuid/v4"
)

const (
	MaxFilesize = 1024 * 1024
)

type S3Uploader struct {
	bucket   string
	s3Client *s3.Client
}

func NewS3Uploader(bucket string, s3Client *s3.Client) *S3Uploader {
	return &S3Uploader{bucket: bucket, s3Client: s3Client}
}

func (s *S3Uploader) urlprefix() string {
	if os.Getenv("USE_MINIO_S3") == "1" {
		return "http://localhost:9000/" + s.bucket + "/"
	}
	return "https://" + s.bucket + ".s3.amazonaws.com/"
}

// Upload takes in JPG bytes
func (s *S3Uploader) Upload(ctx context.Context, prefix string, data []byte) (string, error) {
	if len(data) > MaxFilesize {
		return "", fmt.Errorf("file size is too big; avatar must be a square JPG < %dKB", MaxFilesize/1024)
	}

	if s.s3Client == nil {
		return "", fmt.Errorf("S3 client not initialized")
	}

	if s.bucket == "" {
		return "", fmt.Errorf("S3 bucket not configured")
	}

	uploader := manager.NewUploader(s.s3Client)

	// create a unique id
	suffix := shortuuid.New()[1:10] + ".jpg"
	id := prefix + "-" + suffix

	// cache for a long time as this is a unique id.
	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
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

// UploadVerificationImage uploads verification images with a higher size limit (4MB)
// and a more appropriate error message
func (s *S3Uploader) UploadVerificationImage(ctx context.Context, prefix string, data []byte) (string, error) {
	const maxVerificationImageSize = 4 * 1024 * 1024 // 4MB
	if len(data) > maxVerificationImageSize {
		return "", fmt.Errorf("image must be smaller than 4MB. Please compress or resize your image")
	}

	if s.s3Client == nil {
		return "", fmt.Errorf("S3 client not initialized")
	}

	if s.bucket == "" {
		return "", fmt.Errorf("S3 bucket not configured")
	}

	uploader := manager.NewUploader(s.s3Client)

	// create a unique id
	suffix := shortuuid.New()[1:10] + ".jpg"
	id := prefix + "-" + suffix

	// cache for a long time as this is a unique id.
	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
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

	_, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	return nil
}

// GetPresignedURL generates a temporary presigned URL for private S3 objects
// The URL expires after the specified duration (e.g., 15 minutes)
func (s *S3Uploader) GetPresignedURL(ctx context.Context, url string, expiration time.Duration) (string, error) {
	if !strings.HasPrefix(url, s.urlprefix()) {
		return "", fmt.Errorf("this is not an S3 URL: %v", url)
	}
	key := strings.TrimPrefix(url, s.urlprefix())

	// For local MinIO, create a new client with localhost endpoint for presigning
	// This ensures the signature is calculated for localhost:9000, not minio:9000
	var presignClient *s3.Client
	if os.Getenv("USE_MINIO_S3") == "1" {
		minioEndpoint := os.Getenv("MINIO_S3_ENDPOINT")
		if strings.Contains(minioEndpoint, "minio:") {
			// Create a new client with localhost endpoint for presigning
			// This way the signature will be valid for localhost:9000
			localhostEndpoint := strings.Replace(minioEndpoint, "minio:9000", "localhost:9000", 1)
			presignClient = s3.NewFromConfig(aws.Config{
				Region: "us-east-1",
				Credentials: s.s3Client.Options().Credentials,
				BaseEndpoint: aws.String(localhostEndpoint),
			}, func(o *s3.Options) {
				o.UsePathStyle = true
			})
		} else {
			presignClient = s.s3Client
		}
	} else {
		presignClient = s.s3Client
	}

	presigner := s3.NewPresignClient(presignClient)

	presignedReq, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiration
	})

	if err != nil {
		return "", fmt.Errorf("failed to create presigned URL: %w", err)
	}

	return presignedReq.URL, nil
}
