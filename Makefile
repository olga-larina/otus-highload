BIN_BACKEND_SERVER := "./backend/bin/server"
DOCKER_IMG_BACKEND_SERVER="backend-server:develop"

BIN_BACKEND_CLIENT:= "./backend/bin/client"

DOCKER_IMG_MIGRATOR="backend-migrator:develop"

GIT_HASH := $(shell git log --format="%h" -n 1)
LDFLAGS := -X main.release="develop" -X main.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%S) -X main.gitHash=$(GIT_HASH)

# сгенерировать код по спецификации Open API
.PHONY: generate_from_openapi
generate_from_openapi:
	cd backend && go generate ./...

# скомпилировать бинарные файлы сервиса
.PHONY: build_backend_server
build_backend_server:
	cd backend && go build -v -o $(BIN_BACKEND_SERVER) -ldflags "$(LDFLAGS)" ./cmd/server

# собрать и запустить сервисы с конфигами по умолчанию
.PHONY: run_backend_server
run_backend_server: build_backend_server
	cd backend && $(BIN_BACKEND_SERVER) -config ./configs/server_config_local.yaml

# скомпилировать бинарные файлы проверочного клиента
.PHONY: build_backend_client
build_backend_client:
	cd backend && go build -v -o $(BIN_BACKEND_CLIENT) -ldflags "$(LDFLAGS)" ./cmd/client

# собрать и запустить проверочного клиента
.PHONY: run_backend_client
run_backend_client: build_backend_client
	cd backend && $(BIN_BACKEND_CLIENT)

# применить миграции Postgres (в ручном режиме)
.PHONY: migrate
migrate:
	goose -dir backend/migrations postgres "postgres://otus:password@localhost:5432/backend" up

# откатить миграции Postgres (в ручном режиме)
.PHONY: migrate-down
migrate-down:
	goose -dir backend/migrations postgres "postgres://otus:password@localhost:5432/backend" down

# поднять окружение (только БД master, кеш и очередь)
.PHONY: up-infra
up-infra:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml up -d

# потушить окружение (только БД master, кеш и очередь)
.PHONY: down-infra
down-infra:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml down

# поднять сервисы и окружение (БД master, кеш и очередь)
.PHONY: up
up:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml -f deployments/docker-compose.yaml up -d --build

# потушить сервисы и окружение (БД master, кеш и очередь)
.PHONY: down
down:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml -f deployments/docker-compose.yaml down

# поднять сервисы и окружение с мониторингами (БД master, кеш и очередь+мониторинги)
.PHONY: up-mon
up-mon:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml -f deployments/docker-compose-monitoring.yaml -f deployments/docker-compose.yaml up -d --build

# потушить сервисы и окружение с мониторингами (БД master, кеш и очередь+мониторинги)
.PHONY: down-mon
down-mon:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml -f deployments/docker-compose-monitoring.yaml -f deployments/docker-compose.yaml down

# поднять сервисы и окружение (БД master и реплики, кеш и очередь+мониторинги)
.PHONY: up-replicated
up-replicated:
	docker compose --env-file deployments/.env --env-file deployments/.env_replicated -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-db-replicas.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml -f deployments/docker-compose-monitoring.yaml -f deployments/docker-compose.yaml up -d --build

# потушить сервисы и окружение (БД master и реплики, кеш и очередь+мониторинги)
.PHONY: down-replicated
down-replicated:
	docker compose --env-file deployments/.env --env-file deployments/.env_replicated -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-db-replicas.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml -f deployments/docker-compose-monitoring.yaml -f deployments/docker-compose.yaml down

# поднять сервисы и окружение (citus (1 master + 2 workers + 1 manager), кеш и очередь)
.PHONY: up-sharded
up-sharded:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-sharded.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml -f deployments/docker-compose.yaml up -d --build

# потушить сервисы и окружение (citus (1 master + 2 workers + 1 manager), кеш и очередь)
.PHONY: down-sharded
down-sharded:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-sharded.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml -f deployments/docker-compose.yaml down