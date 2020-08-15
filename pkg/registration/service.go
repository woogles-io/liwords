package registration

import (
	"context"
	"errors"
	"os"

	"github.com/domino14/liwords/pkg/user"
	"github.com/rs/zerolog"

	pb "github.com/domino14/liwords/rpc/api/proto/user_service"
)

type RegistrationService struct {
	userStore user.Store
}

func NewRegistrationService(u user.Store) *RegistrationService {
	return &RegistrationService{userStore: u}
}

// Register registers a new user.
func (rs *RegistrationService) Register(ctx context.Context, r *pb.UserRegistrationRequest) (*pb.RegistrationResponse, error) {
	log := zerolog.Ctx(ctx)
	log.Info().Str("user", r.Username).Str("email", r.Email).Msg("new-user")
	if r.RegistrationCode != os.Getenv("REGISTRATION_CODE") {
		return nil, errors.New("unauthorized")
	}
	err := RegisterUser(ctx, r.Username, r.Password, r.Email, rs.userStore)
	if err != nil {
		return nil, err
	}
	return &pb.RegistrationResponse{}, nil
}
