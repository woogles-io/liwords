package profile

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/domino14/liwords/pkg/apiserver"
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
	user, err := ps.userStore.Get(ctx, r.Username)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
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

	return &pb.ProfileResponse{
		FirstName:       user.Profile.FirstName,
		LastName:        user.Profile.LastName,
		FullName:		 user.RealName(),
		CountryCode:     user.Profile.CountryCode,
		Title:           user.Profile.Title,
		About:           user.Profile.About,
		RatingsJson:     string(ratjson),
		StatsJson:       string(statjson),
		UserId:          user.UUID,
		AvatarUrl:       user.AvatarUrl(),
		AvatarsEditable: ps.avatarService != nil,
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
		Email:			 user.Email,
		FirstName:       user.Profile.FirstName,
		LastName:        user.Profile.LastName,
		CountryCode:     user.Profile.CountryCode,
		AvatarUrl:       user.AvatarUrl(),
		FullName:		 user.RealName(),
		About:			 user.Profile.About,
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

	updateErr := ps.userStore.SetPersonalInfo(ctx, user.UUID, r.Email, r.FirstName, r.LastName, r.CountryCode, r.About)
	if updateErr != nil {
		return nil, twirp.InternalErrorWith(updateErr)
	}


	return &pb.UpdatePersonalInfoResponse{
	}, nil
}

func (ps *ProfileService) GetUsersGameInfo(ctx context.Context, r *pb.UsersGameInfoRequest) (*pb.UsersGameInfoResponse, error) {
	var infos []*pb.UserGameInfo

	for _, uuid := range r.Uuids {
		user, err := ps.userStore.GetByUUID(ctx, uuid)
		if err == nil {
			infos = append(infos, &pb.UserGameInfo{
				Uuid:      uuid,
				AvatarUrl: user.AvatarUrl(),
				Title:     user.Profile.Title,
			})
		}
	}

	return &pb.UsersGameInfoResponse{
		Infos: infos,
	}, nil
}

func (ps *ProfileService) UpdateProfile(ctx context.Context, r *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
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

	err = mod.ActionExists(ctx, ps.userStore, user.UUID, true, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
	if err != nil {
		return nil, err
	}

	err = ps.userStore.SetAbout(ctx, user.UUID, r.About)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &pb.UpdateProfileResponse{}, nil
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

	avatarService := ps.avatarService
	if avatarService == nil {
		return nil, twirp.InternalErrorWith(errors.New("No avatar service available"))
	}

	err = mod.ActionExists(ctx, ps.userStore, user.UUID, true, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
	if err != nil {
		return nil, err
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

	return &pb.RemoveAvatarResponse{
	}, nil
}

