package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lithammer/shortuuid/v4"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/emailer"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/sessions"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"

	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
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
	userStore     user.Store
	sessionStore  sessions.SessionStore
	configStore   config.ConfigStore
	secretKey     string
	mailgunKey    string
	discordToken  string
	argonConfig   config.ArgonConfig
	secureCookies bool
	q             *models.Queries
}

func NewAuthenticationService(u user.Store, ss sessions.SessionStore, cs config.ConfigStore,
	secretKey, mailgunKey string, discordToken string, cfg config.ArgonConfig,
	secureCookies bool, q *models.Queries) *AuthenticationService {
	return &AuthenticationService{
		userStore:     u,
		sessionStore:  ss,
		configStore:   cs,
		secretKey:     secretKey,
		mailgunKey:    mailgunKey,
		discordToken:  discordToken,
		argonConfig:   cfg,
		secureCookies: secureCookies,
		q:             q}
}

func modActionExistsErr(err error) error {
	if ue, ok := err.(*mod.UserModeratedError); ok {
		return apiserver.PermissionDenied(ue.Error())
	} else {
		return apiserver.InternalErr(err)
	}
}

// Login sets a cookie.
func (as *AuthenticationService) Login(ctx context.Context, r *connect.Request[pb.UserLoginRequest],
) (*connect.Response[pb.LoginResponse], error) {

	r.Msg.Username = strings.TrimSpace(r.Msg.Username)
	user, err := as.userStore.Get(ctx, r.Msg.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, apiserver.Unauthenticated("bad login")
	}
	matches, err := ComparePassword(r.Msg.Password, user.Password)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	if !matches {
		return nil, apiserver.Unauthenticated("password incorrect")
	}

	_, err = mod.ActionExists(ctx, as.userStore, user.UUID, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
	if err != nil {
		log.Err(err).Str("username", r.Msg.Username).Str("userID", user.UUID).Msg("action-exists-login")
		return nil, modActionExistsErr(err)
	}

	sess, err := as.sessionStore.New(ctx, user)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	err = apiserver.SetDefaultCookie(ctx, sess.ID, as.secureCookies)

	log.Info().Str("value", sess.ID).Msg("setting-cookie")
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	return connect.NewResponse(&pb.LoginResponse{}), nil
}

// Logout deletes the user session from the store, and also tells the front-end
// to ditch the cookie (yum)
func (as *AuthenticationService) Logout(ctx context.Context, r *connect.Request[pb.UserLogoutRequest],
) (*connect.Response[pb.LogoutResponse], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	err = as.sessionStore.Delete(ctx, sess)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	// Delete the cookie as well.
	err = apiserver.ExpireCookie(ctx, sess.ID, as.secureCookies)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	return connect.NewResponse(&pb.LogoutResponse{}), nil
}

func (as *AuthenticationService) GetSocketToken(ctx context.Context, r *connect.Request[pb.SocketTokenRequest],
) (*connect.Response[pb.SocketTokenResponse], error) {
	var unn, uuid string
	var authed bool

	// Maybe cache this?
	feHash, err := as.configStore.FEHash(ctx)
	if err != nil {
		log.Err(err).Msg("error getting fe-hash")
		// Continue anyway.
	}

	u, err := apiserver.AuthUser(ctx, as.userStore)
	if err != nil {
		// Auth failed - log comprehensive details for debugging
		log.Warn().
			Err(err).
			Str("authError", err.Error()).
			Msg("auth-failed-issuing-anonymous-token")

		ut, err := as.unauthedToken(ctx, feHash)
		if err != nil {
			return nil, apiserver.InternalErr(err)
		}

		log.Info().
			Str("anonymousUID", ut.Cid).
			Str("connectionID", ut.Cid).
			Msg("issued-anonymous-socket-token")

		return connect.NewResponse(ut), nil
	}
	// Otherwise, we are authenticated.

	authed = true
	uuid = u.UUID
	unn = u.Username

	roles, err := rbac.UserRoles(ctx, as.q, u.Username)
	if err != nil {
		return nil, err
	}
	perms := []string{}

	moderator := string(rbac.Moderator)
	admin := string(rbac.Admin)

	for _, r := range roles {
		if r == moderator {
			perms = append(perms, "mod")
		}
		if r == admin {
			perms = append(perms, "adm")
		}
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
		return nil, apiserver.InternalErr(err)
	}
	// create a random connection ID.
	cid := shortuuid.New()[1:10]

	log.Info().
		Str("username", unn).
		Str("userID", uuid).
		Str("connectionID", cid).
		Bool("authenticated", authed).
		Strs("permissions", perms).
		Msg("issued-authenticated-socket-token")

	return connect.NewResponse(&pb.SocketTokenResponse{
		Token:           tokenString,
		Cid:             cid,
		FrontEndVersion: feHash,
	}), nil
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
		return nil, err
	}
	return &pb.SocketTokenResponse{
		Token:           tokenString,
		Cid:             cid,
		FrontEndVersion: feHash,
	}, nil
}

func (as *AuthenticationService) GetSignedCookie(ctx context.Context, r *connect.Request[pb.GetSignedCookieRequest],
) (*connect.Response[pb.SignedCookieResponse], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		// Not authed.
		return nil, apiserver.Unauthenticated("need auth for this endpoint")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":       time.Now().Add(TokenExpiration).Unix(),
		"sessionID": sess.ID,
	})
	tokenString, err := token.SignedString([]byte(as.secretKey))
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	log.Info().Str("username", sess.Username).Msg("got-signed-cookie")
	return connect.NewResponse(&pb.SignedCookieResponse{Jwt: tokenString}), nil
}

func (as *AuthenticationService) InstallSignedCookie(ctx context.Context, r *connect.Request[pb.SignedCookieResponse],
) (*connect.Response[pb.InstallSignedCookieResponse], error) {
	token, err := jwt.Parse(r.Msg.Jwt, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apiserver.InvalidArg(fmt.Sprintf("Unexpected signing method: %v", token.Header["alg"]))
		}
		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(as.secretKey), nil
	})
	if err != nil {
		log.Err(err).Str("token", r.Msg.Jwt).Msg("token-failure")
		return nil, apiserver.InternalErr(err)
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		sessId, ok := claims["sessionID"].(string)
		if !ok {
			return nil, apiserver.InternalErr(errors.New("could not convert claim"))
		}
		log.Info().Msg("install-signed-cookie")
		err = apiserver.SetDefaultCookie(ctx, sessId, as.secureCookies)
		if err != nil {
			return nil, apiserver.InternalErr(err)
		}
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return connect.NewResponse(&pb.InstallSignedCookieResponse{}), nil
}

func (as *AuthenticationService) ResetPasswordStep1(ctx context.Context, r *connect.Request[pb.ResetPasswordRequestStep1],
) (*connect.Response[pb.ResetPasswordResponse], error) {
	email := strings.TrimSpace(r.Msg.Email)
	u, err := as.userStore.GetByEmail(ctx, email)
	if err != nil {
		return nil, apiserver.Unauthenticated(err.Error())
	}

	_, err = mod.ActionExists(ctx, as.userStore, u.UUID, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
	if err != nil {
		log.Err(err).Str("userID", u.UUID).Msg("action-exists-reset-password-step-one")
		return nil, modActionExistsErr(err)
	}

	// Create a token for the reset
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":  time.Now().Add(PasswordResetExpiration).Unix(),
		"uuid": u.UUID,
	})
	tokenString, err := token.SignedString([]byte(as.secretKey))
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	resetURL := "https://woogles.io/password/new?t=" + tokenString

	emailBody := fmt.Sprintf(ResetPasswordTemplate, resetURL, u.Username)
	log.Debug().Str("email-body", emailBody).Msg("generated-body")
	id, err := emailer.SendSimpleMessage(
		as.mailgunKey, email, "Password reset for Woogles.io", emailBody)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	log.Info().Str("id", id).Str("email", email).Msg("sent-password-reset")

	return connect.NewResponse(&pb.ResetPasswordResponse{}), nil
}

func (as *AuthenticationService) ResetPasswordStep2(ctx context.Context, r *connect.Request[pb.ResetPasswordRequestStep2],
) (*connect.Response[pb.ResetPasswordResponse], error) {

	tokenString := r.Msg.ResetCode

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(as.secretKey), nil
	})
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		uuid, ok := claims["uuid"].(string)
		if !ok {
			return nil, apiserver.InvalidArg("wrongly formatted uuid in token")
		}

		_, err = mod.ActionExists(ctx, as.userStore, uuid, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
		if err != nil {
			log.Err(err).Str("userID", uuid).Msg("action-exists-reset-password-step-two")
			return nil, modActionExistsErr(err)
		}

		config := NewPasswordConfig(as.argonConfig.Time, as.argonConfig.Memory, as.argonConfig.Threads, as.argonConfig.Keylen)
		if len(r.Msg.Password) < 8 {
			return nil, apiserver.InvalidArg(errPasswordTooShort.Error())
		}
		hashPass, err := GeneratePassword(config, r.Msg.Password)
		if err != nil {
			return nil, apiserver.InternalErr(err)
		}
		err = as.userStore.SetPassword(ctx, uuid, hashPass)
		if err != nil {
			return nil, apiserver.InternalErr(err)
		}
		return connect.NewResponse(&pb.ResetPasswordResponse{}), nil
	}

	return nil, apiserver.InvalidArg("reset code is invalid; please try again")
}

func (as *AuthenticationService) ChangePassword(ctx context.Context, r *connect.Request[pb.ChangePasswordRequest],
) (*connect.Response[pb.ChangePasswordResponse], error) {
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
		return nil, apiserver.InternalErr(err)
	}

	_, err = mod.ActionExists(ctx, as.userStore, user.UUID, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
	if err != nil {
		log.Err(err).Str("userID", user.UUID).Msg("action-exists-change-password")
		return nil, modActionExistsErr(err)
	}

	matches, err := ComparePassword(r.Msg.OldPassword, user.Password)
	if err != nil {
		return nil, err
	}
	if !matches {
		return nil, apiserver.InvalidArg("your password is incorrect")
	}

	if len(r.Msg.NewPassword) < 8 {
		return nil, apiserver.InvalidArg(errPasswordTooShort.Error())
	}

	config := NewPasswordConfig(as.argonConfig.Time, as.argonConfig.Memory, as.argonConfig.Threads, as.argonConfig.Keylen)
	hashPass, err := GeneratePassword(config, r.Msg.NewPassword)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	err = as.userStore.SetPassword(ctx, user.UUID, hashPass)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	return connect.NewResponse(&pb.ChangePasswordResponse{}), nil
}

func (as *AuthenticationService) NotifyAccountClosure(ctx context.Context, r *connect.Request[pb.NotifyAccountClosureRequest],
) (*connect.Response[pb.NotifyAccountClosureResponse], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	_, err = mod.ActionExists(ctx, as.userStore, sess.UserUUID, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
	if err != nil {
		log.Err(err).Str("userID", sess.UserUUID).Msg("action-exists-account-closure")
		return nil, modActionExistsErr(err)
	}

	// Get the user so we can send the notification with their email address
	user, err := as.userStore.Get(ctx, sess.Username)
	if err != nil {
		return nil, err
	}

	matches, err := ComparePassword(r.Msg.Password, user.Password)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	if !matches {
		return nil, apiserver.Unauthenticated("password incorrect")
	}

	// This action will not need to use the chat store so we can pass the nil value
	err = mod.ApplyActions(ctx, as.userStore, nil, user.UUID, []*ms.ModAction{{
		UserId:   sess.UserUUID,
		Duration: 0,
		Note:     "User initiated account deletion",
		Type:     ms.ModActionType_DELETE_ACCOUNT}})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.NotifyAccountClosureResponse{}), nil
}

func (as *AuthenticationService) GetAPIKey(ctx context.Context, req *connect.Request[pb.GetAPIKeyRequest],
) (*connect.Response[pb.GetAPIKeyResponse], error) {
	user, err := apiserver.AuthUser(ctx, as.userStore)
	if err != nil {
		return nil, err
	}
	var apikey string
	if req.Msg.Reset_ {
		apikey, err = as.userStore.ResetAPIKey(ctx, user.UUID)
	} else {
		apikey, err = as.userStore.GetAPIKey(ctx, user.UUID)
	}
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.GetAPIKeyResponse{Key: apikey}), nil
}
