package organization

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/integrations/organizations"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	"github.com/woogles-io/liwords/pkg/verification"
	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrganizationService struct {
	userStore           user.Store
	queries             *models.Queries
	verificationService *verification.VerificationService
}

func NewOrganizationService(
	u user.Store,
	q *models.Queries,
	vs *verification.VerificationService,
) *OrganizationService {
	return &OrganizationService{
		userStore:           u,
		queries:             q,
		verificationService: vs,
	}
}

// ConnectOrganization connects a user's account to an organization
func (s *OrganizationService) ConnectOrganization(
	ctx context.Context,
	req *connect.Request[pb.ConnectOrganizationRequest],
) (*connect.Response[pb.ConnectOrganizationResponse], error) {
	user, err := apiserver.AuthUser(ctx, s.userStore)
	if err != nil {
		return nil, err
	}

	orgCode := organizations.OrganizationCode(req.Msg.OrganizationCode)
	meta, err := organizations.GetOrganizationMetadata(orgCode)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	// Organizations that require manual verification must use SubmitVerification flow
	if meta.RequiresVerification {
		return nil, apiserver.InvalidArg(
			fmt.Sprintf("%s requires identity verification. Please use the verification submission flow instead.", meta.Name),
		)
	}

	// If the organization doesn't have an API, we can't fetch data
	if !meta.HasAPI {
		return nil, apiserver.InvalidArg(
			fmt.Sprintf("%s does not have an API available for automatic verification", meta.Name),
		)
	}

	// Get the integration to fetch title and name
	integration, err := organizations.GetIntegration(orgCode)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	// Try to fetch title from API
	titleInfo, err := integration.FetchTitle(req.Msg.MemberId, req.Msg.Credentials)
	if err != nil {
		log.Error().Err(err).Str("org", string(orgCode)).Msg("failed to fetch title from API")
		return nil, apiserver.InvalidArg(fmt.Sprintf("failed to fetch title from %s: %v", meta.Name, err))
	}

	// Prepare integration data - full name and member ID come from the organization
	// Use titleInfo.MemberID if available (e.g., ABSP fetches it from profile),
	// otherwise fall back to the request's member ID (e.g., NASPA where user provides it)
	memberID := titleInfo.MemberID
	if memberID == "" {
		memberID = req.Msg.MemberId
	}

	now := time.Now()
	integrationData := organizations.OrganizationIntegrationData{
		MemberID:           memberID,
		FullName:           titleInfo.FullName,
		RawTitle:           titleInfo.RawTitle,
		NormalizedTitle:    titleInfo.NormalizedTitle,
		Verified:           true,
		VerificationMethod: "api",
		LastFetched:        &now,
	}

	// Encrypt credentials if provided
	if meta.RequiresAuth && len(req.Msg.Credentials) > 0 {
		encryptedCreds, err := organizations.EncryptCredentials(req.Msg.Credentials)
		if err != nil {
			return nil, apiserver.InternalErr(fmt.Errorf("failed to encrypt credentials: %w", err))
		}
		integrationData.EncryptedCredentials = encryptedCreds
	}

	// Store in integrations table
	dataJSON, err := integrationData.ToJSON()
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	_, err = s.queries.AddOrUpdateIntegration(ctx, models.AddOrUpdateIntegrationParams{
		UserUuid:        pgtype.Text{String: user.UUID, Valid: true},
		IntegrationName: string(orgCode),
		Data:            dataJSON,
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	// Update profile title with highest normalized title
	if err := s.updateProfileTitle(ctx, user.UUID); err != nil {
		log.Error().Err(err).Msg("failed to update profile title")
	}

	// Build response
	resp := &pb.ConnectOrganizationResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully connected to %s", meta.Name),
		Title:   convertTitleInfoToProto(titleInfo),
	}

	log.Info().
		Str("user_uuid", user.UUID).
		Str("org", string(orgCode)).
		Str("full_name", titleInfo.FullName).
		Msg("organization connected")

	return connect.NewResponse(resp), nil
}

// DisconnectOrganization disconnects a user's account from an organization
func (s *OrganizationService) DisconnectOrganization(
	ctx context.Context,
	req *connect.Request[pb.DisconnectOrganizationRequest],
) (*connect.Response[pb.DisconnectOrganizationResponse], error) {
	user, err := apiserver.AuthUser(ctx, s.userStore)
	if err != nil {
		return nil, err
	}

	// Determine target user (for admin disconnect or self disconnect)
	targetUserUUID := user.UUID
	if req.Msg.Username != nil && *req.Msg.Username != "" {
		// Admin is disconnecting another user's organization
		hasPermission, err := rbac.HasPermission(ctx, s.queries, user.ID, rbac.CanVerifyUserIdentities)
		if err != nil {
			return nil, apiserver.InternalErr(err)
		}
		if !hasPermission {
			return nil, apiserver.PermissionDenied("you do not have permission to disconnect other users' organizations")
		}

		// Get the target user by username
		targetUser, err := s.userStore.Get(ctx, *req.Msg.Username)
		if err != nil {
			return nil, apiserver.InvalidArg("user not found")
		}
		targetUserUUID = targetUser.UUID
	}

	// Get the integration UUID first
	integrationData, err := s.queries.GetIntegrationData(ctx, models.GetIntegrationDataParams{
		UserUuid:        pgtype.Text{String: targetUserUUID, Valid: true},
		IntegrationName: req.Msg.OrganizationCode,
	})
	if err != nil {
		return nil, apiserver.InvalidArg("organization not connected")
	}

	// Delete the integration
	err = s.queries.DeleteIntegration(ctx, models.DeleteIntegrationParams{
		IntegrationUuid: integrationData.Uuid,
		UserUuid:        pgtype.Text{String: targetUserUUID, Valid: true},
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	// Update profile title
	if err := s.updateProfileTitle(ctx, targetUserUUID); err != nil {
		log.Error().Err(err).Msg("failed to update profile title")
	}

	log.Info().
		Str("user_uuid", targetUserUUID).
		Str("org", req.Msg.OrganizationCode).
		Msg("organization disconnected")

	return connect.NewResponse(&pb.DisconnectOrganizationResponse{
		Success: true,
	}), nil
}

// RefreshTitles manually refreshes titles from all connected organizations with APIs
func (s *OrganizationService) RefreshTitles(
	ctx context.Context,
	req *connect.Request[pb.RefreshTitlesRequest],
) (*connect.Response[pb.RefreshTitlesResponse], error) {
	user, err := apiserver.AuthUser(ctx, s.userStore)
	if err != nil {
		return nil, err
	}

	// Get all organization integrations for the user
	integrations, err := s.queries.GetOrganizationIntegrations(ctx, pgtype.Text{String: user.UUID, Valid: true})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	var titles []*pb.OrganizationTitle
	var errors []string

	for _, integ := range integrations {
		orgCode := organizations.OrganizationCode(integ.IntegrationName)
		meta, err := organizations.GetOrganizationMetadata(orgCode)
		if err != nil {
			continue
		}

		// Skip organizations without APIs
		if !meta.HasAPI {
			continue
		}

		// Parse existing data
		var integData organizations.OrganizationIntegrationData
		if err := integData.FromJSON(integ.Data); err != nil {
			log.Error().Err(err).Str("org", string(orgCode)).Msg("failed to parse integration data")
			continue
		}

		// Get integration service
		integration, err := organizations.GetIntegration(orgCode)
		if err != nil {
			continue
		}

		// Decrypt credentials if needed
		var credentials map[string]string
		if integData.EncryptedCredentials != "" {
			credentials, err = organizations.DecryptCredentials(integData.EncryptedCredentials)
			if err != nil {
				errors = append(errors, fmt.Sprintf("failed to decrypt credentials for %s", meta.Name))
				continue
			}
		}

		// Fetch title from API
		titleInfo, err := integration.FetchTitle(integData.MemberID, credentials)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to fetch title from %s", meta.Name))
			log.Error().Err(err).Str("org", string(orgCode)).Msg("failed to fetch title from API")
			continue
		}

		// Update integration data with latest from API
		integData.RawTitle = titleInfo.RawTitle
		integData.NormalizedTitle = titleInfo.NormalizedTitle
		integData.FullName = titleInfo.FullName // Update full name in case it changed
		now := time.Now()
		integData.LastFetched = &now

		dataJSON, err := integData.ToJSON()
		if err != nil {
			continue
		}

		// Update in database
		err = s.queries.UpdateIntegrationData(ctx, models.UpdateIntegrationDataParams{
			Uuid: integ.Uuid,
			Data: dataJSON,
		})
		if err != nil {
			log.Error().Err(err).Str("org", string(orgCode)).Msg("failed to update integration data")
			continue
		}

		titles = append(titles, convertTitleInfoToProto(titleInfo))
	}

	// Update profile title with highest normalized title
	if err := s.updateProfileTitle(ctx, user.UUID); err != nil {
		log.Error().Err(err).Msg("failed to update profile title")
	}

	message := "Titles refreshed successfully"
	if len(errors) > 0 {
		message = fmt.Sprintf("Titles partially refreshed. Errors: %v", errors)
	}

	log.Info().
		Str("user_uuid", user.UUID).
		Int("count", len(titles)).
		Msg("titles refreshed")

	return connect.NewResponse(&pb.RefreshTitlesResponse{
		Titles:  titles,
		Message: message,
	}), nil
}

// GetMyOrganizations returns all organizations the user is connected to
func (s *OrganizationService) GetMyOrganizations(
	ctx context.Context,
	req *connect.Request[pb.GetMyOrganizationsRequest],
) (*connect.Response[pb.GetMyOrganizationsResponse], error) {
	user, err := apiserver.AuthUser(ctx, s.userStore)
	if err != nil {
		return nil, err
	}

	integrations, err := s.queries.GetOrganizationIntegrations(ctx, pgtype.Text{String: user.UUID, Valid: true})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	var titles []*pb.OrganizationTitle

	for _, integ := range integrations {
		orgCode := organizations.OrganizationCode(integ.IntegrationName)
		meta, _ := organizations.GetOrganizationMetadata(orgCode)

		var integData organizations.OrganizationIntegrationData
		if err := integData.FromJSON(integ.Data); err != nil {
			continue
		}

		title := &pb.OrganizationTitle{
			OrganizationCode: string(orgCode),
			OrganizationName: meta.Name,
			MemberId:         integData.MemberID,
			FullName:         integData.FullName,
			RawTitle:         integData.RawTitle,
			NormalizedTitle:  string(integData.NormalizedTitle),
			Verified:         integData.Verified,
		}

		if integData.LastFetched != nil {
			title.LastFetched = timestamppb.New(*integData.LastFetched)
		}

		titles = append(titles, title)
	}

	return connect.NewResponse(&pb.GetMyOrganizationsResponse{
		Titles: titles,
	}), nil
}

// GetPublicOrganizations retrieves public organization information for a user by username
func (s *OrganizationService) GetPublicOrganizations(
	ctx context.Context,
	req *connect.Request[pb.GetPublicOrganizationsRequest],
) (*connect.Response[pb.GetPublicOrganizationsResponse], error) {
	// Check if the requesting user is an admin (can view sensitive fields)
	isAdmin := false
	if requestingUser, err := apiserver.AuthUser(ctx, s.userStore); err == nil {
		hasPermission, _ := rbac.HasPermission(ctx, s.queries, requestingUser.ID, rbac.CanVerifyUserIdentities)
		isAdmin = hasPermission
	}

	// Get user by username
	user, err := s.userStore.Get(ctx, req.Msg.Username)
	if err != nil {
		return nil, apiserver.InvalidArg("user not found")
	}

	integrations, err := s.queries.GetOrganizationIntegrations(ctx, pgtype.Text{String: user.UUID, Valid: true})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	var titles []*pb.OrganizationTitle

	for _, integ := range integrations {
		orgCode := organizations.OrganizationCode(integ.IntegrationName)
		meta, _ := organizations.GetOrganizationMetadata(orgCode)

		var integData organizations.OrganizationIntegrationData
		if err := integData.FromJSON(integ.Data); err != nil {
			continue
		}

		// Only return verified organizations for public view (admins can see all)
		if !isAdmin && !integData.Verified {
			continue
		}

		title := &pb.OrganizationTitle{
			OrganizationCode: string(orgCode),
			OrganizationName: meta.Name,
			NormalizedTitle:  string(integData.NormalizedTitle),
			Verified:         integData.Verified,
		}

		// Include sensitive fields for admins only
		if isAdmin {
			title.MemberId = integData.MemberID
			title.FullName = integData.FullName
			title.RawTitle = integData.RawTitle
			if integData.LastFetched != nil {
				title.LastFetched = timestamppb.New(*integData.LastFetched)
			}
		}

		titles = append(titles, title)
	}

	return connect.NewResponse(&pb.GetPublicOrganizationsResponse{
		Titles: titles,
	}), nil
}

// SubmitVerification submits a verification request for manual verification
func (s *OrganizationService) SubmitVerification(
	ctx context.Context,
	req *connect.Request[pb.SubmitVerificationRequest],
) (*connect.Response[pb.SubmitVerificationResponse], error) {
	user, err := apiserver.AuthUser(ctx, s.userStore)
	if err != nil {
		return nil, err
	}

	orgCode := organizations.OrganizationCode(req.Msg.OrganizationCode)
	meta, err := organizations.GetOrganizationMetadata(orgCode)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	if !meta.RequiresVerification {
		return nil, apiserver.InvalidArg(fmt.Sprintf("%s does not require manual verification", meta.Name))
	}

	// Submit verification request (full name will be fetched from organization)
	imageReader := bytes.NewReader(req.Msg.ImageData)
	verReq, err := s.verificationService.SubmitVerificationRequest(
		ctx,
		user.UUID,
		req.Msg.OrganizationCode,
		req.Msg.MemberId,
		imageReader,
		req.Msg.ImageExtension,
	)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	log.Info().
		Str("user_uuid", user.UUID).
		Str("org", req.Msg.OrganizationCode).
		Int64("request_id", verReq.ID).
		Msg("verification request submitted")

	return connect.NewResponse(&pb.SubmitVerificationResponse{
		Success:   true,
		Message:   "Verification request submitted successfully. It will be reviewed by an administrator.",
		RequestId: verReq.ID,
	}), nil
}

// GetPendingVerifications returns all pending verification requests (admin only)
func (s *OrganizationService) GetPendingVerifications(
	ctx context.Context,
	req *connect.Request[pb.GetPendingVerificationsRequest],
) (*connect.Response[pb.GetPendingVerificationsResponse], error) {
	user, err := apiserver.AuthUser(ctx, s.userStore)
	if err != nil {
		return nil, err
	}

	// Check permission
	hasPermission, err := rbac.HasPermission(ctx, s.queries, user.ID, rbac.CanVerifyUserIdentities)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	if !hasPermission {
		return nil, apiserver.PermissionDenied("you do not have permission to verify user identities")
	}

	// Get pending verifications
	requests, err := s.verificationService.GetPendingVerifications(ctx)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	var protoRequests []*pb.VerificationRequestInfo
	for _, r := range requests {
		// Note: Image URL is not included here to avoid generating presigned URLs for all requests
		// Use GetVerificationImageUrl endpoint to get presigned URL on-demand when needed
		protoRequests = append(protoRequests, &pb.VerificationRequestInfo{
			RequestId:        r.ID,
			UserUuid:         r.UserUuid.String,
			Username:         r.Username.String,
			OrganizationCode: r.IntegrationName,
			MemberId:         r.MemberID,
			FullName:         r.FullName,
			ImageUrl:         "", // Empty - use GetVerificationImageUrl to fetch on-demand
			SubmittedAt:      timestamppb.New(r.SubmittedAt.Time),
			Status:           r.Status,
			Title:            r.Title.String,
			Notes:            r.Notes.String,
		})
	}

	return connect.NewResponse(&pb.GetPendingVerificationsResponse{
		Requests: protoRequests,
	}), nil
}

// GetVerificationImageUrl generates a presigned URL for a verification image (admin only)
func (s *OrganizationService) GetVerificationImageUrl(
	ctx context.Context,
	req *connect.Request[pb.GetVerificationImageUrlRequest],
) (*connect.Response[pb.GetVerificationImageUrlResponse], error) {
	user, err := apiserver.AuthUser(ctx, s.userStore)
	if err != nil {
		return nil, err
	}

	// Check permission
	hasPermission, err := rbac.HasPermission(ctx, s.queries, user.ID, rbac.CanVerifyUserIdentities)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	if !hasPermission {
		return nil, apiserver.PermissionDenied("you do not have permission to verify user identities")
	}

	// Get the verification request
	verificationReq, err := s.verificationService.GetVerificationRequest(ctx, req.Msg.RequestId)
	if err != nil {
		return nil, apiserver.InvalidArg("verification request not found")
	}

	// Generate presigned URL for the verification image (expires in 15 minutes)
	presignedURL, err := s.verificationService.GetPresignedImageURL(ctx, verificationReq.ImageUrl)
	if err != nil {
		log.Error().Err(err).Str("image_url", verificationReq.ImageUrl).Msg("failed to generate presigned URL")
		return nil, apiserver.InternalErr(err)
	}

	return connect.NewResponse(&pb.GetVerificationImageUrlResponse{
		ImageUrl: presignedURL,
	}), nil
}

// ApproveVerification approves a verification request (admin only)
func (s *OrganizationService) ApproveVerification(
	ctx context.Context,
	req *connect.Request[pb.ApproveVerificationRequest],
) (*connect.Response[pb.ApproveVerificationResponse], error) {
	user, err := apiserver.AuthUser(ctx, s.userStore)
	if err != nil {
		return nil, err
	}

	// Check permission
	hasPermission, err := rbac.HasPermission(ctx, s.queries, user.ID, rbac.CanVerifyUserIdentities)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	if !hasPermission {
		return nil, apiserver.PermissionDenied("you do not have permission to verify user identities")
	}

	// Get the verification request
	verReq, err := s.queries.GetVerificationRequest(ctx, req.Msg.RequestId)
	if err != nil {
		return nil, apiserver.InvalidArg("verification request not found")
	}

	// Approve the request
	err = s.verificationService.ApproveVerificationRequest(ctx, req.Msg.RequestId, user.UUID, req.Msg.Notes)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	// Fetch real name from organization if possible
	fullName := verReq.FullName // Default to submitted name
	orgCode := organizations.OrganizationCode(verReq.IntegrationName)
	integration, err := organizations.GetIntegration(orgCode)
	if err == nil {
		// Try to fetch real name from organization (e.g., WESPA HTML scraping)
		if fetchedName, err := integration.GetRealName(verReq.MemberID, nil); err == nil {
			fullName = fetchedName
			log.Info().
				Str("submitted_name", verReq.FullName).
				Str("fetched_name", fetchedName).
				Str("org", string(orgCode)).
				Msg("fetched real name from organization")
		} else {
			log.Warn().Err(err).Str("org", string(orgCode)).Msg("failed to fetch real name from organization, using submitted name")
		}
	}

	// Create the integration for the user
	now := time.Now()
	integrationData := organizations.OrganizationIntegrationData{
		MemberID:           verReq.MemberID,
		FullName:           fullName,
		Verified:           true,
		VerificationMethod: "manual",
		VerifiedAt:         &now,
		VerifiedBy:         user.UUID,
	}

	dataJSON, err := integrationData.ToJSON()
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	_, err = s.queries.AddOrUpdateIntegration(ctx, models.AddOrUpdateIntegrationParams{
		UserUuid:        pgtype.Text{String: verReq.UserUuid.String, Valid: true},
		IntegrationName: verReq.IntegrationName,
		Data:            dataJSON,
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	// Update profile title
	if err := s.updateProfileTitle(ctx, verReq.UserUuid.String); err != nil {
		log.Error().Err(err).Msg("failed to update profile title")
	}

	log.Info().
		Int64("request_id", req.Msg.RequestId).
		Str("reviewer", user.UUID).
		Str("user", verReq.UserUuid.String).
		Msg("verification request approved")

	return connect.NewResponse(&pb.ApproveVerificationResponse{
		Success: true,
		Message: "Verification request approved successfully",
	}), nil
}

// RejectVerification rejects a verification request (admin only)
func (s *OrganizationService) RejectVerification(
	ctx context.Context,
	req *connect.Request[pb.RejectVerificationRequest],
) (*connect.Response[pb.RejectVerificationResponse], error) {
	user, err := apiserver.AuthUser(ctx, s.userStore)
	if err != nil {
		return nil, err
	}

	// Check permission
	hasPermission, err := rbac.HasPermission(ctx, s.queries, user.ID, rbac.CanVerifyUserIdentities)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	if !hasPermission {
		return nil, apiserver.PermissionDenied("you do not have permission to verify user identities")
	}

	// Reject the request
	err = s.verificationService.RejectVerificationRequest(ctx, req.Msg.RequestId, user.UUID, req.Msg.Notes)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	log.Info().
		Int64("request_id", req.Msg.RequestId).
		Str("reviewer", user.UUID).
		Msg("verification request rejected")

	return connect.NewResponse(&pb.RejectVerificationResponse{
		Success: true,
		Message: "Verification request rejected",
	}), nil
}

// ManuallySetOrgMembership manually sets an organization membership (admin only)
// Fetches the real name and title from the organization automatically
func (s *OrganizationService) ManuallySetOrgMembership(
	ctx context.Context,
	req *connect.Request[pb.ManuallySetOrgMembershipRequest],
) (*connect.Response[pb.ManuallySetOrgMembershipResponse], error) {
	user, err := apiserver.AuthUser(ctx, s.userStore)
	if err != nil {
		return nil, err
	}

	// Check permission
	hasPermission, err := rbac.HasPermission(ctx, s.queries, user.ID, rbac.CanVerifyUserIdentities)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	if !hasPermission {
		return nil, apiserver.PermissionDenied("you do not have permission to verify user identities")
	}

	// Get the target user by username
	targetUser, err := s.userStore.Get(ctx, req.Msg.Username)
	if err != nil {
		return nil, apiserver.InvalidArg("user not found")
	}

	// Get the organization integration
	orgCode := organizations.OrganizationCode(req.Msg.OrganizationCode)
	meta, err := organizations.GetOrganizationMetadata(orgCode)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	integration, err := organizations.GetIntegration(orgCode)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	// Fetch name and title from organization
	var fullName string
	var rawTitle string
	var normalizedTitle organizations.NormalizedTitle

	// For organizations with public data (NASPA, ABSP), use FetchTitleWithoutAuth
	// This allows admins to set memberships without requiring user credentials
	switch orgCode {
	case organizations.OrgNASPA:
		// NASPA has a public API that can be used without authentication
		naspaIntegration := integration.(*organizations.NASPAIntegration)
		titleInfo, err := naspaIntegration.FetchTitleWithoutAuth(req.Msg.MemberId)
		if err != nil {
			return nil, apiserver.InvalidArg(fmt.Sprintf("failed to fetch data from NASPA: %v", err))
		}
		fullName = titleInfo.FullName
		rawTitle = titleInfo.RawTitle
		normalizedTitle = titleInfo.NormalizedTitle

	case organizations.OrgABSP:
		// ABSP has a public database that can be used without authentication
		abspIntegration := integration.(*organizations.ABSPIntegration)
		titleInfo, err := abspIntegration.FetchTitleWithoutAuth(req.Msg.MemberId)
		if err != nil {
			return nil, apiserver.InvalidArg(fmt.Sprintf("failed to fetch data from ABSP: %v", err))
		}
		fullName = titleInfo.FullName
		rawTitle = titleInfo.RawTitle
		normalizedTitle = titleInfo.NormalizedTitle

	default:
		// For other orgs (like WESPA), use GetRealName for HTML scraping
		fullName, err = integration.GetRealName(req.Msg.MemberId, nil)
		if err != nil {
			return nil, apiserver.InvalidArg(fmt.Sprintf("failed to fetch name from %s: %v", meta.Name, err))
		}
	}

	// Create the integration with verified status
	now := time.Now()
	integrationData := organizations.OrganizationIntegrationData{
		MemberID:           req.Msg.MemberId,
		FullName:           fullName,
		RawTitle:           rawTitle,
		NormalizedTitle:    normalizedTitle,
		Verified:           true,
		VerificationMethod: "admin",
		VerifiedAt:         &now,
		VerifiedBy:         user.UUID,
		LastFetched:        &now,
	}

	dataJSON, err := integrationData.ToJSON()
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	_, err = s.queries.AddOrUpdateIntegration(ctx, models.AddOrUpdateIntegrationParams{
		UserUuid:        pgtype.Text{String: targetUser.UUID, Valid: true},
		IntegrationName: req.Msg.OrganizationCode,
		Data:            dataJSON,
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	// Update profile title
	if err := s.updateProfileTitle(ctx, targetUser.UUID); err != nil {
		log.Error().Err(err).Msg("failed to update profile title")
	}

	log.Info().
		Str("admin", user.UUID).
		Str("username", req.Msg.Username).
		Str("user_uuid", targetUser.UUID).
		Str("org", req.Msg.OrganizationCode).
		Str("member_id", req.Msg.MemberId).
		Str("full_name", fullName).
		Str("title", string(normalizedTitle)).
		Msg("organization membership manually set")

	return connect.NewResponse(&pb.ManuallySetOrgMembershipResponse{
		Success: true,
		Message: fmt.Sprintf("Organization membership set successfully for %s", fullName),
	}), nil
}

// Helper functions

// GetCachedRealName returns the cached real name from the integrations table
func (s *OrganizationService) GetCachedRealName(ctx context.Context, userUUID string, orgCode string) (string, error) {
	integrationData, err := s.queries.GetIntegrationData(ctx, models.GetIntegrationDataParams{
		UserUuid:        pgtype.Text{String: userUUID, Valid: true},
		IntegrationName: orgCode,
	})
	if err != nil {
		return "", fmt.Errorf("organization not connected: %w", err)
	}

	var integData organizations.OrganizationIntegrationData
	if err := integData.FromJSON(integrationData.Data); err != nil {
		return "", fmt.Errorf("failed to parse integration data: %w", err)
	}

	if integData.FullName == "" {
		return "", fmt.Errorf("full name not available in cached data")
	}

	return integData.FullName, nil
}

func (s *OrganizationService) updateProfileTitle(ctx context.Context, userUUID string) error {
	// Get all organization integrations
	integrations, err := s.queries.GetOrganizationIntegrations(ctx, pgtype.Text{String: userUUID, Valid: true})
	if err != nil {
		return err
	}

	var titles []organizations.NormalizedTitle

	for _, integ := range integrations {
		var integData organizations.OrganizationIntegrationData
		if err := integData.FromJSON(integ.Data); err != nil {
			continue
		}
		if integData.Verified {
			titles = append(titles, integData.NormalizedTitle)
		}
	}

	highestTitle := organizations.GetHighestTitle(titles)

	// Update profile
	return s.queries.UpdateProfileTitle(ctx, models.UpdateProfileTitleParams{
		UserUuid: pgtype.Text{String: userUUID, Valid: true},
		Title:    pgtype.Text{String: string(highestTitle), Valid: highestTitle != ""},
	})
}

func convertTitleInfoToProto(info *organizations.TitleInfo) *pb.OrganizationTitle {
	title := &pb.OrganizationTitle{
		OrganizationCode: string(info.Organization),
		OrganizationName: info.OrganizationName,
		MemberId:         info.MemberID,
		FullName:         info.FullName,
		RawTitle:         info.RawTitle,
		NormalizedTitle:  string(info.NormalizedTitle),
		Verified:         true,
	}

	if info.LastFetched != nil {
		title.LastFetched = timestamppb.New(*info.LastFetched)
	}

	return title
}
