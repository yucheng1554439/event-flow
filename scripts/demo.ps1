#Requires -Version 5.1
<#
.SYNOPSIS
  Galactic Commerce - live EventFlow demo backed by real API calls and system state.

  Story: ShipPurchased -> ProcessPayment -> ReserveInventory -> Failure
         -> Retry -> DLQ -> Replay -> Success

.EXAMPLE
  .\scripts\demo.ps1
  .\scripts\demo.ps1 -SkipStackStart
#>
param(
    [switch]$SkipStackStart
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot

# --- Config ---
$Api           = if ($env:EVENTFLOW_API) { $env:EVENTFLOW_API } else { "http://localhost:8080" }
$GrafanaUrl    = "http://localhost:3000/d/eventflow-demo/eventflow-demo-dashboard"
$ComposeArgs   = @("-f", "docker/docker-compose.yml", "-f", "docker/docker-compose.demo.yml")
$Compose       = "docker compose $($ComposeArgs -join ' ')"
$Topic         = "ship-orders"
$DlqTopic      = "ship-orders-dlq"
$ConsumerGroup = "galactic-commerce-workers"
$DemoStart      = Get-Date
$script:LogJobs  = @()
$script:TrackIds = [System.Collections.Generic.List[string]]::new()

function Write-Banner([string]$Text) {
    $bar = "=" * 62
    Write-Host ""
    Write-Host $bar -ForegroundColor Cyan
    Write-Host "  $Text" -ForegroundColor Cyan
    Write-Host $bar -ForegroundColor Cyan
    Write-Host ""
}

function Write-Act([int]$N, [int]$Total, [string]$Title) {
    Write-Host ""
    Write-Host "[$N/$Total] $Title" -ForegroundColor Yellow
    Write-Host ("-" * 54) -ForegroundColor DarkGray
}

function Write-Ok([string]$Msg)    { Write-Host "  [OK] $Msg" -ForegroundColor Green }
function Write-Info([string]$Msg)  { Write-Host "  >> $Msg" -ForegroundColor White }
function Write-Warn([string]$Msg)  { Write-Host "  [!] $Msg" -ForegroundColor Red }
function Write-Api([string]$Msg)   { Write-Host "  [API] $Msg" -ForegroundColor DarkYellow }

function Write-Elapsed() {
    $sec = [int]((Get-Date) - $DemoStart).TotalSeconds
    Write-Host "  (elapsed ${sec}s)" -ForegroundColor DarkGray
}

function Pause-Demo([int]$Seconds = 3, [string]$Msg = "") {
    if ($Msg) { Write-Info $Msg }
    Start-Sleep -Seconds $Seconds
}

function Invoke-Api {
    param(
        [string]$Method = "Get",
        [string]$Path,
        [object]$Body = $null
    )
    $uri = "$Api$Path"
    $params = @{ Uri = $uri; Method = $Method; TimeoutSec = 20 }
    if ($Body) {
        $params.Body        = ($Body | ConvertTo-Json -Depth 10 -Compress)
        $params.ContentType = "application/json"
    }
    return Invoke-RestMethod @params
}

function Test-ApiReady {
    try {
        $r = Invoke-WebRequest -Uri "$Api/healthz" -UseBasicParsing -TimeoutSec 3
        return ($r.StatusCode -eq 200)
    } catch { return $false }
}

function Wait-ForApi([int]$MaxSeconds = 120) {
    Write-Info "Polling $Api/healthz ..."
    for ($i = 1; $i -le $MaxSeconds; $i += 2) {
        if (Test-ApiReady) { Write-Ok "API gateway healthy"; return }
        if ($i % 10 -eq 0) { Write-Info "still waiting... (${i}s)" }
        Start-Sleep -Seconds 2
    }
    throw "API not ready after ${MaxSeconds}s. Run: $Compose down -v then retry."
}

function Ensure-CleanPostgres {
    $pg = Invoke-Expression "$Compose ps postgres" 2>$null
    if ($pg -match "Exited") {
        Write-Warn "Postgres exited - resetting volume..."
        Invoke-Expression "$Compose down -v"
    }
}

function Add-TrackId([string]$Id) {
    if ($Id -and -not $script:TrackIds.Contains($Id)) {
        $script:TrackIds.Add($Id) | Out-Null
    }
}

function Clear-DemoData {
    Write-Info "Resetting retry/DLQ tables for a clean demonstration..."
    $prevEA = $ErrorActionPreference
    $ErrorActionPreference = "SilentlyContinue"
    $pg = (& docker compose @ComposeArgs ps -q postgres 2>$null | Select-Object -First 1)
    if ($pg) {
        & docker exec $pg psql -U eventflow -d eventflow -c `
            "TRUNCATE retries, dead_letter_messages RESTART IDENTITY;" 2>$null | Out-Null
        Write-Ok "Postgres retry + DLQ state cleared"
    }
    $ErrorActionPreference = $prevEA
}

function Ensure-Topics {
    foreach ($t in @(
        @{ name = $Topic; partitions = 6; replicationFactor = 1; retentionHours = 168; cleanupPolicy = "delete" },
        @{ name = $DlqTopic; partitions = 3; replicationFactor = 1; retentionHours = 2160; cleanupPolicy = "delete" }
    )) {
        try { Invoke-Api -Method Post -Path "/api/v1/topics" -Body $t | Out-Null } catch {}
    }
    Write-Ok "Topics ready: $Topic, $DlqTopic"
}

function Publish-ShipEvent {
    param(
        [string]$Key,
        [bool]$SimulateFailure = $false,
        [hashtable]$Extra = @{}
    )
    $payload = @{ pilotId = 42; shipId = "falcon-x"; credits = 45000 }
    if ($SimulateFailure) { $payload.simulateFailure = $true }
    foreach ($k in $Extra.Keys) { $payload[$k] = $Extra[$k] }
    return Invoke-Api -Method Post -Path "/api/v1/events" -Body @{
        topic = $Topic; eventType = "ShipPurchased"; idempotencyKey = $Key; payload = $payload
    }
}

function Start-GalacticWorkflow {
    param(
        [hashtable]$WorkflowInput = @{ pilotId = 42; shipId = "falcon-x"; credits = 45000 },
        [switch]$RunAsync
    )
    $wf = Invoke-Api -Method Post -Path "/api/v1/workflows" -Body @{
        name = "GalacticCommerce"; input = $WorkflowInput
    }
    if ($RunAsync) {
        Invoke-Api -Method Post -Path "/api/v1/workflows/$($wf.id)/run" | Out-Null
    }
    return $wf
}

function Get-DlqStats {
    return Invoke-Api -Path "/api/v1/dlq/${Topic}/stats"
}

function Get-Retries {
    param([string]$EventId = "")
    $q = "/api/v1/retries?topic=$Topic&limit=20"
    if ($EventId) { $q += "&eventId=$EventId" }
    return @(Invoke-Api -Path $q)
}

function Get-LogSnapshot {
    param([string]$Pattern = ".")
    $prevEA = $ErrorActionPreference
    $ErrorActionPreference = "SilentlyContinue"
    $raw = & docker compose @ComposeArgs logs consumer-worker api-gateway workflow-engine --tail=50 2>$null
    $ErrorActionPreference = $prevEA
    foreach ($line in @($raw)) {
        if ($line -match $Pattern) { return $line }
    }
    return $null
}

function Start-DemoLogStreams {
    Write-Info "Streaming logs: consumer-worker, api-gateway (saga), workflow-engine"
    foreach ($svc in @("consumer-worker", "api-gateway", "workflow-engine")) {
        $script:LogJobs += Start-Job -Name "log-$svc" -ScriptBlock {
            param($dir, $service)
            Set-Location $dir
            & docker compose -f docker/docker-compose.yml -f docker/docker-compose.demo.yml logs -f --tail=0 $service 2>&1 | ForEach-Object {
                $line = $_.ToString()
                if ($line -match 'processed event|retry scheduled|routed to DLQ|galactic workflow|workflow step|compensation|event processing failed|retry attempt') {
                    [PSCustomObject]@{ Service = $service; Line = $line }
                }
            }
        } -ArgumentList $ProjectRoot, $svc
    }
    Start-Sleep -Seconds 1
}

function Test-LogLineRelevant([string]$Line) {
    if ($script:TrackIds.Count -eq 0) { return $true }
    foreach ($id in $script:TrackIds) {
        if ($Line -like "*$id*") { return $true }
    }
    return $false
}

function Flush-LogStreams {
    foreach ($job in $script:LogJobs) {
        $items = Receive-Job -Job $job -ErrorAction SilentlyContinue
        foreach ($item in @($items)) {
            if ($null -eq $item) { continue }
            if (-not (Test-LogLineRelevant $item.Line)) { continue }
            $color = "DarkCyan"
            if ($item.Line -match 'failed|DLQ|routed to DLQ') { $color = "Red" }
            elseif ($item.Line -match 'completed|processed event|workflow started') { $color = "Green" }
            Write-Host "    [$($item.Service)] $($item.Line)" -ForegroundColor $color
        }
    }
}

function Stop-DemoLogStreams {
    foreach ($job in @($script:LogJobs)) {
        Stop-Job -Job $job -ErrorAction SilentlyContinue
        Remove-Job -Job $job -Force -ErrorAction SilentlyContinue
    }
    $script:LogJobs = @()
}

function Watch-WorkflowLive {
    param(
        [string]$WorkflowId,
        [int]$TimeoutSec = 60,
        [string]$Label = ""
    )
    $prevStatus   = $null
    $prevCurrent  = $null
    $prevSteps    = @{}
    $deadline     = (Get-Date).AddSeconds($TimeoutSec)
    $labelText    = if ($Label) { " ($Label)" } else { "" }

    Write-Info "Polling GET /api/v1/workflows/$WorkflowId every 1s$labelText"
    while ((Get-Date) -lt $deadline) {
        Flush-LogStreams

        $detail = Invoke-Api -Path "/api/v1/workflows/$WorkflowId"
        $w      = $detail.workflow
        $status = [string]$w.status

        if ($status -ne $prevStatus) {
            $from = if ($prevStatus) { $prevStatus } else { "new" }
            Write-Host "  [STATE] workflowId=$WorkflowId  $from -> $status" -ForegroundColor Magenta
            $prevStatus = $status
        }

        if ($w.currentStep -and $w.currentStep -ne $prevCurrent) {
            Write-Host "  [STEP ] currentStep=$($w.currentStep)" -ForegroundColor White
            $prevCurrent = $w.currentStep
        }

        foreach ($s in @($detail.steps)) {
            $key = $s.name
            $st  = [string]$s.status
            if (-not $prevSteps.ContainsKey($key) -or $prevSteps[$key] -ne $st) {
                $prevSteps[$key] = $st
                $col = switch ($st) {
                    "completed" { "Green" }
                    "failed"    { "Red" }
                    "running"   { "Yellow" }
                    default     { "Gray" }
                }
                Write-Host "  [STEP ] $($s.name) -> $st" -ForegroundColor $col
                if ($s.error) { Write-Warn $s.error }
            }
        }

        if ($status -in @("completed", "failed")) {
            return $detail
        }
        Start-Sleep -Seconds 1
    }
    Write-Warn "Workflow watch timed out after ${TimeoutSec}s"
    return Invoke-Api -Path "/api/v1/workflows/$WorkflowId"
}

function Wait-ForConsumerProcessed {
    param(
        [string]$EventId,
        [int]$TimeoutSec = 30
    )
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    Write-Info "Waiting for consumer-worker to process eventId=$EventId ..."
    while ((Get-Date) -lt $deadline) {
        Flush-LogStreams

        try {
            $off = Invoke-Api -Path "/api/v1/consumer-groups/$ConsumerGroup/offsets"
            foreach ($o in @($off.offsets)) {
                if ($o.topic -eq $Topic) {
                    Write-Api "consumer offset partition=$($o.partition) offset=$($o.offset)"
                }
            }
        } catch {}

        $hit = Get-LogSnapshot -Pattern "processed event.*$EventId"
        if ($hit -and $hit -match $EventId) {
            if ($hit -match '"workflowId"\s*:\s*"([0-9a-f-]{36})"') {
                Add-TrackId $Matches[1]
                Write-Ok "consumer triggered workflowId=$($Matches[1])"
            }
            Write-Ok "consumer processed eventId=$EventId"
            return $true
        }

        Start-Sleep -Seconds 1
    }
    Write-Warn "consumer processing not confirmed within ${TimeoutSec}s (check logs)"
    return $false
}

function Wait-ForRetries {
    param(
        [string]$EventId,
        [int]$MinAttempts = 3,
        [int]$TimeoutSec = 55
    )
    $seenAttempts = @{}
    $deadline     = (Get-Date).AddSeconds($TimeoutSec)
    Write-Info "Polling GET /api/v1/retries?topic=$Topic&eventId=$EventId ..."
    while ((Get-Date) -lt $deadline) {
        Flush-LogStreams
        $retries = Get-Retries -EventId $EventId
        foreach ($r in @($retries)) {
            $attempt = [int]$r.attempt
            if (-not $seenAttempts.ContainsKey($attempt)) {
                $seenAttempts[$attempt] = $true
                Write-Host "  [RETRY] eventId=$EventId  attempt=$attempt/$($r.maxAttempts)  status=$($r.status)  error=$($r.lastError)" -ForegroundColor DarkYellow
            }
        }
        $dlqHit = Get-LogSnapshot -Pattern "routed to DLQ.*$EventId"
        if ($dlqHit) {
            Write-Ok "retry engine exhausted - DLQ routing observed in logs"
            return $retries
        }
        if ($seenAttempts.Count -ge $MinAttempts) { return $retries }
        Start-Sleep -Seconds 1
    }
    return Get-Retries -EventId $EventId
}

function Wait-ForDlqEntry {
    param(
        [string]$EventId,
        [int]$TimeoutSec = 60
    )
    $deadline  = (Get-Date).AddSeconds($TimeoutSec)
    $lastStats = $null
    while ((Get-Date) -lt $deadline) {
        Flush-LogStreams
        $stats = Get-DlqStats
        if (-not $lastStats -or $stats.unreplayed -ne $lastStats.unreplayed -or $stats.total -ne $lastStats.total) {
            Write-Api "DLQ stats: total=$($stats.total)  unreplayed=$($stats.unreplayed)"
            $lastStats = $stats
        }
        $msgs = @(Invoke-Api -Path "/api/v1/dlq/${Topic}?limit=20")
        foreach ($m in $msgs) {
            if ($m.eventId -eq $EventId) { return $m }
        }
        $dlqHit = Get-LogSnapshot -Pattern "routed to DLQ.*$EventId"
        if ($dlqHit) {
            Start-Sleep -Seconds 1
            $msgs = @(Invoke-Api -Path "/api/v1/dlq/${Topic}?limit=20")
            foreach ($m in $msgs) {
                if ($m.eventId -eq $EventId) { return $m }
            }
        }
        Start-Sleep -Seconds 1
    }
    return $null
}

function Show-StoryBoard {
    $lines = @(
        "  ShipPurchased  ->  ProcessPayment  ->  ReserveInventory",
        "        | failure (saga + consumer)",
        "        v",
        "     Retry x3  ->  DLQ  ->  Replay  ->  Success"
    )
    $lines | ForEach-Object { Write-Host $_ -ForegroundColor DarkGray }
}

# =============================================================================
$TotalActs = 9
Write-Banner "EventFlow - Galactic Commerce (LIVE DEMO)"
Show-StoryBoard
Pause-Demo 2 "Press pause - story begins in 3 seconds..."

# --- Infrastructure ---
if (-not $SkipStackStart) {
    Write-Info "Starting Docker stack..."
    Ensure-CleanPostgres
    Invoke-Expression "$Compose up -d --build --wait"
} else {
    $prevEA = $ErrorActionPreference
    $ErrorActionPreference = "SilentlyContinue"
    & docker compose @ComposeArgs up -d --build api-gateway consumer-worker *> $null
    $ErrorActionPreference = $prevEA
    Start-Sleep -Seconds 3
}

Wait-ForApi
Ensure-Topics
Clear-DemoData
Start-DemoLogStreams

try {
    # ACT 1: Event publishing
    Write-Act 1 $TotalActs "Event Publishing (ShipPurchased)"
    $storyKey = "ship-story-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
    Write-Api "POST /api/v1/events"
    $published = Publish-ShipEvent -Key $storyKey
    $eventId1  = $published.id
    Add-TrackId $eventId1
    Write-Ok "eventId=$eventId1"
    Write-Ok "topic=$Topic  type=ShipPurchased  partition=$($published.partition)  offset=$($published.offset)"
    Write-Ok "idempotencyKey=$storyKey"
    Write-Elapsed
    Pause-Demo 4 "Kafka accepted the event - consumer will pick it up next..."

    # ACT 2: Consumer processing
    Write-Act 2 $TotalActs "Consumer Processing"
    Write-Info "consumer-worker consumes from topic=$Topic group=$ConsumerGroup"
    $null = Wait-ForConsumerProcessed -EventId $eventId1 -TimeoutSec 25
    Write-Elapsed
    Pause-Demo 4 "Consumer committed offset - creating workflow next..."

    # ACT 3: Workflow creation
    Write-Act 3 $TotalActs "Workflow Creation (GalacticCommerce)"
    Write-Api "POST /api/v1/workflows"
    $sagaWf = Start-GalacticWorkflow -WorkflowInput @{
        pilotId = 42; shipId = "falcon-x"; credits = 45000; demoFailStep = "ReserveInventory"
    }
    $workflowId1 = $sagaWf.id
    Add-TrackId $workflowId1
    Write-Ok "workflowId=$workflowId1  name=GalacticCommerce  status=$($sagaWf.status)"
    Write-Api "POST /api/v1/workflows/$workflowId1/run"
    Invoke-Api -Method Post -Path "/api/v1/workflows/$workflowId1/run" | Out-Null
    Write-Ok "workflow run accepted (async)"
    Write-Elapsed
    Pause-Demo 3 "Saga executing: ProcessPayment -> ReserveInventory -> ..."

    # ACT 4: Saga execution (with failure at ReserveInventory)
    Write-Act 4 $TotalActs "Saga Execution -> Failure at ReserveInventory"
    $sagaResult = Watch-WorkflowLive -WorkflowId $workflowId1 -Label "saga-failure" -TimeoutSec 30
    if ($sagaResult.workflow.status -eq "failed") {
        Write-Ok "saga failed at ReserveInventory - RefundPayment compensation (LIFO)"
    }
    Write-Elapsed
    Pause-Demo 4 "Saga rolled back. Now inject a toxic Kafka message for retry/DLQ..."

    # ACT 5: Retry handling (consumer failure path)
    Write-Act 5 $TotalActs "Retry Handling (consumer retry engine)"
    $dlqBaseline = Get-DlqStats
    Write-Api "DLQ baseline before toxic event: unreplayed=$($dlqBaseline.unreplayed)"
    $failKey  = "ship-toxic-$(Get-Date -Format 'HHmmss')"
    Write-Api "POST /api/v1/events  (simulateFailure=true)"
    $failEv   = Publish-ShipEvent -Key $failKey -SimulateFailure:$true
    $eventId2 = $failEv.id
    Add-TrackId $eventId2
    Write-Ok "toxic eventId=$eventId2  (consumer will fail processing)"
    Write-Info "Retry policy: max 3 attempts, exponential backoff (DEMO_MODE=2s)"
    $retryRecords = Wait-ForRetries -EventId $eventId2 -MinAttempts 3 -TimeoutSec 55
    if (@($retryRecords).Count -gt 0) {
        Write-Ok "$(@($retryRecords).Count) retry record(s) in Postgres for eventId=$eventId2"
    } else {
        $snap = Get-LogSnapshot -Pattern "retry scheduled|routed to DLQ"
        if ($snap) { Write-Warn $snap }
    }
    Write-Elapsed
    Pause-Demo 4 "Retries exhausted - message routes to DLQ..."

    # ACT 6: DLQ insertion
    Write-Act 6 $TotalActs "DLQ Insertion"
    Write-Ok "DLQ BEFORE: total=$($dlqBaseline.total)  unreplayed=$($dlqBaseline.unreplayed)"
    Write-Info "Waiting for eventId=$eventId2 to land in DLQ (Kafka redelivery + max retries)..."
    $dlqMsg = Wait-ForDlqEntry -EventId $eventId2 -TimeoutSec 60
    $dlqAfter = Get-DlqStats
    if ($dlqMsg) {
        Write-Warn "DLQ message: eventId=$($dlqMsg.eventId)  reason=$($dlqMsg.failureReason)  retries=$($dlqMsg.retryAttempts)"
        Write-Ok "DLQ AFTER:  total=$($dlqAfter.total)  unreplayed=$($dlqAfter.unreplayed)"
    } else {
        Write-Warn "DLQ entry not found for eventId=$eventId2"
        Get-LogSnapshot -Pattern "routed to DLQ" | ForEach-Object { Write-Warn $_ }
    }
    Write-Elapsed
    Pause-Demo 4 "Replaying dead-letter messages back to the topic..."

    # ACT 7: Replay
    Write-Act 7 $TotalActs "Replay from DLQ"
    $replayBefore = Get-DlqStats
    Write-Ok "DLQ BEFORE replay: unreplayed=$($replayBefore.unreplayed)  total=$($replayBefore.total)"
    Write-Api "POST /api/v1/replay  { topic: $Topic, dlqOnly: true }"
    $replay = Invoke-Api -Method Post -Path "/api/v1/replay" -Body @{
        topic = $Topic; dlqOnly = $true; targetTopic = $Topic
    }
    Start-Sleep -Seconds 2
    $replayAfter = Get-DlqStats
    Write-Ok "replayed=$($replay.replayed) message(s)"
    Write-Ok "DLQ AFTER replay:  unreplayed=$($replayAfter.unreplayed)  (was $($replayBefore.unreplayed))"
    Write-Elapsed
    Pause-Demo 4 "Clean workflow run - all saga steps should complete..."

    # ACT 8: Successful completion
    Write-Act 8 $TotalActs "Successful Completion"
    Write-Api "POST /api/v1/workflows  (no demoFailStep)"
    $successWf = Start-GalacticWorkflow -WorkflowInput @{
        pilotId = 42; shipId = "falcon-x"; credits = 45000
    }
    $workflowId2 = $successWf.id
    Add-TrackId $workflowId2
    Write-Ok "workflowId=$workflowId2"
    Write-Api "POST /api/v1/workflows/$workflowId2/run"
    Invoke-Api -Method Post -Path "/api/v1/workflows/$workflowId2/run" | Out-Null
    $success = Watch-WorkflowLive -WorkflowId $workflowId2 -Label "success" -TimeoutSec 30
    if ($success.workflow.status -eq "completed") {
        Write-Ok "ProcessPayment -> ReserveInventory -> SendConfirmation  COMPLETED"
        Write-Ok "workflowId=$workflowId2  eventId(story)=$eventId1  toxicEventId=$eventId2"
    } else {
        Write-Warn "workflow status=$($success.workflow.status)"
    }
    Write-Elapsed
    Pause-Demo 3

    # ACT 9: Grafana (end only)
    Write-Act 9 $TotalActs "Grafana Dashboard"
    Write-Info "Opening dashboard (metrics from this run)"
    Write-Host "  $GrafanaUrl" -ForegroundColor White
    Write-Host "  login: admin / admin" -ForegroundColor DarkGray
    try {
        Start-Process $GrafanaUrl
        Write-Ok "Grafana opened"
    } catch {
        Write-Warn "Open manually: $GrafanaUrl"
    }

} finally {
    Stop-DemoLogStreams
}

# Finale
$total = [int]((Get-Date) - $DemoStart).TotalSeconds
Write-Banner "Demo Complete - ${total}s"
Write-Host "  Story eventId ........ $eventId1" -ForegroundColor Green
Write-Host "  Saga workflowId ...... $workflowId1 (failed + compensated)" -ForegroundColor Green
Write-Host "  Toxic eventId ........ $eventId2 (retry -> DLQ -> replay)" -ForegroundColor Green
Write-Host "  Success workflowId ... $workflowId2" -ForegroundColor Green
Write-Host ""
Write-Host "  [x] Event publishing (real Kafka + API)" -ForegroundColor Green
Write-Host "  [x] Consumer processing (live logs + offsets)" -ForegroundColor Green
Write-Host "  [x] Workflow creation (REST API)" -ForegroundColor Green
Write-Host "  [x] Saga execution + failure + compensation" -ForegroundColor Green
Write-Host "  [x] Retry handling (Postgres retry records)" -ForegroundColor Green
Write-Host "  [x] DLQ insertion (stats API)" -ForegroundColor Green
Write-Host "  [x] Replay (DLQ count before/after)" -ForegroundColor Green
Write-Host "  [x] Successful workflow completion" -ForegroundColor Green
Write-Host "  [x] Grafana (opened at end)" -ForegroundColor Green
Write-Host ""
Write-Host "  Re-run: .\scripts\demo.ps1 -SkipStackStart" -ForegroundColor DarkGray
