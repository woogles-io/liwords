package verification

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/integrations/organizations"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// S3Uploader is an interface for uploading/deleting files from S3
type S3Uploader interface {
	Upload(ctx context.Context, prefix string, data []byte) (string, error)
	Delete(ctx context.Context, url string) error
	GetPresignedURL(ctx context.Context, url string, expiration time.Duration) (string, error)
}

// VerificationService handles identity verification for organization memberships
type VerificationService struct {
	queries   *models.Queries
	s3Uploader S3Uploader
}

// NewVerificationService creates a new verification service
func NewVerificationService(queries *models.Queries, s3Uploader S3Uploader) *VerificationService {
	return &VerificationService{
		queries:   queries,
		s3Uploader: s3Uploader,
	}
}

// SubmitVerificationRequest submits a new verification request
func (s *VerificationService) SubmitVerificationRequest(
	ctx context.Context,
	userUUID string,
	integrationName string,
	memberID string,
	imageData io.Reader,
	imageExt string,
) (*models.VerificationRequest, error) {
	// Check if user already has a pending verification for this organization
	hasPending, err := s.queries.HasPendingVerification(ctx, models.HasPendingVerificationParams{
		UserUuid:        pgtype.Text{String: userUUID, Valid: true},
		IntegrationName: integrationName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check pending verification: %w", err)
	}

	if hasPending {
		return nil, fmt.Errorf("user already has a pending verification request for %s", integrationName)
	}

	// Get the organization integration to fetch the real name and title
	orgCode := organizations.OrganizationCode(integrationName)
	integration, err := organizations.GetIntegration(orgCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization integration: %w", err)
	}

	// Fetch the real name from the organization (no credentials needed for public data)
	fullName, err := integration.GetRealName(memberID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch name from %s: %w", integrationName, err)
	}

	// Fetch the title from the organization (no credentials needed for public data)
	var title string
	titleInfo, err := integration.FetchTitle(memberID, nil)
	if err == nil && titleInfo != nil {
		title = string(titleInfo.NormalizedTitle)
	}
	// If fetching title fails, we continue without it (title is optional)

	// Read image data into memory
	imageBytes, err := io.ReadAll(imageData)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Upload to S3
	prefix := fmt.Sprintf("verification/%s_%s", userUUID, integrationName)
	imageURL, err := s.s3Uploader.Upload(ctx, prefix, imageBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to upload verification image: %w", err)
	}

	// Create the verification request with the fetched name and title
	req, err := s.queries.CreateVerificationRequest(ctx, models.CreateVerificationRequestParams{
		UserUuid:        pgtype.Text{String: userUUID, Valid: true},
		IntegrationName: integrationName,
		MemberID:        memberID,
		FullName:        fullName, // Name fetched from organization, not user input
		ImageUrl:        imageURL,
		Title:           pgtype.Text{String: title, Valid: title != ""},
	})
	if err != nil {
		// Clean up the uploaded image if database operation fails
		_ = s.s3Uploader.Delete(ctx, imageURL)
		return nil, fmt.Errorf("failed to create verification request: %w", err)
	}

	log.Info().
		Str("user_uuid", userUUID).
		Str("integration", integrationName).
		Str("member_id", memberID).
		Str("full_name", fullName).
		Msg("verification request submitted")

	return &req, nil
}

// ApproveVerificationRequest approves a verification request
func (s *VerificationService) ApproveVerificationRequest(
	ctx context.Context,
	requestID int64,
	reviewerUUID string,
	notes string,
) error {
	// Get the verification request details first
	req, err := s.queries.GetVerificationRequest(ctx, requestID)
	if err != nil {
		return fmt.Errorf("failed to get verification request: %w", err)
	}

	if req.Status != "pending" {
		return fmt.Errorf("verification request is not pending")
	}

	// Approve the request
	err = s.queries.ApproveVerificationRequest(ctx, models.ApproveVerificationRequestParams{
		ID:           requestID,
		ReviewerUuid: pgtype.Text{String: reviewerUUID, Valid: true},
		Notes:        pgtype.Text{String: notes, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to approve verification request: %w", err)
	}

	// Delete the verification image from S3
	if err := s.s3Uploader.Delete(ctx, req.ImageUrl); err != nil {
		log.Error().Err(err).Str("image_url", req.ImageUrl).Msg("failed to delete verification image after approval")
	}

	log.Info().
		Int64("request_id", requestID).
		Str("reviewer", reviewerUUID).
		Str("user_uuid", req.UserUuid.String).
		Str("integration", req.IntegrationName).
		Msg("verification request approved")

	return nil
}

// RejectVerificationRequest rejects a verification request
func (s *VerificationService) RejectVerificationRequest(
	ctx context.Context,
	requestID int64,
	reviewerUUID string,
	notes string,
) error {
	// Get the verification request details first
	req, err := s.queries.GetVerificationRequest(ctx, requestID)
	if err != nil {
		return fmt.Errorf("failed to get verification request: %w", err)
	}

	if req.Status != "pending" {
		return fmt.Errorf("verification request is not pending")
	}

	// Reject the request
	err = s.queries.RejectVerificationRequest(ctx, models.RejectVerificationRequestParams{
		ID:           requestID,
		ReviewerUuid: pgtype.Text{String: reviewerUUID, Valid: true},
		Notes:        pgtype.Text{String: notes, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to reject verification request: %w", err)
	}

	// Delete the verification image from S3
	if err := s.s3Uploader.Delete(ctx, req.ImageUrl); err != nil {
		log.Error().Err(err).Str("image_url", req.ImageUrl).Msg("failed to delete verification image after rejection")
	}

	log.Info().
		Int64("request_id", requestID).
		Str("reviewer", reviewerUUID).
		Str("user_uuid", req.UserUuid.String).
		Str("integration", req.IntegrationName).
		Msg("verification request rejected")

	return nil
}

// GetPendingVerifications retrieves all pending verification requests
func (s *VerificationService) GetPendingVerifications(ctx context.Context) ([]models.GetPendingVerificationsRow, error) {
	return s.queries.GetPendingVerifications(ctx)
}

// GetUserVerificationRequests retrieves all verification requests for a user
func (s *VerificationService) GetUserVerificationRequests(ctx context.Context, userUUID string) ([]models.VerificationRequest, error) {
	return s.queries.GetUserVerificationRequests(ctx, pgtype.Text{String: userUUID, Valid: true})
}

// GetVerificationRequest retrieves a single verification request by ID
func (s *VerificationService) GetVerificationRequest(ctx context.Context, requestID int64) (models.GetVerificationRequestRow, error) {
	return s.queries.GetVerificationRequest(ctx, requestID)
}

// GetPresignedImageURL generates a temporary presigned URL for a verification image
// The URL expires after 15 minutes
func (s *VerificationService) GetPresignedImageURL(ctx context.Context, imageURL string) (string, error) {
	// Normalize the URL to use localhost for presigning
	// (in case it was stored with minio:9000)
	normalizedURL := normalizeMinioURL(imageURL)
	return s.s3Uploader.GetPresignedURL(ctx, normalizedURL, 15*time.Minute)
}

// normalizeMinioURL replaces internal Docker hostname with localhost
func normalizeMinioURL(url string) string {
	return strings.Replace(url, "http://minio:9000", "http://localhost:9000", 1)
}
