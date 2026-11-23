-include .env
export

MIGRATIONS_DIR = migrations
DB_URL = postgres://$(DB_USER):$(DB_PASSWORD)@localhost:$(DB_PORT)/$(DB_NAME)?sslmode=disable

.PHONY: migrate-create migrate-up migrate-down migrate-version migrate-force migrate-docker run test test-cover test-integration test-integration-mac up up-app down logs

migrate-create:
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

migrate-version:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" version

migrate-force:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" force $(version)

run:
	go run ./cmd/api
test:
	go test ./... -race
test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out
test-integration:
	go test -tags=integration ./... -count=1

# Docker Desktop (macOS) socketi standart joyda emas; testcontainers'ga
# shu yerdan beriladi. Linux/CI'da DOCKER_HOST allaqachon to'g'ri bo'ladi.
test-integration-mac:
	DOCKER_HOST=unix://$(HOME)/.docker/run/docker.sock TESTCONTAINERS_RYUK_DISABLED=true \
		go test -tags=integration ./... -count=1

# Lokal Postgres + Redis (API `make run` bilan)
up:
	docker compose up -d
down:
	docker compose --profile app down

# Hammasi Docker'da: Postgres + Redis + migratsiyalar + API
up-app:
	docker compose --profile app up -d --build
logs:
	docker compose --profile app logs -f api
# Migratsiyalarni Docker orqali (lokal `migrate` CLI kerak emas)
migrate-docker:
	docker compose --profile app run --rm migrate
