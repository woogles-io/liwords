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
	userStore             user.Store
	argonConfig           config.ArgonConfig
	emailDebugMode        bool
	skipEmailVerification bool
}

func NewRegistrationService(u user.Store, cfg config.ArgonConfig, emailDebugMode bool, skipEmailVerification bool) *RegistrationService {
	return &RegistrationService{
		userStore:             u,
		argonConfig:           cfg,
		emailDebugMode:        emailDebugMode,
		skipEmailVerification: skipEmailVerification,
	}
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
		rs.userStore, r.Msg.RegistrationCode == codebot, rs.argonConfig, rs.emailDebugMode, rs.skipEmailVerification)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.RegistrationResponse{}), nil
}

// VerifyEmail verifies a user's email address using the token
func (rs *RegistrationService) VerifyEmail(ctx context.Context, r *connect.Request[pb.VerifyEmailRequest],
) (*connect.Response[pb.VerifyEmailResponse], error) {
	log := zerolog.Ctx(ctx)

	err := VerifyUserEmail(ctx, r.Msg.Token, rs.userStore)
	if err != nil {
		log.Error().Err(err).Msg("email-verification-failed")
		return nil, apiserver.InvalidArg(err.Error())
	}

	log.Info().Msg("email-verified")
	return connect.NewResponse(&pb.VerifyEmailResponse{
		Message: "Email verified successfully. You can now log in.",
	}), nil
}

// ResendVerificationEmail resends the verification email to a user
func (rs *RegistrationService) ResendVerificationEmail(ctx context.Context, r *connect.Request[pb.ResendVerificationEmailRequest],
) (*connect.Response[pb.ResendVerificationEmailResponse], error) {
	log := zerolog.Ctx(ctx)

	err := ResendVerificationEmail(ctx, r.Msg.Email, rs.userStore, rs.emailDebugMode)
	if err != nil {
		log.Error().Err(err).Str("email", r.Msg.Email).Msg("resend-verification-failed")
		return nil, apiserver.InvalidArg(err.Error())
	}

	log.Info().Str("email", r.Msg.Email).Msg("verification-email-resent")
	return connect.NewResponse(&pb.ResendVerificationEmailResponse{
		Message: "Verification email sent. Please check your inbox.",
	}), nil
}
