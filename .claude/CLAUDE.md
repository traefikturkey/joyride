# CoreDNS Docker DNS Project

## Project Goal

Replace Joyride (dnsmasq-based Docker DNS) with CoreDNS for dynamic DNS resolution of Docker containers. The key improvement is proper split DNS behavior for EdgeRouter integration.

## Background Context

### Current Setup (Joyride)
- Joyride runs dnsmasq on port 54
- Watches Docker containers via `/var/run/docker.sock`
- Containers with `joyride.host.name=hostname.example.com` labels get DNS entries
- EdgeRouter forwards domain-specific queries to Joyride:
  ```
  server=/ilude.com/192.168.16.61#54
  ```

### The Problem
- When Joyride doesn't know a hostname (e.g., `www.ilude.com` which points to GitHub Pages), dnsmasq returns **REFUSED** or **NXDOMAIN**
- EdgeRouter's dnsmasq accepts REFUSED/NXDOMAIN as final answers and doesn't fall back to public DNS
- Result: External domains fail intermittently

### The Solution
CoreDNS with split DNS behavior - simply don't respond for unknown queries:
```
docker-cluster {
    host_ip 192.168.16.61
    label coredns.host.name
    # NO fallthrough - unknown queries get no response
}
# NO forward plugin - let EdgeRouter handle upstream
```
When CoreDNS doesn't respond, EdgeRouter's dnsmasq times out and queries its upstream DNS servers (public DNS), which correctly resolves external domains like `www.ilude.com` to GitHub Pages.

## Architecture

**Custom CoreDNS Plugin (docker-cluster)**

Single binary with built-in Docker watching:
```
┌─────────────────────────────────────┐
│           CoreDNS                   │
│  ┌─────────────────────────────┐    │
│  │   docker-cluster plugin     │    │
│  │   - Watches Docker socket   │    │
│  │   - Serves A records        │    │
│  │   - No response if unknown  │    │
│  └─────────────────────────────┘    │
└─────────────────────────────────────┘
```

## Key Files

- `plugin/` - CoreDNS plugin source code
  - `records.go` - Thread-safe DNS record storage (atomic.Value)
  - `docker_watcher.go` - Docker event monitoring
  - `docker_cluster.go` - DNS handler (ServeDNS)
  - `setup.go` - Plugin registration and config parsing
- `Dockerfile` - Multi-stage build (auto-fetches latest CoreDNS)
- `Corefile` - CoreDNS configuration
- `docker-compose.yml` - Production deployment
- `docker-compose.test.yml` - Integration tests

## Corefile Configuration

```
.:54 {
    docker-cluster {
        docker_socket unix:///var/run/docker.sock
        host_ip 192.168.16.61
        label coredns.host.name
        label joyride.host.name
        ttl 60
        # NO fallthrough - unknown queries get no response
    }
    errors
    health :8080
    prometheus :9153
}
```

## Environment Details

- Target host: 192.168.16.61 (where Joyride currently runs)
- DNS port: 54 (non-standard, for EdgeRouter integration)
- EdgeRouter: 192.168.16.1 (Ubiquiti EdgeRouter-X)
- Domains: ilude.com, traefikturkey.icu, projectbantam.com

## Label Format

Containers register DNS with labels:
```yaml
labels:
  - "coredns.host.name=myapp.ilude.com"
  # Or for backward compatibility:
  - "joyride.host.name=myapp.ilude.com"
  # Multiple hostnames:
  - "coredns.host.name=app.ilude.com,api.ilude.com"
```

## Success Criteria

1. Container DNS works (same as Joyride)
2. Unknown queries get no response (timeout)
3. EdgeRouter falls back to upstream for unknown queries
4. `www.ilude.com` resolves to GitHub Pages via EdgeRouter's upstream
5. No EdgeRouter config changes needed
6. Drop-in replacement on port 54
