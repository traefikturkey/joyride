# =============================================================================
# Multi-stage Dockerfile for CoreDNS with custom plugins
#
# Targets:
#   dev        - Development image with Go toolchain for quick rebuilds
#   production - Minimal production image (~15MB)
#
# Usage:
#   docker build --target dev -t coredns-dev .
#   docker build --target production -t coredns-docker-cluster .
#   docker build -t coredns-docker-cluster .  # defaults to production
# =============================================================================

# -----------------------------------------------------------------------------
# Stage: base
# Common setup - clones CoreDNS and installs Go dependencies
# This layer is cached and reused by both dev and production builds
# -----------------------------------------------------------------------------
FROM golang:1.24-alpine AS base

RUN apk add --no-cache git curl jq

WORKDIR /build

# Fetch latest CoreDNS release version and clone
# This is cached until the base image changes
RUN COREDNS_VERSION=$(curl -s https://api.github.com/repos/coredns/coredns/releases/latest | jq -r '.tag_name') && \
    echo "Building with CoreDNS ${COREDNS_VERSION}" && \
    echo "${COREDNS_VERSION}" > /coredns-version && \
    git clone --depth 1 --branch ${COREDNS_VERSION} https://github.com/coredns/coredns.git

WORKDIR /build/coredns

# Copy plugin.cfg first (changes less frequently)
COPY plugin.cfg /build/coredns/plugin.cfg

# Add dependencies to CoreDNS go.mod and download
RUN go get github.com/docker/docker@v28.5.2+incompatible && \
    go get github.com/fsnotify/fsnotify@v1.7.0 && \
    go mod tidy && \
    go mod download

# -----------------------------------------------------------------------------
# Stage: dev
# Development image - includes Go toolchain for rapid iteration
# Mounts source code as volume for instant rebuilds without re-downloading deps
# -----------------------------------------------------------------------------
FROM base AS dev

# Install additional dev tools
RUN apk add --no-cache make build-base

# Copy plugin source files
COPY plugins/docker-cluster/*.go /build/coredns/plugin/docker-cluster/
COPY plugins/traefik-externals/*.go /build/coredns/plugin/traefik-externals/

# Generate plugin wiring
RUN go generate

# Build with race detector support (useful for testing)
ENV CGO_ENABLED=1
ENV GOOS=linux

# Default command: rebuild and run
# For interactive development, override with: docker run -it coredns-dev sh
CMD ["sh", "-c", "go build -race -o /coredns . && /coredns -conf /etc/coredns/Corefile"]

# Copy config files for standalone dev testing
COPY Corefile /etc/coredns/Corefile
COPY etc/joyride/hosts.d/hosts /etc/hosts.d/hosts

EXPOSE 54/udp 54/tcp 5454 9153

# -----------------------------------------------------------------------------
# Stage: builder
# Compiles the production binary (static, no CGO)
# -----------------------------------------------------------------------------
FROM base AS builder

# Copy plugin source files
COPY plugins/docker-cluster/*.go /build/coredns/plugin/docker-cluster/
COPY plugins/traefik-externals/*.go /build/coredns/plugin/traefik-externals/

# Generate plugin wiring and build static binary
RUN CGO_ENABLED=0 GOOS=linux go generate && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /coredns .

# -----------------------------------------------------------------------------
# Stage: production
# Minimal runtime image - only the binary and essential runtime deps
# -----------------------------------------------------------------------------
FROM alpine:3.20 AS production

# Install minimal runtime dependencies
# - ca-certificates: HTTPS/TLS support
# - wget: health checks
# - su-exec: privilege dropping (smaller than gosu)
# - iproute2: HOSTIP auto-detection
RUN apk add --no-cache ca-certificates wget su-exec iproute2

# Create non-root user
RUN adduser -D -u 1000 coredns

# Copy the static binary
COPY --from=builder /coredns /coredns

# Create entrypoint script
# - Auto-detects HOSTIP from default route if not set
# - Fixes Docker socket permissions
# - Drops privileges to coredns user
RUN cat > /entrypoint.sh << 'EOF'
#!/bin/sh
set -e

# Auto-detect HOSTIP if not set (requires host networking)
if [ -z "$HOSTIP" ]; then
    HOSTIP=$(ip route get 1 2>/dev/null | awk '{print $7}' | head -1)
    if [ -n "$HOSTIP" ]; then
        export HOSTIP
        echo "[INFO] Auto-detected HOSTIP=$HOSTIP"
    fi
fi

if [ "$(id -u)" = "0" ]; then
    # Try to make Docker socket readable by coredns user
    if [ -S /var/run/docker.sock ]; then
        if chmod 666 /var/run/docker.sock 2>/dev/null; then
            exec su-exec coredns /coredns -conf /etc/coredns/Corefile "$@"
        fi
    fi
    # Chmod failed or no socket - run as root
    exec /coredns -conf /etc/coredns/Corefile "$@"
else
    exec /coredns -conf /etc/coredns/Corefile "$@"
fi
EOF
RUN chmod 755 /entrypoint.sh

# Copy configuration
COPY Corefile /etc/coredns/Corefile
COPY etc/joyride/hosts.d/hosts /etc/hosts.d/hosts

# Set ownership
RUN chown -R coredns:coredns /etc/coredns /etc/hosts.d

EXPOSE 54/udp 54/tcp 5454 9153

HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -q -O- http://localhost:5454/health || exit 1

ENTRYPOINT ["/entrypoint.sh"]
