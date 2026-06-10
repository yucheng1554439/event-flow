#!/usr/bin/env bash
# Galactic Commerce — failure, retry, compensation, DLQ demo
set -euo pipefail
cd "$(dirname "$0")/.."
source scripts/demo-common.sh

wait_for_api

echo "╔══════════════════════════════════════════╗"
echo "║   Demo: Failure → Retry → DLQ → Saga     ║"
echo "╚══════════════════════════════════════════╝"
echo ""

# --- Part 1: Consumer retry → DLQ ---
echo "▶ Publishing event with simulateFailure=true (consumer will fail)..."
FAIL_KEY="ship-fail-$(date +%s)"
curl -sf -X POST "$API/api/v1/events" -H "Content-Type: application/json" -d "{
  \"topic\": \"ship-orders\",
  \"eventType\": \"ShipPurchased\",
  \"idempotencyKey\": \"$FAIL_KEY\",
  \"payload\": {
    \"pilotId\": 99,
    \"shipId\": \"broken-wing\",
    \"credits\": 45000,
    \"simulateFailure\": true
  }
}" | head -c 120
echo ""
echo ""

echo "▶ Watch consumer retries (Retry 1 → 2 → 3 → DLQ)..."
for attempt in 1 2 3; do
  echo "  Retry $attempt — waiting for retry engine..."
  sleep 3
done
sleep 4

echo ""
echo "▶ DLQ contents:"
curl -sf "$API/api/v1/dlq/ship-orders?limit=5" | python3 -m json.tool 2>/dev/null || curl -sf "$API/api/v1/dlq/ship-orders?limit=5"
echo ""

# --- Part 2: Workflow saga failure + compensation ---
echo "▶ Starting GalacticCommerce workflow with ReserveInventory failure..."
WF=$(curl -sf -X POST "$API/api/v1/workflows" -H "Content-Type: application/json" -d '{
  "name": "GalacticCommerce",
  "input": {
    "pilotId": 42,
    "shipId": "falcon-x",
    "credits": 45000,
    "demoFailStep": "ReserveInventory"
  }
}')
WF_ID=$(echo "$WF" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "$WF" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "  Workflow ID: $WF_ID"

curl -sf -X POST "$API/api/v1/workflows/$WF_ID/run" >/dev/null
sleep 4

echo ""
echo "▶ Workflow result (expect compensating → RefundPayment):"
curl -sf "$API/api/v1/workflows/$WF_ID" | python3 -m json.tool 2>/dev/null || curl -sf "$API/api/v1/workflows/$WF_ID"
echo ""

echo "▶ Service logs (compensation + DLQ routing):"
echo "--- consumer-worker ---"
docker compose -f docker/docker-compose.yml -f docker/docker-compose.demo.yml logs --tail=15 consumer-worker 2>/dev/null || true
echo "--- api-gateway (workflow) ---"
docker compose -f docker/docker-compose.yml -f docker/docker-compose.demo.yml logs --tail=10 api-gateway 2>/dev/null || true
echo ""
echo "Compensation: RefundPayment executed (see logs: executing compensation)"
echo "DLQ topic: ship-orders-dlq"
echo ""
echo "Next: ./scripts/demo-replay.sh"
