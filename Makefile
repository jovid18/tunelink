.PHONY: up down api web logs clean build

# Start MySQL and Redis
up:
	docker-compose up -d

# Stop all containers
down:
	docker-compose down

# Run API server (requires MySQL and Redis running)
api:
	cd apps/api && go run ./cmd/api

# Run Web dev server
web:
	cd apps/web && npm run dev

# View logs
logs:
	docker-compose logs -f

# Clean up volumes
clean:
	docker-compose down -v

# Build Docker images
build-api:
	docker build -t tunelink-api ./apps/api

build-web:
	docker build -t tunelink-web ./apps/web

build: build-api build-web

# Run everything in containers
run-all: up
	@echo "Waiting for MySQL to be ready..."
	@sleep 10
	docker-compose up -d
	@echo "Starting API and Web..."
	@echo "API: http://localhost:8080"
	@echo "Web: http://localhost:5173"

# Install dependencies
install:
	cd apps/api && go mod download
	cd apps/web && npm install

# Help
help:
	@echo "Available commands:"
	@echo "  make up        - Start MySQL and Redis containers"
	@echo "  make down      - Stop all containers"
	@echo "  make api       - Run Go API server"
	@echo "  make web       - Run React dev server"
	@echo "  make logs      - View container logs"
	@echo "  make clean     - Remove containers and volumes"
	@echo "  make build     - Build Docker images"
	@echo "  make install   - Install all dependencies"
