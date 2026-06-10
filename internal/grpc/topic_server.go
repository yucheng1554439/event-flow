package grpcserver

import (
	"context"
	"errors"

	eventflowv1 "github.com/eventflow/eventflow/api/gen/go/eventflow/v1"
	"github.com/eventflow/eventflow/internal/topic"
	"github.com/eventflow/eventflow/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TopicServer struct {
	eventflowv1.UnimplementedTopicServiceServer
	svc *topic.Service
}

func NewTopicServer(svc *topic.Service) *TopicServer {
	return &TopicServer{svc: svc}
}

func (s *TopicServer) CreateTopic(ctx context.Context, req *eventflowv1.CreateTopicRequest) (*eventflowv1.Topic, error) {
	t, err := s.svc.Create(ctx, models.CreateTopicRequest{
		Name:              req.Name,
		Partitions:        int(req.Partitions),
		ReplicationFactor: int(req.ReplicationFactor),
		RetentionHours:    int(req.RetentionHours),
		CleanupPolicy:     req.CleanupPolicy,
	})
	if err != nil {
		return nil, mapTopicError(err)
	}
	return topicToProto(t), nil
}

func (s *TopicServer) ListTopics(ctx context.Context, _ *eventflowv1.ListTopicsRequest) (*eventflowv1.ListTopicsResponse, error) {
	topics, err := s.svc.List(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list topics: %v", err)
	}
	out := make([]*eventflowv1.Topic, 0, len(topics))
	for i := range topics {
		out = append(out, topicToProto(&topics[i]))
	}
	return &eventflowv1.ListTopicsResponse{Topics: out}, nil
}

func (s *TopicServer) GetTopic(ctx context.Context, req *eventflowv1.GetTopicRequest) (*eventflowv1.Topic, error) {
	t, err := s.svc.Get(ctx, req.Name)
	if err != nil {
		return nil, mapTopicError(err)
	}
	return topicToProto(t), nil
}

func (s *TopicServer) DeleteTopic(ctx context.Context, req *eventflowv1.DeleteTopicRequest) (*eventflowv1.DeleteTopicResponse, error) {
	if err := s.svc.Delete(ctx, req.Name); err != nil {
		return nil, mapTopicError(err)
	}
	return &eventflowv1.DeleteTopicResponse{Name: req.Name, Deleted: true}, nil
}

func mapTopicError(err error) error {
	switch {
	case errors.Is(err, topic.ErrTopicExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, topic.ErrTopicNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, topic.ErrInvalidTopic):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Errorf(codes.Internal, "%v", err)
	}
}
