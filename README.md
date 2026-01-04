# CoreDNS Docker DNS

A CoreDNS plugin that provides DNS resolution for Docker containers. Drop-in replacement for Joyride (dnsmasq-based Docker DNS).

## Quick Start

Using the pre-built image:

```bash
docker run -d --name coredns-docker \
  -p 54:54/udp -p 54:54/tcp -p 5454:5454 \
  -e HOSTIP=192.168.16.61 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/traefikturkey/joyride:coredns
```

Or build locally with Docker Compose:

```bash
export HOSTIP=192.168.16.61
docker compose up -d
```

## How It Works

1. Watches Docker for containers with DNS labels
2. Serves A records for labeled containers
3. Unknown hostnames get no response (timeout) - allows EdgeRouter to query upstream

## Container Labels

Add labels to your containers to register DNS names:

```yaml
services:
  myapp:
    image: nginx
    labels:
      - "coredns.host.name=myapp.example.com"
```

Multiple hostnames:
```yaml
labels:
  - "coredns.host.name=app.example.com,api.example.com"
```

Joyride compatibility:
```yaml
labels:
  - "joyride.host.name=myapp.example.com"
```

## Static DNS Entries

For hosts that aren't Docker containers (NAS, printers, etc.), add entries to the hosts file:

```bash
vi ./etc/joyride/hosts.d/hosts
```

Use standard hosts file format:
```
192.168.1.10 nas.example.com nas
192.168.1.20 printer.example.com
```

Static entries take priority over Docker container labels. Changes are picked up automatically (no restart required).

## Traefik External Services

The `traefik-externals` plugin automatically creates DNS records from Traefik external service configurations. This is useful when you have services running outside Docker (VMs, bare metal, external APIs) that are proxied through Traefik.

### How It Works

1. Watches a directory for Traefik YAML config files (default: `/etc/traefik/external-enabled`)
2. Parses `Host()` and `HostSNI()` rules to extract hostnames
3. Creates DNS A records pointing those hostnames to your configured `HOSTIP`
4. File changes are picked up automatically (no restart required)

### Example Traefik External Config

Create a file in your Traefik external configs directory:

```yaml
# /etc/traefik/external-enabled/proxmox.yml
http:
  routers:
    proxmox:
      rule: Host(`proxmox.example.com`)
      service: proxmox
      entryPoints:
        - websecure
  services:
    proxmox:
      loadBalancer:
        servers:
          - url: https://192.168.1.100:8006
```

The plugin extracts `proxmox.example.com` from the `Host()` rule and creates a DNS record for it.

### Supported Patterns

```yaml
# Single host
rule: Host(`app.example.com`)

# Multiple hosts
rule: Host(`app.example.com`, `www.example.com`)

# HostSNI for TCP/TLS services
rule: HostSNI(`db.example.com`)

# Environment variable substitution
rule: Host(`app.{{env "DOMAIN"}}`)  # Resolves $DOMAIN at runtime
```

**Not supported:** `HostRegexp()` patterns (cannot enumerate regex matches - a warning is logged)

### Configuration

Mount your Traefik external configs directory:

```yaml
services:
  coredns:
    volumes:
      - /path/to/traefik/external-enabled:/etc/traefik/external-enabled:ro
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TRAEFIK_EXTERNALS_ENABLED` | Enable/disable the plugin | `true` |
| `TRAEFIK_EXTERNALS_DIRECTORY` | Config directory to watch | `/etc/traefik/external-enabled` |
| `TRAEFIK_EXTERNALS_TTL` | TTL for DNS responses | `60` |

### Disabling the Plugin

If you don't use Traefik external services:

```bash
docker run -e TRAEFIK_EXTERNALS_ENABLED=false ...
```

The plugin also gracefully disables itself if the config directory doesn't exist.

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `HOSTIP` | IP address to return for all container DNS records | Auto-detected (host network) |
| `DOCKER_SOCKET` | Docker socket path | `unix:///var/run/docker.sock` |
| `DNS_UNKNOWN_ACTION` | What to do for unknown hostnames: `drop` or `nxdomain` | `drop` |

### Legacy Joyride Compatibility

For backward compatibility with existing joyride configurations:

| Legacy Variable | Maps To | Notes |
|-----------------|---------|-------|
| `JOYRIDE_ENABLE_SERF` | `CLUSTER_ENABLED` | `true` enables clustering |
| `JOYRIDE_NXDOMAIN_ENABLED` | `DNS_UNKNOWN_ACTION` | `true` = nxdomain, `false` = drop |

New variables take precedence if both are set. Deprecation warnings are logged.

### Corefile Options

```
docker-cluster {
    docker_socket unix:///var/run/docker.sock
    host_ip 192.168.16.61
    label coredns.host.name
    label joyride.host.name
    ttl 60
    unknown_action drop
}
```

## Clustering

Multiple CoreDNS nodes can share DNS records using SWIM gossip protocol. Each node watches its local Docker daemon and replicates records to peers.

### Quick Start (Automatic Discovery)

Using Docker Compose with host networking (recommended for clustering):

```bash
# HOSTIP is auto-detected when using host networking
NODE_NAME=node1 docker compose -f docker-compose.host.yml up -d
```

Or manually:

```bash
# Node 1 - HOSTIP auto-detected from default route
docker run -d --network host \
  -e CLUSTER_ENABLED=true \
  -e NODE_NAME=node1 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/traefikturkey/joyride:coredns

# Node 2 (automatically finds Node 1 via broadcast)
docker run -d --network host \
  -e CLUSTER_ENABLED=true \
  -e NODE_NAME=node2 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/traefikturkey/joyride:coredns
```

**Note:** Host networking enables:
- Automatic HOSTIP detection from the default network interface
- UDP broadcast discovery (nodes find each other automatically)
- No need for `CLUSTER_SEEDS` configuration

### Static Seeds (Bridge Networks)

For Docker bridge networks where broadcast doesn't work:

```yaml
environment:
  - CLUSTER_ENABLED=true
  - CLUSTER_SEEDS=192.168.16.61:7946,192.168.16.62:7946
  - NODE_NAME=node3
```

### Cluster Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `CLUSTER_ENABLED` | Enable clustering | `false` |
| `NODE_NAME` | Unique node identifier | Required when clustering |
| `CLUSTER_PORT` | Memberlist gossip port | `7946` |
| `CLUSTER_SEEDS` | Comma-separated seed nodes (host:port) | Empty (use broadcast) |
| `CLUSTER_BIND_ADDR` | Address to bind memberlist | `0.0.0.0` |
| `DISCOVERY_PORT` | UDP port for broadcast discovery | `8889` |
| `CLUSTER_SECRET` | Encryption key for cluster traffic | None |

### How It Works

1. Each node watches its local Docker daemon for labeled containers
2. When a container starts/stops, the node broadcasts the change via SWIM gossip
3. All nodes maintain a merged view of all cluster records
4. Any node can answer DNS queries for any container in the cluster

### Testing Clustering

```bash
# Run 3-node cluster test
make test-cluster
```

## EdgeRouter Setup

Configure EdgeRouter to forward specific domains to CoreDNS:

```bash
configure
set service dns forwarding options "server=/example.com/192.168.16.61#54"
set service dns forwarding options "all-servers"
commit
save
```

**Important:** The `all-servers` option is required. It queries all DNS servers in parallel. When CoreDNS doesn't know a hostname (e.g., `www.example.com` pointing to GitHub Pages), it stays silent, and EdgeRouter uses the upstream response.

Without `all-servers`, EdgeRouter would wait for CoreDNS to respond before trying upstream DNS, causing timeouts for external hostnames.

### Multiple Cluster Nodes

When running a cluster, add each node:

```bash
set service dns forwarding options "server=/example.com/192.168.16.61#54"
set service dns forwarding options "server=/example.com/192.168.16.62#54"
set service dns forwarding options "all-servers"
```

## PiHole / dnsmasq Setup

Joyride runs on port 54 to avoid conflicts with local systemd-resolved. It does not forward DNS requests to another server - instead, configure your main DNS server to forward specific domain queries to it.

```bash
echo -e "server=/example.com/192.168.1.2#54\nall-servers\nno-negcache" | sudo tee -a /etc/dnsmasq.d/03-custom-dns-names.conf
```

Replace:
- `example.com` - your domain
- `192.168.1.2` - IP address of the server running Joyride

Then restart dnsmasq:
```bash
sudo systemctl restart dnsmasq
# Or for PiHole:
pihole restartdns
```

### dnsmasq Options Explained

- **all-servers** - Query all available DNS servers simultaneously and use the first response. Without this, dnsmasq waits for Joyride to respond (or timeout) before trying upstream DNS, causing delays for external hostnames.

- **no-negcache** - Disable caching of negative results (NXDOMAIN). This prevents dnsmasq from caching "not found" responses, which is important when Joyride containers start/stop dynamically.

See the [dnsmasq documentation](https://thekelleys.org.uk/dnsmasq/docs/dnsmasq-man.html) for all available options.

## Testing

```bash
# Query a container hostname
dig @192.168.16.61 -p 54 myapp.example.com

# Run integration tests
docker compose -f docker-compose.test.yml up --abort-on-container-exit
```

## Health Check

```bash
curl http://192.168.16.61:5454/health
```

Health check is on port 5454.

## Version Endpoint

Query build version information:

```bash
curl http://192.168.16.61:8081/version
```

Returns:
```json
{
  "version": "2.4.0",
  "git_commit": "abc1234...",
  "build_time": "2026-01-04T12:00:00Z"
}
```

Override the port with `COREDNS_VERSION_PORT` environment variable (default: `:8081`).

## Logs

View registered hostnames and configuration:

```bash
docker logs coredns-docker-cluster
```

Example output:
```
docker-cluster: host_ip=192.168.16.61 labels=[coredns.host.name joyride.host.name] ttl=60 unknown_action=drop
docker-cluster: docker_socket=unix:///var/run/docker.sock
docker-cluster: connected to Docker daemon
docker-cluster: synced 3 containers with DNS records
docker-cluster: registered hostnames: [app.example.com api.example.com web.example.com]
```

## Forwarding Unknown Queries

If you want CoreDNS to forward unknown queries to upstream DNS (instead of dropping them), use the native `forward` plugin:

```
.:54 {
    docker-cluster {
        host_ip {$HOSTIP}
        fallthrough
    }
    forward . 8.8.8.8 1.1.1.1
    errors
    health :8080
}
```

See the commented example in `Corefile` for details.
