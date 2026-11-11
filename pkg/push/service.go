package push

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/apiserver"
	pb "github.com/woogles-io/liwords/rpc/api/proto/push_service"
)

var errPushOff = connect.NewError(connect.CodeFailedPrecondition, errors.New("push not configured"))

// Service is the push RPCs a page calls to subscribe its browser.
type Service struct {
	// sender is nil when web push is off.
	sender *Sender
}

// NewService returns the push service; a nil sender keeps it off.
func NewService(sender *Sender) *Service {
	return &Service{sender: sender}
}

func (s *Service) GetPushConfig(ctx context.Context, req *connect.Request[pb.GetPushConfigRequest],
) (*connect.Response[pb.PushConfigResponse], error) {
	if s.sender == nil {
		return connect.NewResponse(&pb.PushConfigResponse{}), nil
	}
	return connect.NewResponse(&pb.PushConfigResponse{
		Enabled:   true,
		PublicKey: s.sender.PublicKey(),
	}), nil
}

func (s *Service) Subscribe(ctx context.Context, req *connect.Request[pb.SubscribeRequest],
) (*connect.Response[pb.PushResponse], error) {
	if s.sender == nil {
		return nil, errPushOff
	}
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	err = s.sender.Subscribe(ctx, sess.UserUUID, req.Msg.Endpoint, req.Msg.P256Dh, req.Msg.Auth)
	switch {
	case errors.Is(err, errBadEndpoint) || errors.Is(err, errBadKeys):
		return nil, apiserver.InvalidArg(err.Error())
	case errors.Is(err, errNoUser):
		return nil, apiserver.Unauthenticated(err.Error())
	case err != nil:
		log.Err(err).Str("userID", sess.UserUUID).Msg("web-push-subscribe")
		return nil, apiserver.InternalErr(err)
	}
	return connect.NewResponse(&pb.PushResponse{}), nil
}

// Unsubscribe forgets the endpoint only if it is the signed-in user's, and
// answers the same either way.
func (s *Service) Unsubscribe(ctx context.Context, req *connect.Request[pb.UnsubscribeRequest],
) (*connect.Response[pb.PushResponse], error) {
	if s.sender == nil {
		return nil, errPushOff
	}
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.sender.Unsubscribe(ctx, sess.UserUUID, req.Msg.Endpoint); err != nil {
		log.Err(err).Str("userID", sess.UserUUID).Msg("web-push-unsubscribe")
		return nil, apiserver.InternalErr(err)
	}
	return connect.NewResponse(&pb.PushResponse{}), nil
}
