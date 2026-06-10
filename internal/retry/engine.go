package retry

import (
	"context"
	"math"
	"time"

	"github.com/eventflow/eventflow/internal/storage"
	"github.com/eventflow/eventflow/pkg/config"
	"github.com/eventflow/eventflow/pkg/metrics"
	"github.com/eventflow/eventflow/pkg/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Engine struct {
	store  *storage.PostgresStore
	policy config.RetryPolicy
	log    *zap.Logger
}

func NewEngine(store *storage.PostgresStore, policy config.RetryPolicy, log *zap.Logger) *Engine {
	return &Engine{store: store, policy: policy, log: log}
}

func (e *Engine) Schedule(ctx context.Context, eventID, topic, errMsg string, attempt int) (*models.RetryRecord, error) {
	now := time.Now().UTC()
	if attempt >= e.policy.MaxAttempts {
		return nil, e.moveToDLQ(ctx, eventID, topic, errMsg, attempt)
	}

	next := e.backoff(attempt)
	record := models.RetryRecord{
		ID:          uuid.New().String(),
		EventID:     eventID,
		Topic:       topic,
		Attempt:     attempt + 1,
		MaxAttempts: e.policy.MaxAttempts,
		NextRetryAt: now.Add(next),
		LastError:   errMsg,
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := e.store.CreateRetry(ctx, record); err != nil {
		return nil, err
	}
	metrics.RetryAttemptsTotal.WithLabelValues(topic, "scheduled").Inc()
	e.log.Info("retry scheduled", zap.String("topic", topic), zap.String("eventId", eventID), zap.Int("attempt", record.Attempt), zap.Time("nextRetryAt", record.NextRetryAt))
	return &record, nil
}

func (e *Engine) ProcessPending(ctx context.Context, handler func(ctx context.Context, r models.RetryRecord) error) error {
	records, err := e.store.PendingRetries(ctx, time.Now().UTC())
	if err != nil {
		return err
	}
	for _, r := range records {
		if err := handler(ctx, r); err != nil {
			r.LastError = err.Error()
			r.Attempt++
			r.UpdatedAt = time.Now().UTC()
			if r.Attempt >= r.MaxAttempts {
				r.Status = "failed"
				_ = e.store.UpdateRetry(ctx, r)
				eventType, payload := "unknown", []byte("{}")
				if ev, evErr := e.store.GetEventByID(ctx, r.EventID); evErr == nil && ev != nil {
					eventType = ev.EventType
					payload = ev.Payload
				}
				_ = e.MoveToDLQ(ctx, r.EventID, r.Topic, eventType, payload, err.Error(), r.Attempt)
				metrics.RetryAttemptsTotal.WithLabelValues(r.Topic, "dlq").Inc()
				continue
			}
			r.Status = "pending"
			r.NextRetryAt = time.Now().UTC().Add(e.backoff(r.Attempt))
			_ = e.store.UpdateRetry(ctx, r)
			metrics.RetryAttemptsTotal.WithLabelValues(r.Topic, "failed").Inc()
			continue
		}
		r.Status = "completed"
		r.UpdatedAt = time.Now().UTC()
		_ = e.store.UpdateRetry(ctx, r)
		metrics.RetryAttemptsTotal.WithLabelValues(r.Topic, "success").Inc()
	}
	return nil
}

func (e *Engine) backoff(attempt int) time.Duration {
	delay := float64(e.policy.InitialBackoff) * math.Pow(e.policy.Multiplier, float64(attempt))
	if delay > float64(e.policy.MaxBackoff) {
		delay = float64(e.policy.MaxBackoff)
	}
	return time.Duration(delay)
}

func (e *Engine) moveToDLQ(ctx context.Context, eventID, topic, errMsg string, attempts int) error {
	return e.MoveToDLQ(ctx, eventID, topic, "unknown", []byte("{}"), errMsg, attempts)
}

func (e *Engine) MoveToDLQ(ctx context.Context, eventID, topic, eventType string, payload []byte, errMsg string, attempts int) error {
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	msg := models.DLQMessage{
		ID:            uuid.New().String(),
		OriginalTopic: topic,
		EventID:       eventID,
		EventType:     eventType,
		Payload:       payload,
		FailureReason: errMsg,
		RetryAttempts: attempts,
		FailedAt:      time.Now().UTC(),
	}
	metrics.EventsFailedTotal.WithLabelValues(topic, "retry-engine", "max_retries").Inc()
	metrics.DLQMessagesTotal.WithLabelValues(topic).Inc()
	e.log.Warn("routed to DLQ", zap.String("topic", topic), zap.String("dlqTopic", topic+"-dlq"), zap.String("eventId", eventID), zap.Int("attempts", attempts))
	return e.store.StoreDLQ(ctx, msg)
}

func (e *Engine) MaxAttempts() int { return e.policy.MaxAttempts }
