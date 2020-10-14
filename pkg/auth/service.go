package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/domino14/liwords/pkg/emailer"
	"github.com/domino14/liwords/pkg/sessions"

	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/lithammer/shortuuid"

	"github.com/dgrijalva/jwt-go"
	"github.com/domino14/liwords/pkg/apiserver"

	"github.com/domino14/liwords/pkg/user"

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

type AuthenticationService struct {
	userStore    user.Store
	sessionStore sessions.SessionStore
	secretKey    string
	mailgunKey   string
}

func NewAuthenticationService(u user.Store, ss sessions.SessionStore, secretKey,
	mailgunKey string) *AuthenticationService {
	return &AuthenticationService{userStore: u, sessionStore: ss, secretKey: secretKey,
		mailgunKey: mailgunKey}
}

// Login sets a cookie.
func (as *AuthenticationService) Login(ctx context.Context, r *pb.UserLoginRequest) (*pb.LoginResponse, error) {

	r.Username = strings.TrimSpace(r.Username)
	user, err := as.userStore.Get(ctx, r.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, err
	}
	matches, err := ComparePassword(r.Password, user.Password)
	if !matches {
		return nil, errors.New("password is incorrect")
	}
	sess, err := as.sessionStore.New(ctx, user)
	if err != nil {
		return nil, err
	}

	err = apiserver.SetDefaultCookie(ctx, sess.ID)

	log.Info().Str("value", sess.ID).Msg("setting-cookie")
	if err != nil {
		return nil, err
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
		return nil, err
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
		return nil, err
	}

	return &pb.LogoutResponse{}, nil
}

func (as *AuthenticationService) GetSocketToken(ctx context.Context, r *pb.SocketTokenRequest) (*pb.SocketTokenResponse, error) {
	// This view requires authentication.
	sess, err := apiserver.GetSession(ctx)
	cid := shortuuid.New()[1:10]
	if err != nil {
		// Create an unauth token.
		uuid := shortuuid.New()
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"exp": time.Now().Add(TokenExpiration).Unix(),
			"uid": uuid,
			"unn": entity.DeterministicUsername(uuid),
			"a":   false, // authed
		})
		tokenString, err := token.SignedString([]byte(as.secretKey))
		if err != nil {
			return nil, err
		}
		return &pb.SocketTokenResponse{
			Token: tokenString,
			Cid:   cid,
		}, nil
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(TokenExpiration).Unix(),
		"uid": sess.UserUUID,
		"unn": sess.Username,
		"a":   true,
	})
	tokenString, err := token.SignedString([]byte(as.secretKey))
	if err != nil {
		return nil, err
	}
	return &pb.SocketTokenResponse{
		Token: tokenString,
		Cid:   cid,
	}, nil
}

func (as *AuthenticationService) ResetPasswordStep1(ctx context.Context, r *pb.ResetPasswordRequestStep1) (*pb.ResetPasswordResponse, error) {
	u, err := as.userStore.GetByEmail(ctx, r.Email)
	if err != nil {
		return nil, err
	}

	// Create a token for the reset
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":  time.Now().Add(PasswordResetExpiration).Unix(),
		"uuid": u.UUID,
	})
	tokenString, err := token.SignedString([]byte(as.secretKey))
	if err != nil {
		return nil, err
	}
	resetURL := "https://woogles.io/password/new?t=" + tokenString

	id, err := emailer.SendSimpleMessage(as.mailgunKey, r.Email, "Password reset for Woogles.io",
		fmt.Sprintf(ResetPasswordTemplate, resetURL, u.Username))
	if err != nil {
		return nil, err
	}
	log.Info().Str("id", id).Str("email", r.Email).Msg("sent-password-reset")

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
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		uuid, ok := claims["uuid"].(string)
		if !ok {
			return nil, errors.New("wrongly formatted uuid in token")
		}

		config := NewPasswordConfig(1, 64*1024, 4, 32)
		// XXX: do not hardcode, put in a config file
		hashPass, err := GeneratePassword(config, r.Password)
		if err != nil {
			return nil, err
		}
		err = as.userStore.SetPassword(ctx, uuid, hashPass)
		if err != nil {
			return nil, err
		}
		return &pb.ResetPasswordResponse{}, nil
	}

	return nil, fmt.Errorf("reset code is invalid; please try again")
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
		return nil, err
	}
	matches, err := ComparePassword(r.OldPassword, user.Password)
	if !matches {
		return nil, errors.New("your password is incorrect")
	}

	if len(r.NewPassword) < 8 {
		return nil, errors.New("your password is too short, use 8 or more characters")
	}

	// time, memory, threads, keyLen for argon2:
	config := NewPasswordConfig(1, 64*1024, 4, 32)
	// XXX: do not hardcode, put in a config file
	hashPass, err := GeneratePassword(config, r.NewPassword)
	if err != nil {
		return nil, err
	}
	err = as.userStore.SetPassword(ctx, user.UUID, hashPass)
	if err != nil {
		return nil, err
	}
	return &pb.ChangePasswordResponse{}, nil
}
