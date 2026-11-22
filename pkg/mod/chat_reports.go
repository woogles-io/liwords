package mod

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	pb "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

// reportedOf lists which of msgs someone reported, for chat review to mark.
func (ms *ModService) reportedOf(ctx context.Context, msgs []*ipc.ChatMessage) ([]string, error) {
	ids := make([]uuid.UUID, 0, len(msgs))
	for _, m := range msgs {
		if id, err := uuid.Parse(m.Id); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := ms.queries.ReportedMessages(ctx, ids)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	reported := make([]string, 0, len(rows))
	for _, id := range rows {
		reported = append(reported, id.String())
	}
	return reported, nil
}

// ReportChatMessage puts a chat message in front of the moderators: anyone
// signed in, any message. The same person's second report of a message is no
// new report.
func (ms *ModService) ReportChatMessage(ctx context.Context, req *connect.Request[pb.ReportChatMessageRequest],
) (*connect.Response[pb.ReportChatMessageResponse], error) {
	user, err := apiserver.AuthUser(ctx, ms.userStore)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.MessageId)
	if err != nil {
		return nil, apiserver.InvalidArg("bad message id")
	}
	err = ms.queries.AddChatReport(ctx, models.AddChatReportParams{
		MessageID:    id,
		ReporterUuid: user.UUID,
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	log.Info().Str("reporter", user.UUID).Str("message-id", id.String()).Msg("chat-report")
	return connect.NewResponse(&pb.ReportChatMessageResponse{}), nil
}

// GetChatReports lists the reports no moderator has handled, newest first, or
// the handled ones, the most recently handled first, each with its message,
// deleted or not, its reporter, and who handled it (moderators).
func (ms *ModService) GetChatReports(ctx context.Context, req *connect.Request[pb.GetChatReportsRequest],
) (*connect.Response[pb.ChatReports], error) {
	if _, err := authenticateMod(ctx, ms, nil); err != nil {
		return nil, err
	}
	rows, err := ms.queries.ChatReports(ctx, req.Msg.Handled)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	reports := make([]*pb.ChatReport, 0, len(rows))
	for _, r := range rows {
		report := &pb.ChatReport{
			Id: r.ID,
			Message: &ipc.ChatMessage{
				Id:        r.MessageID.String(),
				Channel:   r.Channel,
				Username:  r.SenderUsername,
				UserId:    r.SenderUuid,
				Message:   r.Message,
				Timestamp: r.SentAt.Time.UnixMilli(),
			},
			Deleted:           r.Deleted,
			ReporterId:        r.ReporterUuid,
			ReporterUsername:  r.ReporterUsername,
			ReportedAt:        timestamppb.New(r.CreatedAt.Time),
			HandledById:       r.HandlerUuid,
			HandledByUsername: r.HandlerUsername,
		}
		if r.HandledAt.Valid {
			report.HandledAt = timestamppb.New(r.HandledAt.Time)
		}
		reports = append(reports, report)
	}
	return connect.NewResponse(&pb.ChatReports{Reports: reports}), nil
}

// HandleChatReport marks a report handled, by the moderator asking. A report
// handled already is no error.
func (ms *ModService) HandleChatReport(ctx context.Context, req *connect.Request[pb.HandleChatReportRequest],
) (*connect.Response[pb.HandleChatReportResponse], error) {
	modUUID, err := authenticateMod(ctx, ms, nil)
	if err != nil {
		return nil, err
	}
	err = ms.queries.HandleChatReport(ctx, models.HandleChatReportParams{
		ModeratorUuid: modUUID,
		ReportID:      req.Msg.ReportId,
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	log.Info().Str("moderator", modUUID).Int64("report-id", req.Msg.ReportId).Msg("chat-report-handled")
	return connect.NewResponse(&pb.HandleChatReportResponse{}), nil
}

// ReopenChatReport opens a handled report again, for a moderator who marked it
// handled by mistake; who had handled it stays in the log
// (chat-report-handled). A report open already is no error.
func (ms *ModService) ReopenChatReport(ctx context.Context, req *connect.Request[pb.ReopenChatReportRequest],
) (*connect.Response[pb.ReopenChatReportResponse], error) {
	modUUID, err := authenticateMod(ctx, ms, nil)
	if err != nil {
		return nil, err
	}
	err = ms.queries.ReopenChatReport(ctx, req.Msg.ReportId)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	log.Info().Str("moderator", modUUID).Int64("report-id", req.Msg.ReportId).Msg("chat-report-reopened")
	return connect.NewResponse(&pb.ReopenChatReportResponse{}), nil
}
