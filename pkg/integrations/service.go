package integrations

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"

	"github.com/google/uuid"
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

		data := make(map[string]any)
		err = json.Unmarshal(integrations[i].Data, &data)
		if err != nil {
			return nil, err
		}
		details := make(map[string]string)
		if integrations[i].IntegrationName == TwitchIntegrationName {
			details["twitch_username"] = data["twitch_username"].(string)
		}
		msg.Integrations[i] = &pb.Integration{
			Uuid:               integrations[i].Uuid.String(),
			IntegrationName:    integrations[i].IntegrationName,
			IntegrationDetails: details,
		}
	}

	return connect.NewResponse(msg), nil

}

func (s *IntegrationService) DeleteIntegration(ctx context.Context, req *connect.Request[pb.DeleteIntegrationRequest]) (
	*connect.Response[pb.DeleteIntegrationResponse], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		// Not authed.
		return nil, apiserver.Unauthenticated("need auth for this endpoint")
	}
	iuuid, err := uuid.Parse(req.Msg.Uuid)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid uuid")
	}
	userUuid := pgtype.Text{String: sess.UserUUID, Valid: true}

	err = s.q.DeleteIntegration(ctx, models.DeleteIntegrationParams{
		UserUuid:        userUuid,
		IntegrationUuid: iuuid,
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.DeleteIntegrationResponse{}), nil
}
