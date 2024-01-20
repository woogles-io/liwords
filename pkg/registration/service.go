package registration

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/twitchtv/twirp"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/user"

	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

type RegistrationService struct {
	userStore   user.Store
	argonConfig config.ArgonConfig
}

func NewRegistrationService(u user.Store, cfg config.ArgonConfig) *RegistrationService {
	return &RegistrationService{userStore: u, argonConfig: cfg}
}

// Register registers a new user.
func (rs *RegistrationService) Register(ctx context.Context, r *pb.UserRegistrationRequest) (*pb.RegistrationResponse, error) {
	log := zerolog.Ctx(ctx)
	log.Info().Str("user", r.Username).Str("email", r.Email).Msg("new-user")

	code := os.Getenv("REGISTRATION_CODE")
	codebot := "bot" + code

	// if r.RegistrationCode != code && r.RegistrationCode != codebot {
	// 	return nil, errors.New("unauthorized")
	// }
	err := RegisterUser(ctx, r.Username, r.Password, r.Email,
		r.FirstName, r.LastName, r.BirthDate, r.CountryCode,
		rs.userStore, r.RegistrationCode == codebot, rs.argonConfig)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.RegistrationResponse{}, nil
}
