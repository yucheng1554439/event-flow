#Requires -Version 5.1
<#
.SYNOPSIS
  Build docs/assets/demo.gif — 10-act EventFlow story (~30 seconds).
.EXAMPLE
  .\scripts\generate-demo-gif.ps1
#>
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$OutGif  = Join-Path $Root "docs\assets\demo.gif"
$WorkDir = Join-Path $env:TEMP "eventflow-demo-gif"
$Frames  = Join-Path $WorkDir "frames"
$Seconds = 3

$Acts = @(
    "docs\demo\screenshots\demo-event-publishing.png",
    "docs\demo\screenshots\demo-event-publishing.png",
    "docs\demo\screenshots\demo-workflow-failure.png",
    "docs\demo\screenshots\demo-workflow-failure.png",
    "docs\demo\screenshots\demo-workflow-failure.png",
    "docs\demo\screenshots\demo-retry-engine.png",
    "docs\demo\screenshots\demo-dlq.png",
    "docs\demo\screenshots\demo-replay.png",
    "docs\demo\screenshots\demo-event-publishing.png",
    "docs\demo\screenshots\demo-grafana.png"
)

if (-not (Get-Command ffmpeg -ErrorAction SilentlyContinue)) {
    throw "ffmpeg not found on PATH"
}

if (Test-Path $WorkDir) { Remove-Item $WorkDir -Recurse -Force }
New-Item -ItemType Directory -Path $Frames -Force | Out-Null

$i = 0
foreach ($rel in $Acts) {
    $i++
    $src = Join-Path $Root $rel
    if (-not (Test-Path $src)) { throw "Missing: $src" }
    $out = Join-Path $Frames ("frame_{0:D2}.png" -f $i)
    & ffmpeg -y -hide_banner -loglevel error -i $src -vf "scale=720:-1:flags=lanczos" $out
    if ($LASTEXITCODE -ne 0) { throw "ffmpeg frame $i failed" }
    Write-Host "Frame $i/10"
}

$listFile = Join-Path $WorkDir "concat.txt"
$lines = New-Object System.Collections.Generic.List[string]
foreach ($n in 1..10) {
    $fp = (Join-Path $Frames ("frame_{0:D2}.png" -f $n)) -replace '\\','/'
    $lines.Add("file '$fp'")
    $lines.Add("duration $Seconds")
}
$fpLast = (Join-Path $Frames "frame_10.png") -replace '\\','/'
$lines.Add("file '$fpLast'")
[System.IO.File]::WriteAllLines($listFile, $lines)

$palette = Join-Path $WorkDir "palette.png"
& ffmpeg -y -hide_banner -loglevel error -f concat -safe 0 -i $listFile `
    -vf "fps=8,scale=720:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=96[p];[s1][p]paletteuse" `
    -loop 0 $OutGif
if ($LASTEXITCODE -ne 0) { throw "gif encode failed" }

$kb = [math]::Round((Get-Item $OutGif).Length / 1KB, 1)
Write-Host "Created $OutGif ($kb KB)"
