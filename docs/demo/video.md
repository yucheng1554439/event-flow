# EventFlow Demo Video

Use this guide to publish a portfolio-quality demo video and embed it in the README.

## Record

```powershell
# Start stack + run the 9-act live demo (~80s)
.\scripts\demo.ps1

# Re-record narrative only
.\scripts\demo.ps1 -SkipStackStart
```

Follow [recording-guide.md](recording-guide.md) for terminal setup, OBS settings, and pacing.

## Suggested narrative (90 seconds)

| Time | Scene | Talk track |
|------|-------|------------|
| 0:00 | Architecture diagram | "EventFlow is a Go event platform on Kafka with saga workflows, retries, DLQ, and replay." |
| 0:10 | `demo.ps1` Act 1–3 | "We publish ShipPurchased, run GalacticCommerce, and watch the saga complete." |
| 0:35 | Failure injection | "A transient failure triggers exponential backoff retries." |
| 0:50 | DLQ stats API | "After max attempts, the event lands in ship-orders-dlq." |
| 1:05 | Replay API | "Operators replay from DLQ and the workflow succeeds." |
| 1:20 | Grafana dashboard | "Prometheus metrics and Grafana show throughput, lag, and DLQ depth." |

## Publish

1. Upload to YouTube (unlisted) or Loom.
2. Export a 15–30s GIF for the README hero: [generate-gifs.md](../assets/generate-gifs.md).
3. Replace the placeholder in README **Demo Video** with your embed URL.

## Embed template

```markdown
[![EventFlow demo](docs/assets/hero-demo-preview.png)](https://www.youtube.com/watch?v=YOUR_VIDEO_ID)
```

Or HTML (GitHub renders in README):

```html
<p align="center">
  <a href="https://www.youtube.com/watch?v=YOUR_VIDEO_ID">
    <img src="docs/assets/hero-demo-preview.png" alt="EventFlow demo video" width="720" />
  </a>
</p>
```
