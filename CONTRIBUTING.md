# Contributing to Joyride

Thank you for your interest in contributing!

## Commit Message Format

This project uses [Conventional Commits](https://www.conventionalcommits.org/) to automate versioning and changelog generation.

### Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

| Type | Description | Version Bump |
|------|-------------|--------------|
| `feat` | New feature | Minor (0.1.0) |
| `fix` | Bug fix | Patch (0.0.1) |
| `docs` | Documentation only | None |
| `style` | Code style (formatting, etc.) | None |
| `refactor` | Code refactoring | None |
| `perf` | Performance improvement | Patch |
| `test` | Adding/updating tests | None |
| `chore` | Maintenance tasks | None |
| `ci` | CI/CD configuration | None |
| `build` | Build system changes | None |

### Breaking Changes

For breaking changes, add `!` after the type or include `BREAKING CHANGE:` in the footer:

```
feat!: redesign configuration format

BREAKING CHANGE: Corefile syntax has changed. See migration guide.
```

Breaking changes trigger a **Major** version bump (1.0.0 → 2.0.0).

### Examples

```bash
# Feature
git commit -m "feat(docker-cluster): add support for multiple hostname labels"

# Bug fix
git commit -m "fix(watcher): handle container restart events correctly"

# Documentation
git commit -m "docs: update Corefile configuration examples"

# Breaking change
git commit -m "feat!: require explicit docker_socket in Corefile"
```

## Development

### Prerequisites
- Go 1.24+
- Docker
- Make

### Running Tests
```bash
make test-unit      # Run unit tests
make test-race      # Run tests with race detector
make test-integration  # Run integration tests
```

### Building
```bash
make build          # Build production image
make build-dev      # Build development image
```
