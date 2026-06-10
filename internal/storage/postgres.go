package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/eventflow/eventflow/pkg/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) Close() { s.pool.Close() }

func (s *PostgresStore) Pool() *pgxpool.Pool { return s.pool }

func (s *PostgresStore) CreateTopic(ctx context.Context, t models.Topic) error {
	cleanup := t.CleanupPolicy
	if cleanup == "" {
		cleanup = "delete"
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO topics (id, name, partitions, replication, retention_hours, compression, cleanup_policy, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, t.ID, t.Name, t.Partitions, t.Replication, t.RetentionHours, t.Compression, cleanup, t.CreatedAt)
	return err
}

func (s *PostgresStore) GetTopic(ctx context.Context, name string) (*models.Topic, error) {
	var t models.Topic
	var cleanup string
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, partitions, replication, retention_hours, compression, COALESCE(cleanup_policy, 'delete'), created_at
		FROM topics WHERE name = $1
	`, name).Scan(&t.ID, &t.Name, &t.Partitions, &t.Replication, &t.RetentionHours, &t.Compression, &cleanup, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	t.CleanupPolicy = cleanup
	t.ReplicationFactor = t.Replication
	return &t, nil
}

func (s *PostgresStore) ListTopics(ctx context.Context) ([]models.Topic, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, partitions, replication, retention_hours, compression, COALESCE(cleanup_policy, 'delete'), created_at
		FROM topics ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []models.Topic
	for rows.Next() {
		var t models.Topic
		var cleanup string
		if err := rows.Scan(&t.ID, &t.Name, &t.Partitions, &t.Replication, &t.RetentionHours, &t.Compression, &cleanup, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.CleanupPolicy = cleanup
		t.ReplicationFactor = t.Replication
		topics = append(topics, t)
	}
	return topics, rows.Err()
}

func (s *PostgresStore) DeleteTopic(ctx context.Context, name string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM topics WHERE name = $1`, name)
	return err
}

func (s *PostgresStore) GetEventByIdempotencyKey(ctx context.Context, key string) (*models.Event, error) {
	var e models.Event
	var idKey *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, topic, partition, "offset", event_type, idempotency_key, payload, published_at
		FROM events WHERE idempotency_key = $1
	`, key).Scan(&e.ID, &e.Topic, &e.Partition, &e.Offset, &e.EventType, &idKey, &e.Payload, &e.PublishedAt)
	if err != nil {
		return nil, err
	}
	if idKey != nil {
		e.IdempotencyKey = *idKey
	}
	return &e, nil
}

func (s *PostgresStore) CountDLQ(ctx context.Context, topic string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM dead_letter_messages WHERE original_topic = $1
	`, topic).Scan(&count)
	return count, err
}

func (s *PostgresStore) CountDLQUnreplayed(ctx context.Context, topic string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM dead_letter_messages
		WHERE original_topic = $1 AND replayed_at IS NULL
	`, topic).Scan(&count)
	return count, err
}

func (s *PostgresStore) ListRetries(ctx context.Context, topic, eventID string, limit int) ([]models.RetryRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, event_id, topic, attempt, max_attempts, next_retry_at, last_error, status, created_at, updated_at
		FROM retries WHERE topic = $1`
	args := []any{topic}
	if eventID != "" {
		query += ` AND event_id = $2`
		args = append(args, eventID)
	}
	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.RetryRecord
	for rows.Next() {
		var r models.RetryRecord
		if err := rows.Scan(&r.ID, &r.EventID, &r.Topic, &r.Attempt, &r.MaxAttempts, &r.NextRetryAt, &r.LastError, &r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *PostgresStore) CountRetries(ctx context.Context, eventID string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM retries WHERE event_id = $1`, eventID).Scan(&count)
	return count, err
}

func (s *PostgresStore) GetMaxRetryAttempt(ctx context.Context, eventID string) (int, error) {
	var attempt int
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(attempt), 0) FROM retries WHERE event_id = $1
	`, eventID).Scan(&attempt)
	return attempt, err
}

func (s *PostgresStore) GetEventByID(ctx context.Context, id string) (*models.Event, error) {
	var e models.Event
	var idKey *string
	err := s.pool.QueryRow(ctx, `
		SELECT id, topic, partition, "offset", event_type, idempotency_key, payload, published_at
		FROM events WHERE id = $1
	`, id).Scan(&e.ID, &e.Topic, &e.Partition, &e.Offset, &e.EventType, &idKey, &e.Payload, &e.PublishedAt)
	if err != nil {
		return nil, err
	}
	if idKey != nil {
		e.IdempotencyKey = *idKey
	}
	return &e, nil
}

func (s *PostgresStore) ListOffsets(ctx context.Context, groupID string) ([]models.ConsumerOffset, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT group_id, topic, partition, "offset" FROM consumer_offsets WHERE group_id = $1
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var offsets []models.ConsumerOffset
	for rows.Next() {
		var o models.ConsumerOffset
		if err := rows.Scan(&o.GroupID, &o.Topic, &o.Partition, &o.Offset); err != nil {
			return nil, err
		}
		offsets = append(offsets, o)
	}
	return offsets, rows.Err()
}

func (s *PostgresStore) StoreEvent(ctx context.Context, e models.Event) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO events (id, topic, partition, "offset", event_type, idempotency_key, payload, headers, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (idempotency_key) DO NOTHING
	`, e.ID, e.Topic, e.Partition, e.Offset, e.EventType, nullIfEmpty(e.IdempotencyKey), e.Payload, headersJSON(e.Headers), e.PublishedAt)
	return err
}

func (s *PostgresStore) UpsertOffset(ctx context.Context, o models.ConsumerOffset) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO consumer_offsets (group_id, topic, partition, "offset", updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (group_id, topic, partition)
		DO UPDATE SET "offset" = EXCLUDED."offset", updated_at = NOW()
	`, o.GroupID, o.Topic, o.Partition, o.Offset)
	return err
}

func (s *PostgresStore) GetOffset(ctx context.Context, groupID, topic string, partition int) (int64, error) {
	var offset int64
	err := s.pool.QueryRow(ctx, `
		SELECT "offset" FROM consumer_offsets
		WHERE group_id = $1 AND topic = $2 AND partition = $3
	`, groupID, topic, partition).Scan(&offset)
	return offset, err
}

func (s *PostgresStore) CreateRetry(ctx context.Context, r models.RetryRecord) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO retries (id, event_id, topic, attempt, max_attempts, next_retry_at, last_error, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, r.ID, r.EventID, r.Topic, r.Attempt, r.MaxAttempts, r.NextRetryAt, r.LastError, r.Status, r.CreatedAt, r.UpdatedAt)
	return err
}

func (s *PostgresStore) UpdateRetry(ctx context.Context, r models.RetryRecord) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE retries SET attempt = $2, next_retry_at = $3, last_error = $4, status = $5, updated_at = $6
		WHERE id = $1
	`, r.ID, r.Attempt, r.NextRetryAt, r.LastError, r.Status, r.UpdatedAt)
	return err
}

func (s *PostgresStore) PendingRetries(ctx context.Context, before time.Time) ([]models.RetryRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, event_id, topic, attempt, max_attempts, next_retry_at, last_error, status, created_at, updated_at
		FROM retries WHERE status = 'pending' AND next_retry_at <= $1
		ORDER BY next_retry_at LIMIT 100
	`, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.RetryRecord
	for rows.Next() {
		var r models.RetryRecord
		if err := rows.Scan(&r.ID, &r.EventID, &r.Topic, &r.Attempt, &r.MaxAttempts, &r.NextRetryAt, &r.LastError, &r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *PostgresStore) StoreDLQ(ctx context.Context, m models.DLQMessage) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO dead_letter_messages (id, original_topic, event_id, event_type, payload, failure_reason, retry_attempts, failed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, m.ID, m.OriginalTopic, m.EventID, m.EventType, m.Payload, m.FailureReason, m.RetryAttempts, m.FailedAt)
	return err
}

func (s *PostgresStore) ListDLQ(ctx context.Context, topic string, limit int) ([]models.DLQMessage, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, original_topic, event_id, event_type, payload, failure_reason, retry_attempts, failed_at, replayed_at
		FROM dead_letter_messages WHERE original_topic = $1
		ORDER BY failed_at DESC LIMIT $2
	`, topic, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []models.DLQMessage
	for rows.Next() {
		var m models.DLQMessage
		if err := rows.Scan(&m.ID, &m.OriginalTopic, &m.EventID, &m.EventType, &m.Payload, &m.FailureReason, &m.RetryAttempts, &m.FailedAt, &m.ReplayedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (s *PostgresStore) MarkDLQReplayed(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE dead_letter_messages SET replayed_at = NOW() WHERE id = $1
	`, id)
	return err
}

func (s *PostgresStore) EventsInRange(ctx context.Context, topic string, partition *int, start, end time.Time) ([]models.Event, error) {
	query := `
		SELECT id, topic, partition, "offset", event_type, idempotency_key, payload, published_at
		FROM events WHERE topic = $1 AND published_at >= $2 AND published_at <= $3
	`
	args := []any{topic, start, end}
	if partition != nil {
		query += ` AND partition = $4`
		args = append(args, *partition)
	}
	query += ` ORDER BY published_at, "offset"`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		var key *string
		if err := rows.Scan(&e.ID, &e.Topic, &e.Partition, &e.Offset, &e.EventType, &key, &e.Payload, &e.PublishedAt); err != nil {
			return nil, err
		}
		if key != nil {
			e.IdempotencyKey = *key
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *PostgresStore) CreateWorkflow(ctx context.Context, w models.Workflow) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO workflows (id, name, status, input, current_step, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, w.ID, w.Name, w.Status, w.Input, w.CurrentStep, w.CreatedAt, w.UpdatedAt)
	return err
}

func (s *PostgresStore) UpdateWorkflow(ctx context.Context, w models.Workflow) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE workflows SET status = $2, current_step = $3, updated_at = $4, completed_at = $5
		WHERE id = $1
	`, w.ID, w.Status, w.CurrentStep, w.UpdatedAt, w.CompletedAt)
	return err
}

func (s *PostgresStore) GetWorkflow(ctx context.Context, id string) (*models.Workflow, error) {
	var w models.Workflow
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, status, input, current_step, created_at, updated_at, completed_at
		FROM workflows WHERE id = $1
	`, id).Scan(&w.ID, &w.Name, &w.Status, &w.Input, &w.CurrentStep, &w.CreatedAt, &w.UpdatedAt, &w.CompletedAt)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *PostgresStore) CreateWorkflowStep(ctx context.Context, step models.WorkflowStep) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO workflow_steps (id, workflow_id, name, status, input, output, compensation, attempt, started_at, completed_at, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, step.ID, step.WorkflowID, step.Name, step.Status, step.Input, step.Output, step.Compensation, step.Attempt, step.StartedAt, step.CompletedAt, step.Error)
	return err
}

func (s *PostgresStore) UpdateWorkflowStep(ctx context.Context, step models.WorkflowStep) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE workflow_steps SET status = $3, output = $4, attempt = $5, started_at = $6, completed_at = $7, error = $8
		WHERE id = $1 AND workflow_id = $2
	`, step.ID, step.WorkflowID, step.Status, step.Output, step.Attempt, step.StartedAt, step.CompletedAt, step.Error)
	return err
}

func (s *PostgresStore) ListWorkflowSteps(ctx context.Context, workflowID string) ([]models.WorkflowStep, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workflow_id, name, status, input, output, compensation, attempt, started_at, completed_at, error
		FROM workflow_steps WHERE workflow_id = $1 ORDER BY started_at NULLS LAST
	`, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []models.WorkflowStep
	for rows.Next() {
		var step models.WorkflowStep
		if err := rows.Scan(&step.ID, &step.WorkflowID, &step.Name, &step.Status, &step.Input, &step.Output, &step.Compensation, &step.Attempt, &step.StartedAt, &step.CompletedAt, &step.Error); err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, rows.Err()
}

func (s *PostgresStore) CreateConsumerGroup(ctx context.Context, g models.ConsumerGroup) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO consumer_groups (id, topic, name, members, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, g.ID, g.Topic, g.Name, g.Members, g.CreatedAt)
	return err
}

func NewTopic(name string, partitions, replication, retention int) models.Topic {
	return models.Topic{
		ID:             uuid.New().String(),
		Name:           name,
		Partitions:     partitions,
		Replication:    replication,
		RetentionHours: retention,
		Compression:    "snappy",
		CreatedAt:      time.Now().UTC(),
	}
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func headersJSON(h map[string]string) []byte {
	if h == nil {
		return []byte("{}")
	}
	b, _ := json.Marshal(h)
	return b
}
