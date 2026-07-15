---
created: 2026-07-15
status: completed
completed: 2026-07-15
---

# Plan: Migrate Joyride to stable Moby client modules

## Context & Motivation

GitHub issue #18 tracks removal of the frozen `github.com/docker/docker v28.5.2+incompatible` module. Current source, module metadata, Dockerfile, CI, and Make targets still use it. Current main CI tests pass, but the production image build fails in the old Docker dependency step.

The issue's original `github.com/moby/moby/v2` target remains beta-only. Moby now publishes stable split modules: `github.com/moby/moby/client v0.5.0` and `github.com/moby/moby/api v1.55.0`. Their stable API changes Ping options, filters, list/inspect result wrappers, and event-stream results. This plan migrates those boundaries on a dedicated branch and verifies the exact CoreDNS/Docker workflow.

## Constraints

- Platform: Windows 11, Git Bash, Docker Desktop; Docker Engine 29.6.1 detected.
- Create all implementation changes on local branch `chore/migrate-moby-client`; do not use a worktree.
- Do not commit, push, open a PR, deploy, edit issue #18, or close it.
- Preserve CoreDNS v1.14.3 for production, CI, and migration acceptance.
- Use stable Moby split modules, not beta `/v2`.
- Preserve watcher behavior and locking; add only a narrow event-channel seam needed for deterministic tests.
- Do not align unrelated dependency versions in this migration. Validate each existing build graph independently.
- Plain root `go test ./...` is invalid because plugins compile after copying into CoreDNS's module path.
- Edited files use ASCII punctuation.

## Risk & Manual Gate Decision

- **Risk level:** medium
- **Blast radius:** personal local repository and all local Docker resources
- **Rollback:** known and scoped
- **Manual approval before action:** not required
- **Manual validation after action:** not required
- **Decision reason:** Work is local and reversible, with no push or deployment. The mounted Docker socket is daemon-admin capability even when mounted read-only; the plan therefore rejects test-name collisions, uses a unique Compose project and image, verifies project resources are removed, and blocks integration if the migrated source adds Docker mutation calls. These controls make automated validation appropriate without a manual gate.

## Alternatives Considered

| Approach | Pros | Cons | Verdict |
|----------|------|------|---------|
| Stable client v0.5.0 and API v1.55.0 modules | Stable, current client/API surface | Requires explicit API adaptation | **Selected** |
| Beta `github.com/moby/moby/v2` | Matches original issue wording | Still beta | Rejected |
| Keep Docker v28 module | No changes | Frozen and current build remains broken | Rejected |
| Broad client adapter | Easier mocking | Unnecessary abstraction | Rejected; use only a narrow event-channel helper if required for tests |

## Objective

On `chore/migrate-moby-client`, Joyride uses the exact stable Moby client/API versions, has no tracked product reference to `github.com/docker/docker`, handles list/inspect/event results correctly, passes focused watcher tests and CoreDNS unit/race tests, builds a uniquely tagged production image, and passes explicit dynamic and split-DNS integration assertions with unconditional cleanup.

## MVP Boundary

Migrate the client API, module/build surfaces, and the existing integration workflow checks needed to prove the migration. Include focused event-channel regression tests and failure-safe cleanup because compile-only acceptance and skipped cleanup would not establish the objective.

## Explicit Deferrals

- Home-lab deployment and host/EdgeRouter DNS checks.
- Commit, push, PR, release, issue edit/closure, and Dependabot alert closure.
- Unrelated memberlist/fsnotify alignment or broad build-system refactoring.
- A disposable Docker daemon/VM; local daemon controls in this plan are sufficient for this personal repository.
- Broader watcher abstraction, concurrency refactor, or exhaustive Docker failure simulation.

## Project Context

- **Language:** Go module declaring Go 1.25, Dockerfile using Go 1.26, plus Make, Compose, and GitHub Actions.
- **Tests:** `make test-unit`, `make test-race`, `make test-integration`.
- **Formatting:** `gofmt`; **whitespace:** `git diff --check`.
- **Lint:** `make lint` exists but depends on an undeclared host `golangci-lint`; it is not an archive gate.

## Review Profile

- **plan_profile:** dependency-migration-with-integration-hardening
- **review_panel_decision:** review-1, post-change review, deterministic audit, and final standalone readiness completed; result `STANDALONE READY`
- **expected reviewer count:** completed with 1 standalone readiness reviewer after the panel reviews
- **selected personas:** fresh-session execution and validation reviewer
- **complexity score:** 6/10
- **risk score:** 4/10
- **expected high-risk areas:** event-channel closure, exact module selection, branch base, Docker socket scope, cleanup, and DNS false positives

## Automation Plan

| Operation | Command/wrapper | Credentials | Mutation boundary | Evidence |
|-----------|-----------------|-------------|-------------------|----------|
| Git preflight | Commands in T1 | existing Git remote access | read-only Git | `evidence/git-preflight.log` |
| Branch creation | `git switch -c chore/migrate-moby-client` | none | branch ref only | branch/status output |
| Source migration | Targeted edits and `gofmt` | none | T2 files only | scoped diff and tests |
| Dependency migration | Targeted edits and Go module commands | public module registry | T3 files only | exact version output |
| Integration hardening | Targeted Make/Compose edits | none | T4 files only | scoped diff |
| Unit/race/build validation | Commands in V2 | public GitHub/module/image registries | caches plus unique image | V2 logs |
| Docker integration | Commands in V3 | local Docker daemon | named test resources only | before/after inventories and V3 log |
| Rollback/cleanup | Compose down plus unique-tag removal | local Docker daemon | test project and unique image only | cleanup log |
| Archive | Commands in F5 | none | plan directory only | archive pre/postcondition log |

## Task Breakdown

| # | Task | Files | Type | Model | Agent | Depends On |
|---|------|-------|------|-------|-------|------------|
| T1 | Create branch from current remote main | 0 | mechanical | small | coding-light | -- |
| V1 | Validate branch isolation | -- | validation | small | validator | T1 |
| T2 | Adapt Moby API and add event regression tests | 2 | feature | medium | backend-dev | V1 |
| T3 | Replace dependency paths and pin acceptance toolchain | 5 | feature | medium | devops-pro | V1 |
| V2 | Validate API, modules, unit/race tests, and image build | -- | validation | medium | validation-lead | T2, T3 |
| T4 | Harden integration assertions and project-scoped cleanup | 2 | feature | medium | devops-pro | V2 |
| V3 | Validate Docker DNS workflow and project cleanup | -- | validation | medium | validation-lead | T4 |

## Execution Waves

### Wave 1

**T1: Create branch from current remote main** [small] -- coding-light
- Files: none.
- Mutation boundary: branch ref only; no source edit, stage, commit, push, or discard.
- Commands:
  1. `test "$(git branch --show-current)" = main`
  2. `test -z "$(git status --porcelain=v1 --untracked-files=all | grep -v '^?? .specs/migrate-moby-client/')"`
  3. `test -z "$(git rev-parse -q --verify MERGE_HEAD)" && test ! -d .git/rebase-merge && test ! -d .git/rebase-apply`
  4. `test "$(git rev-parse main)" = "$(git rev-parse origin/main)"`
  5. `test "$(git rev-parse main)" = "$(git ls-remote origin refs/heads/main | awk '{print $1}')"`
  6. `! git show-ref --verify --quiet refs/heads/chore/migrate-moby-client`
  7. `test -z "$(git ls-remote --heads origin refs/heads/chore/migrate-moby-client)"`
  8. `test ! -e .specs/archive/migrate-moby-client`
  9. `git switch -c chore/migrate-moby-client`
- Pass: all preflight checks pass and current branch becomes `chore/migrate-moby-client` at the recorded current remote-main commit.
- Fail: do not create/edit; record the failed command and preserve all work.

**V1: Validate branch isolation** [small] -- validator
- Blocked by: T1.
- Mutation boundary: checklist/status/evidence only.
- Verify: `test "$(git branch --show-current)" = chore/migrate-moby-client && test "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)"`
- Pass: exit 0; status contains only `.specs/migrate-moby-client/` before implementation.
- Fail: block Wave 2 without switching or deleting branches.

### Wave 2 (parallel)

**T2: Adapt Moby API and add event regression tests** [medium] -- backend-dev
- Blocked by: V1.
- Files: `plugins/docker-cluster/docker_watcher.go`, `plugins/docker-cluster/docker_watcher_test.go`.
- Mutation boundary: these two files only.
- Requirements:
  1. Import `github.com/moby/moby/api/types/{container,events}` and `github.com/moby/moby/client`; remove the old filters import.
  2. Remove deprecated no-op `WithAPIVersionNegotiation`; call `Ping(ctx, client.PingOptions{NegotiateAPIVersion: true})` to preserve immediate negotiation.
  3. Use initialized `client.Filters` and preserve `status=running`, `type=container`, and ORed `start`, `stop`, `die` event values.
  4. Use `ContainerListOptions`, `.Items`, `EventsListOptions`, `.Messages`, `.Err`, `ContainerInspectOptions{}`, and `.Container`.
  5. Handle context cancellation, non-nil errors/EOF, and a closed `EventsResult.Err` channel without a nil-error reconnect loop or blocking forever. A closed Messages channel is not part of the Moby v0.5 contract; any defensive handling must be labeled separately. Extract only the smallest channel-consumer helper needed for deterministic tests.
  6. Extract narrow pure helpers for applying container summaries and inspect responses; `syncContainers` must pass `ContainerListResult.Items`, and start handling must pass `ContainerInspectResult.Container`. Do not introduce a broad client adapter.
  7. Add focused tests for initial summary/list record population, inspect/start record addition, message dispatch, non-nil error and EOF return, closed Err behavior, context cancellation, and stop/die record removal. V3's exact dynamic-label assertions provide the live start/inspect integration check.
- Verify: `gofmt -w plugins/docker-cluster/docker_watcher.go plugins/docker-cluster/docker_watcher_test.go && test -z "$(gofmt -d plugins/docker-cluster/docker_watcher.go plugins/docker-cluster/docker_watcher_test.go)"`
- Pass: formatting is clean and scoped diff implements every requirement; compilation is gated by V2 after T3.
- Fail: correct only the affected API/test seam; do not add a broad adapter.
- Alternative: broad client interface/mocks.
- Rejected because: unnecessary for two direct call sites and channel behavior.

**T3: Replace dependency paths and pin acceptance toolchain** [medium] -- devops-pro
- Blocked by: V1.
- Files: `go.mod`, `go.sum`, `Dockerfile`, `.github/workflows/docker-publish.yml`, `Makefile`.
- Mutation boundary: these five files; unrelated dependency versions remain unchanged.
- Requirements:
  1. Update root module metadata with `go mod edit -droprequire=github.com/docker/docker`, `go mod edit -require=github.com/moby/moby/api@v1.55.0`, `go mod edit -require=github.com/moby/moby/client@v0.5.0`, and `go mod download github.com/moby/moby/api@v1.55.0 github.com/moby/moby/client@v0.5.0`; remove only stale Docker checksum entries.
  2. Replace all five setup occurrences (Dockerfile, CI, three Make blocks) with the exact API/client pair.
  3. Add exact `go list -m` assertions inside every copied-CoreDNS setup after `go get`/`go mod tidy` and before tests/build/audit, so temporary-module MVS drift fails visibly and the selected versions appear in logs.
  4. Pin Make CoreDNS setup to v1.14.3 and its Go test/audit images to Go 1.25. Preserve Dockerfile Go 1.26 and existing unrelated dependency pins.
  5. Add `IMAGE_TAG ?= coredns-docker-cluster` and use `$(IMAGE_TAG)` in the build recipe without changing the default.
- Verify:
  - `go mod verify`
  - `test "$(go list -m -f '{{.Version}}' github.com/moby/moby/client)" = v0.5.0`
  - `test "$(go list -m -f '{{.Version}}' github.com/moby/moby/api)" = v1.55.0`
  - `test "$(git grep -o 'github.com/moby/moby/client@v0.5.0' -- Dockerfile Makefile .github/workflows/docker-publish.yml | wc -l)" -eq 5`
  - `test "$(git grep -o 'github.com/moby/moby/api@v1.55.0' -- Dockerfile Makefile .github/workflows/docker-publish.yml | wc -l)" -eq 5`
  - `make -n build IMAGE_TAG=probe | grep -F -- '-t probe'`
  - `! git grep -n 'github.com/docker/docker'`
- Pass: all commands exit 0 and diff contains no unrelated version changes.
- Fail: fix the exact module/setup mismatch; do not run root `go mod tidy` because the plugin module layout is intentionally external.
- Alternative: centralize all dependency versions.
- Rejected because: broader workflow redesign is not needed for this migration.

### Wave 2 -- Validation Gate

**V2: Validate API, modules, unit/race tests, and image build** [medium] -- validation-lead
- Blocked by: T2, T3.
- Mutation boundary: Go/Docker caches, one persisted unique validation image, evidence, checklist/status.
- Commands:
  1. Re-run T2 formatting and every T3 exact-version check, then run `git diff --check`.
  2. `mkdir -p .specs/migrate-moby-client/evidence`.
  3. Run `set -o pipefail; make test-unit 2>&1 | tee .specs/migrate-moby-client/evidence/test-unit.log` and require exact temporary-module lines for API v1.55.0 and client v0.5.0.
  4. Run `set -o pipefail; make test-race 2>&1 | tee .specs/migrate-moby-client/evidence/test-race.log` and require the same exact module lines and no race report.
  5. Run this as one Git Bash command so the tag cannot be lost between steps:

```bash
set -euo pipefail
EVIDENCE=.specs/migrate-moby-client/evidence
VALIDATION_TAG="joyride-moby-validation-$(git rev-parse --short HEAD)-$(date -u +%Y%m%d%H%M%S)-$$"
printf '%s\n' "$VALIDATION_TAG" > "$EVIDENCE/validation-image-tag.txt"
if docker image inspect "$VALIDATION_TAG" >/dev/null 2>&1; then
  echo "validation tag already exists: $VALIDATION_TAG" >&2
  exit 1
fi
make build IMAGE_TAG="$VALIDATION_TAG" 2>&1 | tee "$EVIDENCE/build.log"
docker image inspect "$VALIDATION_TAG" > "$EVIDENCE/build-image-inspect.json"
```

- Pass: root and copied-CoreDNS modules are exact, focused/existing tests pass, no race appears, the tag did not pre-exist, and the uniquely tagged production image builds.
- Fail: preserve logs and the persisted tag name; remove only that tag if it exists, create a focused fix task, and rerun V2.

### Wave 3

**T4: Harden integration assertions and project-scoped cleanup** [medium] -- devops-pro
- Blocked by: V2.
- Files: `Makefile`, `docker-compose.test.yml`.
- Mutation boundary: these two files only.
- Requirements:
  1. Add `TEST_PROJECT ?= joyride-test` and `TEST_IMAGE ?= joyride-test-coredns`; pass `--project-name "$(TEST_PROJECT)"` to every integration `up` and `down`, and invoke Compose with `TEST_IMAGE="$(TEST_IMAGE)"` so YAML interpolation receives the selected tag. Add `image: ${TEST_IMAGE:-joyride-test-coredns}` to the built coredns service.
  2. Run Compose with `--exit-code-from dns-tester`. Use one POSIX/Git-Bash-compatible recipe with an idempotent `EXIT INT TERM` cleanup trap, capture tester and cleanup statuses, always run project-scoped `down -v --remove-orphans`, and return tester status when nonzero, otherwise cleanup status.
  3. Assert every dynamic labeled hostname resolves exactly to `HOSTIP=192.168.1.100`; empty answers fail.
  4. Replace public unknown names with a reserved `.invalid` name. Capture `dig @coredns -p 54 unknown.invalid +tries=1 +time=2` status/output, require exit status 9 plus a `timed out` diagnostic, and reject successful empty, NXDOMAIN, or REFUSED responses.
  5. Keep static-host checks and existing service topology.
- Verify: `git diff --check -- Makefile docker-compose.test.yml`; `make -n test-integration TEST_PROJECT=probe-project TEST_IMAGE=probe-image` must show the same explicit project on up/down, `--exit-code-from dns-tester`, the cleanup trap, and the recorded image.
- Pass: tester success returns cleanup status; tester failure returns nonzero after cleanup; cleanup failure returns nonzero; interruption invokes scoped cleanup; DNS assertions are value-based.
- Fail: correct the focused workflow logic before V3.
- Alternative: new integration harness.
- Rejected because: the existing Compose workflow can be made truthful with focused edits.

### Wave 3 -- Validation Gate

**V3: Validate Docker DNS workflow and project cleanup** [medium] -- validation-lead
- Blocked by: T4.
- Mutation boundary: only the persisted unique Compose project/image, V2 image, evidence, checklist/status.
- Before running, confirm the migrated source uses only Docker ping/list/inspect/events operations; any Docker mutation method blocks integration.
- Run the complete validation and cleanup in one Git Bash command:

```bash
set -uo pipefail
EVIDENCE=.specs/migrate-moby-client/evidence
mkdir -p "$EVIDENCE" || exit 1
VALIDATION_PROJECT="joyride-moby-$(git rev-parse --short HEAD)-$(date -u +%Y%m%d%H%M%S)-$$"
TEST_IMAGE="${VALIDATION_PROJECT}-coredns"
printf '%s\n' "$VALIDATION_PROJECT" > "$EVIDENCE/validation-project.txt" || exit 1
printf '%s\n' "$TEST_IMAGE" > "$EVIDENCE/integration-image-tag.txt" || exit 1

for name in coredns-test test-app-coredns test-app-joyride test-app-multi dns-tester; do
  if docker container inspect "$name" >/dev/null 2>&1; then
    echo "test container name already exists: $name" >&2
    exit 1
  fi
done
if docker image inspect "$TEST_IMAGE" >/dev/null 2>&1; then
  echo "test image already exists: $TEST_IMAGE" >&2
  exit 1
fi

check_project_resources() {
  local failed=0 ids
  ids=$(docker ps -aq --filter "label=com.docker.compose.project=$VALIDATION_PROJECT") || failed=1
  test -z "$ids" || { echo "project containers remain: $ids"; failed=1; }
  ids=$(docker network ls -q --filter "label=com.docker.compose.project=$VALIDATION_PROJECT") || failed=1
  test -z "$ids" || { echo "project networks remain: $ids"; failed=1; }
  ids=$(docker volume ls -q --filter "label=com.docker.compose.project=$VALIDATION_PROJECT") || failed=1
  test -z "$ids" || { echo "project volumes remain: $ids"; failed=1; }
  return "$failed"
}

check_project_resources || { echo "validation project collision" >&2; exit 1; }
make test-integration TEST_PROJECT="$VALIDATION_PROJECT" TEST_IMAGE="$TEST_IMAGE" > "$EVIDENCE/test-integration.log" 2>&1
TEST_STATUS=$?
printf '%s\n' "$TEST_STATUS" > "$EVIDENCE/test-integration.status" || TEST_STATUS=1
FINAL_STATUS=$TEST_STATUS

RESOURCE_STATUS=0
check_project_resources >> "$EVIDENCE/test-integration.log" 2>&1 || RESOURCE_STATUS=$?
if test "$TEST_STATUS" -ne 0 || test "$RESOURCE_STATUS" -ne 0; then
  docker compose --project-name "$VALIDATION_PROJECT" -f docker-compose.test.yml down -v --remove-orphans >> "$EVIDENCE/test-integration.log" 2>&1 || FINAL_STATUS=1
  FINAL_STATUS=1
fi

for name in coredns-test test-app-coredns test-app-joyride test-app-multi dns-tester; do
  if docker container inspect "$name" >/dev/null 2>&1; then
    echo "test container remains: $name" >> "$EVIDENCE/test-integration.log"
    FINAL_STATUS=1
  fi
done
if docker image inspect "$TEST_IMAGE" >/dev/null 2>&1; then
  docker image rm "$TEST_IMAGE" >> "$EVIDENCE/test-integration.log" 2>&1 || FINAL_STATUS=1
fi
if test -s "$EVIDENCE/validation-image-tag.txt"; then
  VALIDATION_TAG=$(cat "$EVIDENCE/validation-image-tag.txt") || FINAL_STATUS=1
  if test -n "${VALIDATION_TAG:-}" && docker image inspect "$VALIDATION_TAG" >/dev/null 2>&1; then
    docker image rm "$VALIDATION_TAG" >> "$EVIDENCE/test-integration.log" 2>&1 || FINAL_STATUS=1
  fi
else
  echo "missing V2 validation image tag evidence" >> "$EVIDENCE/test-integration.log"
  FINAL_STATUS=1
fi
check_project_resources >> "$EVIDENCE/test-integration.log" 2>&1 || FINAL_STATUS=1
exit "$FINAL_STATUS"
```

- Pass: exit 0; exact dynamic/unknown assertions passed; no project-labeled or fixed-name resources remain; both persisted unique images are absent.
- Fail: block completion, preserve evidence, recover only persisted project/image names, and rerun the original failed boundary.
- Incident transition: the first failed Docker mutation blocks later checks until project cleanup and the failed command pass.

## Dependency Graph

```
Wave 1: T1 -> V1
Wave 2: V1 -> T2, T3 -> V2
Wave 3: V2 -> T4 -> V3
Final: V3 -> F1 -> F2 -> F3 -> F4 -> F5
```

## Success Criteria

1. Stable API/client versions are exact in root and temporary CoreDNS modules; no tracked product file references the old path.
   - Verify: T3 and V2 exact-version commands plus `! git grep -n 'github.com/docker/docker'`.
2. Watcher list, inspect, filter, message, error, closure, and cancellation paths compile and pass focused tests.
   - Verify: `make test-unit && make test-race`.
3. Production image builds without overwriting the default local tag.
   - Verify: V2 unique-tag build and inspect.
4. Labeled containers resolve exactly to `192.168.1.100`, a reserved unknown name times out as split DNS requires, and unique integration resources are removed.
   - Verify: V3 run and project-label postconditions.
5. The local timeout proves only the necessary precondition for EdgeRouter fallback. The deferred deployment outcome remains: EdgeRouter resolves external names upstream without configuration changes.
   - Verify: deferred to home-lab deployment and not required for archive.
6. Work remains uncommitted and isolated on `chore/migrate-moby-client` with only planned implementation, plan, review, and evidence paths changed.
   - Verify: final Git status/diff inventory in F2.

## Validation Contract

### Automation completeness

- Required: yes. All branch, module, test, build, cleanup, evidence, and archive operations have commands above.
- Credentials: existing Git remote access plus public registries; never record credential output.
- Manual validation: not required.
- Deployment validation: not required; local integration is not deployment.

### Required automated validation

- V1 proves branch/base isolation.
- V2 is the single unit/race/module/image gate.
- V3 is the single Docker DNS and cleanup gate.
- `git diff --check` and repository-wide tracked-file old-path checks must pass.
- Any failure leaves the affected checklist item unchecked, sets Execution Status to blocked, records command/evidence, and prevents archive.

### Final Gate Procedures

- **F1 Task-specific verification:** run `test "$(rg -U -P -c '^- \[x\] (?:T[1-4]|V[1-3]):.*\n  - Status: passed\n  - Evidence: (?! --\s*$).+' .specs/migrate-moby-client/plan.md)" -eq 7`. Pass only when the command exits 0; then mark F1 passed with the plan path as evidence.
- **F2 Repo-wide validation:** run `git diff --check`, `! git grep -n 'github.com/docker/docker'`, exact root module checks, and `test "$(git branch --show-current)" = chore/migrate-moby-client`. Then run the following allowlist check and require an empty output file:

```bash
mkdir -p .specs/migrate-moby-client/evidence
git status --porcelain=v1 --untracked-files=all | awk '
{
  path=substr($0,4)
  if (path != "plugins/docker-cluster/docker_watcher.go" &&
      path != "plugins/docker-cluster/docker_watcher_test.go" &&
      path != "go.mod" && path != "go.sum" && path != "Dockerfile" &&
      path != ".github/workflows/docker-publish.yml" &&
      path != "Makefile" && path != "docker-compose.test.yml" &&
      path !~ /^\.specs\/migrate-moby-client\//) print path
}' > .specs/migrate-moby-client/evidence/unexpected-paths.txt
test ! -s .specs/migrate-moby-client/evidence/unexpected-paths.txt
```

Any other tracked or untracked path fails F2.
- **F3 Manual validation:** record `not-required` with the risk decision; no command/user action.
- **F4 Deployment validation:** record `not-required`; confirm no push/deploy/issue mutation occurred.
- **F5 Archive preflight and archive:** first run `test "$(rg -U -P -c '^- \[x\] F[1-4]:.*\n  - Status: passed\n  - Evidence: (?! --\s*$).+' .specs/migrate-moby-client/plan.md)" -eq 4`, `test -d .specs/migrate-moby-client`, and `test ! -e .specs/archive/migrate-moby-client`. If they pass, run `mkdir -p .specs/archive && mv .specs/migrate-moby-client .specs/archive/migrate-moby-client`. Only after the move succeeds, use targeted plan edits on `.specs/archive/migrate-moby-client/plan.md` to set frontmatter `status: completed`, set `completed:` to the actual UTC date, mark F5 checked/passed with the archive path as evidence, and set Execution Status to completed. Then run `test ! -e .specs/migrate-moby-client`, `test -f .specs/archive/migrate-moby-client/plan.md`, `test -d .specs/archive/migrate-moby-client/review-1`, `test -d .specs/archive/migrate-moby-client/evidence`, and verify the archived frontmatter/F5 fields with `rg`. If the move or post-move edit fails, preserve every existing path, mark the readable plan blocked, record both path inventories, and do not retry automatically.

### Archive rule

Archive only after V1-V3 and F1-F4 pass. Do not mark the plan completed before the move. F5 verifies preconditions, moves once, updates only the archived plan, and then verifies postconditions. No implementation item may be checked by review; `/do-it` checks each item immediately after its own pass evidence is written.

## Telemetry & Evidence Contract

Use the execution checklist, Execution Status, review artifacts, and named command logs as durable evidence. Runtime workflow telemetry owns episode and phase records; this plan does not create a plan-specific telemetry writer.

- Store non-secret command logs under `.specs/migrate-moby-client/evidence/`.
- Each checked task or gate must name its validation command and evidence path in the checklist.
- On failure, leave the item unchecked, set its status and Execution Status to blocked, and record the exact failing command plus non-secret log path.
- Do not record tokens, credentials, webhooks, environment dumps, or Docker config.

## Execution Checklist

### Wave 1
- [x] T1: Create branch from current remote main
  - Status: passed
  - Evidence: `.specs/archive/migrate-moby-client/evidence/git-preflight.log`; created `chore/migrate-moby-client` at `8c91d6477c6638607aaebb603b6dd7582aed3caf`
- [x] V1: Validate branch isolation
  - Status: passed
  - Evidence: `.specs/archive/migrate-moby-client/evidence/branch-isolation.log`; branch HEAD equals `origin/main` and only the plan subtree is untracked

### Wave 2
- [x] T2: Adapt Moby API and add event regression tests
  - Status: passed
  - Evidence: `.specs/archive/migrate-moby-client/evidence/t2-validation.log`; formatting, scoped diff, API surface, helpers, and focused test coverage checks passed
- [x] T3: Replace dependency paths and pin acceptance toolchain
  - Status: passed
  - Evidence: `.specs/archive/migrate-moby-client/evidence/go-mod-verify.log` and `.specs/archive/migrate-moby-client/evidence/t3-validation.log`; module verification, exact versions/occurrences, old-path absence, tag override, and whitespace checks passed
- [x] V2: Validate API, modules, unit/race tests, and image build
  - Status: passed
  - Evidence: `.specs/archive/migrate-moby-client/evidence/v2-preflight.log`, `test-unit.log`, `test-race.log`, `build.log`, `build-image-inspect.json`, and `v2-validation.status`; exact modules, unit/race suites, and unique production image passed

### Wave 3
- [x] T4: Harden integration assertions and project-scoped cleanup
  - Status: passed
  - Evidence: `.specs/archive/migrate-moby-client/evidence/t4-dry-run.log`, `t4-compose-config.yml`, and `t4-validation.status`; scoped project/image, exit-code propagation, trap cleanup, exact DNS values, reserved-name timeout, and rendered Compose config checks passed
- [x] V3: Validate Docker DNS workflow and project cleanup
  - Status: passed
  - Evidence: `.specs/archive/migrate-moby-client/evidence/docker-operation-audit.log`, `test-integration.log`, `test-integration.status`, and `v3-validation.status`; exact dynamic DNS, reserved-name timeout, fixed/project resource cleanup, and both unique image removals passed

### Final Gates
- [x] F1: Task-specific verification complete
  - Status: passed
  - Evidence: `.specs/archive/migrate-moby-client/evidence/f1-validation.status`; all seven T1-T4 and V1-V3 checklist records are passed with evidence
- [x] F2: Repo-wide validation complete
  - Status: passed
  - Evidence: `.specs/archive/migrate-moby-client/evidence/f2-validation.status` and `unexpected-paths.txt`; whitespace, old-path absence, exact modules, branch, and changed-path allowlist passed
- [x] F3: Manual validation not required
  - Status: passed
  - Evidence: `Risk & Manual Gate Decision`; local reversible work, unique Docker resources, automated DNS assertions, and verified cleanup make manual validation not required
- [x] F4: Deployment validation not required
  - Status: passed
  - Evidence: `.specs/archive/migrate-moby-client/evidence/f4-validation.status`; HEAD remains at `origin/main`, remote feature branch is absent, staging is empty, and no deployment or issue mutation occurred
- [x] F5: Archive preflight and archive complete
  - Status: passed
  - Evidence: `.specs/archive/migrate-moby-client/`; archive preflight passed, the plan subtree moved once without collision, and archived postconditions passed

## Execution Status

- **Overall status:** completed-and-archived
- **Active phase/task:** complete
- **Next action:** none
- **Blockers:** none
- **Last completed wave/gate:** Final gate F5 passed on 2026-07-15; archived at `.specs/archive/migrate-moby-client/`
- **Completed work:** migrated watcher source/tests, root and copied-CoreDNS dependency graphs, Dockerfile/CI/Make setup, and Compose integration assertions/cleanup
- **Validation summary:** exact modules, unit tests, race tests, unique production image build, dynamic and split-DNS integration, cleanup, whitespace, branch, and changed-path allowlist all passed
- **Manual/deployment gates:** not required; no commit, staging, push, deployment, or issue mutation occurred
- **Last verified baseline:** branch `chore/migrate-moby-client` and `origin/main` are `8c91d6477c6638607aaebb603b6dd7582aed3caf`

## Handoff Notes

- The expected pre-branch untracked subtree is `.specs/migrate-moby-client/`, including review artifacts. Reject unrelated changes without discarding them.
- The Docker socket is an administrative API boundary. Integration is limited to current read/list/inspect/events/ping calls and a unique Compose project; do not run it if source inspection shows any Docker mutation call.
- Persist V2 and V3 project/image names before mutation so cleanup removes only those names.
- Current CI baseline: Actions run `29417753470` had passing tests and failed the old Docker dependency layer.
- Final standalone readiness passed after command, cleanup, checklist, coverage, and archive repairs.
- Leave implementation uncommitted and unpushed. Archive movement is plan-state handling, not a source commit.

## Workflow Eval Record

### Episode

- **schema_version:** 1
- **episode_id:** `2026-07-15T19-41-07Z-do-it-migrate-moby-client`
- **command:** `do-it`
- **artifact_path:** `.specs/archive/migrate-moby-client/plan.md`
- **repo_root:** `C:/Projects/Personal/joyride`
- **started_at:** `2026-07-15T19:41:07Z`
- **completed_at:** `2026-07-15T21:00:22Z`
- **status:** completed
- **classification:** completed-and-archived
- **archive_status:** archived
- **archive_path:** `.specs/archive/migrate-moby-client/`
- **redaction_status:** no_sensitive_output

### Phases

| phase_id | phase_type | depends_on | started_at | completed_at | status | evidence |
|----------|------------|------------|------------|--------------|--------|----------|
| wave-1 | implementation | -- | 2026-07-15T19:41:07Z | 2026-07-15T19:41:18Z | passed | `evidence/git-preflight.log`, `evidence/branch-isolation.log` |
| wave-2 | implementation | wave-1 | 2026-07-15T19:41:18Z | 2026-07-15T20:07:58Z | passed | `evidence/t2-validation.log`, `evidence/t3-validation.log`, `evidence/test-unit.log`, `evidence/test-race.log`, `evidence/build.log` |
| wave-3 | implementation | wave-2 | 2026-07-15T20:07:58Z | 2026-07-15T20:53:40Z | passed | `evidence/t4-validation.status`, `evidence/test-integration.log`, `evidence/v3-validation.status` |
| final-gates | validation | wave-3 | 2026-07-15T20:53:40Z | 2026-07-15T20:55:38Z | passed | `evidence/f1-validation.status`, `evidence/f2-validation.status`, `evidence/f4-validation.status` |
| archive | archive | final-gates | 2026-07-15T20:55:38Z | 2026-07-15T20:56:53Z | passed | `evidence/archive-preflight.log`, `evidence/archive-postconditions.log` |
| post-run-eval | report | archive | 2026-07-15T20:56:53Z | 2026-07-15T21:00:22Z | passed | `evidence/post-run-consistency.log` |

### Validation Events

| event_id | phase_id | task_id | event_type | command_line | exit_code | status | evidence |
|----------|----------|---------|------------|--------------|-----------|--------|----------|
| evt-v1-001 | wave-1 | V1 | validation_result | branch and `origin/main` isolation checks | 0 | passed | `evidence/branch-isolation.log` |
| evt-t3-001 | wave-2 | T3 | command | `go mod verify` | 0 | passed | First combined attempt timed out; isolated rerun passed in `evidence/go-mod-verify.log` |
| evt-v2-001 | wave-2 | V2 | command | `make test-unit` | 0 | passed | `evidence/test-unit.log` |
| evt-v2-002 | wave-2 | V2 | command | `make test-race` | 0 | passed | `evidence/test-race.log`; no race report |
| evt-v2-003 | wave-2 | V2 | command | `make build IMAGE_TAG=<unique-tag>` | 0 | passed | `evidence/build.log`, `evidence/build-image-inspect.json` |
| evt-t4-001 | wave-3 | T4 | validation_result | `make -n test-integration TEST_PROJECT=probe-project TEST_IMAGE=probe-image` | 0 | passed | `evidence/t4-dry-run.log`, `evidence/t4-compose-config.yml` |
| evt-v3-001 | wave-3 | V3 | command | `make test-integration TEST_PROJECT=<unique-project> TEST_IMAGE=<unique-image>` | 0 | passed | `evidence/test-integration.log`, `evidence/test-integration.status` |
| evt-f2-001 | final-gates | F2 | validation_result | whitespace, old-path, exact-module, branch, and changed-path allowlist checks | 0 | passed | `evidence/f2-validation.status`, `evidence/unexpected-paths.txt` |
| evt-f5-001 | archive | F5 | archive_move | move active plan subtree to `.specs/archive/migrate-moby-client/` | 0 | passed | `evidence/archive-postconditions.log` |
| evt-eval-001 | post-run-eval | -- | post_run_eval | checklist, evidence, archive, and image cleanup consistency checks | 0 | passed | `evidence/post-run-consistency.log` |

### Manual and Deployment Gates

- **manual_required:** false
- **risk_level:** medium
- **blast_radius:** personal-repo
- **rollback:** known
- **decision:** not_required
- **decision_reason:** Local reversible changes used unique Docker resources, exact automated DNS assertions, and verified cleanup; no shared or production users were affected.
- **deployment_decision:** not_required
- **deployment_reason:** No push or deployment was in scope; remote feature branch remained absent and HEAD stayed at `origin/main`.

### Compact Eval

- **checklist_completion:** 12/12 passed with non-empty evidence
- **validation_failures_after_review:** 0 implementation failures; one `go mod verify` invocation timed out before an isolated successful rerun
- **blocker_reason:** none
- **friction_tags:** `slow-go-mod-verify`, `hidden-panel-unavailable`
- **missing_evidence:** Hidden evaluator outputs were unavailable because both evaluator launches exited with code 1; deterministic completion evidence is present and passed.
- **improvement_candidate:** Capture subcommand progress and duration before long aggregate validation commands so a timeout identifies the active boundary without process inspection.
- **eval_confidence:** high; checklist, logs, Docker state, Git state, archive postconditions, and cleanup were directly rechecked.
- **execution_outcome:** `{"classification":"completed-and-archived","completed":true,"blocked_by_plan_gap":false,"validation_failures_after_review":0,"manual_gate_ambiguity":false,"archive_issue":false,"missed_by_review":[]}`
- **panel_quality_label:** `{"sizing":"right_sized","reason":"Reviewed plan executed without a plan gap or unresolved gate; the post-run evaluator launch failure did not remove deterministic completion evidence.","confidence":"medium"}`
