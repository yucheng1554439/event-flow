# EventFlow Interview Guide

A structured prep guide for technical interviews — map EventFlow components to common distributed-systems questions.

---

## 30-Second Pitch

> EventFlow is a Go event platform on Apache Kafka. Producers publish through a REST/gRPC gateway; consumer workers process with at-least-once semantics; a workflow engine runs compensating sagas; a retry engine applies exponential backoff and routes poison messages to DLQ topics; operators replay from DLQ or time ranges. State lives in PostgreSQL and Redis; Prometheus and Grafana provide observability. It deploys locally via Docker Compose, to Kubernetes via Helm, and to AWS via Terraform.

---

## Architecture Talking Points

| Component | Interview angle |
|-----------|-----------------|
| **API Gateway** | Single ingress, idempotency keys, topic admin, replay orchestration |
| **Kafka** | Durable log, partition ordering, consumer groups, DLQ topic pairing |
| **Consumer Worker** | Offset commits, handler failures → retry engine |
| **Workflow Engine** | Saga steps, LIFO compensation, Redis distributed locks |
| **Retry Engine** | Exponential backoff, max attempts, DLQ insertion |
| **Replay Service** | Operator-driven recovery without re-deploying code |
| **PostgreSQL** | ACID workflow/retry/DLQ records, audit trail |
| **Redis** | Idempotency dedup (24h TTL), workflow execution locks |

**Diagram to reference:** [architecture.png](diagrams/architecture.png)

---

## Deep-Dive Topics

### 1. Why Kafka instead of a database queue?

- **Durability & replay:** Kafka retains ordered logs; replay is a first-class operation.
- **Throughput:** Partition-level parallelism scales with consumer groups.
- **Decoupling:** Producers and consumers evolve independently.
- **Trade-off:** Operational complexity (brokers, rebalancing) vs. simplicity of Postgres `LISTEN/NOTIFY`.

### 2. Delivery semantics

EventFlow implements **at-least-once**:

1. Producer acks to Kafka before returning success.
2. Consumer commits offset only after handler success (or DLQ routing).
3. **Idempotency keys** in Redis prevent duplicate side effects on redelivery.

**Follow-up:** "How would you move toward exactly-once?" — transactional outbox, idempotent consumers everywhere, or Kafka transactions (higher complexity).

### 3. Saga vs two-phase commit (2PC)

GalacticCommerce saga: `ProcessPayment` → `ReserveInventory` → `SendConfirmation`.

On `ReserveInventory` failure, **RefundPayment** runs (LIFO compensation).

- **Why not 2PC?** 2PC blocks on coordinator failure; sagas favor availability.
- **Trade-off:** Compensation is application-defined; not all steps are truly reversible.

### 4. Retry vs DLQ

| Signal | Action |
|--------|--------|
| Transient error (503, timeout) | Schedule retry with exponential backoff |
| Max attempts exceeded | Insert into `{topic}-dlq` + PostgreSQL `dead_letter_messages` |
| Poison message (bad schema) | DLQ immediately after validation failure |

**Operator APIs:** `GET /api/v1/retries`, `GET /api/v1/dlq/:topic/stats`, `POST /api/v1/replay`.

### 5. Consumer groups & scaling

- Each partition → one consumer in the group.
- Add consumers up to partition count; beyond that, consumers idle.
- Rebalance on member join/leave — discuss stop-the-world vs cooperative sticky assignors.

### 6. Observability

Key metrics to mention in interviews:

- `eventflow_consumer_lag` — backpressure signal
- `eventflow_dlq_messages_total` — poison / sustained failure rate
- `eventflow_workflow_duration_seconds` — saga latency SLO proxy
- `eventflow_retry_attempts_total` — transient failure health

**Diagram:** [observability.png](diagrams/observability.png)

---

## System Design Exercise: "Build an order pipeline"

Use EventFlow as your reference answer skeleton:

1. **Ingress:** REST publish `OrderPlaced` to `orders` topic (12 partitions).
2. **Processing:** Consumer group `order-workers` triggers `OrderFulfillment` saga.
3. **Failure:** Retry 3× with backoff → `orders-dlq`.
4. **Recovery:** Operator replays DLQ after fixing inventory service.
5. **Scale:** Horizontal consumer replicas; partition count set at topic creation.
6. **Observability:** Alert on lag > N for 5m; dashboard for DLQ depth.

---

## Behavioral / Ownership Questions

| Question | EventFlow evidence |
|----------|-------------------|
| "Tell me about a complex system you built" | End-to-end platform: APIs, Kafka, workflows, IaC, CI |
| "How do you handle production incidents?" | DLQ inspection, replay API, Grafana dashboards |
| "How do you test distributed systems?" | Testcontainers integration suite against real Kafka/Postgres |
| "How do you document for others?" | README, case study, recruiter guide, live demo script |

---

## Common Follow-Ups & Answers

**Q: What would you add for real production?**
Auth (mTLS/OAuth2), multi-tenancy, SLO-based alerting, schema registry (Avro/Protobuf evolution), rate limiting, and chaos testing.

**Q: How do you choose partition count?**
Target throughput ÷ per-partition throughput; leave headroom; hard to decrease later.

**Q: Why PostgreSQL AND Redis?**
Postgres for durable queryable state; Redis for fast idempotency and short-lived locks.

**Q: gRPC vs REST?**
REST for human/operators and quick curls; gRPC for service-to-service efficiency and strong typing.

---

## Whiteboard Flows to Practice

1. Publish → consume → retry → DLQ → replay (5 boxes, 2 minutes)
2. Saga happy path + compensation path (state machine)
3. Consumer group rebalance when a pod dies

---

## Related Materials

- [Case Study](case-study.md) — design depth
- [Recruiter Guide](recruiter-guide.md) — non-specialist summary
- [Resume Snippets](resume-snippets.md) — quantified bullets
- [Demo Script](demo/demo-script.md) — live walkthrough
