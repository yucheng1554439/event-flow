package grpcserver

import (
	"context"
	"time"

	eventflowv1 "github.com/eventflow/eventflow/api/gen/go/eventflow/v1"
	"github.com/eventflow/eventflow/internal/storage"
	kafkapkg "github.com/eventflow/eventflow/pkg/kafka"
	"github.com/eventflow/eventflow/pkg/metrics"
	"github.com/eventflow/eventflow/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EventServer struct {
	eventflowv1.UnimplementedEventServiceServer
	store    *storage.PostgresStore
	cache    *storage.RedisCache
	producer *kafkapkg.Producer
}

func NewEventServer(store *storage.PostgresStore, cache *storage.RedisCache, producer *kafkapkg.Producer) *EventServer {
	return &EventServer{store: store, cache: cache, producer: producer}
}

func (s *EventServer) PublishEvent(ctx context.Context, req *eventflowv1.PublishEventRequest) (*eventflowv1.Event, error) {
	if req.Topic == "" || req.EventType == "" {
		return nil, status.Error(codes.InvalidArgument, "topic and event_type required")
	}
	if req.IdempotencyKey != "" {
		dup, err := s.cache.CheckIdempotency(ctx, req.IdempotencyKey, 24*time.Hour)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "idempotency check: %v", err)
		}
		if dup {
			return nil, status.Error(codes.AlreadyExists, "duplicate idempotency key")
		}
	}
	publishReq := models.PublishRequest{
		Topic:          req.Topic,
		EventType:      req.EventType,
		IdempotencyKey: req.IdempotencyKey,
		Payload:        structToRaw(req.Payload),
		Headers:        req.Headers,
	}
	event, err := s.producer.Publish(ctx, publishReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "publish: %v", err)
	}
	_ = s.store.StoreEvent(ctx, *event)
	metrics.EventsPublishedTotal.WithLabelValues(req.Topic).Inc()
	return eventToProto(event), nil
}

func (s *EventServer) PublishBatch(ctx context.Context, req *eventflowv1.PublishBatchRequest) (*eventflowv1.PublishBatchResponse, error) {
	events := make([]models.PublishRequest, 0, len(req.Events))
	for _, e := range req.Events {
		events = append(events, models.PublishRequest{
			Topic: e.Topic, EventType: e.EventType,
			IdempotencyKey: e.IdempotencyKey,
			Payload:        structToRaw(e.Payload),
			Headers:        e.Headers,
		})
	}
	published, err := s.producer.PublishBatch(ctx, events)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "batch publish: %v", err)
	}
	out := make([]*eventflowv1.Event, 0, len(published))
	for _, e := range published {
		_ = s.store.StoreEvent(ctx, *e)
		metrics.EventsPublishedTotal.WithLabelValues(e.Topic).Inc()
		out = append(out, eventToProto(e))
	}
	return &eventflowv1.PublishBatchResponse{Events: out, Count: int32(len(out))}, nil
}
