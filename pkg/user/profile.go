package user

import (
	"context"
	"encoding/json"

	"github.com/twitchtv/twirp"

	pb "github.com/domino14/liwords/rpc/api/proto/user_service"
)

type ProfileService struct {
	userStore Store
}

func NewProfileService(u Store) *ProfileService {
	return &ProfileService{userStore: u}
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
		FirstName:   user.Profile.FirstName,
		LastName:    user.Profile.LastName,
		CountryCode: user.Profile.CountryCode,
		Title:       user.Profile.Title,
		About:       user.Profile.About,
		RatingsJson: string(ratjson),
		StatsJson:   string(statjson),
		UserId:      user.UUID,
	}, nil
}
