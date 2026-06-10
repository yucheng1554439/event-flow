package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/eventflow/eventflow/pkg/models"
	"github.com/google/uuid"
)

type Producer struct {
	producer *kafka.Producer
}

func NewProducer(brokers []string) (*Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":                     joinBrokers(brokers),
		"acks":                                  "all",
		"enable.idempotence":                    true,
		"compression.type":                      "snappy",
		"linger.ms":                             10,
		"batch.size":                            65536,
		"retries":                               10,
		"retry.backoff.ms":                      100,
		"max.in.flight.requests.per.connection": 5,
	})
	if err != nil {
		return nil, fmt.Errorf("create kafka producer: %w", err)
	}
	return &Producer{producer: p}, nil
}

func (p *Producer) Close() {
	p.producer.Flush(5000)
	p.producer.Close()
}

func (p *Producer) Publish(ctx context.Context, req models.PublishRequest) (*models.Event, error) {
	eventID := uuid.New().String()
	payload, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	headers := []kafka.Header{
		{Key: "event_id", Value: []byte(eventID)},
		{Key: "event_type", Value: []byte(req.EventType)},
	}
	if req.IdempotencyKey != "" {
		headers = append(headers, kafka.Header{Key: "idempotency_key", Value: []byte(req.IdempotencyKey)})
	}
	for k, v := range req.Headers {
		headers = append(headers, kafka.Header{Key: k, Value: []byte(v)})
	}

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &req.Topic, Partition: kafka.PartitionAny},
		Key:            []byte(req.IdempotencyKey),
		Value:          payload,
		Headers:        headers,
		Timestamp:      time.Now(),
	}

	delivery := make(chan kafka.Event, 1)
	if err := p.producer.Produce(msg, delivery); err != nil {
		return nil, fmt.Errorf("produce message: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case e := <-delivery:
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			return nil, m.TopicPartition.Error
		}
		return &models.Event{
			ID:             eventID,
			Topic:          req.Topic,
			Partition:      int(m.TopicPartition.Partition),
			Offset:         int64(m.TopicPartition.Offset),
			EventType:      req.EventType,
			IdempotencyKey: req.IdempotencyKey,
			Payload:        req.Payload,
			Headers:        req.Headers,
			PublishedAt:    time.Now().UTC(),
		}, nil
	}
}

func (p *Producer) PublishBatch(ctx context.Context, events []models.PublishRequest) ([]*models.Event, error) {
	results := make([]*models.Event, 0, len(events))
	for _, e := range events {
		ev, err := p.Publish(ctx, e)
		if err != nil {
			return results, err
		}
		results = append(results, ev)
	}
	return results, nil
}

func joinBrokers(brokers []string) string {
	if len(brokers) == 0 {
		return "localhost:9092"
	}
	result := brokers[0]
	for i := 1; i < len(brokers); i++ {
		result += "," + brokers[i]
	}
	return result
}
