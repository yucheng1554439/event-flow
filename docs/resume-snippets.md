# EventFlow Resume Snippets

ATS-friendly bullet points. Copy 2–4 bullets per application. Quantities reference [project-metrics.md](project-metrics.md).

---

## Backend Version

- Built **EventFlow**, a distributed event processing platform in **Go** with **16 REST endpoints** and **gRPC** services for publishing, consuming, and replaying Kafka messages
- Implemented **at-least-once delivery** with consumer group offset tracking, idempotency keys, and **Redis**-backed deduplication
- Designed **saga compensation** workflow engine with LIFO rollback (`ProcessPayment` → `ReserveInventory` → `SendConfirmation`)
- Persisted events, workflows, retries, and DLQ records across **8 PostgreSQL tables** with queryable operator APIs
- Wrote **Testcontainers** integration tests validating publish/consume, retry, DLQ, and replay paths against real dependencies
- Delivered live **GalacticCommerce** demo: publish → saga → failure → retry → DLQ → replay → success in under 90 seconds

---

## Platform Version

- Architected **EventFlow**, a production-style event platform combining **Apache Kafka**, **PostgreSQL**, **Redis**, and **Go** microservices for durable async processing
- Built retry engine with **exponential backoff**, max-attempt DLQ routing, and REST/gRPC APIs for retry inspection and DLQ statistics
- Implemented event **replay service** supporting DLQ-only, time-range, and partition-scoped reprocessing with idempotent replay tracking
- Designed Kafka topic strategy with **6–24 partitions** per domain, paired `{topic}-dlq` topics, and consumer group horizontal scaling
- Exposed **dynamic topic administration** (create/list/delete) via REST and gRPC with Kafka AdminClient integration
- Established **CI/CD pipeline** (GitHub Actions): unit tests, integration suite, Docker builds, Helm lint, Terraform validation
- Delivered **15+ Prometheus metric families** and **2 Grafana dashboards** for throughput, lag, DLQ depth, and saga latency

---

## Infrastructure Version

- Provisioned AWS infrastructure with **Terraform** modules: VPC, **EKS**, **MSK** (Kafka), **RDS** PostgreSQL, and **ElastiCache** Redis
- Packaged EventFlow into a **Helm chart** (api-gateway, consumer-worker, workflow-engine, Postgres, Redis) with configurable values
- Designed **9-container** Docker Compose stack mirroring production: Kafka, Zookeeper, Postgres, Redis, Prometheus, Grafana
- Authored **Kubernetes** base manifests with Kustomize dev overlay for environment-specific deployment
- Configured **Prometheus** scrape targets and **Grafana** provisioning for operations and demo dashboards
- Maintained **~5,500 lines** of application and infrastructure code with reproducible CI and lock files
- Created deployment runbooks and infrastructure demo showcasing DLQ replay, consumer scaling, and workflow failure compensation

---

## Compact Variants (Intern / New Grad)

Use 3–4 bullets from **Backend Version** plus one deployment bullet from **Infrastructure Version** if space allows.

---

## Skills Keywords (ATS)

`Go` `Golang` `Apache Kafka` `PostgreSQL` `Redis` `Docker` `Kubernetes` `Helm` `Terraform` `AWS` `EKS` `MSK` `RDS` `gRPC` `REST API` `Prometheus` `Grafana` `Microservices` `Distributed Systems` `Event-Driven Architecture` `Saga Pattern` `Dead Letter Queue` `CI/CD` `GitHub Actions` `Testcontainers`

---

## One-Line Summary Variants

- **Short:** EventFlow — Kafka event platform with workflows, retries, DLQ, and replay (Go, PostgreSQL, Redis, K8s, Terraform)
- **Impact:** Built a production-grade distributed event processing platform demonstrating saga workflows, at-least-once delivery, and cloud-native deployment
- **Technical:** Go microservices: Kafka ingestion, exponential retry/DLQ, compensating sagas, gRPC/REST APIs, Prometheus/Grafana observability
