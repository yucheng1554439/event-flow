#!/usr/bin/env bash
set -euo pipefail

API="${EVENTFLOW_API:-http://localhost:8080/api/v1}"

curl -s -X POST "$API/events" \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "orders",
    "eventType": "OrderCreated",
    "idempotencyKey": "ord-sample-001",
    "payload": {
      "eventType": "OrderCreated",
      "userId": 123,
      "amount": 45.99
    }
  }' | jq .
