package uploader

import (
	// "bytes"
	"context"
	"errors"
	"os"

	"github.com/lithammer/shortuuid"
)

type UploadService interface {
	Upload(context.Context, string, []byte) (string, error)
}

type XTUploadService struct {
}

func NewXTUploadService() (*XTUploadService) {
	return &XTUploadService{}
}

// Upload takes in JPG bytes
func (s *XTUploadService) Upload(ctx context.Context, prefix string, data []byte) (string, error) {
	id := prefix + "-" + shortuuid.New()[1:10] + ".jpg"

	// Store the file with a name reflective of the user's UUID
	// filename := user.UUID + ".jpg"
	// avatarUrl := "file:///Users/slipkin/Projects/woogles/liwords/" + filename

	f, createErr := os.Create(id)
	if createErr != nil {
		return "", errors.New("Cannot create local file")
	}

	_, writeErr := f.WriteString(string(data))
	if writeErr != nil {
		return "", errors.New("Cannot write local file")
	}

	avatarUrl := "file:///Users/slipkin/Projects/woogles/liwords/" + id
	return avatarUrl, nil
}