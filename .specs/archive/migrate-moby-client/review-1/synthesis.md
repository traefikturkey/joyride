---
date: 2026-07-15
status: synthesis-complete-plan-blocked
---

# Review: Migrate Joyride to stable Moby client modules

## Review Panel
| Reviewer | Base Agent | Assigned Expert Persona | Why selected | Adversarial angle | Artifact |
|----------|------------|-------------------------|--------------|-------------------|----------|
| Completeness | reviewer | Execution-contract auditor | Fresh-session readiness | Missing prerequisites, commands, status, archive | `reviewer.md` |
| Safety | security-reviewer | Local Docker and Git safety reviewer | Docker socket and branch mutations | Cleanup, state overwrite, rollback, permissions | `security-reviewer.md` |
| Scope | product-manager | Maintainer scope reviewer | Smallest sufficient issue migration | Theater, duplication, omitted outcome | `product-manager.md` |
| Go API | backend-dev | Moby v0.5 compatibility reviewer | Changed signatures and channels | Result wrappers, filters, event termination | `backend-moby-api.md` |
| Build | devops-pro | Dockerfile/Make/CI parity reviewer | Repeated dependency setup | Version drift, toolchains, cleanup | `devops-build-ci.md` |
| Verification | qa-engineer | Docker DNS regression reviewer | User-visible and failure-path evidence | False-positive tests and command truth tables | `qa-verification.md` |

Review panel decision: six reviewers selected for complexity 6/10 and risk 4/10. Expected high-risk areas were Moby event semantics, exact module selection, repeated build surfaces, local Docker mutation, and integration-test truthfulness.

## Standard Reviewer Findings
### reviewer
- Accepted: missing `## Execution Status`, stale/hardcoded branch preflight, unreachable cleanup, and undefined final/archive gates.
- Merged duplicate: pinned CoreDNS mismatch.

### security-reviewer
- Accepted: Docker socket grants daemon-admin capability, cleanup is unreachable on failure, validation can overwrite the default image tag, and branch rollback language is incomplete.
- Merged duplicate: latest CoreDNS validation mismatch.

### product-manager
- Accepted: repository-wide old-path check is too narrow and DNS integration exits successfully without asserting dynamic answers.
- Merged duplicate: repeated validation commands.
- Dismissed: removing telemetry conflicts with the required execution-evidence contract.

## Additional Expert Findings
### backend-dev
- Accepted: event-stream termination lacks regression coverage; exact Ping/list/inspect/event result behavior and filter construction need explicit requirements; exact target module versions must be asserted in the compile environment.

### devops-pro
- Accepted: local tests use unpinned CoreDNS and an older test toolchain; failure-safe Compose cleanup is absent.
- Dismissed: aligning unrelated memberlist/fsnotify versions is outside this migration; the production Docker build independently validates its dependency graph.

### qa-engineer
- Accepted: watcher/event behavior lacks focused coverage; presence-only version grep can pass wrong versions; root module versions are not validated.

### Post-change panel
- Accepted and applied: move IMAGE_TAG support before V2; exact copied-module assertions; executable JSONL writer/parser; exact project-scoped Compose commands; `--exit-code-from dns-tester`; collision-resistant persisted tags; test-image cleanup; exact unknown timeout and dynamic answer checks; allowed-path predicate; actual completion date; closed-Err semantics; explicit EdgeRouter deferral.
- Accepted and applied: unique Compose project/image names replace broad all-daemon inventories; cleanup remains inside the workflow under test with recovery teardown only after a failed cleanup postcondition.
- Dismissed: isolated daemon or a new manual gate. Verified source scope uses only read/list/inspect/events/ping operations, execution is explicitly user-authorized on a personal local daemon, and a second approval would not technically constrain socket capability. The plan retains explicit daemon-admin risk and blocks on any mutation call.
- Dismissed: broad list/inspect mocking seam. Exact compile checks plus integration start/inspect assertions cover migrated wrappers; the plan adds focused stop/die and channel tests without a broad adapter.

## Suggested Additional Reviewers
- Git workflow specialist -- only if branch/base policy changes beyond the dynamic preflight.
- Docker Compose specialist -- only if integration hardening requires redesign beyond project-scoped cleanup and assertions.
- Go modules maintainer -- only if root module validation exposes replace/exclude or minimal-version-selection conflicts.

## Bugs (must fix before execution)
1. Add exact dynamic branch/base and local/remote collision checks; remove the stale hardcoded commit gate.
2. Make Compose cleanup unconditional and preserve the primary test exit code.
3. Replace integration checks that accept empty dynamic DNS answers with explicit expected-address assertions.
4. Specify exact Moby v0.5 option/result/filter behavior and add deterministic event-channel regression coverage.
5. Replace presence-only version checks with exact versions/counts and validate root plus copied-CoreDNS module selections.
6. Add `## Execution Status` and exact final/archive gate procedures.
7. Ensure production build validation does not overwrite the user's default local image tag.
8. Widen old-module removal verification to all tracked product files.

## Hardening
1. Pin local CoreDNS tests to the production/CI CoreDNS version.
2. Align local test containers with the module's declared Go version.
3. State that Docker socket access is daemon-admin capability and inventory non-test Docker resources around integration validation.
4. Consolidate duplicate expensive tests into one gate per wave.
5. Define failure evidence and project-resource cleanup checks explicitly.

## Simpler Alternatives / Scope Reductions
1. Keep direct Moby client use; extract only a narrow event-channel consumer if needed for deterministic tests.
2. Validate the Dockerfile independently rather than aligning unrelated dependencies across every setup block.
3. Keep one unit/race/build gate and one integration gate rather than rerunning suites in task criteria and final gates.

## Automation Readiness
- Agent-runnable operational steps: not ready before fixes; branch collision, cleanup, exact-version, and archive commands are incomplete.
- Credential/auth flow clarity: public fetches and local Docker only; no secret is required. Docker socket capability must be described accurately.
- Evidence and archive gates: JSONL destination exists conceptually, but final gate commands and archive pre/postconditions are missing.
- Manual-only steps and justification: none required; local reversible execution remains appropriate after cleanup hardening.

## Contested or Dismissed Findings
1. A separate VM/daemon was rejected as disproportionate. Accepted controls are explicit daemon-admin risk, source-call inspection, pre/post inventory, scoped cleanup, and no manual gate.
2. Aligning memberlist and fsnotify was rejected as unrelated churn. The production Docker build independently validates its graph.
3. Removing telemetry was rejected because the workflow requires machine-readable evidence fields.
4. Reviewer uses of `false positive` for cleanup/version grep were reclassified as substantive defects: those commands could falsely pass or skip cleanup.

## Verification Notes
1. `Makefile:test-integration` has separate `up` and `down` lines; Make skips `down` after a failing `up`.
2. A read-only Docker socket mount does not restrict Docker API mutation capability.
3. Dynamic `dig +short` commands do not compare output and can exit 0 with an empty answer; static checks do compare exact values.
4. Existing `docker_watcher_test.go` covers hostname parsing only.
5. Moby v0.5 signatures and wrappers were verified in downloaded source.
6. Current `main` and `origin/main` both resolve to `8c91d64`; target local and remote branches do not exist now, but execution must check dynamically.

## Reviewer Artifact Status
| Reviewer | Artifact | Status | Notes |
|----------|----------|--------|-------|
| reviewer | `.specs/migrate-moby-client/review-1/reviewer.md` | read | usable; preview ignored |
| security-reviewer | `.specs/migrate-moby-client/review-1/security-reviewer.md` | read | usable |
| product-manager | `.specs/migrate-moby-client/review-1/product-manager.md` | read | usable |
| backend-dev | `.specs/migrate-moby-client/review-1/backend-moby-api.md` | read | usable |
| devops-pro | `.specs/migrate-moby-client/review-1/devops-build-ci.md` | read | usable |
| qa-engineer | `.specs/migrate-moby-client/review-1/qa-verification.md` | read | usable |

## Timing Notes
| Step | Duration | Notes |
|------|----------|-------|
| Initial review panel | 5m19s wall clock | 2026-07-15T15:58:55Z to 16:04:14Z; per-reviewer timing unavailable |
| Artifact reads | timing unavailable | all expected artifacts read |
| Recovery calls | none | all artifacts were actionable |
| Verification | timing unavailable | high-severity claims checked against repository and Moby source |
| Synthesis | timing unavailable | duplicates merged before apply |

## Review Yield
- Total findings across both panels: 45
- Merged must-fix defects: 17
- Merged hardening items: 8
- Duplicate/overlapping findings: 12
- Low-value/theater proposals: 2
- False positives after verification: 0
- Intentionally rejected or narrowed: 6
- Applied: 25 merged fixes/clarifications
- Readiness change: initial plan not ready; revised plan remains blocked because accepted post-change fixes materially changed Docker mutation flow
- Initial per-reviewer yield: reviewer 4 accepted/1 duplicate; security 4 accepted/1 duplicate; product-manager 2 accepted/1 duplicate/1 rejected; backend-dev 4 accepted; devops-pro 2 accepted/1 duplicate/1 rejected; qa-engineer 3 accepted
- Post-change per-reviewer yield: reviewer 5 accepted; security 2 accepted/1 rejected; product-manager 2 accepted/2 duplicate or theater; backend-dev 1 accepted; devops-pro 3 accepted; qa-engineer 3 accepted/1 narrowed

## Panel Quality Inputs
- Task structure: separate integration-hardening wave and focused watcher test file.
- Validation commands: exact module versions/counts, pinned CoreDNS, unique image tag, dynamic DNS assertions, unconditional cleanup, repository-wide old-path check.
- Manual gate: remains no, with corrected Docker socket blast radius and inventories.
- Archive: add final gate commands, active/archive preconditions, postconditions, and Execution Status.
- Automation: branch preflight and failure cleanup become executable.

## Auto-Apply Plan
- Applied fixes artifact: `.specs/migrate-moby-client/review-1/applied-fixes.md`
- Known-blocker fixes artifact: not run/no prior blockers
- Section integrity check: passed after each plan-edit batch; required headings and all T/V/F checklist IDs occur exactly once; no items are checked
- Material-change review: one six-reviewer post-change panel completed and all artifacts read
- Pre-readiness audit: not run because post-change fixes materially changed Compose project/image mutation flow
- Standalone-readiness result: blocked/not run; workflow requires a new `/review-it`
- Repair passes used: 0

## Review Artifact
Wrote full synthesis to: `.specs/migrate-moby-client/review-1/synthesis.md`

## Overall Verdict
**Fix bugs first**

The post-change panel found and corrected command-order, Compose project, image, telemetry, and validation truth-table defects. Those corrections materially changed the mutation flow after the one permitted post-change panel, so this review must stop before pre-readiness and standalone-readiness checks.

## Recommended Next Step
- Run `/review-it .specs/migrate-moby-client/plan.md` again. Do not run `/do-it` until the new review reaches standalone readiness.
