BIN_BACKEND_SOCIAL_SERVER := "./backend/social/bin/server"
DOCKER_IMG_BACKEND_SOCIAL_SERVER="backend-social-server:develop"

BIN_BACKEND_SOCIAL_CLIENT:= "./backend/social/bin/client"

BIN_BACKEND_DIALOG_SERVER := "./backend/dialog/bin/server"
DOCKER_IMG_BACKEND_DIALOG_SERVER="backend-dialog-server:develop"

BIN_BACKEND_VERIFIER_SERVER := "./backend/verifier/bin/server"
DOCKER_IMG_BACKEND_VERIFIER_SERVER="backend-verifier-server:develop"

BIN_BACKEND_DIALOG_GENERATOR := "./backend/dialog/bin/dialog-generator"

DOCKER_IMG_MIGRATOR="backend-migrator:develop"

GIT_HASH := $(shell git log --format="%h" -n 1)
LDFLAGS := -X main.release="develop" -X main.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%S) -X main.gitHash=$(GIT_HASH)

# сгенерировать код по спецификации Open API
.PHONY: generate_from_openapi
generate_from_openapi:
	cd backend/social && go generate ./... && cd ../dialog && go generate ./... && cd ../verifier && go generate ./...

# скомпилировать бинарные файлы сервиса соц.сети
.PHONY: build_backend_social_server
build_backend_social_server:
	cd backend/social && go build -v -o $(BIN_BACKEND_SOCIAL_SERVER) -ldflags "$(LDFLAGS)" ./cmd/server

# собрать и запустить сервис соц.сети с конфигами по умолчанию
.PHONY: run_backend_social_server
run_backend_social_server: build_backend_social_server
	cd backend/social && $(BIN_BACKEND_SOCIAL_SERVER) -config ./configs/server_config_local.yaml

# скомпилировать бинарные файлы проверочного клиента
.PHONY: build_backend_social_client
build_backend_social_client:
	cd backend/social && go build -v -o $(BIN_BACKEND_SOCIAL_CLIENT) -ldflags "$(LDFLAGS)" ./cmd/client

# собрать и запустить проверочного клиента для сервиса соц.сети
.PHONY: run_backend_social_client
run_backend_social_client: build_backend_social_client
	cd backend/social && $(BIN_BACKEND_SOCIAL_CLIENT)

# скомпилировать бинарные файлы сервиса диалогов
.PHONY: build_backend_dialog_server
build_backend_dialog_server:
	cd backend/dialog && go build -v -o $(BIN_BACKEND_DIALOG_SERVER) -ldflags "$(LDFLAGS)" ./cmd/server

# собрать и запустить сервис диалогов с конфигами по умолчанию
.PHONY: run_backend_dialog_server
run_backend_dialog_server: build_backend_dialog_server
	cd backend/dialog && $(BIN_BACKEND_DIALOG_SERVER) -config ./configs/server_config_local.yaml

# скомпилировать бинарные файлы сервиса верификации
.PHONY: build_backend_verifier_server
build_backend_verifier_server:
	cd backend/verifier && go build -v -o $(BIN_BACKEND_VERIFIER_SERVER) -ldflags "$(LDFLAGS)" ./cmd/server

# собрать и запустить сервис верификации с конфигами по умолчанию
.PHONY: run_backend_verifier_server
run_backend_verifier_server: build_backend_verifier_server
	cd backend/verifier && $(BIN_BACKEND_VERIFIER_SERVER) -config ./configs/server_config_local.yaml

# применить миграции Postgres (в ручном режиме)
.PHONY: migrate
migrate:
	goose -dir backend/migrations postgres "postgres://otus:password@localhost:5432/backend" up

# откатить миграции Postgres (в ручном режиме)
.PHONY: migrate-down
migrate-down:
	goose -dir backend/migrations postgres "postgres://otus:password@localhost:5432/backend" down

# сгенерировать данные по диалогам в postgres и tarantool (в ручном режиме)
.PHONY: dialog-generator
dialog-generator:
	cd backend/dialog && go build -v -o $(BIN_BACKEND_DIALOG_GENERATOR) -ldflags "$(LDFLAGS)" ./cmd/dialog-generator && $(BIN_BACKEND_DIALOG_GENERATOR)

# поднять окружение (только БД master, tarantool, кеш и очередь)
.PHONY: up-infra
up-infra:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-tarantool.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml up -d

# потушить окружение (только БД master, tarantool, кеш и очередь)
.PHONY: down-infra
down-infra:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-tarantool.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml down

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

# поднять сервисы и окружение (БД master, tarantool, кеш и очередь+мониторинги)
.PHONY: up-memory
up-memory:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-tarantool.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml -f deployments/docker-compose-monitoring.yaml -f deployments/docker-compose.yaml up -d --build

# потушить сервисы и окружение (БД master, tarantool, кеш и очередь+мониторинги)
.PHONY: down-memory
down-memory:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-tarantool.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml -f deployments/docker-compose-monitoring.yaml -f deployments/docker-compose.yaml down

# поднять сервисы и окружение (БД master и реплики, tarantool, кеш и очередь+мониторинги+несколько реплик сервиса+haproxy, nginx)
.PHONY: up-balancing
up-balancing:
	docker compose --env-file deployments/.env --env-file deployments/.env_balancing -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-db-replicas.yaml -f deployments/docker-compose-tarantool.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml -f deployments/docker-compose-monitoring.yaml -f deployments/docker-compose-balancing.yaml -f deployments/docker-compose.yaml up -d --build

# потушить сервисы и окружение (БД master и реплики, tarantool, кеш и очередь+мониторинги+несколько реплик сервиса+haproxy, nginx)
.PHONY: down-balancing
down-balancing:
	docker compose --env-file deployments/.env --env-file deployments/.env_balancing -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-db-replicas.yaml -f deployments/docker-compose-tarantool.yaml -f deployments/docker-compose-rabbit.yaml -f deployments/docker-compose-redis.yaml -f deployments/docker-compose-monitoring.yaml -f deployments/docker-compose-balancing.yaml -f deployments/docker-compose.yaml down
