#!/usr/bin/env bash
# End-to-end smoke test: boots throwaway Postgres/Redis/NATS, runs migrations,
# starts the real server, and exercises healthz → login → document create/list.
#
# Usage:  bash scripts/smoke.sh
# Ports can be overridden via env (PG_PORT, REDIS_PORT, NATS_PORT, HTTP_PORT).
set -euo pipefail

cd "$(dirname "$0")/.."

PG_PORT="${PG_PORT:-5435}"
REDIS_PORT="${REDIS_PORT:-6381}"
NATS_PORT="${NATS_PORT:-4225}"
NATS_MON_PORT="${NATS_MON_PORT:-8225}"
HTTP_PORT="${HTTP_PORT:-18080}"

PG_NAME="smoke-pg"
REDIS_NAME="smoke-redis"
NATS_NAME="smoke-nats"
SERVER_PID=""

# json_field JSON key → value (first occurrence; handles nested objects).
json_field() {
  echo "$1" | grep -o "\"$2\":\"*[^\",]*" | head -1 | sed -E "s/\"$2\":\"*([^\",]*)/\1/"
}

cleanup() {
  [ -n "$SERVER_PID" ] && kill "$SERVER_PID" 2>/dev/null || true
  docker rm -f "$PG_NAME" "$REDIS_NAME" "$NATS_NAME" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "==> booting infra"
# Docker Desktop's port proxy can briefly hold a just-freed port, so retry.
start_container() { # name, then args for `docker run`
  local name="$1"; shift
  for _ in 1 2 3; do
    docker rm -f "$name" >/dev/null 2>&1 || true
    if docker run -d --rm --name "$name" "$@" >/dev/null 2>&1; then return 0; fi
    sleep 3
  done
  echo "FAIL: could not start $name" >&2
  return 1
}

start_container "$PG_NAME"   -e POSTGRES_DB=docu_flow_smoke -e POSTGRES_USER=aeroxe -e POSTGRES_PASSWORD=secret -p "$PG_PORT:5432" postgres:15-alpine
start_container "$REDIS_NAME" -p "$REDIS_PORT:6379" redis:7-alpine
start_container "$NATS_NAME"  -p "$NATS_PORT:4222" -p "$NATS_MON_PORT:8222" nats:2.10-alpine -js -m 8222

echo "==> waiting for infra"
for _ in $(seq 1 60); do
  docker exec "$PG_NAME" pg_isready -U aeroxe -d docu_flow_smoke >/dev/null 2>&1 \
    && docker exec "$REDIS_NAME" redis-cli ping >/dev/null 2>&1 \
    && curl -sf "http://localhost:${NATS_MON_PORT}/healthz" >/dev/null 2>&1 && break
  sleep 1
done

export DATABASE_URL="postgres://aeroxe:secret@localhost:${PG_PORT}/docu_flow_smoke?sslmode=disable"
export REDIS_URL="redis://localhost:${REDIS_PORT}"
export NATS_URL="nats://localhost:${NATS_PORT}"
export JWT_SECRET="smoke-test-secret-0123456789abcdef0123456789"
export ENV="development"
export PORT="$HTTP_PORT"

echo "==> applying versioned migrations"
GOARCH="${GOARCH:-amd64}" CGO_ENABLED=0 go run cmd/migrate/main.go up

echo "==> building and starting server"
GOARCH="${GOARCH:-amd64}" CGO_ENABLED=0 go build -o bin/docuflow-server-smoke ./cmd/server

# Docker Desktop's port proxy can briefly flap right after a build; retry the
# server process (it dies on bootstrap failure) until it stays healthy.
SERVER_OK=""
for attempt in 1 2 3; do
  ./bin/docuflow-server-smoke & SERVER_PID=$!
  for _ in $(seq 1 45); do
    if ! kill -0 "$SERVER_PID" 2>/dev/null; then break; fi # process died → retry
    code=$(curl -s -X POST "http://localhost:${HTTP_PORT}/api/v1/healthz" -H 'Content-Type: application/json' -d '{}' 2>/dev/null | sed -n 's/.*"code":\([0-9]*\).*/\1/p' || true)
    [ "$code" = "0" ] && { SERVER_OK=1; break 2; }
    sleep 1
  done
  echo "    server attempt $attempt failed, retrying"
  kill "$SERVER_PID" 2>/dev/null || true
  sleep 3
  SERVER_PID=""
done
[ -n "$SERVER_OK" ] || { echo "FAIL: server never became healthy"; exit 1; }
echo "    healthz OK"

echo "==> login as seed admin"
login=$(curl -s -X POST "http://localhost:${HTTP_PORT}/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@aeroxe.io","password":"ChangeMe123!"}')
code=$(json_field "$login" code)
token=$(json_field "$login" token)
[ "$code" = "0" ] && [ -n "$token" ] || { echo "FAIL: login failed: $login"; exit 1; }
echo "    login OK"

echo "==> create document"
create=$(curl -s -X POST "http://localhost:${HTTP_PORT}/api/v1/documents/create" \
  -H 'Content-Type: application/json' -H "Authorization: Bearer $token" \
  -d '{"title":"Smoke Test Document","description":"proves the stack"}')
code=$(json_field "$create" code)
doc_id=$(json_field "$create" id)
[ "$code" = "0" ] || { echo "FAIL: create failed: $create"; exit 1; }
echo "    create OK (id=$doc_id)"

echo "==> list documents"
list=$(curl -s -X POST "http://localhost:${HTTP_PORT}/api/v1/documents/list" \
  -H 'Content-Type: application/json' -H "Authorization: Bearer $token" \
  -d '{"limit":10}')
code=$(json_field "$list" code)
[ "$code" = "0" ] || { echo "FAIL: list failed: $list"; exit 1; }
echo "    list OK"

echo "==> search documents"
search=$(curl -s -X POST "http://localhost:${HTTP_PORT}/api/v1/search/documents" \
  -H 'Content-Type: application/json' -H "Authorization: Bearer $token" \
  -d '{"search":"smoke","limit":10}')
code=$(json_field "$search" code)
[ "$code" = "0" ] || { echo "FAIL: search failed: $search"; exit 1; }
echo "    search OK"

echo "==> analytics endpoints"
for ep in documents storage workflow; do
  an=$(curl -s -X POST "http://localhost:${HTTP_PORT}/api/v1/analytics/$ep" \
    -H 'Content-Type: application/json' -H "Authorization: Bearer $token" -d '{}')
  code=$(json_field "$an" code)
  [ "$code" = "0" ] || { echo "FAIL: analytics/$ep failed: $an"; exit 1; }
  echo "    analytics/$ep OK"
done

echo "==> verification workflow (verification_process saga)"
ver=$(curl -s -X POST "http://localhost:${HTTP_PORT}/api/v1/verifications/create" \
  -H 'Content-Type: application/json' -H "Authorization: Bearer $token" \
  -d "{\"document_id\":\"$doc_id\"}")
code=$(json_field "$ver" code)
ver_id=$(json_field "$ver" id)
[ "$code" = "0" ] || { echo "FAIL: verification create failed: $ver"; exit 1; }
decided=$(curl -s -X POST "http://localhost:${HTTP_PORT}/api/v1/verifications/decide" \
  -H 'Content-Type: application/json' -H "Authorization: Bearer $token" \
  -d "{\"verification_id\":\"$ver_id\",\"decision\":\"verified\"}")
code=$(json_field "$decided" code)
[ "$code" = "0" ] || { echo "FAIL: verification decide failed: $decided"; exit 1; }
echo "    verification saga OK"

echo "==> approval workflow (approval_chain saga)"
me=$(curl -s -X POST "http://localhost:${HTTP_PORT}/api/v1/auth/me" \
  -H 'Content-Type: application/json' -H "Authorization: Bearer $token" -d '{}')
admin_id=$(json_field "$me" id)
chain=$(curl -s -X POST "http://localhost:${HTTP_PORT}/api/v1/approvals/create" \
  -H 'Content-Type: application/json' -H "Authorization: Bearer $token" \
  -d "{\"document_id\":\"$doc_id\",\"approver_ids\":[\"$admin_id\"]}")
code=$(json_field "$chain" code)
appr_id=$(json_field "$chain" id)
[ "$code" = "0" ] || { echo "FAIL: approval create failed: $chain"; exit 1; }
decided=$(curl -s -X POST "http://localhost:${HTTP_PORT}/api/v1/approvals/decide" \
  -H 'Content-Type: application/json' -H "Authorization: Bearer $token" \
  -d "{\"approval_id\":\"$appr_id\",\"decision\":\"approved\"}")
code=$(json_field "$decided" code)
[ "$code" = "0" ] || { echo "FAIL: approval decide failed: $decided"; exit 1; }
echo "    approval saga OK"

echo "==> archive document (archival saga)"
archived=$(curl -s -X PATCH "http://localhost:${HTTP_PORT}/api/v1/documents/update" \
  -H 'Content-Type: application/json' -H "Authorization: Bearer $token" \
  -d "{\"id\":\"$doc_id\",\"status\":\"archived\"}")
code=$(json_field "$archived" code)
[ "$code" = "0" ] || { echo "FAIL: archive failed: $archived"; exit 1; }
echo "    archival OK"

echo "==> readiness"
ready=$(curl -s -X POST "http://localhost:${HTTP_PORT}/api/v1/readyz" -H 'Content-Type: application/json' -d '{}')
echo "    $ready"
[ "$(json_field "$ready" code)" = "0" ] || { echo "FAIL: readiness failed: $ready"; exit 1; }

echo ""
echo "SMOKE TEST PASSED ✔"
