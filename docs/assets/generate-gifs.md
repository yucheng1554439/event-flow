# GIF Generation Guide

Commands to produce README and portfolio GIFs from a live demo recording.

Store output in `docs/assets/`.

---

## Prerequisites

- [ffmpeg](https://ffmpeg.org/) installed and on PATH
- Completed demo recording (MP4) or ability to re-run `demo.ps1`

---

## 1. Hero Demo GIF (README)

Full story arc — publish through Grafana.

```powershell
# Record terminal while running demo
.\scripts\demo.ps1 -SkipStackStart

# Convert last 15 seconds of recording to GIF
ffmpeg -i recording.mp4 -t 15 -vf "fps=8,scale=900:-1:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse" -loop 0 docs/assets/hero-demo.gif
```

**Target:** `docs/assets/hero-demo.gif` — referenced in README hero section.

---

## 2. Workflow Failure GIF

Clip from Act 4 (saga failure + compensation).

```powershell
ffmpeg -i recording.mp4 -ss 00:00:35 -t 8 -vf "fps=10,scale=800:-1:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse" -loop 0 docs/assets/workflow-failure.gif
```

---

## 3. DLQ Replay GIF

Clip from Acts 6–7 (DLQ stats + replay count change).

```powershell
ffmpeg -i recording.mp4 -ss 00:00:55 -t 10 -vf "fps=10,scale=800:-1:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse" -loop 0 docs/assets/dlq-replay.gif
```

---

## 4. Grafana Dashboard GIF

Record browser window showing Grafana dashboard.

```powershell
# Windows: record with OBS, then:
ffmpeg -i grafana-recording.mp4 -t 12 -vf "fps=6,scale=1000:-1:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse" -loop 0 docs/assets/grafana-dashboard.gif
```

**Grafana URL:** http://localhost:3000/d/eventflow-demo/eventflow-demo-dashboard

---

## Optimization

Keep GIFs under 5 MB for GitHub README performance:

```powershell
# Reduce colors
ffmpeg -i input.mp4 -t 10 -vf "fps=8,scale=720:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=128[p];[s1][p]paletteuse" -loop 0 output.gif

# Convert to WebM for smaller size (GitHub supports video in README)
ffmpeg -i recording.mp4 -t 15 -c:v libvpx-vp9 -b:v 0 -crf 35 docs/assets/hero-demo.webm
```

---

## Placeholder Assets

Until recordings exist, use static previews:

| File | Source |
|------|--------|
| `hero-demo-preview.png` | `docs/demo/screenshots/demo-event-publishing.png` |
| `docs/assets/eventflow-banner.png` | Project banner |

Run `.\scripts\demo.ps1 -SkipStackStart` and capture real GIFs to replace placeholders.
