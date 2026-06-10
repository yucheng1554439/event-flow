#!/usr/bin/env bash
set -euo pipefail

API="${EVENTFLOW_API:-http://localhost:8080}"
COMPOSE="docker compose -f docker/docker-compose.yml -f docker/docker-compose.demo.yml"

# Detect curl (Git Bash on Windows may lack it; fall back to wget)
http_ok() {
  local url="$1"
  if command -v curl >/dev/null 2>&1; then
    curl -sf "$url" >/dev/null 2>&1
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O /dev/null "$url" 2>/dev/null
  else
    # PowerShell fallback for Windows without curl
    powershell.exe -NoProfile -Command \
      "try { (Invoke-WebRequest -Uri '$url' -UseBasicParsing -TimeoutSec 3).StatusCode -eq 200 } catch { \$false }" \
      2>/dev/null | grep -qi true
  fi
}

ensure_clean_postgres() {
  # Postgres init scripts run only once; a failed init leaves a broken volume.
  local pg_status
  pg_status=$($COMPOSE ps postgres --format json 2>/dev/null | head -1 || true)
  if echo "$pg_status" | grep -q '"State":"exited"'; then
    echo "Postgres failed previously — resetting database volume..."
    $COMPOSE down -v
    return 0
  fi
  if ! $COMPOSE ps postgres 2>/dev/null | grep -qE 'healthy|running|Up'; then
    if docker volume ls 2>/dev/null | grep -qE 'pgdata|docker.*pgdata'; then
      echo "Postgres not running — resetting database volume for clean init..."
      $COMPOSE down -v
    fi
  fi
}

wait_for_api() {
  echo "Waiting for EventFlow API at ${API}/healthz ..."

  # Wait for infrastructure healthchecks (compose blocks api-gateway until healthy)
  local max_attempts=90
  local attempt=0

  while [ "$attempt" -lt "$max_attempts" ]; do
    attempt=$((attempt + 1))

    if http_ok "${API}/healthz"; then
      echo "API ready (attempt ${attempt})."
      return 0
    fi

    # If api-gateway crashed (e.g. race before healthchecks), restart app services
    if [ $((attempt % 15)) -eq 0 ]; then
      echo "  still waiting... (attempt ${attempt}/${max_attempts}) — restarting app services"
      $COMPOSE up -d api-gateway consumer-worker workflow-engine 2>/dev/null || true
    elif [ $((attempt % 5)) -eq 0 ]; then
      echo "  still waiting... (attempt ${attempt}/${max_attempts})"
    fi

    sleep 2
  done

  echo "" >&2
  echo "ERROR: API not ready after $((max_attempts * 2))s" >&2
  echo "Polling: ${API}/healthz" >&2
  echo "" >&2
  echo "Container status:" >&2
  $COMPOSE ps -a >&2 || true
  echo "" >&2
  echo "Recent logs:" >&2
  $COMPOSE logs --tail=15 postgres api-gateway 2>&1 >&2 || true
  echo "" >&2
  echo "Fix: docker compose -f docker/docker-compose.yml -f docker/docker-compose.demo.yml down -v && ./scripts/demo.sh" >&2
  exit 1
}

create_demo_topics() {
  curl -sf -X POST "$API/api/v1/topics" -H "Content-Type: application/json" \
    -d '{"name":"ship-orders","partitions":6,"replicationFactor":1,"retentionHours":168,"cleanupPolicy":"delete"}' || true
  curl -sf -X POST "$API/api/v1/topics" -H "Content-Type: application/json" \
    -d '{"name":"ship-orders-dlq","partitions":3,"replicationFactor":1,"retentionHours":2160,"cleanupPolicy":"delete"}' || true
}

publish_ship_event() {
  local key="${1:-ship-demo-$(date +%s)}"
  curl -sf -X POST "$API/api/v1/events" -H "Content-Type: application/json" -d "{
    \"topic\": \"ship-orders\",
    \"eventType\": \"ShipPurchased\",
    \"idempotencyKey\": \"$key\",
    \"payload\": {
      \"pilotId\": 42,
      \"shipId\": \"falcon-x\",
      \"credits\": 45000
    }
  }"
}

print_urls() {
  cat <<EOF

REST API:
  $API

Grafana:
  http://localhost:3000  (admin / admin)

Prometheus:
  http://localhost:9091

Workflow Engine:
  http://localhost:8081/healthz

Consumer Metrics:
  http://localhost:8082/metrics

EOF
}
