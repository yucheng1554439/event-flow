#!/usr/bin/env bash
# Galactic Commerce — replay DLQ events and verify success
set -euo pipefail
cd "$(dirname "$0")/.."
source scripts/demo-common.sh

wait_for_api

echo "╔══════════════════════════════════════════╗"
echo "║   Demo: DLQ Replay → Success             ║"
echo "╚══════════════════════════════════════════╝"
echo ""

echo "▶ DLQ before replay:"
BEFORE=$(curl -sf "$API/api/v1/dlq/ship-orders?limit=10")
echo "$BEFORE" | python3 -m json.tool 2>/dev/null || echo "$BEFORE"
echo ""

echo "▶ Replaying events from DLQ to ship-orders..."
REPLAY=$(curl -sf -X POST "$API/api/v1/replay" -H "Content-Type: application/json" \
  -d '{"topic":"ship-orders","dlqOnly":true,"targetTopic":"ship-orders"}')
echo "$REPLAY"
echo ""

echo "▶ Starting successful GalacticCommerce workflow (no failure flags)..."
WF=$(curl -sf -X POST "$API/api/v1/workflows" -H "Content-Type: application/json" -d '{
  "name": "GalacticCommerce",
  "input": {
    "pilotId": 42,
    "shipId": "falcon-x",
    "credits": 45000
  }
}')
WF_ID=$(echo "$WF" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "$WF" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
curl -sf -X POST "$API/api/v1/workflows/$WF_ID/run" >/dev/null
sleep 4

echo "▶ Workflow after replay scenario:"
RESULT=$(curl -sf "$API/api/v1/workflows/$WF_ID")
echo "$RESULT" | python3 -m json.tool 2>/dev/null || echo "$RESULT"

STATUS=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin)['workflow']['status'])" 2>/dev/null || echo "unknown")
echo ""
if [ "$STATUS" = "completed" ]; then
  echo "✓ Workflow succeeded: ProcessPayment → ReserveInventory → SendConfirmation"
else
  echo "Workflow status: $STATUS (check Grafana for metrics)"
fi
echo ""
print_urls
echo "Replay demo complete."
