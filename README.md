<p align="center">
  <img src="docs/assets/eventflow-banner.png" alt="EventFlow" width="720" />
</p>

<h1 align="center">EventFlow</h1>

<p align="center">
  <strong>Production-grade distributed event processing — Kafka ingestion, saga workflows, retries, DLQ, replay, and cloud-native observability.</strong>
</p>

<p align="center">
  <a href="https://github.com/yucheng1554439/event-flow/actions"><img src="https://img.shields.io/github/actions/workflow/status/yucheng1554439/event-flow/ci.yml?branch=main&style=flat&label=CI" alt="CI" /></a>
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/Apache%20Kafka-7.6-231F20?style=flat&logo=apachekafka&logoColor=white" alt="Kafka" />
  <img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat&logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis&logoColor=white" alt="Redis" />
  <img src="https://img.shields.io/badge/Kubernetes-Helm-326CE5?style=flat&logo=kubernetes&logoColor=white" alt="Kubernetes" />
  <img src="https://img.shields.io/badge/Terraform-AWS-844FBA?style=flat&logo=terraform&logoColor=white" alt="Terraform" />
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat" alt="MIT" />
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> ·
  <a href="#live-demo">Live Demo</a> ·
  <a href="#architecture">Architecture</a> ·
  <a href="docs/case-study.md">Case Study</a> ·
  <a href="docs/recruiter-guide.md">Recruiter Guide</a>
</p>

---

## Live Demo

<p align="center">
  <img src="docs/assets/demo.gif" alt="EventFlow live demo — publish, saga, retry, DLQ, replay" width="720" />
</p>

<p align="center">
  <em>ShipPurchased → GalacticCommerce saga → retry → DLQ → replay → Grafana metrics — ~80 seconds, real APIs.</em>
</p>

```powershell
# Full stack + 9-act demo
.\scripts\demo.ps1

# Re-run when stack is already up
.\scripts\demo.ps1 -SkipStackStart
```

| Resource | Link |
|----------|------|
| Demo script | [docs/demo/demo-script.md](docs/demo/demo-script.md) |
| Video guide | [docs/demo/video.md](docs/demo/video.md) |
| Record a GIF | [docs/assets/generate-gifs.md](docs/assets/generate-gifs.md) |

---

## Why EventFlow?

Modern services emit millions of events per day. A single purchase fans out to payment, inventory, and notifications — and failures are inevitable.

EventFlow is the **infrastructure layer** that makes async processing reliable:

| Capability | What operators get |
|------------|-------------------|
| **Durable ingestion** | Kafka topics with idempotent publish APIs |
| **Saga orchestration** | Multi-step workflows with LIFO compensation |
| **Transient failure handling** | Exponential backoff retry engine |
| **Poison message quarantine** | `{topic}-dlq` topics + inspection APIs |
| **Incident recovery** | DLQ and time-range replay without redeploys |
| **Production visibility** | Prometheus metrics + Grafana dashboards |

Built in **Go** with **REST and gRPC**, deployable via **Docker Compose**, **Helm**, and **Terraform (AWS)**.

---

## Architecture

<p align="center">
  <img src="docs/diagrams/architecture.png" alt="EventFlow architecture" width="820" />
</p>

```
Clients ──► API Gateway (REST :8080 / gRPC :9090)
                │
                ├──► Kafka (ship-orders, orders, payments, …)
                │         └──► Consumer Workers ──► Retry Engine ──► DLQ
                ├──► Workflow Engine (sagas + compensation)
                ├──► Replay Service (DLQ / time-range)
                ├──► PostgreSQL (durable state) + Redis (locks, idempotency)
                └──► Prometheus :9091 ──► Grafana :3000
```

| Layer | Components |
|-------|------------|
| **Ingress** | API Gateway — REST `:8080`, gRPC `:9090` |
| **Messaging** | Apache Kafka — partitioned topics + paired DLQ topics |
| **Processing** | Consumer Workers — consumer groups, manual offset commits |
| **Orchestration** | Workflow Engine — saga steps + LIFO compensation |
| **Reliability** | Retry Engine — exponential backoff → DLQ routing |
| **Recovery** | Replay Service — DLQ-only, time-range, partition replay |
| **State** | PostgreSQL (durable) + Redis (locks, idempotency) |
| **Observability** | Prometheus + Grafana dashboards |

<details>
<summary>Additional diagrams</summary>

| Diagram | Description |
|---------|-------------|
| [Workflow Engine](docs/diagrams/workflow-engine.png) | Saga state machine |
| [Retry & DLQ](docs/diagrams/retry-dlq.png) | Failure handling flow |
| [Event Replay](docs/diagrams/event-replay.png) | Replay sequence |
| [Observability](docs/diagrams/observability.png) | Metrics → dashboards pipeline |

Re-render: `.\scripts\render-diagrams.ps1`

</details>

---

## Core Features

| Feature | Description |
|---------|-------------|
| **Topic Administration** | Create, list, delete Kafka topics via REST/gRPC |
| **Event Publishing** | Single + batch publish, idempotency keys, Snappy compression |
| **Consumer Groups** | Partition assignment, offset commits, at-least-once delivery |
| **Workflow Sagas** | Multi-step processes with compensating transactions |
| **Retry Engine** | Exponential backoff, configurable max attempts |
| **Dead Letter Queue** | `{topic}-dlq` with stats and inspection APIs |
| **Event Replay** | DLQ-only, time-range, and partition replay |
| **Observability** | 15+ Prometheus metrics, 2 Grafana dashboards |
| **Cloud Deploy** | Helm chart + Terraform (EKS, MSK, RDS, ElastiCache) |

---

## Quick Start

### Prerequisites

- Docker Desktop (or Docker Engine + Compose v2)
- Go 1.22+ (for local builds)
- Make (optional)

### Start the platform

```bash
# Start Kafka, PostgreSQL, Redis, all services, Prometheus, Grafana
make docker-up

# Or with Galactic Commerce demo overlay
docker compose -f docker/docker-compose.yml -f docker/docker-compose.demo.yml up -d --wait

# Verify
curl http://localhost:8080/healthz
```

### Service endpoints

| Service | URL | Purpose |
|---------|-----|---------|
| REST API | http://localhost:8080 | Topics, events, workflows, replay |
| gRPC | localhost:9090 | Same operations via gRPC |
| Workflow Engine | http://localhost:8081/metrics | Saga metrics |
| Consumer Worker | http://localhost:8082/metrics | Consumer metrics |
| Prometheus | http://localhost:9091 | Metrics scrape UI |
| Grafana | http://localhost:3000 | Dashboards (`admin` / `admin`) |

---

## Example APIs

### Publish an event

```bash
curl -s -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "ship-orders",
    "eventType": "ShipPurchased",
    "idempotencyKey": "ship-001",
    "payload": {"pilotId": 42, "shipId": "falcon-x", "credits": 45000}
  }' | jq
```

**Response:**
```json
{
  "id": "6c26f76b-dd6d-4b1b-9c2a-50a7af5a050f",
  "topic": "ship-orders",
  "partition": 3,
  "offset": 1,
  "eventType": "ShipPurchased",
  "publishedAt": "2026-06-09T23:44:02Z"
}
```

### Create a topic

```bash
curl -s -X POST http://localhost:8080/api/v1/topics \
  -H "Content-Type: application/json" \
  -d '{"name":"ship-orders","partitions":6,"replicationFactor":1,"retentionHours":168}' | jq
```

### Inspect consumer offsets

```bash
curl -s http://localhost:8080/api/v1/consumer-groups/galactic-commerce-workers/offsets | jq
```

---

## Workflow Examples

### GalacticCommerce saga

```bash
# Create workflow
WF=$(curl -s -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{"name":"GalacticCommerce","input":{"pilotId":42,"shipId":"falcon-x","credits":45000}}' \
  | jq -r '.id')

# Run async
curl -s -X POST "http://localhost:8080/api/v1/workflows/$WF/run" | jq

# Poll status
curl -s "http://localhost:8080/api/v1/workflows/$WF" | jq '.workflow.status, .steps[].name, .steps[].status'
```

**Steps:** `ProcessPayment` → `ReserveInventory` → `SendConfirmation`

### Inject saga failure (demo)

```bash
curl -s -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{"name":"GalacticCommerce","input":{"pilotId":99,"demoFailStep":"ReserveInventory"}}' | jq
```

On failure, **RefundPayment** compensation runs (LIFO rollback).

<p align="center">
  <img src="docs/diagrams/workflow-engine.png" alt="Workflow state machine" width="520" />
</p>

---

## Retry and DLQ Examples

```bash
# Publish a failing event
curl -s -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "ship-orders",
    "eventType": "ShipPurchased",
    "idempotencyKey": "toxic-001",
    "payload": {"pilotId": 1, "simulateFailure": true}
  }' | jq -r '.id'

# Inspect retries
curl -s "http://localhost:8080/api/v1/retries?topic=ship-orders&eventId=<EVENT_ID>" | jq

# DLQ stats and messages
curl -s http://localhost:8080/api/v1/dlq/ship-orders/stats | jq
curl -s "http://localhost:8080/api/v1/dlq/ship-orders?limit=5" | jq
```

<p align="center">
  <img src="docs/diagrams/retry-dlq.png" alt="Retry and DLQ flow" width="640" />
</p>

---

## Replay Examples

```bash
# Replay all unreplayed DLQ messages back to source topic
curl -s -X POST http://localhost:8080/api/v1/replay \
  -H "Content-Type: application/json" \
  -d '{"topic":"ship-orders","dlqOnly":true,"targetTopic":"ship-orders"}' | jq

# Replay by time range
curl -s -X POST http://localhost:8080/api/v1/replay \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "orders",
    "startTime": "2026-01-01T00:00:00Z",
    "endTime": "2026-06-01T00:00:00Z"
  }' | jq
```

<p align="center">
  <img src="docs/diagrams/event-replay.png" alt="Replay sequence" width="640" />
</p>

---

## Observability

```bash
# Prometheus targets
open http://localhost:9091

# Grafana demo dashboard (after demo.ps1)
open http://localhost:3000/d/eventflow-demo/eventflow-demo-dashboard
```

<p align="center">
  <img src="docs/diagrams/observability.png" alt="Observability pipeline" width="720" />
</p>

| Metric | Description |
|--------|-------------|
| `eventflow_events_published_total` | Events published per topic |
| `eventflow_events_processed_total` | Consumer throughput |
| `eventflow_consumer_lag` | Partition lag |
| `eventflow_dlq_messages_total` | DLQ insertions |
| `eventflow_retry_attempts_total` | Retry scheduled / failed / dlq |
| `eventflow_workflow_duration_seconds` | Saga step latency |

---

## Scaling and Reliability

### Horizontal scale

| Dimension | Pattern |
|-----------|---------|
| **Publish throughput** | Increase topic partitions; batch publish API |
| **Consume throughput** | Add consumer-worker replicas (≤ partition count) |
| **API ingress** | Stateless api-gateway behind load balancer |
| **Workflow execution** | Redis distributed locks prevent duplicate saga runs |

### Reliability guarantees

- **At-least-once delivery** — producer acks + manual offset commits
- **Idempotency** — Redis deduplication on idempotency keys (24h TTL)
- **Poison messages** — max-retry → DLQ topic + PostgreSQL audit row
- **Partial failure** — saga compensation (LIFO) instead of distributed 2PC
- **Operator recovery** — replay API without code changes

### Production hardening checklist

For real deployments beyond this reference architecture: mTLS/OAuth2, schema registry, rate limiting, SLO-based alerting, multi-AZ Kafka, and chaos testing. See [docs/case-study.md](docs/case-study.md) for trade-off analysis.

---

## Demo Screenshots

<table>
  <tr>
    <td align="center"><img src="docs/demo/screenshots/demo-event-publishing.png" width="320" /><br /><sub>Event Publishing</sub></td>
    <td align="center"><img src="docs/demo/screenshots/demo-workflow-failure.png" width="320" /><br /><sub>Saga Failure</sub></td>
    <td align="center"><img src="docs/demo/screenshots/demo-retry-engine.png" width="320" /><br /><sub>Retry Engine</sub></td>
  </tr>
  <tr>
    <td align="center"><img src="docs/demo/screenshots/demo-dlq.png" width="320" /><br /><sub>DLQ Insertion</sub></td>
    <td align="center"><img src="docs/demo/screenshots/demo-replay.png" width="320" /><br /><sub>DLQ Replay</sub></td>
    <td align="center"><img src="docs/demo/screenshots/demo-grafana.png" width="320" /><br /><sub>Grafana Dashboard</sub></td>
  </tr>
</table>

Regenerate assets: `.\scripts\generate-screenshots.ps1`

---

## Demo Video

Record and publish a portfolio walkthrough:

1. Run `.\scripts\demo.ps1` (~80s)
2. Follow [docs/demo/video.md](docs/demo/video.md) for narrative and OBS settings
3. Upload to YouTube/Loom and embed:

```markdown
[![EventFlow demo](docs/assets/hero-demo-preview.png)](https://www.youtube.com/watch?v=YOUR_VIDEO_ID)
```

---

## Repository Structure

```
EventFlow/
├── cmd/                        # api-gateway, consumer-worker, workflow-engine, demo-generator
├── internal/                   # api, grpc, workflow, retry, replay, storage, topic
├── pkg/                        # config, kafka, metrics, models
├── api/                        # proto, gen/go, openapi
├── migrations/                 # PostgreSQL schema
├── docker/                     # Compose + Dockerfiles
├── deployments/                # k8s, monitoring/grafana
├── helm/eventflow/             # Helm chart
├── terraform/                  # AWS modules (EKS, MSK, RDS, ElastiCache)
├── tests/integration/          # Testcontainers suite
├── docs/
│   ├── diagrams/               # Architecture PNGs (Mermaid source)
│   ├── demo/screenshots/       # README demo captures
│   ├── case-study.md           # System design depth
│   ├── recruiter-guide.md      # Hiring manager overview
│   ├── interview-guide.md      # Interview prep
│   └── resume-snippets.md      # ATS bullet points
└── scripts/                    # demo.ps1, render-diagrams.ps1, generate-screenshots.ps1
```

---

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Kafka as event log** | Durable, ordered-per-partition, replayable |
| **PostgreSQL for state** | ACID workflow/retry/DLQ records, queryable by operators |
| **Redis for locks** | Prevent duplicate workflow execution across replicas |
| **At-least-once delivery** | Simpler than exactly-once; idempotency keys compensate |
| **Embedded retry/replay** | Fewer moving parts; invoked from gateway and consumer |
| **Saga over 2PC** | Availability and partition tolerance in distributed deploys |
| **Convention-based DLQ** | `{topic}-dlq` mirrors industry patterns (SQS, Service Bus) |

Full analysis: [docs/case-study.md](docs/case-study.md)

---

## Resume Bullet Points

- Built **EventFlow**, a distributed event platform in **Go** with Kafka, saga workflows, exponential retry/DLQ, and replay across **16 REST endpoints** and gRPC
- Engineered **at-least-once delivery** with consumer offset tracking, idempotency keys, and Redis deduplication
- Deployed via **Docker Compose**, **Kubernetes/Helm**, and **Terraform** (AWS EKS, MSK, RDS, ElastiCache)

More versions: [docs/resume-snippets.md](docs/resume-snippets.md) (Backend · Platform · Infrastructure)

---

## Build and Test

```bash
make build              # Build all services
make test               # Unit tests
make test-integration   # Testcontainers (Kafka, Postgres, Redis)
make proto              # Generate gRPC stubs
make helm-install       # Deploy Helm chart
```

---

## Documentation

| Document | Audience |
|----------|----------|
| [Case Study](docs/case-study.md) | Engineers — system design depth |
| [Recruiter Guide](docs/recruiter-guide.md) | Hiring managers — plain language |
| [Interview Guide](docs/interview-guide.md) | Candidates — technical interview prep |
| [Resume Snippets](docs/resume-snippets.md) | Portfolio — ATS bullet points |
| [Project Metrics](docs/project-metrics.md) | Portfolio stats |
| [Demo Script](docs/demo/demo-script.md) | Live presentation |
| [Architecture](docs/architecture.md) | Technical reference |
| [Deployment](docs/deployment.md) | Ops runbook |
| [CHANGELOG](CHANGELOG.md) | Release notes |

---

## Release Notes

See [CHANGELOG.md](CHANGELOG.md) for the full history. Latest stable: **v1.0.0** — portfolio release with README, diagrams, recruiter materials, live demo, and contribution guide.

---

## License

MIT — see [LICENSE](LICENSE).

---

<p align="center">
  <sub>EventFlow v1.0.0 — If this looks like a real distributed systems platform, star the repo.</sub>
</p>
