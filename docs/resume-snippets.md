# EventFlow Resume Snippets

ATS-friendly bullet points. Copy 2–4 bullets per role application. Quantities reference [project-metrics.md](project-metrics.md).

---

## Backend Intern Version

- Built **EventFlow**, an event-driven backend platform in **Go** with REST APIs for publishing, consuming, and replaying Kafka messages
- Implemented **PostgreSQL** persistence for events, consumer offsets, and dead-letter queue records across 8 database tables
- Created **Docker Compose** local stack with Kafka, Redis, and 3 microservices for end-to-end development and testing
- Wrote integration tests using **Testcontainers** to validate publish/consume flows against real Kafka and PostgreSQL instances
- Added **Prometheus** metrics and **Grafana** dashboards for event throughput and consumer lag monitoring

---

## New Grad Version

- Designed and implemented **EventFlow**, a distributed event processing platform in **Go** featuring Kafka messaging, saga workflows, exponential backoff retries, and DLQ replay
- Built **16 REST endpoints** and **gRPC services** for topic administration, event publishing, workflow orchestration, and dead-letter queue management
- Engineered **at-least-once delivery** with consumer group offset tracking, idempotency keys, and Redis-backed deduplication
- Implemented **saga compensation** workflow engine with LIFO rollback (ProcessPayment → ReserveInventory → SendConfirmation)
- Deployed via **Docker**, **Kubernetes** manifests, and **Helm** chart; provisioned AWS infrastructure with **Terraform** (EKS, MSK, RDS, ElastiCache)
- Authored live demo script processing ShipPurchased events through retry, DLQ, replay, and successful workflow completion in under 90 seconds

---

## Platform Engineer Version

- Architected **EventFlow**, a production-style event platform combining **Apache Kafka**, **PostgreSQL**, **Redis**, and **Go** microservices for durable async processing at scale
- Designed Kafka topic strategy with 6–24 partitions per domain, paired DLQ topics, and consumer group horizontal scaling patterns
- Built retry engine with **exponential backoff**, max-attempt DLQ routing, and operator APIs for retry inspection and DLQ statistics
- Implemented event **replay service** supporting DLQ-only, time-range, and partition-scoped reprocessing with idempotent replay tracking
- Delivered **observability stack**: 15+ Prometheus metric families, Grafana dashboards, and health endpoints across all services
- Established **CI/CD pipeline** (GitHub Actions) with unit tests, Testcontainers integration suite, Docker builds, Helm lint, and Terraform validation
- Created **Terraform modules** for AWS VPC, EKS, MSK, RDS PostgreSQL, and ElastiCache Redis

---

## Infrastructure Engineer Version

- Built cloud-native **EventFlow** platform with **Terraform** IaC modules for AWS EKS, MSK (Kafka), RDS, ElastiCache, and VPC networking
- Packaged all services into **Helm chart** with configurable values for api-gateway, consumer-worker, workflow-engine, Postgres, and Redis
- Designed **9-container** Docker Compose stack mirroring production topology: Kafka, Zookeeper, Postgres, Redis, Prometheus, Grafana
- Implemented **Kubernetes** base manifests with Kustomize dev overlay for environment-specific deployment
- Configured **Prometheus** scrape targets and **Grafana** provisioning for EventFlow operational and demo dashboards
- Authored deployment runbooks and live infrastructure demo showcasing DLQ replay, consumer scaling, and workflow failure compensation
- Maintained **~5,500 lines** of application and infrastructure code with integration tests against containerized dependencies

---

## Skills Keywords (ATS)

`Go` `Golang` `Apache Kafka` `PostgreSQL` `Redis` `Docker` `Kubernetes` `Helm` `Terraform` `AWS` `EKS` `MSK` `RDS` `gRPC` `REST API` `Prometheus` `Grafana` `Microservices` `Distributed Systems` `Event-Driven Architecture` `Saga Pattern` `Dead Letter Queue` `CI/CD` `GitHub Actions` `Testcontainers`

---

## One-Line Summary Variants

- **Short:** EventFlow — Kafka-based event platform with workflows, retries, DLQ, and replay (Go, PostgreSQL, Redis, K8s, Terraform)
- **Impact:** Built a production-grade distributed event processing platform demonstrating saga workflows, at-least-once delivery, and cloud-native deployment
- **Technical:** Go microservices platform: Kafka ingestion, exponential retry/DLQ, compensating sagas, gRPC/REST APIs, Prometheus/Grafana observability
