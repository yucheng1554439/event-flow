# Changelog

All notable changes to EventFlow are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Production-grade README with demo GIF above the fold, scaling/reliability section, and observability diagram
- Architecture, workflow, retry/DLQ, replay, and observability PNG diagrams
- Demo screenshot assets and `scripts/generate-screenshots.ps1`
- `docs/interview-guide.md` for technical interview preparation
- Reorganized `docs/resume-snippets.md` (Backend, Platform, Infrastructure versions)
- SVG banner and PNG exports for GitHub README

---

## [1.0.0] - 2026-06-10

Portfolio and documentation release — README, architecture diagrams, recruiter materials, and contribution guide.

### Added

- Elite README with architecture overview, quick start, and API examples
- Architecture, workflow, retry/DLQ, and replay Mermaid diagrams (PNG exports)
- Recruiter guide, case study, resume snippets, and readiness checklist
- Demo script documentation, recording guide, and screenshot assets
- `CONTRIBUTING.md` for contributors
- `CHANGELOG.md` with phased release history

[1.0.0]: https://github.com/yucheng1554439/event-flow/releases/tag/v1.0.0

---

## [0.4.0] - 2026-06-09

CI/CD pipeline and integration test hardening.

### Added

- GitHub Actions workflow: lint, unit tests, integration tests, Docker build, Helm validate, Terraform validate
- Shared Testcontainers harness (`tests/integration/setup_test.go`)
- Integration tests: platform, gRPC, failure/retry/DLQ paths
- Kafka test utilities (`pkg/kafka/testutil.go`)
- Terraform provider lock file for reproducible `terraform init`

### Fixed

- Testcontainers `RunContainer` API compatibility (v0.30)
- Terraform HCL formatting and variable block syntax across modules
- Integration test stability: unique DLQ topics, shared container bootstrap, gRPC skip when gateway unavailable

[0.4.0]: https://github.com/yucheng1554439/event-flow/releases/tag/v0.4.0

---

## [0.3.0] - 2026-06-08

Galactic Commerce live demo system.

### Added

- `scripts/demo.ps1` — 9-act live demo with log streaming, workflow polling, DLQ stats, and replay
- Bash demo scripts: `demo.sh`, `demo-common.sh`, `demo-failure.sh`, `demo-replay.sh`
- `cmd/demo-generator` — configurable load and failure injection
- `docker/docker-compose.demo.yml` — demo overlay (`ship-orders`, `galactic-commerce-workers`)
- `GalacticCommerce` saga workflow with failure injection (`demoFailStep`)
- Demo Grafana dashboard (`eventflow-demo.json`)
- Screenshot capture guide (`scripts/capture-demo-screenshots.ps1`)

### Kafka Topics (Demo)

- `ship-orders` with paired `ship-orders-dlq`
- Consumer group `galactic-commerce-workers`

[0.3.0]: https://github.com/yucheng1554439/event-flow/releases/tag/v0.3.0

---

## [0.2.0] - 2026-06-07

Phase 2 — production hardening: gRPC, topic administration, reliability, cloud deploy, and observability.

### Added

#### gRPC & Topic Administration

- Protobuf definitions and generated stubs (`api/proto/`, `api/gen/go/`)
- gRPC services: Topic, Event, Replay, Workflow on port `:9090`
- Dynamic topic CRUD via REST and gRPC (`internal/topic`)
- Kafka AdminClient wrapper (`pkg/kafka/admin.go`)
- Migration `002_add_cleanup_policy.sql`

#### Retry, DLQ & Replay

- Exponential backoff retry engine (`internal/retry`)
- DLQ convention: `{topic}-dlq` + PostgreSQL `dead_letter_messages`
- Replay service: time-range, partition, and DLQ-only replay (`internal/replay`)
- REST: `GET /api/v1/dlq/:topic`, `GET /api/v1/dlq/:topic/stats`, `POST /api/v1/dlq/:topic/replay`
- REST: `GET /api/v1/retries`, `POST /api/v1/replay`

#### Helm & Terraform

- Helm chart at `helm/eventflow/` (api-gateway, consumer-worker, workflow-engine, postgres, redis)
- AWS modules: VPC, EKS, MSK, RDS, ElastiCache
- Dev environment at `terraform/environments/dev/`

#### Observability

- Prometheus metrics (`pkg/metrics`) — events, lag, workflows, DLQ, retries
- Grafana operations dashboard (`eventflow.json`)
- Prometheus scrape config and Grafana provisioning

#### Testing

- Load benchmarks (`tests/load/benchmark_test.go`)
- Proto generation script (`scripts/generate-proto.ps1`)

[0.2.0]: https://github.com/yucheng1554439/event-flow/releases/tag/v0.2.0

---

## [0.1.0] - 2026-06-06

Phase 1 — core distributed event platform foundation.

### Added

- Monorepo layout: `cmd/`, `internal/`, `pkg/`, `api/`, `deployments/`, `docker/`
- REST API gateway (Gin) on `:8080` — events, workflows, consumer groups
- Kafka producer and consumer wrappers with manual offset commits
- Consumer worker with `eventflow-workers` group
- Workflow engine with saga compensation (`OrderFulfillment`)
- PostgreSQL schema (`migrations/001_initial_schema.sql`) — events, workflows, offsets, retries, DLQ
- Redis distributed locks and idempotency support
- Docker Compose local stack (Zookeeper, Kafka, PostgreSQL, Redis, three services)
- Kubernetes base manifests (`deployments/k8s/`)
- OpenAPI specification (`api/openapi/eventflow.yaml`)
- Topic seeding and sample publish scripts

### Kafka Topics (Default)

- `orders`, `payments`, `notifications`, `analytics` with paired DLQ topics

[0.1.0]: https://github.com/yucheng1554439/event-flow/releases/tag/v0.1.0
