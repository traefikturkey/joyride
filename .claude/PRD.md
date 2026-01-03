# PRD: CoreDNS for Dynamic Docker Container DNS

## Overview

Replace Joyride (dnsmasq-based) with CoreDNS for dynamic DNS resolution of Docker containers. CoreDNS provides better fallback handling, plugin extensibility, and native support for forwarding unknown queries to upstream DNS servers.

---

## Problem Statement

### Current State (Joyride)
- Uses dnsmasq to provide DNS for Docker containers
- Containers register via `joyride.host.name` labels
- EdgeRouter forwards domain-specific queries (e.g., `*.ilude.com`) to Joyride
- **Issue**: When Joyride doesn't know a hostname, dnsmasq returns REFUSED or NXDOMAIN
- **Issue**: dnsmasq has no option to forward unknown queries to upstream DNS
- **Issue**: EdgeRouter's dnsmasq doesn't retry on REFUSED/NXDOMAIN responses
- **Result**: External domains (e.g., `www.ilude.com` → GitHub Pages) fail intermittently

### Desired State (CoreDNS)
- Dynamic DNS for Docker containers (same functionality as Joyride)
- Unknown queries forwarded to upstream DNS (8.8.8.8, 1.1.1.1)
- Proper fallback behavior without EdgeRouter workarounds
- Optional: Multi-node support via etcd or similar

---

## Requirements

### Functional Requirements

#### FR-1: Docker Container Discovery
- **FR-1.1**: Watch Docker socket (`/var/run/docker.sock`) for container events
- **FR-1.2**: Register DNS entries for containers with specific labels
- **FR-1.3**: Support configurable label name (default: `coredns.host.name` or `joyride.host.name` for compatibility)
- **FR-1.4**: Remove DNS entries when containers stop
- **FR-1.5**: Support multiple hostnames per container (comma-separated labels)

#### FR-2: DNS Resolution
- **FR-2.1**: Resolve registered container hostnames to host IP address
- **FR-2.2**: Forward unknown queries to upstream DNS servers
- **FR-2.3**: Support A, AAAA, and CNAME record types
- **FR-2.4**: Configurable TTL for responses (default: 60 seconds)

#### FR-3: Network Integration
- **FR-3.1**: Listen on configurable port (default: 54 for EdgeRouter compatibility)
- **FR-3.2**: Listen on configurable interface/IP (default: 0.0.0.0)
- **FR-3.3**: Support UDP and TCP DNS queries

#### FR-4: Upstream Forwarding
- **FR-4.1**: Forward queries for unknown hostnames to upstream DNS
- **FR-4.2**: Configurable upstream DNS servers (default: 8.8.8.8, 1.1.1.1)
- **FR-4.3**: Health checking of upstream servers
- **FR-4.4**: Fallback to secondary upstream if primary fails

#### FR-5: Compatibility
- **FR-5.1**: Drop-in replacement for Joyride (same port, same label support)
- **FR-5.2**: No changes required to EdgeRouter configuration
- **FR-5.3**: Support existing `joyride.host.name` label (configurable)

### Non-Functional Requirements

#### NFR-1: Performance
- DNS query response time < 10ms for local records
- Support 1000+ registered hostnames
- Minimal memory footprint (< 50MB)

#### NFR-2: Reliability
- Graceful handling of Docker daemon restarts
- Automatic reconnection to Docker socket
- No DNS downtime during container churn

#### NFR-3: Observability
- Structured logging (JSON format)
- Prometheus metrics endpoint
- Query logging (optional, configurable)

#### NFR-4: Security
- Read-only Docker socket access
- No privileged container required
- Support for DNS over TLS to upstream (optional)

---

## Architecture

### Option A: CoreDNS with Custom Plugin

```
┌─────────────────────────────────────────────────────────┐
│                    CoreDNS Container                     │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   docker     │  │    file      │  │   forward    │  │
│  │   plugin     │  │   plugin     │  │   plugin     │  │
│  │              │  │  (fallback)  │  │  (upstream)  │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│         │                                    │          │
│         ▼                                    ▼          │
│  ┌──────────────┐                    ┌──────────────┐  │
│  │   Docker     │                    │   Upstream   │  │
│  │   Socket     │                    │  DNS (8.8.8) │  │
│  └──────────────┘                    └──────────────┘  │
└─────────────────────────────────────────────────────────┘
```

**Pros:**
- Single container
- Native CoreDNS plugin ecosystem
- Best performance

**Cons:**
- Requires custom plugin development or finding existing docker plugin

### Option B: CoreDNS with External Hosts File Generator

```
┌─────────────────┐       ┌─────────────────┐
│  Docker Watcher │       │    CoreDNS      │
│   (Go/Python)   │──────▶│                 │
│                 │ hosts │  ┌───────────┐  │
│  Watches labels │ file  │  │   hosts   │  │
│  Writes /hosts  │       │  │  plugin   │  │
└─────────────────┘       │  └───────────┘  │
        │                 │  ┌───────────┐  │
        ▼                 │  │  forward  │  │
┌─────────────────┐       │  │  plugin   │  │
│  Docker Socket  │       │  └───────────┘  │
└─────────────────┘       └─────────────────┘
```

**Pros:**
- Uses standard CoreDNS (no custom plugins)
- Simpler to implement and maintain
- Can reuse Joyride's Docker watching logic

**Cons:**
- Two processes/containers
- File-based sync (slight delay)
- Requires shared volume for hosts file

### Option C: CoreDNS with etcd Backend (Multi-Node)

```
┌─────────────────┐       ┌─────────────────┐
│  Docker Watcher │       │    CoreDNS      │
│     Node 1      │──┐    │     Node 1      │
└─────────────────┘  │    └─────────────────┘
                     │            │
┌─────────────────┐  │    ┌───────▼───────┐
│  Docker Watcher │──┼───▶│     etcd      │
│     Node 2      │  │    │   cluster     │
└─────────────────┘  │    └───────▲───────┘
                     │            │
┌─────────────────┐  │    ┌─────────────────┐
│  Docker Watcher │──┘    │    CoreDNS      │
│     Node 3      │       │     Node 2      │
└─────────────────┘       └─────────────────┘
```

**Pros:**
- Multi-node support (replaces Joyride's Serf)
- Centralized DNS database
- High availability

**Cons:**
- More complex infrastructure
- Requires etcd cluster
- Overkill for single-node setups

### Recommended: Option A (Custom CoreDNS Plugin)

After reviewing joyride-python's SWIM implementation and existing CoreDNS Docker plugins, **Option A is recommended**:

**Single `docker-cluster` plugin** that combines:
1. Docker socket watching for container labels
2. SWIM clustering via HashiCorp memberlist (same protocol as Serf)
3. Built-in `fallthrough` to CoreDNS's `forward` plugin for upstream DNS

```
┌─────────────────────────────────────────────────────────────┐
│                    CoreDNS Container                        │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐ │
│  │  docker-cluster plugin                                │ │
│  │  ┌─────────────────┐    ┌──────────────────────────┐ │ │
│  │  │ Docker Watcher  │    │ memberlist (SWIM)        │ │ │
│  │  │ - socket events │◄──►│ - cluster sync           │ │ │
│  │  │ - label extract │    │ - gossip DNS records     │ │ │
│  │  └─────────────────┘    └──────────────────────────┘ │ │
│  │              │                      │                 │ │
│  │              └──────┬───────────────┘                 │ │
│  │                     ▼                                 │ │
│  │           [In-memory DNS records]                     │ │
│  │                     │                                 │ │
│  │              ServeDNS()                               │ │
│  └───────────────────────────────────────────────────────┘ │
│                        │ fallthrough                        │
│                        ▼                                    │
│  ┌───────────────────────────────────────────────────────┐ │
│  │  forward plugin (built-in) → 8.8.8.8, 1.1.1.1        │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

**Advantages over Option B:**
- Single container (no sidecar process)
- No file-based sync delays
- Native clustering support (replaces Joyride's Serf)
- Lock-free DNS reads via atomic pointer swap
- CoreDNS's `forward` plugin handles upstream DNS (solves the main problem)

---

## CoreDNS Configuration

### Corefile (Single Node)

```
.:54 {
    # Custom docker-cluster plugin
    docker-cluster {
        docker_socket unix:///var/run/docker.sock
        host_ip 192.168.16.61
        label coredns.host.name
        label joyride.host.name
        ttl 60
        fallthrough
    }

    # Forward unknown queries to upstream DNS
    forward . 8.8.8.8 1.1.1.1 {
        health_check 30s
    }

    # Optional: Query logging (trade-off: volume vs debugging)
    # log

    # Error logging
    errors

    # Health check endpoint
    health :8080

    # Prometheus metrics
    prometheus :9153

    # Cache responses
    cache 300
}
```

### Corefile (Multi-Node Cluster)

```
.:54 {
    docker-cluster {
        docker_socket unix:///var/run/docker.sock
        host_ip 192.168.16.61
        label coredns.host.name
        label joyride.host.name
        ttl 60

        # Clustering via memberlist (SWIM protocol)
        cluster_enabled true
        cluster_port 7946
        cluster_seeds 192.168.16.62:7946,192.168.16.63:7946
        node_name coredns-node-1

        fallthrough
    }

    forward . 8.8.8.8 1.1.1.1
    errors
    health :8080
    prometheus :9153
    cache 300
}
```

---

## Docker Compose

### Single Node Deployment

```yaml
services:
  coredns:
    build: .
    container_name: coredns-docker-cluster
    restart: unless-stopped
    user: "1000:1000"
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE
    read_only: true
    security_opt:
      - no-new-privileges:true
    ports:
      - "54:54/udp"
      - "54:54/tcp"
      - "8080:8080"   # Health check
      - "9153:9153"   # Prometheus metrics
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./Corefile:/etc/coredns/Corefile:ro
    deploy:
      resources:
        limits:
          memory: 128M
          cpus: '0.5'
    healthcheck:
      test: ["CMD", "wget", "-q", "-O-", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 3
```

### Multi-Node Cluster Deployment

```yaml
services:
  coredns-node1:
    build: .
    environment:
      - NODE_NAME=node1
      - HOSTIP=192.168.16.61
      - CLUSTER_SEEDS=192.168.16.62:7946
    ports:
      - "54:54/udp"
      - "54:54/tcp"
      - "192.168.16.61:7946:7946/udp"  # Bind memberlist to internal IP
      - "192.168.16.61:7946:7946/tcp"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./Corefile:/etc/coredns/Corefile:ro
```

---

## Migration Plan

### Phase 1: Parallel Deployment
1. Deploy CoreDNS alongside Joyride (different port, e.g., 55)
2. Test container DNS resolution against CoreDNS
3. Test upstream forwarding for unknown domains
4. Verify no regressions

### Phase 2: EdgeRouter Cutover
1. Update EdgeRouter to point to CoreDNS:
   ```
   set service dns forwarding options 'server=/ilude.com/192.168.16.61#55'
   ```
2. Monitor for issues
3. Keep Joyride running as fallback

### Phase 3: Joyride Decommission
1. Stop Joyride container
2. Move CoreDNS to port 54
3. Update EdgeRouter back to port 54:
   ```
   set service dns forwarding options 'server=/ilude.com/192.168.16.61#54'
   ```
4. Remove Joyride

### Rollback Plan
1. Stop CoreDNS
2. Start Joyride
3. Revert EdgeRouter to Joyride port

---

## Success Criteria

1. **Container DNS**: All containers with labels resolve correctly
2. **Upstream Forwarding**: `www.ilude.com` resolves to GitHub Pages via upstream
3. **No REFUSED/NXDOMAIN**: Unknown queries forward upstream, never refuse
4. **Performance**: Query latency < 10ms for local, < 50ms for upstream
5. **Reliability**: Zero DNS downtime during container restarts
6. **Compatibility**: No EdgeRouter configuration changes required (same port)

---

## Open Questions

1. ~~**Multi-node**: Is Serf-like multi-node support needed?~~ **RESOLVED**: Yes, using HashiCorp memberlist (SWIM protocol) - same as Serf uses internally.
2. **IPv6**: Should container DNS support AAAA records?
3. **Wildcard**: Should `*.app.ilude.com` wildcards be supported?
4. **Health checks**: Should containers only get DNS if healthy?
5. **TTL**: What TTL is appropriate for container records? (default: 60s)

---

## Timeline

| Phase | Description | Duration |
|-------|-------------|----------|
| 1 | Docker watcher implementation | - |
| 2 | CoreDNS configuration & testing | - |
| 3 | Parallel deployment & validation | - |
| 4 | EdgeRouter cutover | - |
| 5 | Joyride decommission | - |

---

## References

- [CoreDNS Documentation](https://coredns.io/manual/toc/)
- [CoreDNS hosts plugin](https://coredns.io/plugins/hosts/)
- [CoreDNS forward plugin](https://coredns.io/plugins/forward/)
- [Joyride GitHub](https://github.com/traefikturkey/joyride)
- [dnsmasq limitations](https://github.com/openwrt/openwrt/issues/5808)
