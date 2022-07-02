package profile

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/user"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"

	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	pb "github.com/domino14/liwords/rpc/api/proto/user_service"
)

type ProfileService struct {
	userStore     user.Store
	avatarService user.UploadService
}

func NewProfileService(u user.Store, us user.UploadService) *ProfileService {
	return &ProfileService{userStore: u, avatarService: us}
}

func (ps *ProfileService) GetRatings(ctx context.Context, r *pb.RatingsRequest) (*pb.RatingsResponse, error) {
	user, err := ps.userStore.Get(ctx, r.Username)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	ratings := user.Profile.Ratings

	b, err := json.Marshal(ratings)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &pb.RatingsResponse{
		Json: string(b),
	}, nil
}

func (ps *ProfileService) GetStats(ctx context.Context, r *pb.StatsRequest) (*pb.StatsResponse, error) {
	user, err := ps.userStore.Get(ctx, r.Username)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	stats := user.Profile.Stats

	b, err := json.Marshal(stats)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &pb.StatsResponse{
		Json: string(b),
	}, nil
}

func (ps *ProfileService) GetProfile(ctx context.Context, r *pb.ProfileRequest) (*pb.ProfileResponse, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		// This is fine. No, not that meme. It's really fine.
		sess = nil
	}

	user, err := ps.userStore.Get(ctx, r.Username)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}

	permaban, possiblyForceLogout := mod.ActionExists(ctx, ps.userStore, user.UUID, true, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})

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
			return nil, twirp.InternalErrorWith(err)
		}

		if !viewer.IsMod && !viewer.IsAdmin {
			// If this is the user's profile, telling them
			// 'record not found' may make them suspicious.
			// Instead, give a subtler message that is more
			// likely to prompt a logout.
			if subjectIsMe {
				return nil, possiblyForceLogout
			} else {
				return nil, twirp.NewError(twirp.InvalidArgument, "record not found")
			}
		}
	}

	ratings := user.Profile.Ratings
	ratjson, err := json.Marshal(ratings)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	stats := user.Profile.Stats
	statjson, err := json.Marshal(stats)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
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

	return &pb.ProfileResponse{
		FirstName:   childProof(user.Profile.FirstName),
		LastName:    childProof(user.Profile.LastName),
		BirthDate:   concealIf(!subjectIsMe, user.Profile.BirthDate),
		FullName:    childProof(user.RealName()),
		CountryCode: user.Profile.CountryCode,
		Title:       user.Profile.Title,
		About:       childProof(user.Profile.About),
		RatingsJson: string(ratjson),
		StatsJson:   string(statjson),
		UserId:      user.UUID,
		AvatarUrl:   childProof(user.AvatarUrl()),
		SilentMode:  user.Profile.SilentMode,
	}, nil
}

func (ps *ProfileService) GetPersonalInfo(ctx context.Context, r *pb.PersonalInfoRequest) (*pb.PersonalInfoResponse, error) {
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
		return nil, twirp.InternalErrorWith(err)
	}

	return &pb.PersonalInfoResponse{
		Email:       user.Email,
		FirstName:   user.Profile.FirstName,
		LastName:    user.Profile.LastName,
		BirthDate:   user.Profile.BirthDate,
		CountryCode: user.Profile.CountryCode,
		AvatarUrl:   user.AvatarUrl(),
		FullName:    user.RealName(),
		About:       user.Profile.About,
		SilentMode:  user.Profile.SilentMode,
	}, nil
}

func (ps *ProfileService) UpdatePersonalInfo(ctx context.Context, r *pb.UpdatePersonalInfoRequest) (*pb.UpdatePersonalInfoResponse, error) {
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
		return nil, twirp.InternalErrorWith(err)
	}

	updateErr := ps.userStore.SetPersonalInfo(ctx, user.UUID, r.Email, r.FirstName, r.LastName, r.BirthDate, r.CountryCode, r.About, r.SilentMode)
	if updateErr != nil {
		return nil, twirp.InternalErrorWith(updateErr)
	}

	return &pb.UpdatePersonalInfoResponse{}, nil
}

func (ps *ProfileService) UpdateAvatar(ctx context.Context, r *pb.UpdateAvatarRequest) (*pb.UpdateAvatarResponse, error) {
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
		return nil, twirp.InternalErrorWith(err)
	}

	_, err = mod.ActionExists(ctx, ps.userStore, user.UUID, true, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
	if err != nil {
		return nil, err
	}

	avatarService := ps.avatarService
	if avatarService == nil {
		return nil, twirp.InternalErrorWith(errors.New("No avatar service available"))
	}

	oldUrl := user.AvatarUrl()

	avatarUrl, err := avatarService.Upload(ctx, user.UUID, r.JpgData)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	// Remember the URL in the database
	updateErr := ps.userStore.SetAvatarUrl(ctx, user.UUID, avatarUrl)
	if updateErr != nil {
		return nil, twirp.InternalErrorWith(updateErr)
	}

	// Delete old URL
	if oldUrl != "" {
		err = avatarService.Delete(ctx, oldUrl)
		if err != nil {
			// Don't crash.
			log.Err(err).Msg("error-deleting-old-avatar")
		}
	}

	return &pb.UpdateAvatarResponse{
		AvatarUrl: avatarUrl,
	}, nil
}

func (ps *ProfileService) RemoveAvatar(ctx context.Context, r *pb.RemoveAvatarRequest) (*pb.RemoveAvatarResponse, error) {
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
		return nil, twirp.InternalErrorWith(err)
	}

	avatarService := ps.avatarService
	if avatarService == nil {
		return nil, twirp.InternalErrorWith(errors.New("No avatar service available"))
	}

	// Clear the URL in the database
	updateErr := ps.userStore.SetAvatarUrl(ctx, user.UUID, "")
	if updateErr != nil {
		return nil, twirp.InternalErrorWith(updateErr)
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

	return &pb.RemoveAvatarResponse{}, nil
}

func (ps *ProfileService) GetBriefProfiles(ctx context.Context, r *pb.BriefProfilesRequest) (*pb.BriefProfilesResponse, error) {
	// this endpoint should work without login

	response, err := ps.userStore.GetBriefProfiles(ctx, r.UserIds)
	if err != nil {
		return nil, err
	}

	return &pb.BriefProfilesResponse{Response: response}, nil
}
