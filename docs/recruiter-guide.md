# EventFlow — Recruiter Guide

A plain-language overview for hiring managers, recruiters, and non-specialist technical reviewers.

---

## What EventFlow Is

**EventFlow** is a distributed event processing platform — software that receives, routes, stores, and processes high-volume business events (like "order placed" or "payment completed") across multiple servers reliably.

Think of it as the **infrastructure layer** behind modern apps: the part that makes sure messages are never lost, failures are retried, bad messages are quarantined, and complex multi-step processes complete correctly.

It is a **portfolio / reference implementation** built to production standards: real APIs, real databases, real message queues, real monitoring, and real cloud deployment configs.

---

## Problem Solved

Modern applications generate millions of events per day. A single "purchase" might trigger:

1. Charge the customer
2. Reserve inventory
3. Send confirmation email

If step 2 fails after step 1 succeeds, you need **compensation** (refund). If a server crashes mid-processing, you need **retries**. If a message is permanently broken, you need a **dead-letter queue** so engineers can inspect and replay it later.

EventFlow solves all of this in one cohesive platform — not as disconnected tutorials, but as integrated services you can run locally or deploy to Kubernetes/AWS.

---

## Technical Challenges Demonstrated

| Challenge | How EventFlow Addresses It |
|-----------|---------------------------|
| Message durability | Apache Kafka with persistent topics |
| Duplicate messages | Idempotency keys + Redis deduplication |
| Partial failures | Saga pattern with LIFO compensation |
| Transient errors | Exponential backoff retry engine |
| Poison messages | Dead-letter queue with replay API |
| Operational visibility | Prometheus metrics + Grafana dashboards |
| Multi-environment deploy | Docker Compose, Helm, Terraform |

---

## Scale Characteristics

EventFlow is designed with **production scale patterns** even when running locally:

- **Partitioned Kafka topics** (6–24 partitions) for horizontal throughput
- **Consumer groups** for parallel processing across partitions
- **Stateless API gateway** — scales horizontally behind a load balancer
- **PostgreSQL** for durable workflow, retry, and DLQ state
- **Redis** for distributed workflow locking
- **AWS Terraform modules** for EKS + MSK + RDS at cloud scale

The included load generator (`cmd/demo-generator`) can publish 1,000+ events for throughput demonstrations.

---

## Distributed Systems Concepts

EventFlow demonstrates concepts commonly discussed in **senior backend and platform interviews**:

- At-least-once delivery semantics
- Consumer offset management
- Event-driven architecture
- Saga / compensation transactions
- Dead-letter queues and replay
- Observability-driven operations (metrics, dashboards)
- Infrastructure as code (Terraform, Helm)
- gRPC and REST API design
- Integration testing with real dependencies (Testcontainers)

---

## Interview Topics Covered

A candidate who built EventFlow can speak confidently about:

1. **Why Kafka?** — Durability, ordering per partition, replay capability
2. **Retry vs DLQ** — When to retry, when to quarantine
3. **Workflow sagas** — Multi-step processes with rollback
4. **Idempotency** — Preventing duplicate side effects
5. **Consumer groups** — Partition assignment and rebalancing
6. **Observability** — What to metric, what to alert on
7. **Deployment** — Local → Kubernetes → AWS progression

---

## Estimated Engineering Level

| Dimension | Assessment |
|-----------|------------|
| **Overall** | Strong **new grad to mid-level platform/backend** portfolio project |
| **With live demo** | Presents as **mid-level** — shows end-to-end ownership |
| **With case study depth** | Approaches **senior** systems design articulation |
| **Production deployment** | Reference architecture; would need hardening for real production (auth, multi-tenancy, SLOs) |

**Best fit roles:** Backend Engineer, Platform Engineer, Infrastructure Engineer, Distributed Systems Engineer (entry to mid).

---

## How to Evaluate in 5 Minutes

1. **Read the README** — architecture diagram and quick start
2. **Run the demo** — `.\scripts\demo.ps1` (~80 seconds)
3. **Open Grafana** — http://localhost:3000 (admin/admin)
4. **Skim the case study** — [case-study.md](case-study.md)

---

## Related Documents

- [System Design Case Study](case-study.md)
- [Interview Guide](interview-guide.md)
- [Resume Snippets](resume-snippets.md)
- [Project Metrics](project-metrics.md)
- [Demo Script](demo/demo-script.md)
