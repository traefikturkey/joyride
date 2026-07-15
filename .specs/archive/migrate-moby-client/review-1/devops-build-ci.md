# DevOps Build CI Review

## Finding 1
- Category: substantive defect
- Severity: high
- Confidence: high
- Evidence: Severity rationale: the planned validation can pass against a materially different dependency graph than the image build, so it does not prove the failed CI build path is fixed. The Dockerfile installs `github.com/hashicorp/memberlist@v0.5.1` and `github.com/fsnotify/fsnotify@v1.7.0` (Dockerfile:50-52), while the root module requires memberlist v0.5.4 and fsnotify v1.10.1 (go.mod:8-15), and CI installs fsnotify v1.10.0 (docker-publish.yml:56-59). T3 only requires alignment for the new Moby modules and explicitly preserves unrelated versions, leaving these existing setup drifts unresolved. `make test-unit` and `make test-race` therefore do not validate the Dockerfile's dependency setup.
- Required fix: Either make Dockerfile, CI, and all Make setup blocks use the same memberlist and fsnotify versions as go.mod, or explicitly validate each entry point independently, including a production Docker build. Add the non-Moby dependency parity to T3's acceptance criteria.

## Finding 2
- Category: substantive defect
- Severity: high
- Confidence: high
- Evidence: Severity rationale: local test success can conceal CoreDNS compatibility failures in the production image. Dockerfile and CI pin CoreDNS v1.14.3 (Dockerfile:20-22; docker-publish.yml:25-27), but both Make unit and race tests query the latest CoreDNS release at runtime (Makefile:28-31 and 45-48). The plan's `make test-unit`, `make test-race`, and integration checks consequently do not exercise the pinned source used by the image build.
- Required fix: Pin Make's unit, race, and audit setup to the same v1.14.3 value, or add an explicit test command that clones and tests exactly v1.14.3 before treating V2 as passed. Keep network-resolved latest-version checks separate from migration acceptance.

## Finding 3
- Category: substantive defect
- Severity: high
- Confidence: high
- Evidence: Severity rationale: the documented cleanup guarantee is false on the failure path and can leave containers, networks, and volumes behind. `test-integration` runs `docker compose ... up --build --abort-on-container-exit` followed by a separate `down -v` recipe line (Makefile:62-64). GNU Make does not execute the second line when the first command exits nonzero, which is the expected result for a failing integration test or build. V2 invokes this target and the plan says cleanup must occur after success or failure, but no trap or unconditional cleanup wrapper is specified.
- Required fix: Make `test-integration` use an unconditional cleanup trap around `docker compose up`, or have V2 run cleanup in a failure-safe shell step and verify it. Apply the same guarantee to any validation path that can fail after creating Compose resources.

## Finding 4
- Category: substantive defect
- Severity: medium
- Confidence: high
- Evidence: Severity rationale: the toolchain matrix is inconsistent and can produce false local confidence or an earlier/later failure than CI and Docker. `go.mod` declares Go 1.25.0 (go.mod:3), CI uses Go 1.25 (docker-publish.yml:39-43), the Dockerfile builds with golang:1.26 (Dockerfile:17), and Make unit/race/audit containers use golang:1.24-alpine (Makefile:27, 44, and 113). The plan records these values but only states they fit; it does not require a compatibility check or explain why Go 1.24 is supported for the migrated module set.
- Required fix: Standardize the test/build toolchain where feasible, preferably on the module's declared Go version, or document and verify the intentionally supported range by running the exact Make, CI-equivalent, and Docker paths. Do not treat one successful toolchain as evidence for the others.
