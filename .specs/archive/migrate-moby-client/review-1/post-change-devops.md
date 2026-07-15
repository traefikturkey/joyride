---
reviewer: post-change:devops
status: complete
finding_count: 3
---

# Findings

## Finding 1
- Category: substantive defect
- Severity: high
- Confidence: high
- Evidence: Severity rationale: V2 cannot pass in the stated execution order, so it cannot prove the uniquely tagged image requirement. V2 runs `VALIDATION_TAG=...; make build IMAGE_TAG="$VALIDATION_TAG"` before T4. The current `Makefile` `build` recipe hard-codes `-t coredns-docker-cluster` and ignores `IMAGE_TAG`; `make -n build IMAGE_TAG=probe` confirms that it still emits `docker build --target production -t coredns-docker-cluster .`. T4 is the task that adds `IMAGE_TAG`, but it is blocked by V2. Therefore `docker image inspect "$VALIDATION_TAG"` in V2 fails after overwriting the default tag.
- Required fix: Move the `IMAGE_TAG ?= coredns-docker-cluster` Makefile change into T3 (or a prerequisite task before V2), update T3's mutation boundary accordingly, and retain T4 only for integration hardening. Re-run `make -n build IMAGE_TAG=probe` as a V2 precondition and require its output to contain `-t probe`.

## Finding 2
- Category: substantive defect
- Severity: medium
- Confidence: high
- Evidence: Severity rationale: V3 can report matching container, network, and volume inventories while leaving a new integration-build image behind. `make test-integration` runs `docker compose -f docker-compose.test.yml up --build`; the Compose file has a `coredns` service with `build: .` and no `image:` override. `docker compose ... down -v --remove-orphans` removes containers, networks, and volumes but does not remove the image built for that service. V3 inventories only containers, networks, and volumes, then removes only the separately recorded V2 validation tag. The integration-built image is neither inventoried nor removed, despite the plan's local-Docker-resource cleanup claim.
- Required fix: Give the test Compose build a run-specific, recorded image tag and remove that exact tag during V3 cleanup, or add an image before/after inventory and explicitly remove only the image created by the test project. Do not use broad image pruning.

## Finding 3
- Category: process defect
- Severity: medium
- Confidence: high
- Evidence: Severity rationale: the V3 failure-path cleanup and exit-status contract is not executable as documented and can be bypassed by shell error handling. V3 says to capture `make test-integration` status, always run Compose `down`, and preserve the primary or cleanup failure status, but provides no shell wrapper or command. A validator using an errexit shell will stop at a failing `make test-integration` before the direct cleanup command; a validator without one can accidentally return the final inventory command status instead of the failed test status. T4 states the desired Make truth table but likewise supplies no concrete recipe to inspect.
- Required fix: Specify one POSIX/Git-Bash-compatible recipe for T4 and one V3 invocation wrapper that uses `set +e`, captures the primary status, always executes `docker compose -f docker-compose.test.yml down -v --remove-orphans`, captures cleanup status, and exits primary status when nonzero otherwise cleanup status. Require V3 to record both statuses and fail if either required condition is nonzero.
