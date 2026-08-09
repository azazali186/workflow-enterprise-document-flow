# AeroXe DocuFlow — Backend (Go)

Hertz + GORM (PostgreSQL) + Redis + NATS JetStream modular monolith implementing the
README business model: documents, verification, approval chains, versioning,
storage, templates, categories, access control, login/audit logs and reports —
with complete RBAC seeded from the live route table.

## Quick start

```bash
cd backend
docker compose up -d postgres redis nats   # or reuse a local stack
cp .env.example .env                       # fill DATABASE_URL, JWT_SECRET
make run                                    # migrates, seeds admin + RBAC, serves
```

- Swagger UI: `http://localhost:8080/swagger/index.html`
- Seed admin: `admin@aeroxe.io` / `ChangeMe123!` (env overridable)
- Regenerate docs after changing annotations: `make swagger`

> **Build tags**: the Makefile builds with `-tags "stdjson gjson"` to route Hertz
> off `bytedance/sonic`, which does not link against Go ≥ 1.24 (stdlib linkname
> restrictions) and has no 32-bit support. This also guarantees a pure-Go build.

## API conventions

- **Methods**: only `POST`, `PATCH`, `DELETE` — **no GET, no PUT**, no path
  variables, no query params. Every endpoint takes a JSON body (see `/swagger`).
- **Envelope**: `{"code":0,"message":"success","data":...}` — business code in
  body (0 = ok, else HTTP-style 4xx/5xx).
- **Pagination**: every `*/list` endpoint accepts `cursor`, `limit` (≤100),
  `filters` (whitelisted exact-match), `date_from`/`date_to`, `sort_by`,
  `sort_dir`, `search`. Response includes `items`, `pagination` (next_cursor,
  has_more, limit, returned_count, total_count) and entity-specific `summary`.
- **Auth**: `Authorization: Bearer <token>`. Single-sign-on via Redis
  (`admin_token:<user_id>`); sessions rotate and renew. Locked/deleted accounts
  lose access on the next request.
- **RBAC**: every route is scanned at startup and upserted as a permission
  (`METHOD path`). Assign permissions to roles, roles to users. `super_admin`
  bypasses all checks. `POST /api/v1/permissions/sync` re-seeds.
- **Row-level access**: documents are readable by their owner, by any active
  access grant (`/accesses/grant`, per-user or per-role, `read`/`write`/
  `approve`) and by `super_admin`. The same rule scopes `documents/list` and
  `search/documents` (including counts and summaries); `documents/update` and
  `documents/delete` additionally require owner, a `write`/`approve` grant or
  `super_admin`. Denials answer 404 so document existence is never revealed.

## Architecture

```
cmd/server → bootstrap (DI) → router
                                ├─ middleware: recovery, metrics, request-log,
                                │   rate-limit (Redis), CORS, auth (JWT+SSO), RBAC
                                ├─ handler: one file per entity (swagger annotated)
                                ├─ service: auth, rbac, document, verification,
                                │   approval, storage, search, analytics, report,
                                │   audit, generic crud
                                ├─ repository: generic GORM repo (cursor pagination,
                                │   filters, sort, summary) + per-entity config
                                ├─ model: UUID v7 PK, snake_case, soft delete
                                ├─ saga: Redis-backed state machine (the 4 README
                                │   sagas: document_upload, verification_process,
                                │   approval_chain, archival)
                                └─ outbox: transactional events → NATS JetStream
```

Cross-cutting (all GORM, no raw SQL):

| Concern | Where |
|---|---|
| Transactional outbox | `internal/pkg/outbox` → JetStream `docuflow.>` |
| Circuit breaker | `internal/pkg/breaker` (NATS publish) |
| Retry/backoff | `internal/pkg/retry` (exponential + jitter) |
| Distributed locking | `internal/pkg/lock` (verification/approval/outbox) |
| Distributed cache | `internal/pkg/cache` (sessions, permissions, entity cache) |
| Prometheus | `POST /metrics` (requests, latency, in-flight, outbox gauge) |
| Request tracing | `X-Request-ID` middleware → outbox events → worker logs |
| Body size limit | `server.WithMaxRequestBodySize` (env `MAX_BODY_SIZE_MB`) |
| Rate limiting | Redis fixed window, per user/IP (proxy-aware: `TRUSTED_PROXIES`) |
| Audit trail | before/after JSON, sensitive keys redacted |
| Login logs | success/failure with ip/user-agent/reason |
| Encryption at rest | AES-256-GCM for sensitive fields, bcrypt passwords |
| Row-level access | owner / active grant (user or role) / super_admin on document reads & writes |
| Business-code metrics | requests labelled by envelope `code` so error alerting can fire |
| Dead-letter queue | `DOCUFLOW_DLQ` JetStream stream for events the worker gives up on |

## Production checklist

**Done in code:** typed errors, no secrets in logs, request logging without
bodies, fail-fast config, env-tunable connection pooling
(`DB_MAX_OPEN_CONNS`/`DB_MAX_IDLE_CONNS`), graceful shutdown, idempotent
seed (the bootstrap admin is ensured by email even on a non-empty DB),
request tracing, JWT issuer + signing-method validation, client-supplied
object keys validated as safe relative paths (`filepath.IsLocal`), WebSocket
handshakes gated on the account status, metrics labelled by the envelope's
business code (so HTTP-level alerting sees 5xx), and a JetStream dead-letter
stream for events the worker gives up on. **Swagger UI is off by default in
production** — set `SWAGGER_ENABLED=true` to expose it (it is on in
development). **Versioned migrations** (golang-migrate, embedded SQL under
`internal/database/migrations`) replace AutoMigrate for Postgres — run
`make migrate-up` before deploy; sqlite test setups still use AutoMigrate.
**Production config is enforced at startup**: `ENV=production` rejects the
dev-default `JWT_SECRET`/`ENCRYPTION_KEY`/`ADMIN_PASSWORD` values and wildcard
`CORS_ORIGINS`; `TRUSTED_PROXIES` gates proxy headers. The worker consumes
`docuflow.>` JetStream events (durable, manual ack, dedupe) and drives the
upload saga with **real, pluggable integrations**: `VIRUS_SCANNER=clamav`
streams binaries to a clamd daemon (INSTREAM) and `INDEXER=opensearch`
publishes documents to a search cluster — both default to safe noops. A
failed saga is now terminal: an infected or checksum-less document is never
advanced or finalized.

**Ops checklist:**

- [ ] **Secrets**: provision real `JWT_SECRET`, `ENCRYPTION_KEY`,
      `ADMIN_PASSWORD` via a secret manager; never the compose defaults
- [ ] **Deployment**: container image + k8s manifests (incl. `ingress.yaml`
      with TLS) / compose `app` service with `ENV=production`; review
      `TRUSTED_PROXIES` CIDRs for your network
- [ ] **Pipeline integrations**: deploy clamd (`VIRUS_SCANNER=clamav`) and an
      OpenSearch cluster (`INDEXER=opensearch`) when those capabilities are
      required
- [ ] **S3 storage**: `storages/upload` writes to local disk; configure the
      S3-compatible endpoint env vars for object storage
- [ ] **Observability**: load `deploy/prometheus/alerting-rules.yml` into
      Prometheus, wire Alertmanager, ship logs
- [ ] **Load test**: `k6 run scripts/load.js`; verify pagination + summary
      query plans
- [ ] **Security review**: dependency updates (govulncheck runs in CI)

**CI runs:** vet, `-race` unit tests, image build, golangci-lint (config in
`.golangci.yml`, incl. gosec), govulncheck, CodeQL (`.github/workflows/codeql.yml`),
a Trivy container scan of the built image, Dependabot dependency updates
(`.github/dependabot.yml`), and an **integration job** that runs
`-tags integration` against real Postgres/Redis/NATS-JetStream service
containers (validating the versioned migrations and the outbox → JetStream →
worker pipeline).

## Tests

```bash
make test               # unit tests (sqlite in-memory + miniredis)
make test-integration   # requires real Postgres/Redis/NATS
make vet
make migrate-up         # versioned golang-migrate up + RBAC seed (Postgres)
bash scripts/smoke.sh   # end-to-end: throwaway infra + real server + API flow
```

## Search & analytics

Two dedicated README modules round out the service layer:

- **`SearchService`** — `POST /api/v1/search/documents`: keyword full-text
  search over documents (`search` term required) plus the same whitelisted
  filters (`status`, `category_id`, `owner_id`), sort, date range and cursor
  pagination as the list endpoints.
- **`AnalyticsService`** — three endpoints:
  - `POST /api/v1/analytics/documents` — documents by status, by category
    (uncategorized included) and the per-day creation trend
  - `POST /api/v1/analytics/storage` — total bytes, per-provider bytes and
    the per-day upload trend
  - `POST /api/v1/analytics/workflow` — zero-filled status funnel plus the
    pending verification / approval backlogs

All analytics queries are portable GORM (no SQL date functions), so they run
unchanged on Postgres and the sqlite test harness.

## Sagas

All four README sagas are orchestrated (Redis state + NATS events):

| Saga | Started | Steps |
|---|---|---|
| `document_upload` | on document create | upload → virus_scan → metadata_extraction → storage → indexing (worker-driven) |
| `verification_process` | on verification create | authenticity_check → document_update (completed on decision) |
| `approval_chain` | on approval chain create | routing (immediate) → decision (completed when every level is decided) |
| `archival` | on archive status transition | archive (completed synchronously) |

Saga advancement on the synchronous domains (verification, approval,
archival) is best-effort: a decision on a legacy record without a saga never
fails the request.

## Request tracing

Every HTTP request gets a `X-Request-ID` (forwarded when the client supplies a
valid one, otherwise generated). The id is echoed on the response, added to
access logs, stored on transactional outbox rows (`000002_outbox_trace_id`)
and carried into worker processing logs, so a single API call can be followed
through the whole async pipeline.

```bash
curl -H 'X-Request-ID: my-trace-1' -X POST .../api/v1/auth/login | jq -r '.message'
# worker logs will carry request_id=my-trace-1
```

## Database migrations

Migrations live in `internal/database/migrations` as plain SQL and run through
golang-migrate (`cmd/migrate`). The server applies pending migrations on
startup in Postgres mode (idempotent) and `make migrate-up` does the same
standalone:

```bash
make migrate-up                     # up + seed RBAC/admin
go run cmd/migrate/main.go down     # roll back one migration
go run cmd/migrate/main.go version  # show current schema version
```

For local sqlite (unit tests) AutoMigrate remains the mechanism.

> **Note**: databases first created by AutoMigrate (before this change) are
> missing the foreign keys the SQL migration defines; `CREATE TABLE IF NOT
> EXISTS` skips them silently. Recreate dev schemas with `make migrate-up`
> (drop the DB or run `down` then `up`) so dev and prod stay in sync.

Coverage is high on the core pkgs (breaker 84%, jwt 84%, pagination 83%,
retry 80%, crypto 75%); service/repository layers are covered by sqlite-backed
tests, handler layer via Hertz `ut.PerformRequest`.
