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

# собрать образ миграций
.PHONY: build-img-migrator
build-img-migrator:
	docker build \
		--build-arg=LDFLAGS="$(LDFLAGS)" \
		-t $(DOCKER_IMG_MIGRATOR) \
		-f backend/build/migrator/Dockerfile .

# запустить образ миграций
.PHONY: run-img-migrator
run-img-migrator: build-img-migrator
	docker run -d $(DOCKER_IMG_MIGRATOR)

# собрать образ сервис
.PHONY: build-img
build-img:
	docker build \
		--build-arg=LDFLAGS="$(LDFLAGS)" \
		-t $(DOCKER_IMG_BACKEND_SERVER) \
		-f backend/build/server/Dockerfile .

# запустить образ сервиса
.PHONY: run-img
run-img: build-img
	docker run -d $(DOCKER_IMG_BACKEND_SERVER)

# применить миграции Postgres (в ручном режиме)
.PHONY: migrate
migrate:
	goose -dir migrations postgres "postgres://otus:password@localhost:5432/backend" up

# откатить миграции Postgres (в ручном режиме)
.PHONY: migrate-down
migrate-down:
	goose -dir migrations postgres "postgres://otus:password@localhost:5432/backend" down

# поднять окружение (только БД master)
.PHONY: up-infra
up-infra:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml up -d

# потушить окружение (только БД master)
.PHONY: down-infra
down-infra:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml down

# поднять сервисы и окружение (БД master)
.PHONY: up
up:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose.yaml up -d --build

# потушить сервисы и окружение (БД master)
.PHONY: down
down:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose.yaml down

# поднять сервисы и окружение (БД master+мониторинги)
.PHONY: up-mon
up-mon:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-monitoring.yaml -f deployments/docker-compose.yaml up -d --build

# потушить сервисы и окружение (БД master+мониторинги)
.PHONY: down-mon
down-mon:
	docker compose --env-file deployments/.env -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-monitoring.yaml -f deployments/docker-compose.yaml down

# поднять сервисы и окружение (БД master и реплики+мониторинги)
.PHONY: up-replicated
up-replicated:
	docker compose --env-file deployments/.env_replicated -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-db-replicas.yaml -f deployments/docker-compose-monitoring.yaml -f deployments/docker-compose.yaml up -d --build

# потушить сервисы и окружение (БД master и реплики+мониторинги)
.PHONY: down-replicated
down-replicated:
	docker compose --env-file deployments/.env_replicated -f deployments/docker-compose-db-master.yaml -f deployments/docker-compose-db-replicas.yaml -f deployments/docker-compose-monitoring.yaml -f deployments/docker-compose.yaml down