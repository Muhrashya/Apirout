.PHONY: all build run test clean docker-build docker-up docker-down release help

APP_NAME = apirout
VERSION ?= 1.0.0
ROLE ?= traditional

all: build

help:
	@echo "Apirout - Open Source Instant API Engine & Gateway"
	@echo "Self-Hosted Instant API Engine & Multi-Plane Gateway Architecture"
	@echo ""
	@echo "Available commands:"
	@echo "  make build             Build the unified Go binary and web console"
	@echo "  make release           Run release workflow and tag version (e.g. make release VERSION=v1.0.0)"
	@echo "  make run-traditional   Run Apirout in traditional mode (All-in-One Gateway + Console)"
	@echo "  make run-cp            Run Apirout in Control-Plane mode (Console & Management API only)"
	@echo "  make run-dp            Run Apirout in Data-Plane mode (Stateless Gateway only)"
	@echo "  make docker-build      Build the multi-stage Docker image"
	@echo "  make docker-up         Start PostgreSQL, Redis, and Apirout via Docker Compose"
	@echo "  make docker-down       Stop all containers and networks"
	@echo "  make migrate           Run database schema migrations"
	@echo "  make create-admin      Create a new super admin account interactively"

build:
	@echo "Building web console..."
	cd web && npm install && npm run build
	@echo "Building Apirout engine binary..."
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/$(APP_NAME) ./cmd/apirout

run-traditional:
	APIROUT_ROLE=traditional go run ./cmd/apirout

run-cp:
	APIROUT_ROLE=control_plane go run ./cmd/apirout

run-dp:
	APIROUT_ROLE=data_plane go run ./cmd/apirout

migrate:
	go run ./cmd/apirout migrate

create-admin:
	go run ./cmd/apirout create-admin

docker-build:
	docker build --build-arg VERSION=$(VERSION) -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

release:
	@powershell -ExecutionPolicy Bypass -File scripts/release.ps1 -Version $(VERSION)

clean:
	rm -rf bin/ web/dist
