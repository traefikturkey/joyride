.PHONY: build test test-unit test-integration run clean check-coredns-version build-latest

# Default target
all: build

# Build the Docker image
build:
	docker build -t coredns-docker-cluster .

# Build with no cache (forces fresh CoreDNS fetch)
build-latest:
	docker build --no-cache -t coredns-docker-cluster .

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	go test -v -race ./plugin/...

# Run integration tests
test-integration:
	docker compose -f docker-compose.test.yml up --build --abort-on-container-exit
	docker compose -f docker-compose.test.yml down -v

# Run cluster tests (Phase 2)
test-cluster:
	docker compose -f docker-compose.cluster-test.yml up --build --abort-on-container-exit
	docker compose -f docker-compose.cluster-test.yml down -v

# Run development environment
run:
	docker compose up --build

# Run in background
run-detached:
	docker compose up --build -d

# Stop and clean up
clean:
	docker compose down -v
	docker compose -f docker-compose.test.yml down -v 2>/dev/null || true
	docker compose -f docker-compose.cluster-test.yml down -v 2>/dev/null || true

# Check latest CoreDNS version
check-coredns-version:
	@echo "Latest CoreDNS release:"
	@curl -s https://api.github.com/repos/coredns/coredns/releases/latest | grep '"tag_name"' | cut -d'"' -f4

# Format Go code
fmt:
	go fmt ./plugin/...

# Lint Go code
lint:
	golangci-lint run ./plugin/...

# Generate test coverage report
coverage:
	go test -coverprofile=coverage.out ./plugin/...
	go tool cover -html=coverage.out -o coverage.html

# View logs
logs:
	docker compose logs -f

# DNS query test (requires dig)
dns-test:
	@echo "Testing container DNS..."
	dig @localhost -p 54 test.example.com +short
	@echo "Testing upstream forwarding..."
	dig @localhost -p 54 www.google.com +short
