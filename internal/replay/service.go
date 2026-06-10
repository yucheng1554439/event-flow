package replay

import (
	"context"
	"fmt"
	"time"

	"github.com/eventflow/eventflow/internal/storage"
	kafkapkg "github.com/eventflow/eventflow/pkg/kafka"
	"github.com/eventflow/eventflow/pkg/models"
	"go.uber.org/zap"
)

type Service struct {
	store    *storage.PostgresStore
	producer *kafkapkg.Producer
	log      *zap.Logger
}

func NewService(store *storage.PostgresStore, producer *kafkapkg.Producer, log *zap.Logger) *Service {
	return &Service{store: store, producer: producer, log: log}
}

func (s *Service) Replay(ctx context.Context, req models.ReplayRequest) (int, error) {
	target := req.Topic
	if req.TargetTopic != "" {
		target = req.TargetTopic
	}

	if req.DLQOnly {
		return s.replayDLQ(ctx, req.Topic, target)
	}

	start := time.Unix(0, 0).UTC()
	end := time.Now().UTC()
	if req.StartTime != nil {
		start = *req.StartTime
	}
	if req.EndTime != nil {
		end = *req.EndTime
	}

	events, err := s.store.EventsInRange(ctx, req.Topic, req.Partition, start, end)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, e := range events {
		_, err := s.producer.Publish(ctx, models.PublishRequest{
			Topic:     target,
			EventType: e.EventType,
			Payload:   e.Payload,
			Headers:   map[string]string{"replay": "true", "original_event_id": e.ID},
		})
		if err != nil {
			return count, fmt.Errorf("replay event %s: %w", e.ID, err)
		}
		count++
	}
	s.log.Info("replay completed", zap.String("topic", req.Topic), zap.Int("count", count))
	return count, nil
}

func (s *Service) replayDLQ(ctx context.Context, sourceTopic, target string) (int, error) {
	msgs, err := s.store.ListDLQ(ctx, sourceTopic, 1000)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, m := range msgs {
		if m.ReplayedAt != nil {
			continue
		}
		_, err := s.producer.Publish(ctx, models.PublishRequest{
			Topic:     target,
			EventType: m.EventType,
			Payload:   m.Payload,
			Headers:   map[string]string{"replay": "true", "dlq_id": m.ID},
		})
		if err != nil {
			return count, err
		}
		_ = s.store.MarkDLQReplayed(ctx, m.ID)
		count++
	}
	return count, nil
}
