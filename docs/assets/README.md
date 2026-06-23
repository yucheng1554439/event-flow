# EventFlow Assets

Visual assets for README, portfolio, and social sharing.

| File | Purpose |
|------|---------|
| `eventflow-banner.svg` / `.png` | README header banner |
| `demo.gif` | Above-the-fold live demo animation in README |
| `hero-demo-preview.png` | Static preview / video embed thumbnail |
| `generate-gifs.md` | ffmpeg commands for custom GIF exports |

## Regenerate

```powershell
.\scripts\generate-screenshots.ps1   # terminal mockups + banner PNG
.\scripts\render-diagrams.ps1        # architecture / workflow / DLQ / replay / observability PNGs
.\scripts\generate-demo-gif.ps1      # demo.gif from screenshots (requires ffmpeg)
```

## Demo video

Record `.\scripts\demo.ps1`, then follow [../demo/video.md](../demo/video.md) to publish and embed in README.
