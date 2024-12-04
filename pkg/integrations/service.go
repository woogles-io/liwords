package integrations

import (
	"context"

	"connectrpc.com/connect"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

type IntegrationService struct {
	q *models.Queries
}

func NewIntegrationService(q *models.Queries) *IntegrationService {
	return &IntegrationService{q}
}

func (s *IntegrationService) GetIntegrations(ctx context.Context, req *connect.Request[pb.GetIntegrationsRequest]) (
	*connect.Response[pb.IntegrationsResponse], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		// Not authed.
		return nil, apiserver.Unauthenticated("need auth for this endpoint")
	}

	integrations, err := s.q.GetIntegrations(ctx, pgtype.Text{String: sess.UserUUID, Valid: true})
	if err != nil {
		return nil, err
	}

	msg := &pb.IntegrationsResponse{Integrations: make([]*pb.Integration, len(integrations))}
	for i := range integrations {
		msg.Integrations[i] = &pb.Integration{
			Uuid:            integrations[i].Uuid.String(),
			IntegrationName: integrations[i].IntegrationName,
		}
	}

	return connect.NewResponse(msg), nil

}
