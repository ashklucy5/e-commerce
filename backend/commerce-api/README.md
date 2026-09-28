Commerce API

Production-oriented B2B e-commerce backend written in Go with PostgreSQL, Redis, Atlas migrations, background workers, and provider-neutral integration seams.

This repository is the backend freeze candidate. Feature development is complete; remaining work should be limited to defect fixes, final validation, and deployment/integration configuration.

Runtime stack

Go 1.26.4

Gin HTTP router

PostgreSQL 18

pgvector

Redis

pgx/pgxpool

Atlas migration library and Atlas CLI tooling

S3-compatible object storage (optional)

Main processes

The production runtime is split into three long-running programs:

cmd/api        HTTP API; applies embedded Atlas migrations on startup
cmd/scheduler  Periodically enqueues expiry, reconciliation, and notification jobs
cmd/worker     Consumes Redis jobs and runs payment/notification/order handlers

Run them in separate terminals:

go run ./cmd/api
go run ./cmd/scheduler
go run ./cmd/worker

The API health endpoints are:

GET /health/live
GET /health/ready

/health/ready checks both PostgreSQL and Redis.

Local setup

Install Go 1.26.4, PostgreSQL 18, Redis, and the pgvector PostgreSQL extension.

Copy .env.example to .env and set the required local values.

Create the PostgreSQL database/user represented by the configured POSTGRES_* variables.

Start Redis.

Start the API. The API validates and applies embedded Atlas migrations before serving traffic.

Start the scheduler and worker for background processing.

Verify GET /health/ready returns HTTP 200.

PowerShell example:

Copy-Item .env.example .env
go run ./cmd/api

In two additional terminals:

go run ./cmd/scheduler
go run ./cmd/worker

Docker Compose

The backend can run as a five-service Docker Compose stack:

postgres    PostgreSQL 18 with pgvector and pg_trgm
redis       persistent Redis queue/cache service
api         commerce HTTP API
scheduler   payment-expiry, reconciliation, and notification scheduler
worker      Redis job consumer

api, scheduler, and worker all use the same multi-stage Dockerfile image. The image builds every direct executable under cmd/*, so operational commands can also be run as one-off containers without maintaining separate Dockerfiles.

The database service uses the current PostgreSQL 18 volume layout and persists /var/lib/postgresql. On first initialization it runs:

deploy/database-bootstrap.sql

which enables pg_trgm and vector before the API applies the embedded Atlas migrations.

On a fresh stack, scheduler and worker wait for the API health check before starting. This makes the API migration gate authoritative and prevents background jobs from racing a not-yet-migrated database.

Start the Docker stack

Docker Compose reads the repository .env for both Compose interpolation and application environment variables. Keep the existing application values, but Docker overrides the service-to-service addresses internally:

POSTGRES_HOST=postgres
POSTGRES_PORT=5432
REDIS_ADDR=redis:6379
HTTP_ADDR=:8080

The default host bindings are intentionally separate from the normal host-installed PostgreSQL/Redis ports:

API         127.0.0.1:8081
PostgreSQL  127.0.0.1:5433
Redis       127.0.0.1:6380

They can be changed with DOCKER_API_BIND, DOCKER_API_PORT, DOCKER_POSTGRES_PORT, and DOCKER_REDIS_PORT in .env.

From PowerShell:

docker compose config --quiet
docker compose build --pull
docker compose up -d --remove-orphans
docker compose ps

Readiness should then be available at:

http://127.0.0.1:8081/health/ready

Follow the application processes with:

docker compose logs -f api scheduler worker

Stop containers without deleting data:

docker compose down

Reset the Docker PostgreSQL and Redis data completely:

docker compose down -v --remove-orphans

The reset command is destructive and should only be used when a fresh local Docker database/queue is intended.

Docker backend validation

A Windows validation script is included:

.\scripts\docker_backend_validation.ps1

It validates:

Docker and Compose availability

Compose configuration

production image build

PostgreSQL and Redis health

vector and pg_trgm extension availability

automatic Atlas migration startup

API /health/ready

API, scheduler, and worker process health

application-to-Redis queue connectivity through queue-inspect

API/scheduler/worker restart recovery

The script intentionally leaves the stack running so the final full cross-module black-box E2E can run against the containerized backend.

One-off operational commands

The runtime image contains all direct cmd/* programs. Examples:

docker compose run --rm api /app/bin/seed
docker compose run --rm api /app/bin/admin-bootstrap
docker compose run --rm api /app/bin/support-bootstrap
docker compose run --rm api /app/bin/search-reindex
docker compose run --rm api /app/bin/storage-check
docker compose run --rm api /app/bin/queue-inspect

The bootstrap commands continue to use the corresponding ADMIN_BOOTSTRAP_* and SUPPORT_BOOTSTRAP_* environment variables from .env.

Required startup configuration

The API always requires:

POSTGRES_HOST
POSTGRES_DB
POSTGRES_USER
POSTGRES_PASSWORD
ADMIN_ENCRYPTION_KEY_ID
ADMIN_ENCRYPTION_KEY_BASE64
INVOICE_MERCHANT_NAME

It also requires at least one valid invoice email:

INVOICE_BUSINESS_EMAIL
or
INVOICE_MERCHANT_EMAIL

POSTGRES_PORT defaults to 5432 and POSTGRES_SSLMODE defaults to disable.

The complete current configuration surface, defaults, scheduler/worker tuning, delivery settings, storage settings, search settings, metrics settings, and bootstrap variables are documented in .env.example.

Browser origins and trusted proxies

Customer/storefront CORS is controlled by:

APP_ALLOWED_ORIGINS

Admin CORS is controlled separately by:

ADMIN_ALLOWED_ORIGINS

Both default to localhost origins in development and must be explicitly configured in production. Wildcard origins are rejected.

TRUSTED_PROXIES is a comma-separated Gin trusted-proxy list. Leave it empty unless the API is actually behind a trusted reverse proxy/load balancer.

API surface

Public/customer API

Most customer-facing endpoints live under:

/api/v1

Major route families include:

customer registration/login/refresh/logout and verification flows

customer profile, addresses, account/security/preferences

categories

catalog/home feed and product detail

text/semantic/image-aware search seams

cart

checkout and delivery-method selection

orders and order history

payment status/initiation

returns and refunds

public/customer reviews

wishlist

customer delivery/tracking

Anonymous browsing is supported for catalog/category/search surfaces where authentication is not required. Account-bound operations remain authenticated.

Payment webhooks

Provider webhook ingress is intentionally outside customer authentication middleware:

POST /api/v1/payments/webhooks/:provider

Provider-specific signature verification/processing is still required before a real payment provider can be enabled.

CRM

Authenticated customer CRM routes are under:

/api/v1/crm

They cover support cases, messages, and bulk-stock-request escalation.

Human support

Staff/support routes are under:

/api/v1/support

They include staff authentication plus support inbox/claim/reply/internal-note/escalation/resolve flows.

Admin API

Admin routes are under:

/api/v1/admin

Admin authentication is protected by network policy, rate limits, MFA/session controls, CSRF protection, and permission-based RBAC.

Major admin route families include:

Admin authentication/MFA/session management

catalog/category/product writes

catalog import and media upload management

inventory

orders/payments/returns/refunds

promotions

analytics

customers and review moderation

staff, roles, and permissions

CRM/support administration

warehouse and fulfillment operations

delivery/tracking/receipt confirmation

invoices

finance expenses and profit/loss reporting

Payments

Payment-method switches:

COD_ENABLED
BKASH_ENABLED
NAGAD_ENABLED
ROCKET_ENABLED
BANK_TRANSFER_ENABLED

COD is implemented and can be enabled independently.

The bKash, Nagad, Rocket, and bank-transfer seams exist, but no fake provider adapters are registered. Do not enable a method in production until its real provider adapter and credentials are implemented and wired into both the API and worker registries.

The scheduler enqueues:

order.payment_expiration
payment.reconciliation
notification.outbox

The worker consumes those same job types.

Notifications and OTP

The notification outbox, templates, dispatcher, scheduler job, worker job, persistence, retries, audit state, and provider-neutral channel registry are implemented.

No fake production email/SMS/messenger provider is registered. Without a real notification provider, outbox delivery is intentionally recorded as skipped/provider unavailable rather than falsely marked delivered.

Customer registration OTP behavior is currently:

development: OTP is written to the API log so the flow can be runtime-tested;

production: delivery fails closed until a real SMS provider is integrated.

Delivery and warehouse

The backend contains the provider-neutral logistics/warehouse seam, including warehouse locations, allocations/reservations, pick/pack/ready-for-handoff, handoffs, inbound-shipment milestones, shipment lifecycle, tracking events, delivery verification, and receipt confirmation.

Real courier APIs and live GPS tracking remain deferred integrations.

Finance and invoices

The backend persists immutable order/invoice financial snapshots, invoice records/items, an auditable expense ledger, and profit/loss analytics.

Historical orders created before frozen COGS snapshots may legitimately report incomplete COGS coverage. In that case gross/net profit remains null rather than fabricating historical cost.

Semantic search

Semantic search is disabled by default:

SEARCH_SEMANTIC_ENABLED=false

When enabled, VOYAGE_API_KEY is required. Current embedding defaults are documented in .env.example.

The OpenAI API is not used as the semantic-search provider.

Metrics

Prometheus metrics are configurable with:

METRICS_ENABLED
METRICS_PATH
METRICS_TOKEN

Development defaults to metrics enabled. Production defaults to disabled; if production metrics are enabled, a bearer token of at least 32 characters is required.

Default path:

/internal/metrics

Object storage

Supported storage modes:

disabled
s3

When STORAGE_PROVIDER=s3, endpoint, bucket, access key, and secret key are required. Presigned upload/download TTLs and path-style behavior are configurable through .env.example.

Database migrations

The API uses the embedded migration directory:

db/migrations

and applies pending migrations during API startup under a PostgreSQL advisory lock.

The declarative Atlas schema is:

db/schema.pg.hcl

For Atlas CLI operations on Windows, use:

.\scripts\atlas.ps1 migrate status --env local

The script reads the local .env and constructs ATLAS_DB_URL for atlas.hcl.

Command inventory

Every direct executable under cmd/* is part of the build gate:

admin-bootstrap
api
catalog-import-apply
catalog-import-check
catalog-import-stage
queue-inspect
scheduler
search-reindex
seed
storage-check
support-bootstrap
worker

Examples:

go run ./cmd/seed
go run ./cmd/admin-bootstrap
go run ./cmd/support-bootstrap
go run ./cmd/search-reindex
go run ./cmd/storage-check
go run ./cmd/queue-inspect

Bootstrap commands read their corresponding ADMIN_BOOTSTRAP_* or SUPPORT_BOOTSTRAP_* environment variables.

Static/build gate

Useful Make targets:

make fmt
make tidy-check
make vet
make test
make build
make check

Equivalent Go commands:

go fmt ./...
go mod tidy -diff
go vet ./...
go test ./... -count=1
go build ./cmd/...

The Windows final-validation script performs the stricter freeze gate and dynamically builds every direct cmd/* executable:

.\scripts\final_backend_validation.ps1

Before running it, stop Air while formatting all Go files, then restart exactly one API instance at the configured E2E base URL.

Deferred integrations (non-blocking for backend freeze)

The following are intentionally outside the backend-freeze feature scope:

real bKash/Nagad/Rocket payment providers

real courier APIs and live GPS

real email/SMS/SMTP/messenger provider

fraud/risk system

Try-On / visual AI provider integration

The existing provider-neutral seams should remain truthful when these integrations are absent; no fake success behavior should be introduced merely to satisfy a test.

Freeze rule

Do not add another backend feature workstream unless final audit/E2E exposes a concrete correctness, security, data-integrity, or operational defect. After the final static/build, Atlas, and black-box E2E gates pass, the backend can be frozen and frontend work can proceed against the documented API/co