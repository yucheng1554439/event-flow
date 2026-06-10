# EventFlow — 3-Minute Talk Track

*For backend interviews, career fairs, internship recruiting, and system design discussions.*

---

## 1. Problem (20 sec)

> "Modern backends need durable events, retries, and workflows — but wiring Kafka, queues, and orchestration yourself takes months. **EventFlow** is a platform that gives you all of that in one stack."

---

## 2. Architecture (25 sec)

> "Producers hit our API Gateway — REST and gRPC. Events land in **Kafka** topics. **Consumer groups** process them in parallel. A **workflow engine** runs multi-step sagas. Failures go through a **retry engine** with exponential backoff, then a **dead letter queue**. **Prometheus and Grafana** show everything in real time."

*Show architecture diagram from demo-guide.md*

---

## 3. Event Publishing (20 sec)

> "Watch this — a pilot buys a spaceship. One POST publishes `ShipPurchased` to the `ship-orders` topic with an **idempotency key** so duplicate clicks never double-charge."

```bash
./scripts/demo.sh
```

---

## 4. Consumer Groups (20 sec)

> "The **galactic-commerce-workers** consumer group reads from Kafka. Partitions scale horizontally — add workers, get more throughput. Offsets are committed only after success — **at-least-once delivery**."

*Point to consumer lag panel in Grafana*

---

## 5. Workflow Execution (25 sec)

> "On success, we trigger the **GalacticCommerce** saga: ProcessPayment → ReserveInventory → SendConfirmation. State is persisted in PostgreSQL — if a worker crashes, we resume."

*Show workflow steps in API response*

---

## 6. Retry Handling (25 sec)

> "What if processing fails? We don't lose the event. Retry 1… 2… 3 with exponential backoff. Metadata lives in Postgres so we can audit every attempt."

```bash
./scripts/demo-failure.sh
```

---

## 7. DLQ Handling (20 sec)

> "After max retries, the message moves to **ship-orders-dlq** — not deleted, not silent. Operators inspect, debug, and decide when to replay."

*Show DLQ API response*

---

## 8. Replay (20 sec)

> "Bug fixed? One replay command re-publishes DLQ events. The workflow completes successfully. This is how production systems recover without manual data fixes."

```bash
./scripts/demo-replay.sh
```

---

## 9. Monitoring (15 sec)

> "The **EventFlow Demo Dashboard** shows events per second, consumer lag, retry count, DLQ volume, and workflow success rate — everything an SRE needs."

*Open Grafana → EventFlow Demo Dashboard*

---

## 10. Scalability (15 sec)

> "This runs locally in Docker today. In production: **Kubernetes + Helm**, **Terraform** for AWS MSK, RDS, and ElastiCache. Topics scale by partitions; consumers scale by replicas. Same patterns as Kafka at LinkedIn or Uber scale — simplified for teams that want to ship fast."

---

## Closing (10 sec)

> "One command: `./scripts/demo.sh`. Full lifecycle — publish, consume, workflow, retry, DLQ, replay, metrics. Questions?"

---

**Total: ~3 minutes** (stretch to 5 with live Grafana and failure demo)
