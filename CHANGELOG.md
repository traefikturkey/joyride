# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Version endpoint at `:8081/version` with build info (configurable via `COREDNS_VERSION_PORT`)
- Release-Please automation for semantic versioning
- OCI image labels for container metadata

### Changed
- Pinned all GitHub Actions to SHA for supply chain security

### Fixed
- HTTP server graceful shutdown (prevents goroutine leak)
- HTTP server timeouts for DoS prevention

---

## [2.3.0] - 2026-01-04

### Added
- `traefik-externals` plugin for Traefik external service DNS resolution
- Multi-stage Docker build with separate dev and production targets

### Changed
- Reorganized plugins into `plugins/` directory structure

### Fixed
- Deadlock in ClusterManager.Stop()
- Docker build dependency ordering (move install after plugin copy)

---

## [2.2.0] - 2026-01-03

### Added
- `make audit` target for vulnerability scanning
- SECURITY.md security policy

### Fixed
- Updated dependencies to address 9 Dependabot security alerts
- Updated indirect dependencies for remaining CVEs

### Dependencies
- Bump github.com/coredns/coredns from 1.12.0 to 1.12.4

---

## [2.1.0] - 2026-01-03

### Added
- Static DNS entries via CoreDNS hosts plugin
- PiHole/dnsmasq setup instructions in README
- Unit tests for environment variable overrides
- Integration tests for static hosts file feature
- Unit tests in GitHub Actions workflow

### Changed
- Renamed `HOST_IP` environment variable to `HOSTIP`

### Fixed
- Move hosts plugin before docker-cluster in plugin chain
- CI: Pin Docker client version in test workflow
- CI: Discord notification configuration

---

## [2.0.0] - 2025-12-30

### Added
- **Complete rewrite using CoreDNS** - Go-based plugin architecture replacing Ruby/dnsmasq
- `docker-cluster` plugin for dynamic Docker container DNS resolution
- GitHub Actions CI/CD pipeline for Docker image publishing
- Prometheus metrics endpoint (`:9153`)
- Health check endpoint (`:5454/health`)
- Cluster mode with **memberlist** gossip protocol for multi-node deployments
- Automatic NODE_NAME generation from hostname when clustering enabled
- Backward compatibility for legacy `joyride.host.name` labels
- Support for `coredns.host.name` container labels

### Changed
- Health check port changed from 8080 to 5454
- Split DNS behavior: unknown queries get no response (timeout) instead of NXDOMAIN/REFUSED

### Fixed
- Health plugin lameduck URL by using explicit bind address

### Notes
- Drop-in replacement for Joyride v1.x on port 54
- Designed for EdgeRouter split DNS setups

---

## Legacy Versions (ruby branch)

### [1.5.0] - 2025-07-31 [serf branch]
- Serf-based clustering for multi-node DNS synchronization
- Option to enable/disable Serf (off by default)
- Podman support for development

### [1.4.0] - 2026-01-03 [ruby branch]
- Auto-detect HOSTIP in host network mode
- AAAA record filtering

### [1.3.0] - 2025-06-01 [ruby branch]
- NXDOMAIN as configurable option
- dnsmasq returns NXDOMAIN for unknowns instead of REFUSED

### [1.2.0] - 2024-04-24 [ruby branch]
- Static hosts file support via `/etc/hosts.d` volume mount
- Discord notifications for image publishing

### [1.1.0] - 2024-02-11 [ruby branch]
- Allow `/etc/hosts.d` to be bind mounted
- GitHub Actions workflow for Docker image publishing

### [1.0.0] - 2021-09-04 [ruby branch]
- Initial Ruby/dnsmasq-based Docker DNS resolution
- Support for `joyride.host.name` container labels
- Port 54 for EdgeRouter integration

---

## Branch Strategy

| Branch | Version | Description |
|--------|---------|-------------|
| `main` | 2.x | CoreDNS/Go rewrite (current) |
| `serf` | 1.5.x | Ruby + Serf clustering (legacy) |
| `ruby` | 1.0-1.4.x | Original Ruby/dnsmasq (legacy) |
