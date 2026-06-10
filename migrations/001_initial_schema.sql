-- EventFlow PostgreSQL Schema
-- Supports events, workflows, retries, DLQ, and consumer offsets

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Topics (metadata; Kafka holds actual topic data)
CREATE TABLE topics (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL UNIQUE,
    partitions      INT NOT NULL DEFAULT 6 CHECK (partitions > 0),
    replication     INT NOT NULL DEFAULT 3 CHECK (replication > 0),
    retention_hours INT NOT NULL DEFAULT 168,
    compression     VARCHAR(32) NOT NULL DEFAULT 'snappy',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_topics_name ON topics (name);

-- Event audit log (complements Kafka for replay and inspection)
CREATE TABLE events (
    id              UUID PRIMARY KEY,
    topic           VARCHAR(255) NOT NULL,
    partition       INT NOT NULL,
    "offset"        BIGINT NOT NULL,
    event_type      VARCHAR(255) NOT NULL,
    idempotency_key VARCHAR(255) UNIQUE,
    payload         JSONB NOT NULL,
    headers         JSONB DEFAULT '{}',
    published_at    TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_topic_time ON events (topic, published_at);
CREATE INDEX idx_events_topic_partition ON events (topic, partition, "offset");
CREATE INDEX idx_events_type ON events (event_type);
CREATE INDEX idx_events_published_at ON events (published_at DESC);

-- Consumer groups
CREATE TABLE consumer_groups (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    topic       VARCHAR(255) NOT NULL,
    name        VARCHAR(255) NOT NULL UNIQUE,
    members     INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_consumer_groups_topic ON consumer_groups (topic);

-- Consumer offset tracking (at-least-once delivery)
CREATE TABLE consumer_offsets (
    group_id    VARCHAR(255) NOT NULL,
    topic       VARCHAR(255) NOT NULL,
    partition   INT NOT NULL,
    "offset"    BIGINT NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, topic, partition)
);

CREATE INDEX idx_consumer_offsets_group ON consumer_offsets (group_id);

-- Retry metadata
CREATE TABLE retries (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id        UUID NOT NULL,
    topic           VARCHAR(255) NOT NULL,
    attempt         INT NOT NULL DEFAULT 1,
    max_attempts    INT NOT NULL DEFAULT 5,
    next_retry_at   TIMESTAMPTZ NOT NULL,
    last_error      TEXT,
    status          VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_retries_pending ON retries (status, next_retry_at) WHERE status = 'pending';
CREATE INDEX idx_retries_event ON retries (event_id);

-- Dead letter queue
CREATE TABLE dead_letter_messages (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    original_topic  VARCHAR(255) NOT NULL,
    event_id        UUID NOT NULL,
    event_type      VARCHAR(255) NOT NULL,
    payload         JSONB NOT NULL,
    failure_reason  TEXT NOT NULL,
    retry_attempts  INT NOT NULL DEFAULT 0,
    failed_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    replayed_at     TIMESTAMPTZ
);

CREATE INDEX idx_dlq_topic ON dead_letter_messages (original_topic, failed_at DESC);
CREATE INDEX idx_dlq_unreplayed ON dead_letter_messages (original_topic) WHERE replayed_at IS NULL;

-- Workflows
CREATE TABLE workflows (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name          VARCHAR(255) NOT NULL,
    status        VARCHAR(32) NOT NULL DEFAULT 'pending',
    input         JSONB NOT NULL,
    current_step  VARCHAR(255),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ
);

CREATE INDEX idx_workflows_status ON workflows (status);
CREATE INDEX idx_workflows_name ON workflows (name, created_at DESC);

-- Workflow steps
CREATE TABLE workflow_steps (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id   UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name          VARCHAR(255) NOT NULL,
    status        VARCHAR(32) NOT NULL DEFAULT 'pending',
    input         JSONB,
    output        JSONB,
    compensation  VARCHAR(255),
    attempt       INT NOT NULL DEFAULT 1,
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    error         TEXT
);

CREATE INDEX idx_workflow_steps_workflow ON workflow_steps (workflow_id, started_at);

-- Seed default topics
INSERT INTO topics (name, partitions, replication, retention_hours) VALUES
    ('orders', 12, 3, 168),
    ('payments', 6, 3, 720),
    ('notifications', 3, 3, 72),
    ('analytics', 24, 3, 336),
    ('orders-dlq', 3, 3, 2160),
    ('payments-dlq', 3, 3, 2160),
    ('notifications-dlq', 1, 3, 2160),
    ('analytics-dlq', 3, 3, 2160)
ON CONFLICT (name) DO NOTHING;
