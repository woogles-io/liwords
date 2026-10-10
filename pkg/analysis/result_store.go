package analysis

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	macondo "github.com/domino14/macondo/gen/api/proto/macondo"
)

// ResultStore holds analysis results outside the database, one object per
// accepted result.
type ResultStore interface {
	Put(ctx context.Context, key string, result *macondo.GameAnalysisResult) error
	Get(ctx context.Context, key string) (*macondo.GameAnalysisResult, error)
	Delete(ctx context.Context, key string) error
}

// ResultKey returns a new object key for a result of jobID. Every accepted
// result gets its own key, so storing one never overwrites another: a
// submission CompleteJob rejects can't replace the accepted result, and a
// reanalysis only swaps the job's key once it is accepted. Keys are grouped by
// game so all of a game's objects share a prefix.
func ResultKey(gameID string, jobID uuid.UUID) string {
	return fmt.Sprintf("analyses/%s/%s-%d.json.gz", gameID, jobID, time.Now().UnixNano())
}

// unmarshalResult reads a stored result. DiscardUnknown handles legacy results
// with fields removed in newer macondo versions (e.g. avgSpreadLoss).
func unmarshalResult(raw []byte) (*macondo.GameAnalysisResult, error) {
	result := &macondo.GameAnalysisResult{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, result); err != nil {
		return nil, err
	}
	return result, nil
}

// S3ResultStore stores results as gzipped protojson, like game histories
// (pkg/stores/game/s3.go).
type S3ResultStore struct {
	client *s3.Client
	bucket string
}

func NewS3ResultStore(client *s3.Client, bucket string) *S3ResultStore {
	return &S3ResultStore{client: client, bucket: bucket}
}

func (s *S3ResultStore) Put(ctx context.Context, key string, result *macondo.GameAnalysisResult) error {
	ctx, span := tracer.Start(ctx, "analysis.ResultStore.Put")
	defer span.End()
	span.SetAttributes(attribute.String("s3.key", key))

	raw, err := protojson.Marshal(result)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(raw); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}

	_, err = manager.NewUploader(s.client).Upload(ctx, &s3.PutObjectInput{
		Bucket:          aws.String(s.bucket),
		Key:             aws.String(key),
		Body:            bytes.NewReader(buf.Bytes()),
		ContentType:     aws.String("application/json"),
		ContentEncoding: aws.String("gzip"),
	})
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("analysis-s3-put %s: %w", key, err)
	}
	return nil
}

func (s *S3ResultStore) Get(ctx context.Context, key string) (*macondo.GameAnalysisResult, error) {
	ctx, span := tracer.Start(ctx, "analysis.ResultStore.Get")
	defer span.End()
	span.SetAttributes(attribute.String("s3.key", key))

	buf := manager.NewWriteAtBuffer(nil)
	_, err := manager.NewDownloader(s.client).Download(ctx, buf, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("analysis-s3-get %s: %w", key, err)
	}
	gr, err := gzip.NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("analysis-s3-gzip %s: %w", key, err)
	}
	defer gr.Close()
	raw, err := io.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("analysis-s3-read %s: %w", key, err)
	}
	result, err := unmarshalResult(raw)
	if err != nil {
		return nil, fmt.Errorf("analysis-s3-unmarshal %s: %w", key, err)
	}
	return result, nil
}

func (s *S3ResultStore) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("analysis-s3-delete %s: %w", key, err)
	}
	return nil
}

// ErrResultNotFound is returned by MemoryResultStore.Get for a missing key.
var ErrResultNotFound = errors.New("analysis result not found")

// MemoryResultStore is an in-memory ResultStore for tests.
type MemoryResultStore struct {
	mu      sync.Mutex
	objects map[string]*macondo.GameAnalysisResult
	// FailPut and FailGet make calls fail, to exercise error paths.
	FailPut, FailGet bool
	// BeforePut, when set, runs at the start of each Put.
	BeforePut func()
}

func NewMemoryResultStore() *MemoryResultStore {
	return &MemoryResultStore{objects: map[string]*macondo.GameAnalysisResult{}}
}

func (m *MemoryResultStore) Put(_ context.Context, key string, result *macondo.GameAnalysisResult) error {
	if m.BeforePut != nil {
		m.BeforePut()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FailPut {
		return errors.New("memory store: put failed")
	}
	m.objects[key] = proto.Clone(result).(*macondo.GameAnalysisResult)
	return nil
}

func (m *MemoryResultStore) Get(_ context.Context, key string) (*macondo.GameAnalysisResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FailGet {
		return nil, errors.New("memory store: get failed")
	}
	r, ok := m.objects[key]
	if !ok {
		return nil, ErrResultNotFound
	}
	return proto.Clone(r).(*macondo.GameAnalysisResult), nil
}

func (m *MemoryResultStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, key)
	return nil
}

// Keys returns the stored keys.
func (m *MemoryResultStore) Keys() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := make([]string, 0, len(m.objects))
	for k := range m.objects {
		keys = append(keys, k)
	}
	return keys
}
