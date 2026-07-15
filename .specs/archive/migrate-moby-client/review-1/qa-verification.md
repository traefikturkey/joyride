# QA Verification Review

## Finding 1
- Category: substantive defect
- Severity: high
- Confidence: high
- Evidence: Severity rationale: the highest-risk compatibility boundary is not behaviorally tested, so a compile-only migration can ship broken Docker discovery or event handling. `plugins/docker-cluster/docker_watcher_test.go` only tests `extractHostnames`; no test constructs or exercises `ContainerList`, `ContainerInspect`, `Events`, event message delivery, or the event error channel. T2 accepts `make test-unit`, but that command runs the existing plugin tests, which do not cover `DockerWatcher` lifecycle or any Moby result wrapper. The plan explicitly identifies event error channels, list/inspect results, and event streaming as migration risks, yet V2 only repeats the same suite and the Compose workflow checks DNS queries without asserting the watcher API path.
- Required fix: Add deterministic watcher-focused tests or a named integration assertion that proves initial container listing, start/stop event processing, inspect label extraction, and a non-nil event-stream error causes reconnect/cleanup rather than silent success. Make these tests part of the T2/V2 acceptance commands and fail on closed or errored event channels as appropriate.

## Finding 2
- Category: false positive
- Severity: high
- Confidence: high
- Evidence: Severity rationale: the version-alignment acceptance can pass while the requested exact versions are wrong or incomplete. The command `rg -n 'github.com/moby/moby/(client|api)@|github.com/moby/moby/(client|api) v' go.mod Dockerfile Makefile .github/workflows/docker-publish.yml` only proves that matching text exists somewhere. It does not require `client v0.5.0` and `api v1.55.0`, does not require both modules in each of the three Makefile blocks, and does not reject a block containing only one module. The pass statement requires all of those conditions, but the command truth table does not encode them.
- Required fix: Replace the presence-only grep with exact assertions for the two requested versions and expected block count, for example extract all dependency-install blocks and compare each to the exact pair, then separately verify exact `go.mod` requirements. Include a failing fixture or shell check showing that a wrong version and a missing API line return nonzero.

## Finding 3
- Category: substantive defect
- Severity: medium
- Confidence: high
- Evidence: Severity rationale: the documented validation does not prove the root module metadata is usable or that the changed `go.mod` is consistent with the implementation. Every build and test entry point clones CoreDNS into `/tmp/coredns`, copies plugin sources there, and runs `go get` against the cloned module; the repository root `go.mod` is never used by `make test-unit`, `make test-race`, `make build`, or `make test-integration`. Plain `go test ./...` is explicitly disallowed, and no root-module `go mod verify`, `go list`, or equivalent check is specified. Thus a malformed or incomplete root dependency migration can pass all V2 runtime checks while the plan still claims the repository compiles against the stable modules.
- Required fix: Add a deterministic root-module validation that actually loads the affected package/dependency graph, or narrow the objective and acceptance wording to the CoreDNS-clone build path. If the root module is intended to be authoritative, make the validation run against it and assert the exact Moby packages resolve at the pinned versions rather than only validating the separately cloned CoreDNS module.
