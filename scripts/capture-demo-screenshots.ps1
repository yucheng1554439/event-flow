#Requires -Version 5.1
<#
.SYNOPSIS
  Capture demo screenshots after running the live demo.
  Requires Windows (PrintScreen to file) or manual capture — see docs/demo/recording-guide.md
.EXAMPLE
  .\scripts\demo.ps1 -SkipStackStart
  .\scripts\capture-demo-screenshots.ps1
#>
$out = Join-Path (Split-Path -Parent $PSScriptRoot) "docs\demo\screenshots"
Write-Host "Save screenshots to: $out"
Write-Host ""
Write-Host "Recommended captures after demo.ps1:"
Write-Host "  1. demo-event-publishing.png  — Act 1 eventId + offset"
Write-Host "  2. demo-workflow-failure.png  — Act 4 saga failure"
Write-Host "  3. demo-retry-engine.png      — Act 5 retry API output"
Write-Host "  4. demo-dlq.png               — Act 6 DLQ stats"
Write-Host "  5. demo-replay.png            — Act 7 replay count"
Write-Host "  6. demo-grafana.png           — Act 9 Grafana dashboard"
Write-Host ""
Write-Host "See docs/demo/recording-guide.md for OBS/ffmpeg workflow."
