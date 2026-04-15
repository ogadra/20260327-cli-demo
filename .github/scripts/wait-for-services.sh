#!/usr/bin/env bash
# Wait for integration-e2e services to become ready.
# Called from .github/workflows/integration.yml.
set -euo pipefail

COMPOSE="docker compose --env-file .env.example --profile integration --profile front"

wait_with_retry() {
  local description="$1"
  local check_cmd="$2"
  local max_attempts="${3:-60}"
  local interval="${4:-3}"

  echo "Waiting for ${description}..."
  for i in $(seq 1 "$max_attempts"); do
    if eval "$check_cmd"; then
      echo "${description} ready"
      return 0
    fi
    if [ "$i" -eq "$max_attempts" ]; then
      echo "${description} did not become ready in time"
      $COMPOSE logs
      exit 1
    fi
    sleep "$interval"
  done
}

broker_ready() {
  curl -sf http://localhost:80/health > /dev/null 2>&1
}

runners_ready() {
  local logs count
  logs=$(docker compose --env-file .env.example --profile integration logs runner-1 runner-2 2>/dev/null)
  count=$(echo "$logs" | grep -c "server listening on" || true)
  [ "$count" -ge 2 ]
}

frontend_ready() {
  curl -sf http://localhost:5173 > /dev/null 2>&1
}

wait_with_retry "broker health via nginx" broker_ready 60 3 &
pid_broker=$!
wait_with_retry "both runners" runners_ready 60 3 &
pid_runners=$!
wait_with_retry "frontend" frontend_ready 30 3 &
pid_frontend=$!

fail=0
wait "$pid_broker" || fail=1
wait "$pid_runners" || fail=1
wait "$pid_frontend" || fail=1

if [ "$fail" -ne 0 ]; then
  exit 1
fi

echo "All services ready"
