package registration

import (
	"context"
	"os"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"

	"github.com/woogles-io/liwords/pkg/apiserver"
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
func (rs *RegistrationService) Register(ctx context.Context, r *connect.Request[pb.UserRegistrationRequest],
) (*connect.Response[pb.RegistrationResponse], error) {
	log := zerolog.Ctx(ctx)
	log.Info().Str("user", r.Msg.Username).Str("email", r.Msg.Email).Msg("new-user")

	code := os.Getenv("REGISTRATION_CODE")
	codebot := "bot" + code

	// if r.RegistrationCode != code && r.RegistrationCode != codebot {
	// 	return nil, errors.New("unauthorized")
	// }
	err := RegisterUser(ctx, r.Msg.Username, r.Msg.Password, r.Msg.Email,
		r.Msg.FirstName, r.Msg.LastName, r.Msg.BirthDate, r.Msg.CountryCode,
		rs.userStore, r.Msg.RegistrationCode == codebot, rs.argonConfig)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.RegistrationResponse{}), nil
}
