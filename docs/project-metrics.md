# EventFlow Project Metrics

Estimated statistics for portfolio and resume context. Figures reflect the v1.0.0 codebase.

## Codebase

| Metric | Count | Notes |
|--------|------:|-------|
| **Go (application)** | ~3,250 | Excludes generated protobuf stubs |
| **SQL (migrations)** | ~120 | PostgreSQL schema + seeds |
| **Terraform** | ~240 | AWS modules (VPC, EKS, MSK, RDS, ElastiCache) |
| **YAML (K8s/Helm/CI)** | ~1,050 | Compose, Helm, K8s, Prometheus, CI |
| **Protobuf / OpenAPI** | ~800 | gRPC definitions + REST spec |
| **Total (est.)** | **~5,500** | Excludes `go.sum`, generated `api/gen/` |

## Services

| Service | Binary | Port(s) | Role |
|---------|--------|---------|------|
| api-gateway | `cmd/api-gateway` | 8080, 9090 | REST + gRPC ingress |
| workflow-engine | `cmd/workflow-engine` | 8081 | Saga orchestration |
| consumer-worker | `cmd/consumer-worker` | 8082 | Kafka consumption |
| demo-generator | `cmd/demo-generator` | — | Load / failure injection |

## Kafka Topics

| Topic | Partitions (default) | Purpose |
|-------|---------------------:|---------|
| orders | 12 | Order lifecycle events |
| payments | 6 | Payment events |
| notifications | 3 | Notification dispatch |
| analytics | 24 | Analytics pipeline |
| ship-orders | 6 | Galactic Commerce demo |
| `{topic}-dlq` | 1–3 | Dead-letter per source topic |

## Database Tables

| Table | Purpose |
|-------|---------|
| topics | Topic metadata registry |
| events | Published event audit log |
| consumer_groups | Consumer group registry |
| consumer_offsets | Per-partition offset tracking |
| retries | Retry scheduling metadata |
| dead_letter_messages | DLQ persistence |
| workflows | Workflow instances |
| workflow_steps | Per-step execution state |

**Total: 8 tables**

## APIs

### REST (`/api/v1`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/topics` | Create topic |
| GET | `/topics` | List topics |
| GET | `/topics/:name` | Get topic |
| DELETE | `/topics/:name` | Delete topic |
| POST | `/events` | Publish event |
| POST | `/events/batch` | Batch publish |
| POST | `/consumer-groups` | Register consumer group |
| GET | `/consumer-groups/:id/offsets` | Get offsets |
| GET | `/dlq/:topic` | List DLQ messages |
| GET | `/dlq/:topic/stats` | DLQ statistics |
| POST | `/dlq/:topic/replay` | Replay DLQ |
| GET | `/retries` | List retry records |
| POST | `/replay` | Replay events |
| POST | `/workflows` | Create workflow |
| POST | `/workflows/:id/run` | Run workflow |
| GET | `/workflows/:id` | Get workflow + steps |

**Total: 16 REST endpoints**

### gRPC

- TopicService, EventService, ReplayService, WorkflowService (port 9090)

## Tests

| Suite | Count | Tooling |
|-------|------:|---------|
| Unit tests | 2 | `engine_test`, `service_test` |
| Integration tests | 13 | Testcontainers (Kafka, Postgres, Redis) |
| Load benchmarks | 1 | `tests/load/benchmark_test.go` |

## Containers (Docker Compose)

| Container | Image / Build |
|-----------|---------------|
| zookeeper | confluentinc/cp-zookeeper |
| kafka | confluentinc/cp-kafka |
| postgres | postgres:16-alpine |
| redis | redis:7-alpine |
| api-gateway | Built from Dockerfile |
| consumer-worker | Built from Dockerfile |
| workflow-engine | Built from Dockerfile |
| prometheus | prom/prometheus |
| grafana | grafana/grafana |

**Total: 9 containers** in full local stack

## Cloud Resources (Terraform / AWS)

| Resource | Module |
|----------|--------|
| VPC + subnets | `terraform/modules/vpc` |
| EKS cluster | `terraform/modules/eks` |
| MSK (Kafka) | `terraform/modules/msk` |
| RDS PostgreSQL | `terraform/modules/rds` |
| ElastiCache Redis | `terraform/modules/elasticache` |

## Observability

- **15+ Prometheus metric families** (publish, consume, retry, DLQ, workflow)
- **2 Grafana dashboards** (operations + demo)

## CI/CD

- GitHub Actions: Go lint, unit tests, integration tests, Docker build, Helm lint, Terraform validate

---

*Last updated: v1.0.0*
