# Joyride

Joyride is a standalone DNS service for your network, packaged as a ready-to-run Docker container. It watches Docker and automatically adds or removes DNS records as labeled containers start or stop. It runs on CoreDNS and can also serve static host entries.

By default, Joyride answers for the names it knows on port 54 and leaves unknown queries unanswered. Run it alongside your network's main DNS resolver: forward a dedicated subdomain for your services to Joyride and leave other queries with your existing resolver. Set `DNS_UNKNOWN_ACTION=nxdomain` if you want an immediate negative response for unknown names instead of a timeout.

## Quick Start

### 1. Run Joyride

Replace `192.168.16.61` with the LAN address clients should use to reach your apps, typically your Docker host or reverse proxy. Joyride returns this address in the IPv4 (A) records it creates from labels.

```bash
docker run -d --name joyride --restart unless-stopped \
  -p 54:54/udp -p 54:54/tcp -p 5454:5454 \
  -e HOSTIP=192.168.16.61 \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  ghcr.io/traefikturkey/joyride:latest
```

### 2. Label an app container

Add a DNS label to a service and start or recreate its container on the same Docker host as Joyride. Replace `apps.example.com` with the dedicated subdomain you want Joyride to serve.

```yaml
services:
  myapp:
    image: nginx
    labels:
      - "coredns.host.name=myapp.apps.example.com"
```

### 3. Verify the record

Query the LAN address of the machine running Joyride on port 54. The answer should be the configured `HOSTIP`, which may be a different address if you use a separate reverse proxy.

```bash
dig @192.168.16.61 -p 54 myapp.apps.example.com A +short
```

### 4. Connect your network resolver

Configure your router or DNS resolver to forward `apps.example.com` to Joyride on TCP and UDP port 54. See [Network DNS Setup](#network-dns-setup) for examples. Then query the same hostname through your normal resolver to verify that forwarding works.

## Container Labels

Use `coredns.host.name` to give a container one or more DNS names.

Multiple hostnames:
```yaml
labels:
  - "coredns.host.name=app.example.com,api.example.com"
```

Also accepted:
```yaml
labels:
  - "joyride.host.name=myapp.example.com"
```

## Network DNS Setup

Replace `apps.example.com` with your dedicated subdomain and `192.168.16.61` with the address of the machine running Joyride. The `server=` rules below send queries for that subdomain only to the configured Joyride server or cluster nodes. With these rules alone, names that Joyride does not know will not fall back to public DNS.

### EdgeRouter

```bash
configure
set service dns forwarding options "server=/apps.example.com/192.168.16.61#54"
commit
save
```

For a cluster, add one `server=/apps.example.com/<node-ip>#54` rule per Joyride node.

### Pi-hole or dnsmasq

Add a domain-specific rule to your resolver's configuration:

```ini
server=/apps.example.com/192.168.16.61#54
```

For standalone dnsmasq, add it in a configured directory such as `/etc/dnsmasq.d/` and restart the service. On Pi-hole v6, add it to `misc.dnsmasq_lines` under **Settings > All settings**. See the [dnsmasq manual](https://thekelleys.org.uk/dnsmasq/docs/dnsmasq-man.html) for forwarding options.

## Static DNS Entries

For hosts that aren't Docker containers (NAS, printers, etc.), add entries to a file named `hosts`. For `docker run`, add `-v /path/to/hosts.d:/etc/hosts.d:ro` and put the file in that directory. The repository's Docker Compose setup already mounts `./etc/joyride/hosts.d`.

Use standard hosts file format:
```
192.168.1.10 nas.example.com nas
192.168.1.20 printer.example.com
```

Static entries take priority over Docker container labels. Changes are picked up automatically (no restart required).

## Traefik External Services

The `traefik-externals` plugin can create DNS records for services proxied through Traefik, including services outside Docker (VMs, bare metal, or external APIs). With the shipped `Corefile`, the Docker plugin stops unknown queries before they reach this plugin. To use Traefik records, mount your YAML directory and use a custom `Corefile` with `fallthrough` in `docker-cluster`. Without a `forward` directive, Joyride then returns `SERVFAIL` for unknown names instead of staying silent. See [Forwarding Unknown Queries](#forwarding-unknown-queries) to send those queries to an upstream resolver.

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

Mount your Traefik external configs directory in the `coredns` service in the repository's Compose file:

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
| `HOSTIP` | IPv4 address returned for names from Docker labels and Traefik configs | Set explicitly on a bridge network; can be auto-detected with host networking |
| `DOCKER_SOCKET` | Docker socket path | `unix:///var/run/docker.sock` |
| `DNS_UNKNOWN_ACTION` | What to do for unknown hostnames: `drop` or `nxdomain` | `drop` |

### Deprecated Environment Variables

| Variable | Use Instead | Notes |
|----------|-------------|-------|
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

Multiple Joyride nodes can share DNS records using the SWIM gossip protocol. Each node watches its local Docker daemon and replicates records to peers.

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
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  ghcr.io/traefikturkey/joyride:latest

# Node 2 (automatically finds Node 1 via broadcast)
docker run -d --network host \
  -e CLUSTER_ENABLED=true \
  -e NODE_NAME=node2 \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  ghcr.io/traefikturkey/joyride:latest
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

### Inter-VLAN Clustering

When clustering nodes across different VLANs or subnets, UDP broadcast discovery will not work because broadcasts don't cross network boundaries. You must use static `CLUSTER_SEEDS` to connect nodes.

```
VLAN 10 (10.0.10.x)            VLAN 20 (10.0.20.x)            VLAN 30 (10.0.30.x)
┌─────────────────┐            ┌─────────────────┐            ┌─────────────────┐
│ node-web-1      │────────────│ node-seed       │────────────│ node-db-1       │
│ node-web-2      │            │ (seed node)     │            │ node-db-2       │
└─────────────────┘            └─────────────────┘            └─────────────────┘
```

**Seed Node** (10.0.20.10): starts first, no `CLUSTER_SEEDS`:

```yaml
services:
  joyride:
    image: ghcr.io/traefikturkey/joyride:latest
    network_mode: host
    environment:
      - HOSTIP=10.0.20.10
      - CLUSTER_ENABLED=true
      - NODE_NAME=node-seed
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

**Member Nodes**: point to the seed node:

```yaml
services:
  joyride:
    image: ghcr.io/traefikturkey/joyride:latest
    network_mode: host
    environment:
      - HOSTIP=10.0.10.11
      - CLUSTER_ENABLED=true
      - NODE_NAME=node-web-1
      - CLUSTER_SEEDS=10.0.20.10:7946
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

#### Startup Order

**Single seed:** The seed node must be running before member nodes start. Members will log `connection refused` errors until the seed is available.

**Multiple seeds (recommended):** For resilience, configure 2-3 seed nodes. Any running seed allows new nodes to join, eliminating strict startup ordering:

```yaml
environment:
  - CLUSTER_SEEDS=10.0.20.10:7946,10.0.10.11:7946
```

#### Firewall Requirements

Ensure these ports are open between all cluster nodes:

| Port | Protocol | Purpose |
|------|----------|---------|  
| 7946 | TCP/UDP | Memberlist gossip |
| 54 | TCP/UDP | DNS (default Joyride port) |

**Note:** If you have configured Joyride to listen on port 53 (standard DNS port), open port 53 instead of 54.

#### Encryption

For traffic crossing untrusted network segments, enable cluster encryption by setting `CLUSTER_SECRET` to the same value on all nodes:

```yaml
environment:
  - CLUSTER_SECRET=your-shared-secret-key
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

### Cluster Troubleshooting

| Log Message | Cause | Solution |
|-------------|-------|----------|
| `failed to join <ip>:7946: connection refused` | Seed node not running or firewall blocking port 7946 | Ensure seed node is started first; verify port 7946 is open (TCP/UDP) between nodes |
| `Refuting a dead message` | Normal during rejoin; another node had marked this node as dead | No action needed; the node is announcing it's back online |
| `failed to join` on startup (multiple seeds) | Some seed nodes unavailable | Expected if not all seeds are running; node will use available seeds |
| Nodes not seeing each other's DNS records | Clustering not enabled on all nodes | Verify `CLUSTER_ENABLED=true` is set on every node |
| `memberlist: Initiating push/pull sync` | Normal protocol operation | Informational; nodes are synchronizing state |

## Testing

To build and run Joyride from a local checkout, set `HOSTIP` and run `docker compose up -d --build`.

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

With bridge networking, publish the version port with `-p 8081:8081` when starting the container. With host networking, no port mapping is needed.

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
docker logs joyride  # Use your container name if different
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

To have Joyride forward queries it cannot answer instead of dropping them, copy the shipped [Corefile](Corefile) and make these changes inside the `.:54` server block:

1. Add `fallthrough` inside `docker-cluster` so misses reach the next plugin instead of using `DNS_UNKNOWN_ACTION`.
2. Keep `fallthrough` inside `traefik-externals` if you use it.
3. Add a `forward` directive, for example `forward . <upstream-resolver-ip>`. Replace the placeholder with the IP address of a resolver that does not send these queries back to Joyride. Do not point it at the network resolver configured above to forward the subdomain to Joyride.

Mount your custom file at `/etc/coredns/Corefile` and recreate the container. With `docker run`, add `-v /path/to/Corefile:/etc/coredns/Corefile:ro`; the repository's Compose file already mounts `./Corefile`.
