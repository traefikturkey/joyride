# Stage 1: Build CoreDNS with custom docker-cluster plugin
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make curl jq

WORKDIR /build

# Fetch latest CoreDNS release version and clone
RUN COREDNS_VERSION=$(curl -s https://api.github.com/repos/coredns/coredns/releases/latest | jq -r '.tag_name') && \
    echo "Building with CoreDNS ${COREDNS_VERSION}" && \
    git clone --depth 1 --branch ${COREDNS_VERSION} https://github.com/coredns/coredns.git

WORKDIR /build/coredns

# Copy custom plugin source files
COPY plugin/*.go /build/coredns/plugin/docker-cluster/

# Copy custom plugin.cfg that includes docker-cluster
COPY plugin.cfg /build/coredns/plugin.cfg

# Add Docker client dependency to CoreDNS go.mod
RUN go get github.com/docker/docker@v27.0.0+incompatible && \
    go mod tidy

# Generate plugin wiring and build
RUN CGO_ENABLED=0 GOOS=linux go generate && \
    CGO_ENABLED=0 GOOS=linux go build -o coredns .

# Stage 2: Runtime image
FROM alpine:3.20

# Install runtime dependencies
# - ca-certificates: HTTPS support
# - wget: health checks
# - su-exec: drop privileges (like gosu but smaller)
# - iproute2: auto-detect HOSTIP from default route
RUN apk add --no-cache ca-certificates wget su-exec iproute2

# Create non-root user
RUN adduser -D -u 1000 coredns

# Copy the CoreDNS binary
COPY --from=builder /build/coredns/coredns /coredns

# Create entrypoint script inline
# - Fixes Docker socket permissions (chmod 666)
# - Auto-detects HOSTIP from default route if not set
# - Drops privileges to coredns user via su-exec
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
            # Success - drop privileges and run as coredns user
            exec su-exec coredns /coredns -conf /etc/coredns/Corefile "$@"
        fi
    fi
    # Chmod failed (read-only mount) or no socket - run as root
    exec /coredns -conf /etc/coredns/Corefile "$@"
else
    # Already non-root, just run
    exec /coredns -conf /etc/coredns/Corefile "$@"
fi
EOF
RUN chmod 755 /entrypoint.sh

# Copy default Corefile
COPY Corefile /etc/coredns/Corefile

# Copy default hosts file for static DNS entries
COPY etc/joyride/hosts.d/hosts /etc/hosts.d/hosts

# Set ownership
RUN chown -R coredns:coredns /etc/coredns /etc/hosts.d

# Expose DNS ports, health check, and metrics
EXPOSE 54/udp 54/tcp 5454 9153

# Health check on port 5454
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -q -O- http://localhost:5454/health || exit 1

# Start as root, entrypoint drops to coredns user after fixing permissions
ENTRYPOINT ["/entrypoint.sh"]
