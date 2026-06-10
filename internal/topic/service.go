package topic

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	kafkapkg "github.com/eventflow/eventflow/pkg/kafka"
	"github.com/eventflow/eventflow/pkg/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var topicNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

var (
	ErrTopicExists    = errors.New("topic already exists")
	ErrTopicNotFound  = errors.New("topic not found")
	ErrInvalidTopic   = errors.New("invalid topic name or configuration")
)

type Service struct {
	repo  *Repository
	admin *kafkapkg.Admin
	log   *zap.Logger
}

func NewService(repo *Repository, admin *kafkapkg.Admin, log *zap.Logger) *Service {
	return &Service{repo: repo, admin: admin, log: log}
}

func (s *Service) Create(ctx context.Context, req models.CreateTopicRequest) (*models.Topic, error) {
	if err := validateCreateRequest(req); err != nil {
		return nil, err
	}

	replication := req.ReplicationFactor
	if replication == 0 {
		replication = req.Replication
	}
	if replication == 0 {
		replication = 1
	}

	cfg := kafkapkg.TopicConfig{
		Name:              req.Name,
		Partitions:        req.Partitions,
		ReplicationFactor: replication,
		RetentionHours:    req.RetentionHours,
		CleanupPolicy:     req.CleanupPolicy,
	}

	if _, err := s.repo.Get(ctx, req.Name); err == nil {
		return nil, ErrTopicExists
	}

	if err := s.admin.CreateTopic(ctx, cfg); err != nil {
		return nil, fmt.Errorf("kafka create topic: %w", err)
	}
	if err := s.admin.WaitForTopic(ctx, req.Name); err != nil {
		return nil, err
	}

	topic := models.Topic{
		ID:                uuid.New().String(),
		Name:              cfg.Name,
		Partitions:        cfg.Partitions,
		Replication:       cfg.ReplicationFactor,
		ReplicationFactor: cfg.ReplicationFactor,
		RetentionHours:    cfg.RetentionHours,
		CleanupPolicy:     cfg.CleanupPolicy,
		Compression:       "snappy",
	}
	if err := s.repo.Create(ctx, topic); err != nil {
		return nil, fmt.Errorf("persist topic: %w", err)
	}

	s.log.Info("topic created", zap.String("name", topic.Name), zap.Int("partitions", topic.Partitions))
	return &topic, nil
}

func (s *Service) Get(ctx context.Context, name string) (*models.Topic, error) {
	topic, err := s.repo.Get(ctx, name)
	if err != nil {
		return nil, ErrTopicNotFound
	}
	exists, err := s.admin.TopicExists(ctx, name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrTopicNotFound
	}
	return topic, nil
}

func (s *Service) List(ctx context.Context) ([]models.Topic, error) {
	return s.repo.List(ctx)
}

func (s *Service) Delete(ctx context.Context, name string) error {
	if _, err := s.repo.Get(ctx, name); err != nil {
		return ErrTopicNotFound
	}
	if err := s.admin.DeleteTopic(ctx, name); err != nil {
		return fmt.Errorf("kafka delete topic: %w", err)
	}
	if err := s.repo.Delete(ctx, name); err != nil {
		return fmt.Errorf("delete topic metadata: %w", err)
	}
	s.log.Info("topic deleted", zap.String("name", name))
	return nil
}

func validateCreateRequest(req models.CreateTopicRequest) error {
	if req.Name == "" || !topicNamePattern.MatchString(req.Name) {
		return fmt.Errorf("%w: name must be alphanumeric with . _ -", ErrInvalidTopic)
	}
	if req.Partitions < 0 || req.Partitions > 1000 {
		return fmt.Errorf("%w: partitions must be 1-1000", ErrInvalidTopic)
	}
	if req.Partitions == 0 {
		req.Partitions = 6
	}
	policy := strings.ToLower(req.CleanupPolicy)
	if policy != "" && policy != "delete" && policy != "compact" {
		return fmt.Errorf("%w: cleanupPolicy must be delete or compact", ErrInvalidTopic)
	}
	return nil
}
