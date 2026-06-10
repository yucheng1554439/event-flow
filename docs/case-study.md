# EventFlow: A Distributed Event Processing Platform

*A system design case study — v1.0.0*

---

## Problem

Modern services communicate asynchronously. A single user action fans out into payment processing, inventory updates, notifications, and analytics. Failures are inevitable: networks blip, databases time out, downstream APIs return 503.

Without a dedicated event platform, teams reinvent the same primitives in every service:

- Retry loops with inconsistent backoff
- Ad-hoc "error tables" instead of dead-letter queues
- Brittle multi-step scripts instead of compensating workflows
- No unified way to replay traffic after incidents

**EventFlow** consolidates these primitives into one platform: publish events, consume reliably, orchestrate multi-step workflows, retry transient failures, quarantine poison messages, and replay on demand.

---

## Architecture

EventFlow follows a **hub-and-spoke** model centered on Apache Kafka:

```
Producers → API Gateway → Kafka → Consumer Workers → Side effects
                ↓                        ↓
           PostgreSQL              Retry Engine → DLQ
                ↓
         Workflow Engine (sagas)
                ↓
         Prometheus → Grafana
```

The **API Gateway** is the single ingress for REST and gRPC. It publishes to Kafka, manages topic metadata, exposes replay and workflow APIs, and records events in PostgreSQL for audit and replay.

**Consumer Workers** are horizontally scalable Kafka consumers. On failure, they delegate to the **Retry Engine**. On success, they may trigger workflows (e.g., `ShipPurchased` → `GalacticCommerce`).

The **Workflow Engine** runs saga-style multi-step processes with persistent state and Redis-backed distributed locks.

See the full diagram: [diagrams/architecture.png](diagrams/architecture.png).

---

## Kafka Design

### Topic strategy

Each business domain gets a dedicated topic with partition count tuned to expected throughput:

| Topic | Partitions | Rationale |
|-------|----------:|-----------|
| orders | 12 | High volume, parallel consumers |
| analytics | 24 | Write-heavy, embarrassingly parallel |
| notifications | 3 | Lower volume, latency-sensitive |

### DLQ convention

Every source topic has a paired `{topic}-dlq` topic. This mirrors AWS SQS DLQ patterns and keeps poison messages out of the hot path while preserving them for inspection.

### Producer guarantees

- **At-least-once delivery** to Kafka (producer acks)
- **Idempotency keys** deduplicated via Redis (24h TTL)
- **Snappy compression** for payload efficiency
- **Batch publish API** for throughput

### Partition ordering

Ordering is guaranteed **per partition key**. EventFlow uses idempotency keys and business IDs as Kafka message keys where ordering matters.

---

## Consumer Groups

Consumer groups provide **horizontal scale** and **fault tolerance**:

- Each partition is assigned to exactly one consumer in the group
- On consumer failure, Kafka rebalances partitions to surviving members
- Offsets are committed to PostgreSQL only after successful handler execution

This gives at-least-once semantics: a crash between processing and commit may redeliver the message, which is why idempotency and retry tracking matter.

The demo consumer group `galactic-commerce-workers` processes `ship-orders` and auto-triggers the GalacticCommerce workflow on success.

---

## Retry Strategy

Transient failures (network timeouts, 503s, simulated demo failures) should not immediately land in the DLQ.

EventFlow implements **exponential backoff**:

```
delay = initial_backoff × multiplier^attempt  (capped at max_backoff)
```

Default policy: 3 max attempts, 2s initial backoff (demo mode).

### Flow

1. Handler fails → record retry in PostgreSQL
2. Retry poller processes pending records when `next_retry_at` elapses
3. If attempts exceed max → route to DLQ

Retry metadata is queryable via `GET /api/v1/retries?topic=&eventId=`.

See: [diagrams/retry-dlq.png](diagrams/retry-dlq.png).

---

## DLQ Design

Dead-letter messages are stored in **two places**:

1. **PostgreSQL** `dead_letter_messages` — queryable, supports `replayed_at` tracking
2. **Kafka** `{topic}-dlq` — streaming access for downstream tooling

Each DLQ record captures:

- Original `event_id` and `event_type`
- Full payload (for replay fidelity)
- `failure_reason` and `retry_attempts`
- `failed_at` / `replayed_at` timestamps

Stats API: `GET /api/v1/dlq/:topic/stats` returns `{ total, unreplayed }`.

---

## Workflow Engine

Workflows are **state machines** with explicit steps and compensations.

### GalacticCommerce saga

```
ProcessPayment → ReserveInventory → SendConfirmation
     ↓ compensate: RefundPayment
```

On step failure, completed steps are compensated in **LIFO order** (reverse execution order).

### State persistence

- `workflows` table: status, current step, input/output
- `workflow_steps` table: per-step status, timing, errors
- Redis lock: prevents duplicate concurrent runs of the same workflow ID

### API

```bash
POST /api/v1/workflows        # Create (status: pending)
POST /api/v1/workflows/:id/run  # Execute async
GET  /api/v1/workflows/:id      # Poll status + steps
```

See: [diagrams/workflow-engine.png](diagrams/workflow-engine.png).

---

## Replay Design

Replay is an **operator-facing** capability for incident recovery and testing.

### Modes

| Mode | Use case |
|------|----------|
| DLQ-only | Reprocess quarantined messages |
| Time range | Replay historical window |
| Partition | Targeted partition replay |

### Safety

- DLQ records marked `replayed_at` to prevent double-replay
- Replay headers (`replay: true`, `dlq_id`) tag reprocessed messages
- Republish targets configurable via `targetTopic`

See: [diagrams/event-replay.png](diagrams/event-replay.png).

---

## Scaling Strategy

| Component | Scale approach |
|-----------|---------------|
| API Gateway | Horizontal pods behind load balancer (stateless) |
| Consumer Workers | Add replicas within consumer group |
| Kafka | Increase partitions (requires planning) |
| PostgreSQL | Read replicas for inspection APIs; connection pooling |
| Redis | Cluster mode for lock throughput |
| Workflow Engine | Horizontal with distributed locks |

Terraform modules provision **AWS MSK** (Kafka), **EKS** (compute), **RDS** (Postgres), and **ElastiCache** (Redis) for cloud deployments.

---

## Tradeoffs

| Decision | Benefit | Cost |
|----------|---------|------|
| Kafka over SQS | Ordering, replay, throughput | Operational complexity |
| PostgreSQL for state | ACID, queryable DLQ/retries | Write latency vs pure KV |
| At-least-once delivery | Simpler than exactly-once | Requires idempotency |
| Embedded retry/replay | Fewer services to deploy | Shared fate with gateway |
| Saga compensation vs 2PC | Availability, partition tolerance | Eventual consistency |

---

## Lessons Learned

1. **Make failures visible early.** Retry and DLQ metrics in Grafana caught demo issues before they became silent data loss.
2. **Context propagation matters.** Workflow goroutines must not inherit cancelled HTTP request contexts.
3. **Count attempts, not rows.** DLQ routing requires tracking max retry attempt, not retry table row count.
4. **Demo mode is documentation.** A live scripted demo (`demo.ps1`) communicates the platform faster than architecture slides.
5. **Testcontainers pays off.** Integration tests against real Kafka and Postgres caught schema and serialization bugs unit tests missed.

---

*EventFlow v1.0.0 — MIT License*
