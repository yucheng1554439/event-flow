$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$protoc = Get-Command protoc -ErrorAction SilentlyContinue
if (-not $protoc) {
    Write-Host "protoc not found; using Docker..."
    docker run --rm `
        -v "${root}:/workspace" `
        -w /workspace `
        namely/protoc-all:1.51_1 `
        -I api/proto `
        --go_out=api/gen/go --go_opt=paths=source_relative `
        --go-grpc_out=api/gen/go --go-grpc_opt=paths=source_relative `
        api/proto/eventflow/v1/common.proto `
        api/proto/eventflow/v1/topic.proto `
        api/proto/eventflow/v1/event.proto `
        api/proto/eventflow/v1/replay.proto `
        api/proto/eventflow/v1/workflow.proto
} else {
    protoc -I api/proto `
        --go_out=api/gen/go --go_opt=paths=source_relative `
        --go-grpc_out=api/gen/go --go-grpc_opt=paths=source_relative `
        api/proto/eventflow/v1/common.proto `
        api/proto/eventflow/v1/topic.proto `
        api/proto/eventflow/v1/event.proto `
        api/proto/eventflow/v1/replay.proto `
        api/proto/eventflow/v1/workflow.proto
}
Write-Host "Proto generation complete."
