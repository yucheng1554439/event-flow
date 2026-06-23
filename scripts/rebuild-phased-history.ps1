# Rebuild git history as phased commits without co-author trailers.
# Usage: .\scripts\rebuild-phased-history.ps1 [-Force]

param([switch]$Force)

$ErrorActionPreference = "Stop"
$WarningPreference = "SilentlyContinue"
$git = "C:\Program Files\Git\mingw64\bin\git.exe"
$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

function New-CleanCommit {
    param(
        [string]$Subject,
        [string]$Body,
        [string]$Parent
    )
    $tree = (& $git write-tree).Trim()
    if (-not $tree) { throw "write-tree failed" }

    $ts = [int][double]::Parse((Get-Date -UFormat %s))
    $identity = "yucheng1554439 <zyucheng2004@gmail.com> $ts -0700"

    $lines = New-Object System.Collections.Generic.List[string]
    $lines.Add("tree $tree")
    if ($Parent) { $lines.Add("parent $Parent") }
    $lines.Add("author $identity")
    $lines.Add("committer $identity")
    $lines.Add("")
    $lines.Add($Subject)
    if ($Body) {
        $lines.Add("")
        $lines.Add($Body)
    }

    $commitFile = Join-Path $repoRoot ".git\PHASE_COMMIT_OBJ"
    [System.IO.File]::WriteAllText($commitFile, ($lines -join "`n") + "`n", [System.Text.UTF8Encoding]::new($false))
    $hash = (& $git hash-object -w -t commit $commitFile).Trim()
    if (-not $hash) { throw "hash-object failed" }
    return $hash
}

$phase1 = @(
    ".gitignore", "go.mod", "go.sum", "LICENSE", "Makefile",
    "cmd/api-gateway", "cmd/consumer-worker", "cmd/workflow-engine",
    "internal/api", "internal/storage", "internal/workflow",
    "pkg/config", "pkg/kafka/producer.go", "pkg/kafka/consumer.go", "pkg/models",
    "api/openapi",
    "migrations/001_initial_schema.sql",
    "docker/Dockerfile.api-gateway", "docker/Dockerfile.consumer-worker", "docker/Dockerfile.workflow-engine",
    "docker/docker-compose.yml",
    "deployments/k8s",
    "scripts/seed-topics.sh", "scripts/publish-sample.sh"
)

$phase2 = @(
    "internal/grpc", "internal/topic", "internal/retry", "internal/replay",
    "pkg/kafka/admin.go", "pkg/metrics",
    "api/proto", "api/gen",
    "migrations/002_add_cleanup_policy.sql",
    "helm",
    "terraform/modules", "terraform/environments/dev/main.tf", "terraform/environments/dev/variables.tf",
    "docker/prometheus.yml",
    "deployments/monitoring/grafana/dashboards/eventflow.json",
    "deployments/monitoring/grafana/provisioning",
    "scripts/generate-proto.ps1",
    "tests/load"
)

$demo = @(
    "cmd/demo-generator",
    "docker/docker-compose.demo.yml",
    "scripts/demo.ps1", "scripts/demo.sh", "scripts/demo-common.sh",
    "scripts/demo-failure.sh", "scripts/demo-replay.sh", "scripts/capture-demo-screenshots.ps1",
    "deployments/monitoring/grafana/dashboards/eventflow-demo.json"
)

$ci = @(
    ".github/workflows",
    "tests/integration",
    "pkg/kafka/testutil.go",
    "terraform/environments/dev/.terraform.lock.hcl"
)

$docs = @(
    "README.md", "CHANGELOG.md", "CONTRIBUTING.md", "docs",
    "scripts/render-diagrams.ps1", "scripts/rebuild-phased-history.ps1"
)

$commits = @(
    @{
        Paths = $phase1
        Subject = "feat(phase-1): core event platform foundation"
        Body = @"
Kafka ingestion, REST API gateway, consumer workers, PostgreSQL/Redis state,
OrderFulfillment saga workflows, Docker Compose, and Kubernetes base manifests.
"@
        Tag = "v0.1.0"
    },
    @{
        Paths = $phase2
        Subject = "feat(phase-2): gRPC, topics, retry/DLQ/replay, Helm, Terraform, observability"
        Body = @"
Protobuf/gRPC services, dynamic topic administration, exponential backoff retries,
DLQ and replay APIs, Helm chart, AWS Terraform modules, Prometheus metrics, and Grafana.
"@
        Tag = "v0.2.0"
    },
    @{
        Paths = $demo
        Subject = "feat(demo): Galactic Commerce live demo system"
        Body = @"
Nine-act demo scripts, demo-generator, ship-orders topic overlay, GalacticCommerce saga,
and demo Grafana dashboard.
"@
        Tag = "v0.3.0"
    },
    @{
        Paths = $ci
        Subject = "ci: GitHub Actions pipeline and integration test suite"
        Body = @"
Lint, unit, integration, Docker build, Helm validate, and Terraform validate jobs
with Testcontainers harness and provider lock file.
"@
        Tag = "v0.4.0"
    },
    @{
        Paths = $docs
        Subject = "docs: portfolio documentation, diagrams, and v1.0.0 release"
        Body = @"
README, architecture diagrams, recruiter materials, demo docs, CHANGELOG, and CONTRIBUTING guide.
"@
        Tag = "v1.0.0"
    }
)

if (-not $Force) {
    Write-Host "This rewrites main into 5 phased commits and creates release tags."
    Write-Host "A backup branch 'backup-before-phased-history' will be created."
    Write-Host "Re-run with -Force to execute."
    exit 0
}

& $git branch -f backup-before-phased-history main 2>$null
& $git checkout --orphan phased-main
& $git rm -rf --cached . 2>$null | Out-Null

$parent = $null
$tagMap = @{}

foreach ($c in $commits) {
    foreach ($p in $c.Paths) {
        $prev = $ErrorActionPreference
        $ErrorActionPreference = "Continue"
        & $git add -- $p 2>&1 | Out-Null
        $ErrorActionPreference = $prev
        if ($LASTEXITCODE -ne 0) { throw "git add failed for $p" }
    }
    $hash = New-CleanCommit -Subject $c.Subject -Body $c.Body -Parent $parent
    & $git reset --hard $hash | Out-Null
    $tagMap[$c.Tag] = $hash
    $parent = $hash
    Write-Host "Created $($c.Tag): $hash - $($c.Subject)"
}

foreach ($tag in $tagMap.Keys) {
    & $git tag -f $tag $tagMap[$tag]
}

& $git branch -M main
Write-Host ""
Write-Host "Done. Tags: $($tagMap.Keys -join ', ')"
Write-Host "Backup: backup-before-phased-history"
Write-Host "Push: git push --force-with-lease origin main; git push --force origin --tags"
