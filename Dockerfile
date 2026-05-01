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
# Clones CoreDNS source - cached until base image changes
# -----------------------------------------------------------------------------
# Pinned by digest for reproducibility (Dependabot keeps this current).
FROM golang:1.25-alpine@sha256:5caaf1cca9dc351e13deafbc3879fd4754801acba8653fa9540cea125d01a71f AS base

RUN apk add --no-cache git

# Pinned CoreDNS version. Keep in sync with go.mod and the test job in
# .github/workflows/docker-publish.yml. Bump deliberately, not implicitly.
ARG COREDNS_VERSION=v1.14.3

WORKDIR /build

RUN echo "Building with CoreDNS ${COREDNS_VERSION}" && \
    echo "${COREDNS_VERSION}" > /coredns-version && \
    git clone --depth 1 --branch ${COREDNS_VERSION} https://github.com/coredns/coredns.git

WORKDIR /build/coredns

# Copy plugin.cfg first (changes less frequently)
COPY plugin.cfg /build/coredns/plugin.cfg

# -----------------------------------------------------------------------------
# Stage: deps
# Copies plugin source and installs dependencies
# This layer is cached and reused by both dev and builder stages
# -----------------------------------------------------------------------------
FROM base AS deps

# Copy plugin source files (including subdirectories like version/)
COPY plugins/docker-cluster/ /build/coredns/plugin/docker-cluster/
COPY plugins/traefik-externals/ /build/coredns/plugin/traefik-externals/

# Add dependencies and download (must be after plugin copy so go mod tidy works)
RUN go get github.com/docker/docker@v28.5.2+incompatible && \
    go get github.com/hashicorp/memberlist@v0.5.1 && \
    go get github.com/fsnotify/fsnotify@v1.7.0 && \
    go mod tidy && \
    go mod download

# -----------------------------------------------------------------------------
# Stage: dev
# Development image - includes Go toolchain for rapid iteration
# Mounts source code as volume for instant rebuilds without re-downloading deps
# -----------------------------------------------------------------------------
FROM deps AS dev

# Install additional dev tools
RUN apk add --no-cache make build-base

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
FROM deps AS builder

# Build arguments for version injection
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# Generate plugin wiring and build static binary
RUN CGO_ENABLED=0 GOOS=linux go generate && \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w \
    -X github.com/coredns/coredns/plugin/docker-cluster/version.Version=${VERSION} \
    -X github.com/coredns/coredns/plugin/docker-cluster/version.GitCommit=${GIT_COMMIT} \
    -X github.com/coredns/coredns/plugin/docker-cluster/version.BuildTime=${BUILD_TIME}" \
    -o /coredns .

# -----------------------------------------------------------------------------
# Stage: production
# Minimal runtime image - only the binary and essential runtime deps
# -----------------------------------------------------------------------------
FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d AS production

# Receive build args for labels
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

# OCI image labels
LABEL org.opencontainers.image.title="CoreDNS Docker Cluster"
LABEL org.opencontainers.image.description="CoreDNS with docker-cluster and traefik-externals plugins"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.created="${BUILD_TIME}"
LABEL org.opencontainers.image.source="https://github.com/traefikturkey/joyride"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# Install minimal runtime dependencies
# - ca-certificates: HTTPS/TLS support
# - wget: health checks
# - su-exec: privilege dropping (smaller than gosu)
# - iproute2: HOSTIP auto-detection
# - libcap: setcap, so the non-root coredns user can bind privileged ports (:54)
RUN apk add --no-cache ca-certificates wget su-exec iproute2 libcap

# Create non-root user
RUN adduser -D -u 1000 coredns

# Copy the static binary and grant it the file capability needed to bind
# privileged ports (DNS on :54) when running as the non-root coredns user.
# NET_BIND_SERVICE is already in the container's bounding set via cap_add,
# so this does not escalate beyond what compose already grants.
COPY --from=builder /coredns /coredns
RUN setcap cap_net_bind_service=+ep /coredns

# Create entrypoint script
# - Auto-detects HOSTIP from default route if not set
# - Grants coredns user access to the Docker socket via group membership
#   (matches the socket's GID instead of chmod 666, so the socket stays
#   readable only to root + that group, not world)
# - Drops privileges to coredns user
RUN cat > /entrypoint.sh << 'EOF'
#!/bin/sh
set -e

if [ -z "$HOSTIP" ]; then
    HOSTIP=$(ip route get 1 2>/dev/null | awk '{print $7}' | head -1)
    if [ -n "$HOSTIP" ]; then
        export HOSTIP
        echo "[INFO] Auto-detected HOSTIP=$HOSTIP"
    fi
fi

if [ "$(id -u)" = "0" ]; then
    if [ -S /var/run/docker.sock ]; then
        SOCK_GID=$(stat -c '%g' /var/run/docker.sock)
        if ! getent group "$SOCK_GID" >/dev/null 2>&1; then
            addgroup -g "$SOCK_GID" dockersock
        fi
        SOCK_GROUP=$(getent group "$SOCK_GID" | cut -d: -f1)
        addgroup coredns "$SOCK_GROUP" >/dev/null 2>&1 || true
        exec su-exec coredns /coredns -conf /etc/coredns/Corefile "$@"
    fi
    exec su-exec coredns /coredns -conf /etc/coredns/Corefile "$@"
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
