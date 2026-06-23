#!/usr/bin/env bash
# Mirror GitHub Actions CI locally. Requires: go 1.22+, docker, terraform, helm.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "==> go vet"
CGO_ENABLED=1 go vet ./...

echo "==> unit tests"
CGO_ENABLED=1 go test ./internal/... ./pkg/... -count=1 -race -timeout 5m

echo "==> integration tests (requires Docker)"
CGO_ENABLED=1 go test -tags=integration ./tests/integration/... -count=1 -timeout 15m

echo "==> docker build"
docker build -f docker/Dockerfile.api-gateway -t eventflow/api-gateway:ci .
docker build -f docker/Dockerfile.consumer-worker -t eventflow/consumer-worker:ci .
docker build -f docker/Dockerfile.workflow-engine -t eventflow/workflow-engine:ci .

echo "==> helm validate"
helm lint ./helm/eventflow
helm template eventflow ./helm/eventflow > /dev/null

echo "==> terraform validate"
terraform fmt -check -recursive terraform/
(cd terraform/environments/dev && terraform init -backend=false -input=false && terraform validate -no-color)

echo "All CI checks passed."
