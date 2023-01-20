package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/twitchtv/twirp"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/emailer"
	"github.com/domino14/liwords/pkg/sessions"

	"github.com/rs/zerolog/log"

	"github.com/lithammer/shortuuid"

	"github.com/dgrijalva/jwt-go"
	"github.com/domino14/liwords/pkg/apiserver"

	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/user"

	ipc "github.com/domino14/liwords/rpc/api/proto/ipc"
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

const AccountClosureTemplate = `
Dear Woogles Administrators,

The following user has deleted their account:

User:  %s
Email: %s

`

var errPasswordTooShort = errors.New("your password is too short, use 8 or more characters")

type AuthenticationService struct {
	userStore    user.Store
	sessionStore sessions.SessionStore
	configStore  config.ConfigStore
	secretKey    string
	mailgunKey   string
	discordToken string
	argonConfig  config.ArgonConfig
}

func NewAuthenticationService(u user.Store, ss sessions.SessionStore, cs config.ConfigStore,
	secretKey, mailgunKey string, discordToken string, cfg config.ArgonConfig) *AuthenticationService {
	return &AuthenticationService{
		userStore:    u,
		sessionStore: ss,
		configStore:  cs,
		secretKey:    secretKey,
		mailgunKey:   mailgunKey,
		discordToken: discordToken,
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

	_, err = mod.ActionExists(ctx, as.userStore, user.UUID, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
	if err != nil {
		log.Err(err).Str("username", r.Username).Str("userID", user.UUID).Msg("action-exists-login")
		return nil, err
	}

	sess, err := as.sessionStore.New(ctx, user)
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
	err = apiserver.ExpireCookie(ctx, sess.ID)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &pb.LogoutResponse{}, nil
}

func (as *AuthenticationService) GetSocketToken(ctx context.Context, r *pb.SocketTokenRequest) (*pb.SocketTokenResponse, error) {
	var unn, uuid string
	var authed bool

	// Maybe cache this?
	feHash, err := as.configStore.FEHash(ctx)
	if err != nil {
		log.Err(err).Msg("error getting fe-hash")
		// Continue anyway.
	}

	u, err := apiserver.AuthUser(ctx, apiserver.CookieFirst, as.userStore)
	if err != nil {
		return as.unauthedToken(ctx, feHash)
	} else {
		authed = true
		uuid = u.UUID
		unn = u.Username
	}
	perms := []string{}
	if u.IsAdmin {
		perms = append(perms, "adm")
	}
	if u.IsDirector {
		perms = append(perms, "dir")
	}
	if u.IsMod {
		perms = append(perms, "mod")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":   time.Now().Add(TokenExpiration).Unix(),
		"uid":   uuid,
		"unn":   unn,
		"a":     authed,
		"cs":    u.IsChild(),
		"perms": strings.Join(perms, ","),
	})
	tokenString, err := token.SignedString([]byte(as.secretKey))
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	// create a random connection ID.
	cid := shortuuid.New()[1:10]
	return &pb.SocketTokenResponse{
		Token:           tokenString,
		Cid:             cid,
		FrontEndVersion: feHash,
	}, nil
}

func (as *AuthenticationService) unauthedToken(ctx context.Context, feHash string) (*pb.SocketTokenResponse, error) {
	cid := shortuuid.New()
	uid := "anon-" + cid
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":   time.Now().Add(TokenExpiration).Unix(),
		"uid":   uid,
		"unn":   uid,
		"a":     false, // not authed
		"cs":    ipc.ChildStatus_UNKNOWN,
		"perms": "",
	})
	tokenString, err := token.SignedString([]byte(as.secretKey))
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &pb.SocketTokenResponse{
		Token:           tokenString,
		Cid:             cid,
		FrontEndVersion: feHash,
	}, nil
}

func (as *AuthenticationService) GetSignedCookie(ctx context.Context, r *pb.GetSignedCookieRequest) (*pb.SignedCookieResponse, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		// Not authed.
		return nil, twirp.NewError(twirp.Unauthenticated, "need auth for this endpoint")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":       time.Now().Add(TokenExpiration).Unix(),
		"sessionID": sess.ID,
	})
	tokenString, err := token.SignedString([]byte(as.secretKey))
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	log.Info().Str("username", sess.Username).Msg("got-signed-cookie")
	return &pb.SignedCookieResponse{Jwt: tokenString}, nil
}

func (as *AuthenticationService) InstallSignedCookie(ctx context.Context, r *pb.SignedCookieResponse) (*pb.InstallSignedCookieResponse, error) {
	token, err := jwt.Parse(r.Jwt, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(as.secretKey), nil
	})
	if err != nil {
		log.Err(err).Str("token", r.Jwt).Msg("token-failure")
		return nil, twirp.InternalErrorWith(err)
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		sessId, ok := claims["sessionID"].(string)
		if !ok {
			return nil, twirp.InternalError("could not convert claim")
		}
		log.Info().Msg("install-signed-cookie")
		err = apiserver.SetDefaultCookie(ctx, sessId)
		if err != nil {
			return nil, twirp.InternalErrorWith(err)
		}
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return &pb.InstallSignedCookieResponse{}, nil
}

func (as *AuthenticationService) ResetPasswordStep1(ctx context.Context, r *pb.ResetPasswordRequestStep1) (*pb.ResetPasswordResponse, error) {
	email := strings.TrimSpace(r.Email)
	u, err := as.userStore.GetByEmail(ctx, email)
	if err != nil {
		return nil, twirp.NewError(twirp.Unauthenticated, err.Error())
	}

	_, err = mod.ActionExists(ctx, as.userStore, u.UUID, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
	if err != nil {
		log.Err(err).Str("userID", u.UUID).Msg("action-exists-reset-password-step-one")
		return nil, err
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

	emailBody := fmt.Sprintf(ResetPasswordTemplate, resetURL, u.Username)
	log.Debug().Str("email-body", emailBody).Msg("generated-body")
	id, err := emailer.SendSimpleMessage(
		as.mailgunKey, email, "Password reset for Woogles.io", emailBody)
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

		_, err = mod.ActionExists(ctx, as.userStore, uuid, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
		if err != nil {
			log.Err(err).Str("userID", uuid).Msg("action-exists-reset-password-step-two")
			return nil, err
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

	_, err = mod.ActionExists(ctx, as.userStore, user.UUID, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
	if err != nil {
		log.Err(err).Str("userID", user.UUID).Msg("action-exists-change-password")
		return nil, err
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

func (as *AuthenticationService) NotifyAccountClosure(ctx context.Context, r *pb.NotifyAccountClosureRequest) (*pb.NotifyAccountClosureResponse, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	_, err = mod.ActionExists(ctx, as.userStore, sess.UserUUID, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
	if err != nil {
		log.Err(err).Str("userID", sess.UserUUID).Msg("action-exists-account-closure")
		return nil, err
	}

	// Get the user so we can send the notification with their email address
	user, err := as.userStore.Get(ctx, sess.Username)
	if err != nil {
		return nil, err
	}

	matches, err := ComparePassword(r.Password, user.Password)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	if !matches {
		return nil, twirp.NewError(twirp.Unauthenticated, "password incorrect")
	}

	// This action will not need to use the chat store so we can pass the nil value
	err = mod.ApplyActions(ctx, as.userStore, nil, []*ms.ModAction{{
		UserId:   sess.UserUUID,
		Duration: 0,
		Note:     "User initiated account deletion",
		Type:     ms.ModActionType_DELETE_ACCOUNT}})
	if err != nil {
		return nil, err
	}

	return &pb.NotifyAccountClosureResponse{}, nil
}

func (as *AuthenticationService) GetAPIKey(ctx context.Context, req *pb.GetAPIKeyRequest) (*pb.GetAPIKeyResponse, error) {
	user, err := apiserver.AuthUser(ctx, apiserver.CookieOnly, as.userStore)
	if err != nil {
		return nil, twirp.NewError(twirp.Unauthenticated, "did not authenticate")
	}
	var apikey string
	if req.Reset_ {
		apikey, err = as.userStore.ResetAPIKey(ctx, user.UUID)
	} else {
		apikey, err = as.userStore.GetAPIKey(ctx, user.UUID)
	}
	if err != nil {
		return nil, err
	}
	return &pb.GetAPIKeyResponse{
		Key: apikey,
	}, nil
}
