# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.4.1](https://github.com/traefikturkey/joyride/compare/v2.4.0...v2.4.1) (2026-01-04)


### Bug Fixes

* **ci:** Correct discord-webhook-notify SHA ([6fb6363](https://github.com/traefikturkey/joyride/commit/6fb6363581e25b32313b566a916a8011c32e75c0))

## [2.4.0](https://github.com/traefikturkey/joyride/compare/v2.3.0...v2.4.0) (2026-01-04)


### Features

* Add Release-Please automation and project documentation ([e6573db](https://github.com/traefikturkey/joyride/commit/e6573dbadaceb72291a20a81d51b538f3515a674))
* Add static DNS entries via CoreDNS hosts plugin ([0fba687](https://github.com/traefikturkey/joyride/commit/0fba687e7f4a0806b8bd3ba90700075d9ed5f9c7))
* Add version endpoint with build info ([f74ca7d](https://github.com/traefikturkey/joyride/commit/f74ca7d4e060df3a78ebba130e65ad032bd85ef2))
* **make:** Add audit target for vulnerability scanning ([75a5b60](https://github.com/traefikturkey/joyride/commit/75a5b60cb14e90c137571ce028333b1508ed02ac))
* **plugin:** Add traefik-externals plugin for Traefik external service DNS ([f6b5048](https://github.com/traefikturkey/joyride/commit/f6b50483bd2e25aa10699d333feca9307a6efbc2))


### Bug Fixes

* Address crash risks and improve error handling ([680d4b5](https://github.com/traefikturkey/joyride/commit/680d4b51705f1a19b6cc11dbdd6642fe18681210))
* **ci:** Check secret instead of variable for Discord notification ([f12b57a](https://github.com/traefikturkey/joyride/commit/f12b57aaeb40c499f8be096328bbb7e3545be3c5))
* **ci:** Correct release-please-action SHA ([2441809](https://github.com/traefikturkey/joyride/commit/2441809fba61b94730119cc137201af01195c086))
* **ci:** Exclude cluster tests requiring UDP networking ([0f2fbff](https://github.com/traefikturkey/joyride/commit/0f2fbffd83fe6a9e2c4f1f753dd4803f9fede70d))
* **ci:** Pin Docker client version in test workflow ([8d857a8](https://github.com/traefikturkey/joyride/commit/8d857a81be33ffafd4ec6cdb0ddf1060d193f416))
* **ci:** Remove condition from Discord notification ([31b9616](https://github.com/traefikturkey/joyride/commit/31b96168db8d9473d1dd8467c1ac7e44631ee591))
* **ci:** Use continue-on-error for Discord notification ([76d4813](https://github.com/traefikturkey/joyride/commit/76d48134d02e8e2f4ee8d31fcd0cdaaafc811eec))
* **ci:** Use manifest file for release-please versioning ([544d7d4](https://github.com/traefikturkey/joyride/commit/544d7d4499352c7c09fd779ef8876eb505ad0280))
* **cluster:** Prevent deadlock in ClusterManager.Stop() ([2595a69](https://github.com/traefikturkey/joyride/commit/2595a695f88a758601b60e902259d95b488e854c))
* **deps:** Update indirect dependencies for remaining CVEs ([b83a25c](https://github.com/traefikturkey/joyride/commit/b83a25cc58ec0734e8e0043c403515d5793d4d41))
* **docker:** Move dependency install after plugin copy ([73254d1](https://github.com/traefikturkey/joyride/commit/73254d1e0235267430b4e664c78e3b9ca2e1f9a3))
* Move hosts plugin before docker-cluster in plugin chain ([dd82903](https://github.com/traefikturkey/joyride/commit/dd829035e9a242b3833f06016c33fa492ce804d2))
* **security:** Update dependencies to address 9 Dependabot alerts ([9b2ff79](https://github.com/traefikturkey/joyride/commit/9b2ff79348f5505f42542ca833cff6dd51e0f599))

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
