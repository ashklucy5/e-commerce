GO ?= go
DOCKER_COMPOSE ?= docker compose

.PHONY: fmt tidy tidy-check vet test build check run-api run-worker run-scheduler seed admin-bootstrap support-bootstrap search-reindex storage-check queue-inspect docker-config docker-build docker-up docker-down docker-logs docker-ps docker-reset

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy

tidy-check:
	$(GO) mod tidy -diff

vet:
	$(GO) vet ./...

test:
	$(GO) test ./... -count=1

# Compile every executable under cmd/* without emitting repository binaries.
build:
	$(GO) build ./cmd/...

check: tidy-check vet test build

run-api:
	$(GO) run ./cmd/api

run-worker:
	$(GO) run ./cmd/worker

run-scheduler:
	$(GO) run ./cmd/scheduler

seed:
	$(GO) run ./cmd/seed

admin-bootstrap:
	$(GO) run ./cmd/admin-bootstrap

support-bootstrap:
	$(GO) run ./cmd/support-bootstrap

search-reindex:
	$(GO) run ./cmd/search-reindex

storage-check:
	$(GO) run ./cmd/storage-check

queue-inspect:
	$(GO) run ./cmd/queue-inspect

# Docker Compose uses the same production image for API, scheduler, and worker.
docker-config:
	$(DOCKER_COMPOSE) config --quiet

docker-build:
	$(DOCKER_COMPOSE) build

docker-up:
	$(DOCKER_COMPOSE) up -d --remove-orphans

docker-down:
	$(DOCKER_COMPOSE) down

docker-logs:
	$(DOCKER_COMPOSE) logs -f api scheduler worker

docker-ps:
	$(DOCKER_COMPOSE) ps

# Destructive: removes the local Docker PostgreSQL/Redis data volumes.
docker-reset:
	$(DOCKER_COMPOSE) down -v --remove-orphans