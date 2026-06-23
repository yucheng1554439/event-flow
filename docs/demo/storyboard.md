# EventFlow Demo — 10-Act Storyboard

**Total runtime:** ~30 seconds (GIF) · ~80 seconds (live `demo.ps1`)  
**GIF asset:** [docs/assets/demo.gif](../assets/demo.gif)  
**Regenerate GIF:** `.\scripts\generate-demo-gif.ps1`

---

## Story arc

```
Publish → Consume → Workflow → Failure → Compensate
    → Retry → DLQ → Replay → Success → Grafana
```

| Act | Scene | What happens | API / signal |
|-----|-------|--------------|--------------|
| **1** | Event published | `ShipPurchased` accepted by Kafka | `POST /api/v1/events` |
| **2** | Consumer processes event | Worker commits offset, may trigger workflow | Consumer logs, `GET .../offsets` |
| **3** | Workflow starts | `GalacticCommerce` saga begins | `POST /api/v1/workflows`, `/run` |
| **4** | Saga failure | `ReserveInventory` step fails | Workflow status `failed` |
| **5** | Compensation executes | `RefundPayment` rolls back completed steps (LIFO) | Compensation logs |
| **6** | Retry engine | Toxic event scheduled with exponential backoff | `GET /api/v1/retries` |
| **7** | DLQ insertion | Max retries exceeded; message quarantined | `GET /api/v1/dlq/:topic/stats` |
| **8** | Replay | Operator replays unreplayed DLQ messages | `POST /api/v1/replay` |
| **9** | Successful completion | Clean saga: all three steps `completed` | `GET /api/v1/workflows/:id` |
| **10** | Grafana dashboard | Throughput, lag, DLQ depth visualized | Grafana demo dashboard |

---

## Run live

```powershell
# Full stack + demo
.\scripts\demo.ps1

# Stack already up
.\scripts\demo.ps1 -SkipStackStart
```

---

## Capture assets

| Asset | Command |
|-------|---------|
| **demo.gif** (README) | `.\scripts\generate-demo-gif.ps1` |
| Screenshots | `.\scripts\capture-demo-screenshots.ps1` |
| Terminal recording | See [recording-guide.md](recording-guide.md) |
| Replace GIF from MP4 | See [generate-gifs.md](../assets/generate-gifs.md) |

---

## Screenshot map

| Act | File |
|-----|------|
| 1, 2, 9 | `screenshots/demo-event-publishing.png` |
| 3, 4, 5 | `screenshots/demo-workflow-failure.png` |
| 6 | `screenshots/demo-retry-engine.png` |
| 7 | `screenshots/demo-dlq.png` |
| 8 | `screenshots/demo-replay.png` |
| 10 | `screenshots/demo-grafana.png` |

After re-recording the live demo, update screenshots then rerun `generate-demo-gif.ps1`.
