package tournament

// Clubs and tournaments are closely related. Clubs can create club sessions,
// which are interchangeable with tournaments.
/*
import (
	"context"

	"github.com/lithammer/shortuuid"

	"github.com/domino14/liwords/pkg/entity"
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	pb "github.com/domino14/liwords/rpc/api/proto/tournament_service"
)

func NewClub(ctx context.Context, clubStore ClubStore, name string,
	description string, slug string, directors *entity.TournamentPersons,
	defaultSettings *realtime.GameRequest) (*entity.Club, error) {

	executiveDirector, err := getExecutiveDirector(directors)
	if err != nil {
		return nil, err
	}

	id := shortuuid.New()

	entClub := &entity.Club{Name: name, Description: description,
		Directors: directors, ExecutiveDirector: executiveDirector, UUID: id,
		Slug: slug, DefaultSettings: defaultSettings}

	return entClub, nil
}

type ClubService struct {
	clubStore       ClubStore
	tournamentStore TournamentStore
}

func NewClubService(c ClubStore, t TournamentStore) *ClubService {
	return &ClubService{clubStore: c, tournamentStore: t}
}

func (cs *ClubService) NewClub(ctx context.Context, req *pb.NewClubRequest) (*pb.NewClubResponse, error) {

	return nil, nil
}

func (cs *ClubService) GetClubMetadata(ctx context.Context, req *pb.GetClubMetadataRequest) (*pb.ClubMetadataResponse, error) {

	return nil, nil
}

func (cs *ClubService) SetClubMetadata(ctx context.Context, req *pb.SetClubMetadataRequest) (*pb.ClubResponse, error) {

	return nil, nil
}

// Implement this later:
// func (cs *ClubService) AddDirectors(ctx context.Context, req *pb.TournamentPersons) (*pb.ClubResponse, error) {
// 	err := AddDirectors(ctx, cs.clubStore, req.Id, convertPersonsToStringMap(req))
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &pb.ClubResponse{}, nil
// }

// func (cs *ClubService) RemoveDirectors(ctx context.Context, req *pb.TournamentPersons) (*pb.ClubResponse, error) {
// 	err := RemoveDirectors(ctx, cs.clubStore, req.Id, convertPersonsToStringMap(req))
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &pb.ClubResponse{}, nil
// }

func (cs *ClubService) CreateSession(ctx context.Context, req *pb.NewClubSessionRequest) (*pb.NewClubSessionResponse, error) {
	return nil, nil
}

func (cs *ClubService) GetRecentSessions(ctx context.Context, req *pb.RecentClubSessionsRequest) (*pb.ClubSessionsResponse, error) {
	return nil, nil
}
*/
