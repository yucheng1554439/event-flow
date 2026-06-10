# EventFlow Live Demo Script

**Duration:** ~80 seconds (with pauses)  
**Command:** `.\scripts\demo.ps1` or `.\scripts\demo.ps1 -SkipStackStart`

---

## Story Arc

```
ShipPurchased → ProcessPayment → ReserveInventory → Failure
      → Retry (×3) → DLQ → Replay → Success → Grafana
```

Every step uses **real API calls** and **live system state** — not simulated output.

---

## Act 1 — Event Publishing

**Say:** "A pilot purchases a ship. We publish a ShipPurchased event to Kafka."

**Show:**
- `POST /api/v1/events`
- Printed `eventId`, partition, offset, idempotency key

**API:**
```bash
curl -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -d '{"topic":"ship-orders","eventType":"ShipPurchased","idempotencyKey":"demo-001","payload":{"pilotId":42,"shipId":"falcon-x","credits":45000}}'
```

![Event Publishing](screenshots/demo-event-publishing.png)
*Figure 1: Event accepted by Kafka with partition and offset.*

---

## Act 2 — Consumer Processing

**Say:** "The consumer worker picks up the message and commits its offset."

**Show:**
- Live logs from `consumer-worker`
- `GET /api/v1/consumer-groups/galactic-commerce-workers/offsets`

---

## Act 3 — Workflow Creation

**Say:** "We create a GalacticCommerce saga with intentional failure at ReserveInventory."

**Show:**
- `POST /api/v1/workflows` → `workflowId`
- `POST /api/v1/workflows/:id/run`

---

## Act 4 — Saga Failure

**Say:** "ProcessPayment succeeds. ReserveInventory fails. RefundPayment compensates."

**Show:**
- Workflow polled every 1 second
- State transitions: `pending → running → failed`
- Step status: ProcessPayment completed, ReserveInventory failed

![Workflow Failure](screenshots/demo-workflow-failure.png)
*Figure 2: Saga failure with LIFO compensation.*

---

## Act 5 — Retry Engine

**Say:** "We publish a toxic event. The retry engine schedules 3 attempts with exponential backoff."

**Show:**
- `POST /api/v1/events` with `simulateFailure: true`
- `GET /api/v1/retries?topic=ship-orders&eventId=...`
- Live retry logs

![Retry Engine](screenshots/demo-retry-engine.png)
*Figure 3: Retry attempts tracked in PostgreSQL.*

---

## Act 6 — DLQ Insertion

**Say:** "After max retries, the message is quarantined in the dead-letter queue."

**Show:**
- `GET /api/v1/dlq/ship-orders/stats` — before/after counts
- DLQ message details: eventId, failure reason, retry count

![DLQ](screenshots/demo-dlq.png)
*Figure 4: Message routed to DLQ after retry exhaustion.*

---

## Act 7 — Replay

**Say:** "Operators replay DLQ messages back to the source topic."

**Show:**
- `POST /api/v1/replay` with `dlqOnly: true`
- Unreplayed count: 1 → 0

![Replay](screenshots/demo-replay.png)
*Figure 5: DLQ replay restores unreplayed count to zero.*

---

## Act 8 — Successful Completion

**Say:** "A clean workflow runs all three saga steps to completion."

**Show:**
- ProcessPayment → ReserveInventory → SendConfirmation: **completed**
- Final workflowId and eventIds summary

---

## Act 9 — Grafana (End Only)

**Say:** "Metrics from this run are visible in Grafana."

**Open:** http://localhost:3000/d/eventflow-demo/eventflow-demo-dashboard (admin/admin)

![Grafana](screenshots/demo-grafana.png)
*Figure 6: EventFlow demo dashboard with throughput and DLQ metrics.*

---

## Troubleshooting

| Issue | Fix |
|-------|-----|
| API not ready | `docker compose -f docker/docker-compose.yml -f docker/docker-compose.demo.yml up -d --wait` |
| Stale DLQ data | Demo auto-clears retry/DLQ tables at start |
| No consumer logs | Ensure `consumer-worker` uses demo overlay (`CONSUMER_TOPIC=ship-orders`) |

---

## Related

- [Recording Guide](recording-guide.md)
- [Demo Talk Track](../demo-talk-track.md)
