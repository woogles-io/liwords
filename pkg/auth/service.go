package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

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

type AuthenticationService struct {
	userStore    user.Store
	sessionStore user.SessionStore
	secretKey    string
}

func NewAuthenticationService(u user.Store, ss user.SessionStore, secretKey string) *AuthenticationService {
	return &AuthenticationService{userStore: u, sessionStore: ss, secretKey: secretKey}
}

// Login sets a cookie.
func (as *AuthenticationService) Login(ctx context.Context, r *pb.UserLoginRequest) (*pb.LoginResponse, error) {
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
	err = apiserver.SetCookie(ctx, &http.Cookie{
		Name:  "sessionid",
		Value: sess.ID,
		// Tell the browser the cookie expires after a year, but the actual
		// session ID in the database will expire sooner than that.
		// We will write middleware to extend the expiration length but maybe
		// it's ok to require the user to log in once a year.
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		HttpOnly: true,
	})
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
		Name:     "sessionid",
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
	})
	if err != nil {
		return nil, err
	}
	return &pb.LogoutResponse{}, nil
}

func (as *AuthenticationService) GetSocketToken(ctx context.Context, r *pb.SocketTokenRequest) (*pb.SocketTokenResponse, error) {
	// This view requires authentication.
	sess, err := apiserver.GetSession(ctx)
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
	}, nil
}

func (as *AuthenticationService) ResetPassword(ctx context.Context, r *pb.ResetPasswordRequestStep1) (*pb.ResetPasswordResponse, error) {
	return nil, nil
}

func (as *AuthenticationService) ChangePassword(ctx context.Context, r *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	// This view requires authentication.
	sess, err := apiserver.GetSession(ctx)

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
