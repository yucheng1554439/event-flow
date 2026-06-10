# Screen Recording Guide

Tips for recording a polished EventFlow demo for portfolios, LinkedIn, or interviews.

---

## Before Recording

1. **Terminal setup**
   - Font: Cascadia Code, JetBrains Mono, or Fira Code — 14–16pt
   - Theme: Dark (matches Grafana screenshots)
   - Window width: 1200px minimum for readable API output

2. **Stack health**
   ```powershell
   docker compose -f docker/docker-compose.yml -f docker/docker-compose.demo.yml up -d --wait
   curl http://localhost:8080/healthz
   ```

3. **Close distractions**
   - Mute notifications
   - Hide unrelated browser tabs
   - Use a clean desktop background

---

## Recording Tools

| Tool | Platform | Notes |
|------|----------|-------|
| OBS Studio | Win/Mac/Linux | Free, best quality |
| Windows Game Bar | Win | Win+G, quick captures |
| asciinema | Linux/Mac | Terminal-only, lightweight |
| Loom | All | Fast sharing |

---

## Recommended Flow

1. **Intro (10s)** — Show README architecture diagram or `docs/diagrams/architecture.png`
2. **Run demo (80s)** — `.\scripts\demo.ps1 -SkipStackStart`
3. **Grafana (20s)** — Pan across dashboard panels
4. **Outro (10s)** — Show GitHub repo structure or case study link

**Total:** ~2 minutes

---

## Camera / Audio

- Optional face cam in corner — increases engagement on LinkedIn
- Narrate each act using [demo-script.md](demo-script.md) talk track
- Speak slowly during API polling pauses — viewers need time to read output

---

## Post-Production

```bash
# Trim silence (ffmpeg)
ffmpeg -i raw.mp4 -af silenceremove=start_periods=1:start_silence=0.5:start_threshold=-40dB trimmed.mp4

# Generate GIF for README (10s clip, 800px wide)
ffmpeg -i trimmed.mp4 -t 10 -vf "fps=10,scale=800:-1:flags=lanczos" -loop 0 docs/assets/hero-demo.gif
```

See [../assets/generate-gifs.md](../assets/generate-gifs.md) for full GIF workflow.

---

## Export Checklist

- [ ] 1080p MP4 for YouTube/LinkedIn
- [ ] 800px GIF for README hero (under 5MB)
- [ ] 6 screenshots saved to `docs/demo/screenshots/`
- [ ] Grafana dashboard visible in final frame
- [ ] eventId and workflowId visible in terminal output

---

## Upload Suggestions

- **GitHub** — Attach GIF to README via `docs/assets/`
- **LinkedIn** — 2-min technical demo post with architecture diagram carousel
- **Interview** — Share repo link + offer live `demo.ps1` run
