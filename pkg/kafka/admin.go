package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/eventflow/eventflow/pkg/models"
)

type Admin struct {
	client *kafka.AdminClient
}

type TopicConfig struct {
	Name              string
	Partitions        int
	ReplicationFactor int
	RetentionHours    int
	CleanupPolicy     string
}

func NewAdmin(brokers []string) (*Admin, error) {
	client, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": joinBrokers(brokers),
	})
	if err != nil {
		return nil, fmt.Errorf("create admin client: %w", err)
	}
	return &Admin{client: client}, nil
}

func (a *Admin) Close() {
	a.client.Close()
}

func (a *Admin) CreateTopic(ctx context.Context, cfg TopicConfig) error {
	if cfg.Partitions <= 0 {
		cfg.Partitions = 6
	}
	if cfg.ReplicationFactor <= 0 {
		cfg.ReplicationFactor = 1
	}
	if cfg.RetentionHours <= 0 {
		cfg.RetentionHours = 168
	}
	if cfg.CleanupPolicy == "" {
		cfg.CleanupPolicy = "delete"
	}

	retentionMs := fmt.Sprintf("%d", cfg.RetentionHours*3600*1000)
	spec := kafka.TopicSpecification{
		Topic:             cfg.Name,
		NumPartitions:     cfg.Partitions,
		ReplicationFactor: cfg.ReplicationFactor,
		Config: map[string]string{
			"retention.ms":   retentionMs,
			"cleanup.policy": cfg.CleanupPolicy,
			"compression.type": "snappy",
		},
	}

	results, err := a.client.CreateTopics(ctx, []kafka.TopicSpecification{spec})
	if err != nil {
		return fmt.Errorf("create topic %s: %w", cfg.Name, err)
	}
	for _, r := range results {
		if r.Error.Code() != kafka.ErrNoError && r.Error.Code() != kafka.ErrTopicAlreadyExists {
			return fmt.Errorf("create topic %s: %v", r.Topic, r.Error)
		}
	}
	return nil
}

func (a *Admin) DeleteTopic(ctx context.Context, name string) error {
	results, err := a.client.DeleteTopics(ctx, []string{name})
	if err != nil {
		return fmt.Errorf("delete topic %s: %w", name, err)
	}
	for _, r := range results {
		if r.Error.Code() != kafka.ErrNoError && r.Error.Code() != kafka.ErrUnknownTopicOrPart {
			return fmt.Errorf("delete topic %s: %v", r.Topic, r.Error)
		}
	}
	return nil
}

func (a *Admin) TopicExists(ctx context.Context, name string) (bool, error) {
	meta, err := a.client.GetMetadata(&name, false, 5000)
	if err != nil {
		return false, err
	}
	_, ok := meta.Topics[name]
	return ok, nil
}

func (a *Admin) ListKafkaTopics(ctx context.Context) (map[string]kafka.TopicMetadata, error) {
	meta, err := a.client.GetMetadata(nil, false, 5000)
	if err != nil {
		return nil, err
	}
	return meta.Topics, nil
}

func (a *Admin) WaitForTopic(ctx context.Context, name string) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		exists, err := a.TopicExists(ctx, name)
		if err == nil && exists {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return fmt.Errorf("topic %s not ready", name)
}

func TopicFromConfig(cfg TopicConfig) models.Topic {
	return models.Topic{
		Name:            cfg.Name,
		Partitions:      cfg.Partitions,
		Replication:     cfg.ReplicationFactor,
		RetentionHours:  cfg.RetentionHours,
		CleanupPolicy:   cfg.CleanupPolicy,
		Compression:     "snappy",
		CreatedAt:       time.Now().UTC(),
	}
}
