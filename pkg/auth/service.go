package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/twitchtv/twirp"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/emailer"
	"github.com/domino14/liwords/pkg/sessions"

	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/lithammer/shortuuid"

	"github.com/dgrijalva/jwt-go"
	"github.com/domino14/liwords/pkg/apiserver"

	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/user"

	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	pb "github.com/domino14/liwords/rpc/api/proto/user_service"
)

// A little bit of grace period in case we have to redeploy the socket or something.
const TokenExpiration = 60 * time.Second
const PasswordResetExpiration = 24 * time.Hour

const ResetPasswordTemplate = `
Dear Woogles.io user,

You recently requested a password reset. If this wasn't you, you can ignore this email.

Otherwise, please visit the following URL to reset your password. This URL expires within the next 24 hours:

%s

Note your username: %s

Love,

The Woogles.io team
`

var errPasswordTooShort = errors.New("your password is too short, use 8 or more characters")

type AuthenticationService struct {
	userStore    user.Store
	sessionStore sessions.SessionStore
	configStore  config.ConfigStore
	secretKey    string
	mailgunKey   string
	argonConfig  config.ArgonConfig
}

func NewAuthenticationService(u user.Store, ss sessions.SessionStore, cs config.ConfigStore,
	secretKey, mailgunKey string, cfg config.ArgonConfig) *AuthenticationService {
	return &AuthenticationService{
		userStore:    u,
		sessionStore: ss,
		configStore:  cs,
		secretKey:    secretKey,
		mailgunKey:   mailgunKey,
		argonConfig:  cfg}
}

// Login sets a cookie.
func (as *AuthenticationService) Login(ctx context.Context, r *pb.UserLoginRequest) (*pb.LoginResponse, error) {

	r.Username = strings.TrimSpace(r.Username)
	user, err := as.userStore.Get(ctx, r.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, twirp.NewError(twirp.Unauthenticated, "bad login")
	}
	matches, err := ComparePassword(r.Password, user.Password)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	if !matches {
		return nil, twirp.NewError(twirp.Unauthenticated, "password incorrect")
	}
	sess, err := as.sessionStore.New(ctx, user)
	if err != nil {
		return nil, err
	}

	err = mod.ActionExists(ctx, as.userStore, user.UUID, ms.ModActionType_SUSPEND_ACCOUNT)
	if err != nil {
		return nil, err
	}

	err = apiserver.SetDefaultCookie(ctx, sess.ID)

	log.Info().Str("value", sess.ID).Msg("setting-cookie")
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &pb.LoginResponse{}, nil
}

// Logout deletes the user session from the store, and also tells the front-end
// to ditch the cookie (yum)
func (as *AuthenticationService) Logout(ctx context.Context, r *pb.UserLogoutRequest) (*pb.LogoutResponse, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	err = as.sessionStore.Delete(ctx, sess)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	// Delete the cookie as well.
	err = apiserver.SetCookie(ctx, &http.Cookie{
		Name:     "session",
		Value:    sess.ID,
		MaxAge:   -1,
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(-100 * time.Hour),
	})
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &pb.LogoutResponse{}, nil
}

func (as *AuthenticationService) GetSocketToken(ctx context.Context, r *pb.SocketTokenRequest) (*pb.SocketTokenResponse, error) {
	// This view requires authentication.
	sess, err := apiserver.GetSession(ctx)
	cid := shortuuid.New()[1:10]
	var unn, uuid string
	var authed bool
	if err != nil {
		authed = false
		uuid = shortuuid.New()
		unn = entity.DeterministicUsername(uuid)
	} else {
		authed = true
		uuid = sess.UserUUID
		unn = sess.Username
	}

	// Create an unauth token.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(TokenExpiration).Unix(),
		"uid": uuid,
		"unn": unn,
		"a":   authed,
	})
	tokenString, err := token.SignedString([]byte(as.secretKey))
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	// Maybe cache this?
	feHash, err := as.configStore.FEHash(ctx)
	if err != nil {
		log.Err(err).Msg("error getting fe-hash")
		// Continue anyway.
	}

	return &pb.SocketTokenResponse{
		Token:           tokenString,
		Cid:             cid,
		FrontEndVersion: feHash,
	}, nil

}

func (as *AuthenticationService) ResetPasswordStep1(ctx context.Context, r *pb.ResetPasswordRequestStep1) (*pb.ResetPasswordResponse, error) {
	email := strings.TrimSpace(r.Email)
	u, err := as.userStore.GetByEmail(ctx, email)
	if err != nil {
		return nil, twirp.NewError(twirp.Unauthenticated, err.Error())
	}

	// Create a token for the reset
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":  time.Now().Add(PasswordResetExpiration).Unix(),
		"uuid": u.UUID,
	})
	tokenString, err := token.SignedString([]byte(as.secretKey))
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	resetURL := "https://woogles.io/password/new?t=" + tokenString

	id, err := emailer.SendSimpleMessage(as.mailgunKey, email, "Password reset for Woogles.io",
		fmt.Sprintf(ResetPasswordTemplate, resetURL, u.Username))
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	log.Info().Str("id", id).Str("email", email).Msg("sent-password-reset")

	return &pb.ResetPasswordResponse{}, nil
}

func (as *AuthenticationService) ResetPasswordStep2(ctx context.Context, r *pb.ResetPasswordRequestStep2) (*pb.ResetPasswordResponse, error) {

	tokenString := r.ResetCode

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(as.secretKey), nil
	})
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		uuid, ok := claims["uuid"].(string)
		if !ok {
			return nil, twirp.NewError(twirp.Malformed, "wrongly formatted uuid in token")
		}

		config := NewPasswordConfig(as.argonConfig.Time, as.argonConfig.Memory, as.argonConfig.Threads, as.argonConfig.Keylen)
		if len(r.Password) < 8 {
			return nil, twirp.NewError(twirp.InvalidArgument, errPasswordTooShort.Error())
		}
		// XXX: do not hardcode, put in a config file
		hashPass, err := GeneratePassword(config, r.Password)
		if err != nil {
			return nil, twirp.InternalErrorWith(err)
		}
		err = as.userStore.SetPassword(ctx, uuid, hashPass)
		if err != nil {
			return nil, twirp.InternalErrorWith(err)
		}
		return &pb.ResetPasswordResponse{}, nil
	}

	return nil, twirp.InternalErrorWith(errors.New("reset code is invalid; please try again"))
}

func (as *AuthenticationService) ChangePassword(ctx context.Context, r *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	// This view requires authentication.
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := as.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		// The username should maybe not be in the session? We can't change
		// usernames easily.
		return nil, twirp.InternalErrorWith(err)
	}
	matches, err := ComparePassword(r.OldPassword, user.Password)
	if !matches {
		return nil, twirp.NewError(twirp.InvalidArgument, "your password is incorrect")
	}

	if len(r.NewPassword) < 8 {
		return nil, twirp.NewError(twirp.InvalidArgument, errPasswordTooShort.Error())
	}

	config := NewPasswordConfig(as.argonConfig.Time, as.argonConfig.Memory, as.argonConfig.Threads, as.argonConfig.Keylen)
	hashPass, err := GeneratePassword(config, r.NewPassword)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	err = as.userStore.SetPassword(ctx, user.UUID, hashPass)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &pb.ChangePasswordResponse{}, nil
}
