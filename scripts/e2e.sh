#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$PROJECT_ROOT/docker-compose.yaml"
API_URL="http://localhost:8080"
TIMEOUT=30

cleanup() {
  echo "==> Tearing down..."
  docker compose -f "$COMPOSE_FILE" down -v 2>/dev/null || true
}
trap cleanup EXIT

echo "==> Building and starting API..."
docker compose -f "$COMPOSE_FILE" up -d --build

echo "==> Waiting for API to be healthy (${TIMEOUT}s timeout)..."
for i in $(seq 1 "$TIMEOUT"); do
  if curl -sf "$API_URL/health" > /dev/null 2>&1; then
    echo "    API ready after ${i}s"
    break
  fi
  if [ "$i" -eq "$TIMEOUT" ]; then
    echo "ERROR: API not ready after ${TIMEOUT}s"
    echo "==> Container logs:"
    docker compose -f "$COMPOSE_FILE" logs
    exit 1
  fi
  sleep 1
done

echo "==> Running e2e tests..."
(cd "$PROJECT_ROOT/e2e" && go test -v -count=1 ./...)
TEST_EXIT=$?

exit $TEST_EXIT
