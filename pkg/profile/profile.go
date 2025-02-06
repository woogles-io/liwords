package profile

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	userservices "github.com/woogles-io/liwords/pkg/user/services"

	"github.com/rs/zerolog/log"

	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

type ProfileService struct {
	userStore     user.Store
	avatarService userservices.UploadService
	queries       *models.Queries
}

func NewProfileService(u user.Store, us userservices.UploadService, q *models.Queries) *ProfileService {
	return &ProfileService{userStore: u, avatarService: us, queries: q}
}

func modActionExistsErr(err error) error {
	if ue, ok := err.(*mod.UserModeratedError); ok {
		return apiserver.PermissionDenied(ue.Error())
	} else {
		return apiserver.InternalErr(err)
	}
}

func (ps *ProfileService) GetRatings(ctx context.Context, r *connect.Request[pb.RatingsRequest],
) (*connect.Response[pb.RatingsResponse], error) {
	user, err := ps.userStore.Get(ctx, r.Msg.Username)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	ratings := user.Profile.Ratings

	b, err := json.Marshal(ratings)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	return connect.NewResponse(&pb.RatingsResponse{
		Json: string(b),
	}), nil
}

func (ps *ProfileService) GetStats(ctx context.Context, r *connect.Request[pb.StatsRequest],
) (*connect.Response[pb.StatsResponse], error) {
	user, err := ps.userStore.Get(ctx, r.Msg.Username)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	stats := user.Profile.Stats

	b, err := json.Marshal(stats)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	return connect.NewResponse(&pb.StatsResponse{
		Json: string(b),
	}), nil
}

func (ps *ProfileService) GetProfile(ctx context.Context, r *connect.Request[pb.ProfileRequest],
) (*connect.Response[pb.ProfileResponse], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		// This is fine. No, not that meme. It's really fine.
		sess = nil
	}

	user, err := ps.userStore.Get(ctx, r.Msg.Username)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	permaban, possiblyForceLogoutErr := mod.ActionExists(ctx, ps.userStore, user.UUID, true, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})

	subjectIsMe := sess != nil && sess.UserUUID == user.UUID

	// If the user is permabanned, the profile is hidden to unprivileged users
	if permaban {
		sess, err := apiserver.GetSession(ctx)
		if err != nil {
			return nil, err
		}

		viewer, err := ps.userStore.Get(ctx, sess.Username)
		if err != nil {
			log.Err(err).Msg("getting-user")
			return nil, apiserver.InternalErr(err)
		}

		privilegedViewer, err := ps.queries.HasPermission(ctx, models.HasPermissionParams{
			UserID:     int32(viewer.ID),
			Permission: string(rbac.CanModerateUsers),
		})

		if !privilegedViewer {
			// If this is the user's profile, telling them
			// 'record not found' may make them suspicious.
			// Instead, give a subtler message that is more
			// likely to prompt a logout.
			if subjectIsMe {
				return nil, modActionExistsErr(possiblyForceLogoutErr)
			} else {
				return nil, apiserver.InvalidArg("record not found")
			}
		}
	}

	ratings := user.Profile.Ratings
	ratjson, err := json.Marshal(ratings)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	stats := user.Profile.Stats
	statjson, err := json.Marshal(stats)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	subjectIsAdult := entity.IsAdult(user.Profile.BirthDate, time.Now())
	concealIf := func(b bool, s string) string {
		if b {
			return ""
		} else {
			return s
		}
	}
	childProof := func(s string) string { return concealIf(!(subjectIsMe || subjectIsAdult), s) }

	return connect.NewResponse(&pb.ProfileResponse{
		FirstName:       childProof(user.Profile.FirstName),
		LastName:        childProof(user.Profile.LastName),
		BirthDate:       concealIf(!subjectIsMe, user.Profile.BirthDate),
		FullName:        childProof(user.RealName()),
		CountryCode:     user.Profile.CountryCode,
		Title:           user.Profile.Title,
		About:           childProof(user.Profile.About),
		RatingsJson:     string(ratjson),
		StatsJson:       string(statjson),
		UserId:          user.UUID,
		AvatarUrl:       childProof(user.AvatarUrl()),
		AvatarsEditable: ps.avatarService != nil,
	}), nil
}

func (ps *ProfileService) GetPersonalInfo(ctx context.Context, r *connect.Request[pb.PersonalInfoRequest],
) (*connect.Response[pb.PersonalInfoResponse], error) {
	// This view requires authentication.
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ps.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		// The username should maybe not be in the session? We can't change
		// usernames easily.
		return nil, apiserver.InternalErr(err)
	}

	return connect.NewResponse(&pb.PersonalInfoResponse{
		Email:       user.Email,
		FirstName:   user.Profile.FirstName,
		LastName:    user.Profile.LastName,
		BirthDate:   user.Profile.BirthDate,
		CountryCode: user.Profile.CountryCode,
		AvatarUrl:   user.AvatarUrl(),
		FullName:    user.RealName(),
		About:       user.Profile.About,
	}), nil
}

func (ps *ProfileService) UpdatePersonalInfo(ctx context.Context, r *connect.Request[pb.UpdatePersonalInfoRequest],
) (*connect.Response[pb.UpdatePersonalInfoResponse], error) {
	// This view requires authentication.
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ps.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		// The username should maybe not be in the session? We can't change
		// usernames easily.
		return nil, apiserver.InternalErr(err)
	}

	updateErr := ps.userStore.SetPersonalInfo(ctx, user.UUID, r.Msg.Email, r.Msg.FirstName,
		r.Msg.LastName, r.Msg.BirthDate, r.Msg.CountryCode, r.Msg.About)
	if updateErr != nil {
		return nil, apiserver.InternalErr(updateErr)
	}

	return connect.NewResponse(&pb.UpdatePersonalInfoResponse{}), nil
}

func (ps *ProfileService) UpdateAvatar(ctx context.Context, r *connect.Request[pb.UpdateAvatarRequest],
) (*connect.Response[pb.UpdateAvatarResponse], error) {
	// This view requires authentication.
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ps.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		// The username should maybe not be in the session? We can't change
		// usernames easily.
		return nil, apiserver.InternalErr(err)
	}

	_, err = mod.ActionExists(ctx, ps.userStore, user.UUID, true, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
	if err != nil {
		return nil, modActionExistsErr(err)
	}

	avatarService := ps.avatarService
	if avatarService == nil {
		return nil, apiserver.InternalErr(errors.New("no avatar service available"))
	}

	oldUrl := user.AvatarUrl()

	avatarUrl, err := avatarService.Upload(ctx, user.UUID, r.Msg.JpgData)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	// Remember the URL in the database
	updateErr := ps.userStore.SetAvatarUrl(ctx, user.UUID, avatarUrl)
	if updateErr != nil {
		return nil, apiserver.InternalErr(updateErr)
	}

	// Delete old URL
	if oldUrl != "" {
		err = avatarService.Delete(ctx, oldUrl)
		if err != nil {
			// Don't crash.
			log.Err(err).Msg("error-deleting-old-avatar")
		}
	}

	return connect.NewResponse(&pb.UpdateAvatarResponse{
		AvatarUrl: avatarUrl,
	}), nil
}

func (ps *ProfileService) RemoveAvatar(ctx context.Context, r *connect.Request[pb.RemoveAvatarRequest],
) (*connect.Response[pb.RemoveAvatarResponse], error) {
	// This view requires authentication.
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ps.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		// The username should maybe not be in the session? We can't change
		// usernames easily.
		return nil, apiserver.InternalErr(err)
	}

	avatarService := ps.avatarService
	if avatarService == nil {
		return nil, apiserver.InternalErr(errors.New("No avatar service available"))
	}

	// Clear the URL in the database
	updateErr := ps.userStore.SetAvatarUrl(ctx, user.UUID, "")
	if updateErr != nil {
		return nil, apiserver.InternalErr(updateErr)
	}

	// Delete old URL
	oldUrl := user.AvatarUrl()
	if oldUrl != "" {
		err = avatarService.Delete(ctx, oldUrl)
		if err != nil {
			// Don't crash.
			log.Err(err).Msg("error-deleting-old-avatar")
		}
	}

	return connect.NewResponse(&pb.RemoveAvatarResponse{}), nil
}

func (ps *ProfileService) GetBriefProfiles(ctx context.Context, r *connect.Request[pb.BriefProfilesRequest],
) (*connect.Response[pb.BriefProfilesResponse], error) {
	// this endpoint should work without login
	// pc, file, line, ok := runtime.Caller(0)
	// var span trace.Span
	// if ok {
	// 	_, span = otel.Tracer("test-for-now").Start(ctx, "in-gbp", trace.WithAttributes(
	// 		// Capture caller file and line information
	// 		attribute.String("code.file", file),
	// 		attribute.String("code.func", runtime.FuncForPC(pc).Name()),
	// 		attribute.Int("code.line", line),
	// 	))
	// 	defer span.End()
	// }
	// span.AddEvent("fetching-profiles")
	response, err := ps.userStore.GetBriefProfiles(ctx, r.Msg.UserIds)
	if err != nil {
		return nil, err
	}
	// span.AddEvent("got-profiles")

	return connect.NewResponse(&pb.BriefProfilesResponse{Response: response}), nil
}
