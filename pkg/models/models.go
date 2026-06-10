package models

import (
	"encoding/json"
	"time"
)

type Topic struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Partitions        int       `json:"partitions"`
	Replication       int       `json:"replication"`
	ReplicationFactor int       `json:"replicationFactor,omitempty"`
	RetentionHours    int       `json:"retentionHours"`
	CleanupPolicy     string    `json:"cleanupPolicy,omitempty"`
	Compression       string    `json:"compression"`
	CreatedAt         time.Time `json:"createdAt"`
}

type CreateTopicRequest struct {
	Name              string `json:"name"`
	Partitions        int    `json:"partitions"`
	Replication       int    `json:"replication"`
	ReplicationFactor int    `json:"replicationFactor"`
	RetentionHours    int    `json:"retentionHours"`
	CleanupPolicy     string `json:"cleanupPolicy"`
}

type Event struct {
	ID             string          `json:"id"`
	Topic          string          `json:"topic"`
	Partition      int             `json:"partition"`
	Offset         int64           `json:"offset"`
	EventType      string          `json:"eventType"`
	IdempotencyKey string          `json:"idempotencyKey,omitempty"`
	Payload        json.RawMessage `json:"payload"`
	Headers        map[string]string `json:"headers,omitempty"`
	PublishedAt    time.Time       `json:"publishedAt"`
}

type PublishRequest struct {
	Topic          string          `json:"topic"`
	EventType      string          `json:"eventType"`
	IdempotencyKey string          `json:"idempotencyKey,omitempty"`
	Payload        json.RawMessage `json:"payload"`
	Headers        map[string]string `json:"headers,omitempty"`
}

type BatchPublishRequest struct {
	Events []PublishRequest `json:"events"`
}

type ConsumerGroup struct {
	ID        string    `json:"id"`
	Topic     string    `json:"topic"`
	Name      string    `json:"name"`
	Members   int       `json:"members"`
	CreatedAt time.Time `json:"createdAt"`
}

type ConsumerOffset struct {
	GroupID   string `json:"groupId"`
	Topic     string `json:"topic"`
	Partition int    `json:"partition"`
	Offset    int64  `json:"offset"`
}

type RetryRecord struct {
	ID            string    `json:"id"`
	EventID       string    `json:"eventId"`
	Topic         string    `json:"topic"`
	Attempt       int       `json:"attempt"`
	MaxAttempts   int       `json:"maxAttempts"`
	NextRetryAt   time.Time `json:"nextRetryAt"`
	LastError     string    `json:"lastError"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type DLQMessage struct {
	ID            string          `json:"id"`
	OriginalTopic string          `json:"originalTopic"`
	EventID       string          `json:"eventId"`
	EventType     string          `json:"eventType"`
	Payload       json.RawMessage `json:"payload"`
	FailureReason string          `json:"failureReason"`
	RetryAttempts int             `json:"retryAttempts"`
	FailedAt      time.Time       `json:"failedAt"`
	ReplayedAt    *time.Time      `json:"replayedAt,omitempty"`
}

type ReplayRequest struct {
	Topic      string     `json:"topic"`
	Partition  *int       `json:"partition,omitempty"`
	StartTime  *time.Time `json:"startTime,omitempty"`
	EndTime    *time.Time `json:"endTime,omitempty"`
	DLQOnly    bool       `json:"dlqOnly"`
	TargetTopic string    `json:"targetTopic,omitempty"`
}

type WorkflowStatus string

const (
	WorkflowPending    WorkflowStatus = "pending"
	WorkflowRunning    WorkflowStatus = "running"
	WorkflowCompleted  WorkflowStatus = "completed"
	WorkflowFailed     WorkflowStatus = "failed"
	WorkflowCompensating WorkflowStatus = "compensating"
	WorkflowCancelled  WorkflowStatus = "cancelled"
)

type Workflow struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Status      WorkflowStatus `json:"status"`
	Input       json.RawMessage `json:"input"`
	CurrentStep string         `json:"currentStep"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	CompletedAt *time.Time     `json:"completedAt,omitempty"`
}

type WorkflowStep struct {
	ID           string          `json:"id"`
	WorkflowID   string          `json:"workflowId"`
	Name         string          `json:"name"`
	Status       string          `json:"status"`
	Input        json.RawMessage `json:"input"`
	Output       json.RawMessage `json:"output,omitempty"`
	Compensation string          `json:"compensation,omitempty"`
	Attempt      int             `json:"attempt"`
	StartedAt    *time.Time      `json:"startedAt,omitempty"`
	CompletedAt  *time.Time      `json:"completedAt,omitempty"`
	Error        string          `json:"error,omitempty"`
}

type CreateWorkflowRequest struct {
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}
