# EventFlow System Design

## Consistency Model

| Component | Consistency | Notes |
|-----------|-------------|-------|
| Kafka | Per-partition ordering | Total order only within partition |
| PostgreSQL | Strong (ACID) | Workflow state, offsets, retries |
| Redis | Eventual | Idempotency cache, queue depth, locks |
| Cross-service | Eventual | Outbox pattern recommended for Phase 3 |

## Throughput Optimization

1. **Producer batching:** `linger.ms=10`, `batch.size=65536`, Snappy compression.
2. **Idempotent producer:** `enable.idempotence=true`, `acks=all`.
3. **Consumer parallelism:** One consumer instance per partition (max).
4. **Connection pooling:** pgxpool for PostgreSQL, shared Kafka producer per gateway instance.
5. **Analytics topic:** 24 partitions for maximum write fan-out.

## Replay Mechanisms

### By Time Range
```sql
SELECT * FROM events
WHERE topic = 'orders'
  AND published_at BETWEEN '2026-01-01' AND '2026-01-30'
ORDER BY published_at, offset;
```
Events re-published to target topic with `replay: true` header.

### By Partition
Filter `partition = N` for surgical replay after consumer bug fix.

### DLQ Replay
Unreplayed messages from `dead_letter_messages` re-published; `replayed_at` timestamp set.

## Saga Pattern

EventFlow implements **orchestration-based sagas**:

- Central workflow engine coordinates steps.
- Each step has an optional `Compensation` function.
- On failure, compensations run in reverse order (LIFO).
- State machine: `pending → running → completed | compensating → failed`.

Example compensation chain for failed `ReserveInventory`:
1. Compensate `ProcessPayment` → refund
2. Mark workflow `failed`

## Distributed Workflow Execution

- Workflows created via API, executed asynchronously.
- Redis lock (`workflow_lock:{id}`) ensures single executor.
- Steps persisted before and after execution for crash recovery.
- On restart: query `workflows WHERE status = 'running'` and resume from `current_step`.

## Database Scaling Considerations

```sql
-- Future: partition events table by month
CREATE TABLE events_2026_01 PARTITION OF events
  FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
```

- **Indexes:** `(topic, published_at)` for replay; `(status, next_retry_at)` partial index for retry poller.
- **Archival:** Move events older than retention to S3 via pg_cron.
- **Read replicas:** Route replay queries to replica; writes to primary.

## Network Partition Behavior

- **Kafka:** Minority partition consumers lose leadership; rebalance occurs.
- **Consumers:** `session.timeout.ms=10000` — if heartbeat fails, partition reassigned.
- **No event loss:** Uncommitted messages remain in Kafka; redelivered after recovery.
- **Workflows:** Lock TTL expires; another worker can resume from persisted state.

## gRPC vs REST

Both ingress paths share the same service layer:

- REST: Gin handlers in `internal/api` and `internal/topic`
- gRPC: Generated servers in `internal/grpc` calling identical topic, replay, workflow, and storage code

Idempotency, metrics, and persistence behave identically across transports.

## Integration Test Design

Testcontainers spin up isolated Kafka (Confluent Local 7.5), PostgreSQL 16, and Redis 7 per test package. Migrations `001` + `002` applied before tests. Topics use unique names to avoid cross-test pollution. Replication factor is `1` for single-broker test Kafka.

## Helm Topology

```
helm/eventflow
├── api-gateway (Deployment + Service :80/:9090)
├── consumer-worker (Deployment, scales to partition count)
├── workflow-engine (Deployment)
├── postgres (StatefulSet, optional)
└── redis (Deployment, optional)
```

External Kafka (MSK) is configured via `kafkaBootstrapServers` value; embedded Postgres/Redis suitable for dev clusters only.
