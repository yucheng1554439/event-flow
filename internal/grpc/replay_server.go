package grpcserver

import (
	"context"

	eventflowv1 "github.com/eventflow/eventflow/api/gen/go/eventflow/v1"
	"github.com/eventflow/eventflow/internal/replay"
	"github.com/eventflow/eventflow/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ReplayServer struct {
	eventflowv1.UnimplementedReplayServiceServer
	svc *replay.Service
}

func NewReplayServer(svc *replay.Service) *ReplayServer {
	return &ReplayServer{svc: svc}
}

func (s *ReplayServer) ReplayEvents(ctx context.Context, req *eventflowv1.ReplayEventsRequest) (*eventflowv1.ReplayEventsResponse, error) {
	replayReq := models.ReplayRequest{
		Topic:       req.Topic,
		DLQOnly:     req.DlqOnly,
		TargetTopic: req.TargetTopic,
		StartTime:   timeFromProto(req.StartTime),
		EndTime:     timeFromProto(req.EndTime),
	}
	if req.Partition != nil {
		p := int(*req.Partition)
		replayReq.Partition = &p
	}
	count, err := s.svc.Replay(ctx, replayReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "replay: %v", err)
	}
	return &eventflowv1.ReplayEventsResponse{Replayed: int32(count)}, nil
}
