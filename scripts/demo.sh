#!/usr/bin/env bash
# Galactic Commerce — happy-path demo (3–5 min presentation)
set -euo pipefail
cd "$(dirname "$0")/.."
source scripts/demo-common.sh

echo "╔══════════════════════════════════════════╗"
echo "║   EventFlow — Galactic Commerce Demo     ║"
echo "╚══════════════════════════════════════════╝"
echo ""

ensure_clean_postgres

echo "Starting EventFlow stack..."
$COMPOSE up -d --build --wait 2>/dev/null || $COMPOSE up -d --build

wait_for_api

echo "Creating demo topics (ship-orders, ship-orders-dlq)..."
create_demo_topics

echo ""
echo "Publishing ShipPurchased Event..."
RESP=$(publish_ship_event "ship-123-$(date +%s)")
echo "$RESP" | head -c 200
echo ""
echo ""

echo "Workflow Started (GalacticCommerce via consumer)..."
sleep 3

echo "Generating metrics burst..."
for i in $(seq 1 20); do
  publish_ship_event "ship-metrics-$i-$(date +%s)" >/dev/null 2>&1 || true
  sleep 0.1
done

echo ""
echo "Demo Started"
print_urls
echo "Publishing ShipPurchased Event... ✓"
echo "Workflow Started... ✓"
echo ""
echo "Demo Ready — open Grafana → EventFlow Demo Dashboard"
echo "Next: ./scripts/demo-failure.sh  |  ./scripts/demo-replay.sh"
