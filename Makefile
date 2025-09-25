.PHONY: help build build-scheduler clean test proto proto-scheduler deps install-deps install-deps-proto run-scheduler format lint dev-setup check-tools mock-install mock-gen copy-env

# OS Detection
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    SHELL_CMD := powershell -Command
    RM_FILE := Remove-Item -Force
    RM_DIR := Remove-Item -Recurse -Force
    MKDIR := New-Item -ItemType Directory -Force
    TEST_PATH := Test-Path
    WHICH_CMD := Get-Command
    NULL_DEV := $$null
    PATH_SEP := ;
    EXE_EXT := .exe
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Linux)
        DETECTED_OS := Linux
    endif
    ifeq ($(UNAME_S),Darwin)
        DETECTED_OS := Mac
    endif
    SHELL_CMD := sh -c
    RM_FILE := rm -f
    RM_DIR := rm -rf
    MKDIR := mkdir -p
    TEST_PATH := test -e
    WHICH_CMD := command -v
    NULL_DEV := /dev/null
    PATH_SEP := :
    EXE_EXT :=
endif

PROTOC_VERSION := 3.21.12
GO_VERSION := 1.24.5

help:
	@echo "Available commands:"
	@echo "  build              - Build all services"
	@echo "  build-scheduler    - Build scheduler service"
	@echo "  clean              - Clean build artifacts"
	@echo "  test               - Run tests"
	@echo "  proto              - Generate protobuf files"
	@echo "  proto-scheduler    - Generate scheduler protobuf"
	@echo "  deps               - Download dependencies"
	@echo "  install-deps       - Install required tools"
	@echo "  install-deps-proto - Install protoc using system package manager"
	@echo "  run-scheduler      - Run scheduler service"
	@echo "  format             - Format code"
	@echo "  lint               - Run linter"
	@echo "  dev-setup          - Setup development environment"
	@echo "  check-tools        - Check tools"
	@echo "  docker-up-force        - Force Docker Compose up (recreate and rebuild)"
	@echo "  docker-up-workers      - Start services with N workers (usage: make docker-up-workers N=3)"
	@echo "  docker-up-fetcher      - Start fetcher service components (scheduler, orchestrator, worker, postgres, redis, rabbitmq)"
	@echo "  race-detect            - Run race condition detection on all services"
	@echo "  swagger-gen            - Generate Swagger documentation for search service"
	@echo "  swagger-install        - Install Swagger code generator tool"
	@echo "  k6-install             - Install K6 load testing tool"
	@echo "  k6-test-hotel-id       - Run K6 load test for hotel-by-id endpoint"
	@echo "  k6-test-hotel-id-quick - Run quick K6 load test for hotel-by-id endpoint (2 minutes)"
	@echo "  mock-install           - Install uber-go/mock (mockgen) tool"
	@echo "  mock-gen               - Generate mocks for interfaces"
	@echo "  copy-env               - Copy .env.example to .env"

build: build-scheduler build-orchestrator build-worker build-search

build-scheduler:
	@echo "Building scheduler..."
	cd fetcher-service && go build -o ../bin/scheduler ./cmd/scheduler

build-orchestrator:
	@echo "Building orchestrator..."
	cd fetcher-service && go build -o ../bin/orchestrator ./cmd/orchestrator

build-worker:
	@echo "Building worker..."
	cd fetcher-service && go build -o ../bin/worker ./cmd/worker

build-search:
	@echo "Building search service..."
	cd search-service && go build -o ../bin/search-api ./cmd/api

clean:
	@echo "Cleaning build artifacts..."
ifeq ($(DETECTED_OS),Windows)
	@$(SHELL_CMD) "if ($(TEST_PATH) bin) { $(RM_DIR) bin }"
	@$(SHELL_CMD) "Get-ChildItem *.exe -ErrorAction SilentlyContinue | $(RM_FILE)"
else
	@$(RM_DIR) bin 2>$(NULL_DEV) || true
	@$(RM_FILE) *.exe 2>$(NULL_DEV) || true
endif
	go clean

test:
	@echo "Running tests..."
	cd fetcher-service && go test ./...
	@echo "Running search service tests..."
	cd search-service && go test ./...

proto:
	@echo "Generating protobuf files..."
ifeq ($(DETECTED_OS),Windows)
	@$(SHELL_CMD) "if (!($(WHICH_CMD) protoc -ErrorAction SilentlyContinue)) { Write-Host 'Error: protoc not found. Please install Protocol Buffers compiler.'; Write-Host 'Install from: https://github.com/protocolbuffers/protobuf/releases'; exit 1 }"
	@$(SHELL_CMD) "if (!($(WHICH_CMD) protoc-gen-go -ErrorAction SilentlyContinue)) { Write-Host 'Installing protoc-gen-go...'; go install google.golang.org/protobuf/cmd/protoc-gen-go@latest }"
	@$(SHELL_CMD) "if (!($(WHICH_CMD) protoc-gen-go-grpc -ErrorAction SilentlyContinue)) { Write-Host 'Installing protoc-gen-go-grpc...'; go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest }"
	@$(SHELL_CMD) "$(MKDIR) -Path 'fetcher-service\\proto\\orchestrator'"
	@$(SHELL_CMD) "$(MKDIR) -Path 'fetcher-service\\proto\\scheduler'"
	@$(SHELL_CMD) "$(MKDIR) -Path 'search-service\\proto'"
else
	@$(WHICH_CMD) protoc >$(NULL_DEV) 2>&1 || (echo "Error: protoc not found. Please install Protocol Buffers compiler." && echo "Install from: https://github.com/protocolbuffers/protobuf/releases" && exit 1)
	@$(WHICH_CMD) protoc-gen-go >$(NULL_DEV) 2>&1 || (echo "Installing protoc-gen-go..." && go install google.golang.org/protobuf/cmd/protoc-gen-go@latest)
	@$(WHICH_CMD) protoc-gen-go-grpc >$(NULL_DEV) 2>&1 || (echo "Installing protoc-gen-go-grpc..." && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest)
	@$(MKDIR) fetcher-service/proto/orchestrator
	@$(MKDIR) fetcher-service/proto/scheduler
	@$(MKDIR) search-service/proto
endif
	@echo "Generating orchestrator protobuf..."
	cd fetcher-service && protoc --go_out=. --go-grpc_out=. --go_opt=module=github.com/victoragudo/hotel-management-system/fetcher-service --go-grpc_opt=module=github.com/victoragudo/hotel-management-system/fetcher-service proto/orchestrator.proto
	@echo "Generating scheduler protobuf..."
	cd fetcher-service && protoc --go_out=. --go-grpc_out=. --go_opt=module=github.com/victoragudo/hotel-management-system/fetcher-service --go-grpc_opt=module=github.com/victoragudo/hotel-management-system/fetcher-service proto/scheduler.proto
	@echo "Protobuf generation complete!"

deps:
	@echo "Downloading dependencies..."
	cd fetcher-service && go mod download && go mod tidy
	@echo "Downloading search service dependencies..."
	cd search-service && go mod download && go mod tidy

install-deps:
	@echo "Installing development tools..."
ifeq ($(DETECTED_OS),Windows)
	@$(SHELL_CMD) "if (!($(WHICH_CMD) protoc -ErrorAction SilentlyContinue)) { if ($(WHICH_CMD) choco -ErrorAction SilentlyContinue) { Write-Host 'Trying to install protoc using Chocolatey...'; try { choco install protoc -y } catch { Write-Host 'Chocolatey installation failed, trying Scoop...'; if ($(WHICH_CMD) scoop -ErrorAction SilentlyContinue) { scoop install protobuf } else { Write-Host 'Alternative installation methods:'; Write-Host '1. Install Chocolatey as admin: https://chocolatey.org/install'; Write-Host '2. Install Scoop: https://scoop.sh'; Write-Host '3. Manual: Download protoc from https://github.com/protocolbuffers/protobuf/releases' } } } elseif ($(WHICH_CMD) scoop -ErrorAction SilentlyContinue) { Write-Host 'Installing protoc using Scoop...'; scoop install protobuf } else { Write-Host 'No package managers found. Manual installation required:'; Write-Host '1. Download from: https://github.com/protocolbuffers/protobuf/releases'; Write-Host '2. Extract protoc.exe to a directory in your PATH'; Write-Host '3. Or install Chocolatey/Scoop first' } } else { Write-Host 'protoc already installed' }"
else
	@$(WHICH_CMD) protoc >$(NULL_DEV) 2>&1 || { \
		echo "protoc not found. Trying to install..."; \
		if $(WHICH_CMD) apt-get >$(NULL_DEV) 2>&1; then \
			echo "Installing protoc using apt..."; \
			sudo apt-get update && sudo apt-get install -y protobuf-compiler; \
		elif $(WHICH_CMD) brew >$(NULL_DEV) 2>&1; then \
			echo "Installing protoc using Homebrew..."; \
			brew install protobuf; \
		elif $(WHICH_CMD) yum >$(NULL_DEV) 2>&1; then \
			echo "Installing protoc using yum..."; \
			sudo yum install -y protobuf-compiler; \
		elif $(WHICH_CMD) pacman >$(NULL_DEV) 2>&1; then \
			echo "Installing protoc using pacman..."; \
			sudo pacman -S protobuf; \
		else \
			echo "No supported package manager found. Manual installation required:"; \
			echo "Download from: https://github.com/protocolbuffers/protobuf/releases"; \
		fi; \
	}
endif
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

install-deps-proto:
	@echo "Installing protoc..."
ifeq ($(DETECTED_OS),Windows)
	@$(SHELL_CMD) "if (!($(WHICH_CMD) choco -ErrorAction SilentlyContinue)) { Write-Host 'Error: Chocolatey not found. Please install Chocolatey first or install protoc manually.' } else { choco install protoc -y }"
else
	@if $(WHICH_CMD) apt-get >$(NULL_DEV) 2>&1; then \
		echo "Installing protoc using apt..."; \
		sudo apt-get update && sudo apt-get install -y protobuf-compiler; \
	elif $(WHICH_CMD) brew >$(NULL_DEV) 2>&1; then \
		echo "Installing protoc using Homebrew..."; \
		brew install protobuf; \
	elif $(WHICH_CMD) yum >$(NULL_DEV) 2>&1; then \
		echo "Installing protoc using yum..."; \
		sudo yum install -y protobuf-compiler; \
	elif $(WHICH_CMD) pacman >$(NULL_DEV) 2>&1; then \
		echo "Installing protoc using pacman..."; \
		sudo pacman -S protobuf; \
	else \
		echo "No supported package manager found. Manual installation required:"; \
		echo "Download from: https://github.com/protocolbuffers/protobuf/releases"; \
	fi
endif

format:
	@echo "Formatting code..."
	cd fetcher-service && go fmt ./...
	@echo "Formatting search service code..."
	cd search-service && go fmt ./...

lint:
	@echo "Running golangci-lint..."
ifeq ($(DETECTED_OS),Windows)
	@$(SHELL_CMD) "if (!($(WHICH_CMD) golangci-lint -ErrorAction SilentlyContinue)) { Write-Host 'golangci-lint not found. Installing...'; if ($(WHICH_CMD) choco -ErrorAction SilentlyContinue) { choco install golangci-lint -y } elseif ($(WHICH_CMD) scoop -ErrorAction SilentlyContinue) { scoop install golangci-lint } else { Write-Host 'No package manager found. Installing via go install...'; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest } }"
	@$(SHELL_CMD) "if (!($(WHICH_CMD) golangci-lint -ErrorAction SilentlyContinue)) { Write-Host 'Error: golangci-lint is not available in PATH after installation attempt.'; exit 1 }"
else
	@$(WHICH_CMD) golangci-lint >$(NULL_DEV) 2>&1 || { \
		echo "golangci-lint not found. Installing..."; \
		if $(WHICH_CMD) brew >$(NULL_DEV) 2>&1; then \
			echo "Installing golangci-lint using Homebrew..."; \
			brew install golangci-lint; \
		elif $(WHICH_CMD) apt-get >$(NULL_DEV) 2>&1; then \
			echo "Installing golangci-lint using go install..."; \
			go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		else \
			echo "Installing golangci-lint using go install..."; \
			go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		fi; \
	}
	@$(WHICH_CMD) golangci-lint >$(NULL_DEV) 2>&1 || (echo "Error: golangci-lint is not available in PATH after installation attempt." && exit 1)
endif
	cd fetcher-service && golangci-lint run ./...
	@echo "Running golangci-lint on search service..."
	cd search-service && golangci-lint run ./...


dev-setup: install-deps deps
	@echo "Development environment setup complete!"

check-tools:
	@echo "Checking required tools..."
ifeq ($(DETECTED_OS),Windows)
	@$(SHELL_CMD) "if (!($(WHICH_CMD) go -ErrorAction SilentlyContinue)) { Write-Host 'Go not found'; exit 1 }"
	@$(SHELL_CMD) "if (!($(WHICH_CMD) protoc -ErrorAction SilentlyContinue)) { Write-Host 'protoc not found - install from https://github.com/protocolbuffers/protobuf/releases' }"
	@$(SHELL_CMD) "if (!($(WHICH_CMD) docker -ErrorAction SilentlyContinue)) { Write-Host 'Docker not found' }"
else
	@$(WHICH_CMD) go >$(NULL_DEV) 2>&1 || (echo "Go not found" && exit 1)
	@$(WHICH_CMD) protoc >$(NULL_DEV) 2>&1 || echo "protoc not found - install from https://github.com/protocolbuffers/protobuf/releases"
	@$(WHICH_CMD) docker >$(NULL_DEV) 2>&1 || echo "Docker not found"
endif
	@echo "Tool check complete!"

# ----- Docker Compose -----

COMPOSE_FILE := docker-compose.yml
ENV_FILE := .env

# Docker availability check function
check-docker:
ifeq ($(DETECTED_OS),Windows)
	@$(SHELL_CMD) "docker info 2>$$null | Out-Null; if ($$LASTEXITCODE -ne 0) { Write-Host 'Error: Docker daemon is not accessible or Docker Desktop is not running'; Write-Host 'Common causes:'; Write-Host '  - Docker Desktop is not started'; Write-Host '  - Docker Desktop service is not running properly'; Write-Host '  - Windows Docker named pipe is not accessible'; Write-Host 'Solutions:'; Write-Host '  1. Start Docker Desktop application'; Write-Host '  2. Restart Docker Desktop if already running'; Write-Host '  3. Check Windows Services for Docker Desktop Service'; Write-Host '  4. If Docker is not installed, install from https://www.docker.com/products/docker-desktop'; exit 1 } else { Write-Host 'Docker daemon is accessible and running' }"
else
	@docker info >/dev/null 2>&1 || (echo "Error: Docker daemon is not accessible or not running" && echo "Please start Docker daemon and try again" && echo "If Docker is not installed, please install Docker from https://docs.docker.com/get-docker/" && exit 1)
	@echo "Docker daemon is accessible and running"
endif

.PHONY: check-docker docker-up docker-down docker-restart docker-logs docker-ps docker-build docker-up-force docker-up-workers docker-up-fetcher docker-ext-up docker-ext-down docker-ext-logs docker-ext-ps

docker-up: check-docker
	@echo "Starting all services with 2 worker instances with Docker Compose..."
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) up -d --build --scale worker=3

docker-down: check-docker
	@echo "Stopping and removing services..."
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) down

docker-build: check-docker
	@echo "Building images (no cache)..."
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) build --no-cache

docker-restart: check-docker
	@echo "Restarting services..."
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) restart

docker-logs: check-docker
	@echo "Tailing logs..."
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) logs -f

docker-ps: check-docker
	@echo "Services status:"
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) ps

docker-up-force: check-docker
	@echo "Forcing Docker Compose up (recreate and rebuild)..."
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) up -d --build --force-recreate

docker-up-workers: check-docker
	@echo "Starting services with N workers..."
	@echo "Usage: make docker-up-workers N=<number_of_workers>"
ifeq ($(N),)
	@echo "Error: Please specify the number of workers using N parameter"
	@echo "Example: make docker-up-workers N=3"
	@exit 1
endif
	@echo "Starting all services with $(N) worker instances..."
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) up -d --build --scale worker=$(N)

docker-up-fetcher: check-docker
	@echo "Starting fetcher service components (scheduler, orchestrator, worker, postgres, redis, rabbitmq)..."
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) up -d --build scheduler orchestrator worker postgres redis rabbitmq

docker-up-search: check-docker
	@echo "Starting searcher service components (typesense, postgres, redis)..."
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) up -d --build typesense postgres redis

# ----- External resources only (DB, RabbitMQ, Redis) -----

docker-ext-up: check-docker
	@echo "Starting external resources (postgres, rabbitmq, redis)..."
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) up -d --build postgres rabbitmq redis typesense

docker-ext-down: check-docker
	@echo "Stopping and removing external resources..."
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) rm -sfv postgres rabbitmq redis typesense || true

docker-ext-logs: check-docker
	@echo "Tailing logs for external resources..."
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) logs -f postgres rabbitmq redis typesense

docker-ext-ps: check-docker
	@echo "External resources status:"
	docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) ps postgres rabbitmq redis typesense

race-detect:
	@echo "Running race condition detection on all services..."
	@echo "Building and running race detection for scheduler..."
	cd fetcher-service && go build -race -o ../bin/scheduler-race ./cmd/scheduler
	@echo "Building and running race detection for orchestrator..."
	cd fetcher-service && go build -race -o ../bin/orchestrator-race ./cmd/orchestrator
	@echo "Building and running race detection for worker..."
	cd fetcher-service && go build -race -o ../bin/worker-race ./cmd/worker
	@echo "Building and running race detection for search service..."
	cd search-service && go build -race -o ../bin/search-api-race ./cmd/api
	@echo "Running tests with race detection enabled..."
	cd fetcher-service && go test -race ./...
	@echo "Running search service tests with race detection enabled..."
	cd search-service && go test -race ./...
	@echo "Race condition detection complete!"
	@echo "Race-enabled binaries created in bin/ directory:"
ifeq ($(DETECTED_OS),Windows)
	@$(SHELL_CMD) "Get-ChildItem bin\\*-race* | Select-Object Name, Length"
else
	@ls -la bin/*-race* 2>$(NULL_DEV) || echo "Race binaries created successfully"
endif

# ----- Swagger Documentation -----

swagger-install:
	@echo "Installing Swagger code generator tool..."
ifeq ($(DETECTED_OS),Windows)
	@$(SHELL_CMD) "if (!($(WHICH_CMD) swag -ErrorAction SilentlyContinue)) { Write-Host 'Installing swag...'; go install github.com/swaggo/swag/cmd/swag@latest } else { Write-Host 'swag is already installed' }"
else
	@$(WHICH_CMD) swag >$(NULL_DEV) 2>&1 || (echo "Installing swag..." && go install github.com/swaggo/swag/cmd/swag@latest)
endif
	@echo "Swagger installation complete!"

swagger-gen: swagger-install
	@echo "Generating Swagger documentation for search service..."
ifeq ($(DETECTED_OS),Windows)
	@$(SHELL_CMD) "$(MKDIR) -Path 'search-service\\docs'"
else
	@$(MKDIR) search-service/docs
endif
	cd search-service && swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
	@echo "Swagger documentation generated successfully!"
	@echo "Documentation available at:"
	@echo "  - JSON: search-service/docs/swagger.json"
	@echo "  - YAML: search-service/docs/swagger.yaml"
	@echo "  - Swagger UI: http://localhost:8080/swagger/index.html (when service is running)"

# ----- K6 Load Testing -----

k6-install:
	@echo "Installing K6 load testing tool..."
ifeq ($(DETECTED_OS),Windows)
	@$(SHELL_CMD) "if (!($(WHICH_CMD) k6 -ErrorAction SilentlyContinue)) { Write-Host 'Installing k6...'; if ($(WHICH_CMD) choco -ErrorAction SilentlyContinue) { choco install k6 -y } elseif ($(WHICH_CMD) scoop -ErrorAction SilentlyContinue) { scoop install k6 } elseif ($(WHICH_CMD) winget -ErrorAction SilentlyContinue) { winget install k6 } else { Write-Host 'No package manager found. Please install k6 manually from https://k6.io/docs/getting-started/installation/' } } else { Write-Host 'k6 is already installed' }"
else
	@$(WHICH_CMD) k6 >$(NULL_DEV) 2>&1 || { \
		echo "k6 not found. Installing..."; \
		if $(WHICH_CMD) brew >$(NULL_DEV) 2>&1; then \
			echo "Installing k6 using Homebrew..."; \
			brew install k6; \
		elif $(WHICH_CMD) apt-get >$(NULL_DEV) 2>&1; then \
			echo "Installing k6 using apt..."; \
			sudo gpg -k; \
			sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69; \
			echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list; \
			sudo apt-get update; \
			sudo apt-get install k6; \
		else \
			echo "No supported package manager found. Manual installation required:"; \
			echo "Download from: https://k6.io/docs/getting-started/installation/"; \
		fi; \
	}
endif
	@echo "K6 installation complete!"

k6-test-hotel-id: k6-install
	@echo "Running K6 load test for hotel-by-id endpoint..."
	@echo "Make sure the search service is running on localhost:8080"
	k6 run k6-tests/hotel-by-id-load-test.js

k6-test-hotel-id-quick: k6-install
	@echo "Running quick K6 load test for hotel-by-id endpoint (2 minutes)..."
	@echo "Make sure the search service is running on localhost:8080"
	k6 run --duration 2m --vus 10 k6-tests/hotel-by-id-load-test.js

# ----- Mock Generation -----

mock-install:
	@echo "Installing uber-go/mock (mockgen) tool..."
ifeq ($(DETECTED_OS),Windows)
	@$(SHELL_CMD) "if (!($(WHICH_CMD) mockgen -ErrorAction SilentlyContinue)) { Write-Host 'Installing mockgen...'; go install go.uber.org/mock/mockgen@latest } else { Write-Host 'mockgen is already installed' }"
else
	@$(WHICH_CMD) mockgen >$(NULL_DEV) 2>&1 || (echo "Installing mockgen..." && go install go.uber.org/mock/mockgen@latest)
endif
	@echo "Mock generation tool installation complete!"

mock-gen: mock-install
	@echo "Generating mocks for interfaces..."
ifeq ($(DETECTED_OS),Windows)
	@$(SHELL_CMD) "$(MKDIR) -Path 'fetcher-service\\internal\\mocks'"
	@$(SHELL_CMD) "$(MKDIR) -Path 'search-service\\internal\\mocks'"
else
	@$(MKDIR) fetcher-service/internal/mocks
	@$(MKDIR) search-service/internal/mocks
endif
	@echo "Generating fetcher-service mocks..."
	cd fetcher-service && mockgen -source=internal/worker/ports/api_client_port.go -destination=internal/mocks/mock_api_client.go -package=mocks
	cd fetcher-service && mockgen -source=internal/worker/ports/cache_port.go -destination=internal/mocks/mock_cache.go -package=mocks
	cd fetcher-service && mockgen -source=internal/infrastructure/queue/rabbitmq_consumer.go -destination=internal/mocks/mock_consumer.go -package=mocks
	@echo "Generating search-service mocks..."
	cd search-service && mockgen -source=internal/domain/hotel/repository.go -destination=internal/mocks/mock_repository.go -package=mocks
	cd search-service && mockgen -source=internal/domain/search/search.go -destination=internal/mocks/mock_search.go -package=mocks
	@echo "Mock generation complete!"
	@echo "Generated mocks:"
	@echo "  - fetcher-service/internal/mocks/mock_api_client.go"
	@echo "  - fetcher-service/internal/mocks/mock_cache.go"
	@echo "  - fetcher-service/internal/mocks/mock_consumer.go"
	@echo "  - search-service/internal/mocks/mock_repository.go"
	@echo "  - search-service/internal/mocks/mock_search.go"

# ----- Environment Setup -----

copy-env:
	@echo "Copying .env.example to .env..."
ifeq ($(DETECTED_OS),Windows)
	@$(SHELL_CMD) "if ($(TEST_PATH) .env.example) { Copy-Item .env.example .env -Force; Write-Host '.env file created successfully from .env.example' } else { Write-Host 'Error: .env.example file not found'; exit 1 }"
else
	@if $(TEST_PATH) .env.example; then \
		cp .env.example .env; \
		echo ".env file created successfully from .env.example"; \
	else \
		echo "Error: .env.example file not found"; \
		exit 1; \
	fi
endif
