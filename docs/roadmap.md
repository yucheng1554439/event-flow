# EventFlow Implementation Roadmap

## Phase 1 — Foundation (Weeks 1–4) ✅ Scaffolded

- [x] Monorepo structure (`cmd`, `internal`, `pkg`, `api`, `deployments`, `terraform`, `docker`)
- [x] PostgreSQL schema with indexes
- [x] REST API (Gin) + gRPC proto definitions
- [x] Kafka producer/consumer wrappers (idempotent producer, manual commit consumer)
- [x] Retry engine with exponential backoff
- [x] DLQ storage and replay API
- [x] Workflow engine with saga compensation
- [x] Prometheus metrics + Grafana dashboard
- [x] Docker Compose local stack
- [x] Kubernetes base manifests
- [x] Terraform modules (VPC, EKS, MSK, RDS, ElastiCache)

## Phase 2 — Production Hardening (Weeks 5–8) ✅

- [x] Full gRPC codegen — TopicService, EventService, ReplayService, WorkflowService
- [x] Kafka topic auto-provisioning via AdminClient (`internal/topic`)
- [x] Topic DELETE API + cleanup policy configuration
- [x] Integration tests with Testcontainers (Kafka, PostgreSQL, Redis)
- [x] CI/CD pipeline (lint, unit, integration, Docker, Helm, Terraform)
- [x] Helm chart (`helm/eventflow`)
- [ ] Consumer group rebalancing metrics and lag exporter
- [ ] Idempotent consumer handlers (dedup table)
- [ ] Workflow scheduler (poll `pending` workflows)

## Phase 3 — Advanced Features (Weeks 9–12)

- [ ] Transactional outbox for exactly-once publish
- [ ] Event sourcing read models
- [ ] Airflow-inspired DAG workflows (parallel steps, cron triggers)
- [ ] Multi-tenant topic isolation
- [ ] Schema registry (Avro/Protobuf) integration
- [ ] Cross-region replication
- [ ] Operator CLI (`eventflowctl`)

## Phase 4 — Enterprise (Weeks 13+)

- [ ] RBAC and API key management
- [ ] Audit logging
- [ ] SLA-based alerting (PagerDuty)
- [ ] Cost attribution per topic/tenant
- [ ] Chaos engineering test suite

## Interview Discussion Points

1. **Why Kafka over SQS alone?** Ordered partitions, replay, high throughput, consumer groups — SQS lacks native replay and ordering at scale.
2. **At-least-once vs exactly-once:** EventFlow defaults to at-least-once with idempotency keys — pragmatic for most systems; exactly-once requires transactional outbox.
3. **Saga vs 2PC:** Choreography via events is decoupled but hard to debug; orchestration (EventFlow workflow engine) gives visibility and compensation control.
4. **DLQ design:** Per-topic DLQ (`orders-dlq`) isolates failures; 90-day retention supports forensic analysis.
5. **Partition count:** Too few = bottleneck; too many = overhead. Start with 6–12, scale with throughput metrics.
6. **Consumer lag:** Primary scaling signal; alert when lag > threshold for > 5 minutes.
7. **Replay safety:** Replay events carry `replay: true` header; consumers must handle idempotently.
8. **Workflow locks:** Redis `SETNX` prevents duplicate execution; TTL prevents deadlocks on worker crash.
