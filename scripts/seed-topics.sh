#!/usr/bin/env bash
set -euo pipefail

API="${EVENTFLOW_API:-http://localhost:8080/api/v1}"

for topic in orders payments notifications analytics; do
  curl -s -X POST "$API/topics" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"$topic\",\"partitions\":6,\"replication\":1,\"retentionHours\":168}" || true
  curl -s -X POST "$API/topics" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"${topic}-dlq\",\"partitions\":3,\"replication\":1,\"retentionHours\":2160}" || true
done

echo "Topics seeded."
