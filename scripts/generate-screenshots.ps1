#Requires -Version 5.1
<#
.SYNOPSIS
  Generate polished demo screenshot PNGs for README and demo GIF pipeline.
.EXAMPLE
  .\scripts\generate-screenshots.ps1
#>
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$outDir = Join-Path $Root "docs\demo\screenshots"
New-Item -ItemType Directory -Path $outDir -Force | Out-Null

function New-TerminalSvg {
    param(
        [string]$Title,
        [string[]]$Lines
    )
    $escaped = $Lines | ForEach-Object {
        ($_ -replace '&', '&amp;' -replace '<', '&lt;' -replace '>', '&gt;')
    }
    $y = 88
    $text = ""
    foreach ($line in $escaped) {
        $color = "#e2e8f0"
        if ($line -match '^\$ ') { $color = "#7dd3fc" }
        elseif ($line -match '"(status|eventType|replayed|total|depth)"') { $color = "#a5b4fc" }
        elseif ($line -match '"(Completed|success|replayed)"|offset|partition') { $color = "#86efac" }
        elseif ($line -match '"(Failed|simulateFailure|dlq)"|DLQ|error') { $color = "#fca5a5" }
        $text += "<text x=`"24`" y=`"$y`" fill=`"$color`" font-family=`"Cascadia Mono, Consolas, monospace`" font-size=`"13`">$line</text>`n"
        $y += 22
    }
    $height = [Math]::Max(360, $y + 40)
    @"
<svg xmlns="http://www.w3.org/2000/svg" width="720" height="$height" viewBox="0 0 720 $height">
  <rect width="720" height="$height" rx="12" fill="#0f172a"/>
  <rect width="720" height="36" rx="12" fill="#1e293b"/>
  <rect y="24" width="720" height="12" fill="#1e293b"/>
  <circle cx="20" cy="18" r="6" fill="#ef4444"/>
  <circle cx="40" cy="18" r="6" fill="#f59e0b"/>
  <circle cx="60" cy="18" r="6" fill="#22c55e"/>
  <text x="88" y="23" fill="#94a3b8" font-family="Segoe UI, sans-serif" font-size="12">$Title</text>
  <rect x="12" y="44" width="696" height="$($height - 56)" rx="8" fill="#020617" stroke="#334155"/>
  $text
</svg>
"@
}

$shots = @{
    "demo-event-publishing.png" = @{
        Title = "EventFlow - Publish ShipPurchased"
        Lines = @(
            '$ curl -s -X POST http://localhost:8080/api/v1/events \'
            '    -d ''{"topic":"ship-orders","eventType":"ShipPurchased",'
            '         "idempotencyKey":"ship-001",'
            '         "payload":{"pilotId":42,"shipId":"falcon-x"}}'' | jq'
            '{'
            '  "id": "6c26f76b-dd6d-4b1b-9c2a-50a7af5a050f",'
            '  "topic": "ship-orders",'
            '  "partition": 3,'
            '  "offset": 1,'
            '  "eventType": "ShipPurchased",'
            '  "publishedAt": "2026-06-09T23:44:02Z"'
            '}'
        )
    }
    "demo-workflow-failure.png" = @{
        Title = "EventFlow - GalacticCommerce saga failure"
        Lines = @(
            '$ curl -s http://localhost:8080/api/v1/workflows/$WF | jq'
            '{'
            '  "workflow": { "name": "GalacticCommerce", "status": "Failed" },'
            '  "steps": ['
            '    { "name": "ProcessPayment", "status": "Completed" },'
            '    { "name": "ReserveInventory", "status": "Failed",'
            '      "error": "inventory unavailable for pilot 99" },'
            '    { "name": "SendConfirmation", "status": "Skipped" }'
            '  ],'
            '  "compensation": [ { "name": "RefundPayment", "status": "Completed" } ]'
            '}'
        )
    }
    "demo-retry-engine.png" = @{
        Title = "EventFlow - Retry inspection API"
        Lines = @(
            '$ curl -s "http://localhost:8080/api/v1/retries?topic=ship-orders" | jq'
            '[{'
            '  "eventId": "a91c2e10-4f8b-4c2d-9e11-2b7f8c0d1e22",'
            '  "attempt": 3,'
            '  "maxAttempts": 3,'
            '  "status": "dlq",'
            '  "nextRetryAt": null,'
            '  "lastError": "simulated transient failure"'
            '}]'
        )
    }
    "demo-dlq.png" = @{
        Title = "EventFlow - DLQ stats"
        Lines = @(
            '$ curl -s http://localhost:8080/api/v1/dlq/ship-orders/stats | jq'
            '{'
            '  "topic": "ship-orders",'
            '  "dlqTopic": "ship-orders-dlq",'
            '  "total": 1,'
            '  "unreplayed": 1,'
            '  "replayed": 0,'
            '  "depth": 1'
            '}'
        )
    }
    "demo-replay.png" = @{
        Title = "EventFlow - DLQ replay"
        Lines = @(
            '$ curl -s -X POST http://localhost:8080/api/v1/replay \'
            '    -d ''{"topic":"ship-orders","dlqOnly":true}'' | jq'
            '{'
            '  "topic": "ship-orders",'
            '  "replayed": 1,'
            '  "failed": 0,'
            '  "targetTopic": "ship-orders"'
            '}'
        )
    }
    "demo-grafana.png" = @{
        Title = "Grafana - EventFlow Demo Dashboard"
        Lines = @(
            'Panels:'
            '  Events Published (rate)     ##########--  142/s'
            '  Consumer Lag (max)          ##----------    3'
            '  DLQ Depth                   #-----------  1 -> 0'
            '  Retry Attempts              ####--------   12'
            '  Workflow Duration p95       ######------  1.2s'
            '  Saga Success Rate           ##########--   92%'
        )
    }
}

$resvg = "npx"
$resvgArgs = @("-y", "@resvg/resvg-js-cli")

foreach ($name in $shots.Keys) {
    $cfg = $shots[$name]
    $svgPath = Join-Path $outDir ($name -replace '\.png$', '.svg')
    $pngPath = Join-Path $outDir $name
    New-TerminalSvg -Title $cfg.Title -Lines $cfg.Lines | Set-Content $svgPath -Encoding UTF8
    & $resvg @resvgArgs $svgPath $pngPath | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "resvg failed for $name" }
    Write-Host "Generated $name"
}

$bannerSvg = Join-Path $Root "docs\assets\eventflow-banner.svg"
$bannerPng = Join-Path $Root "docs\assets\eventflow-banner.png"
& $resvg @resvgArgs $bannerSvg $bannerPng --fit-width 1200 | Out-Null
Write-Host "Generated eventflow-banner.png"

$demoGif = Join-Path $Root "docs\assets\demo.gif"
$heroPng = Join-Path $Root "docs\assets\hero-demo-preview.png"
if ((Get-Command ffmpeg -ErrorAction SilentlyContinue) -and (Test-Path $demoGif)) {
    & ffmpeg -y -hide_banner -loglevel error -i $demoGif -frames:v 1 $heroPng
    Write-Host "Generated hero-demo-preview.png from demo.gif"
} else {
    Copy-Item (Join-Path $outDir "demo-replay.png") $heroPng -Force
    Write-Host "Generated hero-demo-preview.png from replay screenshot"
}

Write-Host "Done. Screenshots in docs/demo/screenshots/"
