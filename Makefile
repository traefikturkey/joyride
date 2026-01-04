.PHONY: build build-dev build-latest test test-unit test-race test-integration run run-dev clean check-coredns-version audit

# Default target
all: build

# Build production image (minimal, ~15MB)
build:
	docker build --target production -t coredns-docker-cluster .

# Build development image (includes Go toolchain for quick rebuilds)
build-dev:
	docker build --target dev -t coredns-dev .

# Build with no cache (forces fresh CoreDNS fetch)
build-latest:
	docker build --no-cache --target production -t coredns-docker-cluster .

# Run all tests
test: test-unit test-integration

# Run unit tests (inside Docker with CoreDNS environment)
test-unit:
	@echo "Running unit tests in Docker..."
	@MSYS_NO_PATHCONV=1 docker run --rm -v "$$(pwd)":/src -w /src golang:1.24-alpine sh -c '\
		apk add --no-cache git curl jq >/dev/null 2>&1 && \
		COREDNS_VERSION=$$(curl -s https://api.github.com/repos/coredns/coredns/releases/latest | jq -r ".tag_name") && \
		echo "Testing with CoreDNS $$COREDNS_VERSION" && \
		git clone --depth 1 --branch $$COREDNS_VERSION https://github.com/coredns/coredns.git /tmp/coredns 2>/dev/null && \
		cp -r plugins/docker-cluster /tmp/coredns/plugin/docker-cluster && \
		cp -r plugins/traefik-externals /tmp/coredns/plugin/traefik-externals && \
		cp plugin.cfg /tmp/coredns/plugin.cfg && \
		cd /tmp/coredns && \
		go get github.com/docker/docker@v28.5.2+incompatible && \
		go get github.com/fsnotify/fsnotify@v1.7.0 && \
		go mod tidy && \
		echo "Running traefik-externals tests..." && \
		go test -v -timeout 60s ./plugin/traefik-externals/... && \
		echo "Running docker-cluster tests..." && \
		go test -v -timeout 60s ./plugin/docker-cluster/...'

# Run unit tests with race detector (slower but catches race conditions)
test-race:
	@echo "Running unit tests with race detector in Docker..."
	@MSYS_NO_PATHCONV=1 docker run --rm -v "$$(pwd)":/src -w /src golang:1.24-alpine sh -c '\
		apk add --no-cache git curl jq build-base >/dev/null 2>&1 && \
		COREDNS_VERSION=$$(curl -s https://api.github.com/repos/coredns/coredns/releases/latest | jq -r ".tag_name") && \
		echo "Testing with CoreDNS $$COREDNS_VERSION (race detection enabled)" && \
		git clone --depth 1 --branch $$COREDNS_VERSION https://github.com/coredns/coredns.git /tmp/coredns 2>/dev/null && \
		cp -r plugins/docker-cluster /tmp/coredns/plugin/docker-cluster && \
		cp -r plugins/traefik-externals /tmp/coredns/plugin/traefik-externals && \
		cp plugin.cfg /tmp/coredns/plugin.cfg && \
		cd /tmp/coredns && \
		go get github.com/docker/docker@v28.5.2+incompatible && \
		go get github.com/fsnotify/fsnotify@v1.7.0 && \
		go mod tidy && \
		echo "Running tests with race detector..." && \
		CGO_ENABLED=1 go test -v -race -timeout 120s ./plugin/docker-cluster/... ./plugin/traefik-externals/...'

# Run integration tests
test-integration:
	docker compose -f docker-compose.test.yml up --build --abort-on-container-exit
	docker compose -f docker-compose.test.yml down -v

# Run cluster tests (Phase 2)
test-cluster:
	docker compose -f docker-compose.cluster-test.yml up --build --abort-on-container-exit
	docker compose -f docker-compose.cluster-test.yml down -v

# Run production environment
run:
	docker compose up --build

# Run in background
run-detached:
	docker compose up --build -d

# Run dev container with source mounted for quick rebuilds
# Usage: make run-dev
# Then inside container: go build -o /coredns . && /coredns -conf /etc/coredns/Corefile
run-dev: build-dev
	@MSYS_NO_PATHCONV=1 docker run -it --rm \
		-v "$$(pwd)/plugins/docker-cluster":/build/coredns/plugin/docker-cluster \
		-v "$$(pwd)/plugins/traefik-externals":/build/coredns/plugin/traefik-externals \
		-v "$$(pwd)/Corefile":/etc/coredns/Corefile \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-p 54:54/udp -p 54:54/tcp -p 5454:5454 -p 9153:9153 \
		-e HOSTIP=$${HOSTIP:-192.168.1.100} \
		--network host \
		coredns-dev sh

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
	go fmt ./plugins/...

# Lint Go code
lint:
	golangci-lint run ./plugins/...

# Audit dependencies for known vulnerabilities (uses govulncheck via Docker)
audit:
	@echo "Scanning for vulnerabilities..."
	@MSYS_NO_PATHCONV=1 docker run --rm -v "$$(pwd)":/src -w /src golang:1.24-alpine sh -c '\
		apk add --no-cache git curl jq >/dev/null 2>&1 && \
		go install golang.org/x/vuln/cmd/govulncheck@latest && \
		COREDNS_VERSION=$$(curl -s https://api.github.com/repos/coredns/coredns/releases/latest | jq -r ".tag_name") && \
		git clone --depth 1 --branch $$COREDNS_VERSION https://github.com/coredns/coredns.git /tmp/coredns 2>/dev/null && \
		cp -r plugins/docker-cluster /tmp/coredns/plugin/docker-cluster && \
		cp -r plugins/traefik-externals /tmp/coredns/plugin/traefik-externals && \
		cp plugin.cfg /tmp/coredns/plugin.cfg && \
		cd /tmp/coredns && \
		go get github.com/docker/docker@v28.5.2+incompatible && \
		go get github.com/fsnotify/fsnotify@v1.7.0 && \
		go mod tidy && \
		govulncheck ./...'

# Generate test coverage report
coverage:
	go test -coverprofile=coverage.out ./plugins/...
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
