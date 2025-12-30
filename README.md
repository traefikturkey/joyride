# CoreDNS Docker DNS

A CoreDNS plugin that provides DNS resolution for Docker containers. Drop-in replacement for Joyride (dnsmasq-based Docker DNS).

## Quick Start

```bash
# Set your host IP
export HOST_IP=192.168.16.61

# Run with Docker Compose
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

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `HOST_IP` | IP address to return for all container DNS records | Required |
| `DOCKER_SOCKET` | Docker socket path | `unix:///var/run/docker.sock` |
| `DNS_UNKNOWN_ACTION` | What to do for unknown hostnames: `drop` or `nxdomain` | `drop` |

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
# HOST_IP is auto-detected when using host networking
NODE_NAME=node1 docker compose -f docker-compose.host.yml up -d
```

Or manually:

```bash
# Node 1 - HOST_IP auto-detected from default route
docker run -d --network host \
  -e CLUSTER_ENABLED=true \
  -e NODE_NAME=node1 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  coredns-docker-cluster

# Node 2 (automatically finds Node 1 via broadcast)
docker run -d --network host \
  -e CLUSTER_ENABLED=true \
  -e NODE_NAME=node2 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  coredns-docker-cluster
```

**Note:** Host networking enables:
- Automatic HOST_IP detection from the default network interface
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

## Testing

```bash
# Query a container hostname
dig @192.168.16.61 -p 54 myapp.example.com

# Run integration tests
docker compose -f docker-compose.test.yml up --abort-on-container-exit
```

## Health Check

```bash
curl http://192.168.16.61:8080/health
```

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
        host_ip {$HOST_IP}
        fallthrough
    }
    forward . 8.8.8.8 1.1.1.1
    errors
    health :8080
}
```

See the commented example in `Corefile` for details.
