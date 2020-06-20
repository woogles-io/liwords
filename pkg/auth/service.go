package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/domino14/liwords/pkg/apiserver"

	"github.com/domino14/liwords/pkg/user"

	pb "github.com/domino14/liwords/rpc/api/proto"
)

const TokenExpiration = 15 * time.Second

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
		Expires: time.Now().Add(365 * 24 * time.Hour),
	})
	if err != nil {
		return nil, err
	}
	return &pb.LoginResponse{}, nil
}

func (as *AuthenticationService) Logout(ctx context.Context, r *pb.UserLogoutRequest) (*pb.LogoutResponse, error) {

	return nil, nil
}

func (as *AuthenticationService) GetSocketToken(ctx context.Context, r *pb.SocketTokenRequest) (*pb.SocketTokenResponse, error) {
	// This view requires authentication.
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(TokenExpiration).Unix(),
		"iss": "liwords",
		"aud": "liwords-socket",
		"uid": sess.UserUUID,
		"unn": sess.Username,
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
