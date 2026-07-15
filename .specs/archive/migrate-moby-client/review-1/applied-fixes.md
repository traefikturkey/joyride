---
date: 2026-07-15
status: applied-plan-blocked-new-review
---

# Applied Fixes

| Finding | Category | Target sections | Edit intent | Checklist impact |
|---------|----------|-----------------|-------------|------------------|
| Dynamic branch/base and collision checks | bug | Automation, T1/V1 | Replace hardcoded commit with executable local/upstream/remote checks | T1/V1 retained, criteria revised |
| Failure-safe Compose cleanup | bug | T4/V3, Validation | Change Make integration target in implementation and verify cleanup on all paths | Add T4/V3 |
| Dynamic DNS assertions | bug | T4/V3, Success | Require exact labeled-host answers rather than command exit alone | Add T4/V3 |
| Exact Moby API behavior and event tests | bug | T2/V2 | Specify options, wrappers, filters, closed/error channels, focused tests | T2 grows to 2 files |
| Exact module selection | bug | T3/V2 | Assert exact root/copied module versions and install-block counts | T3/V2 revised |
| Execution status and archive procedures | process | Validation, Checklist, Execution Status | Add deterministic final gates and archive pre/postconditions | F1-F5 receive commands |
| Unique build tag | hardening | T4/V3 | Parameterize Make image tag and use a run-specific validation tag | Add T4/V3 |
| Repository-wide removal check | bug | T3/V2, Success | Use tracked-file `git grep` instead of a hand-maintained path list | Existing checks revised |
| Pinned CoreDNS and Go test image | hardening | T3/V2 | Pin local setup to v1.14.3 and Go 1.25 | T3 criteria revised |
| Docker socket blast radius/inventory | hardening | Risk, V3 | Describe daemon-admin access; snapshot and compare non-test resources | V3 criteria revised |
| Duplicate validation | hardening | Waves, Validation | One validator gate per wave; final gates reference recorded evidence | No extra task checkbox |
| Evidence destinations | hardening | Automation, Telemetry, final gates | Name JSONL/log evidence and failure behavior | F1-F5 revised |
| Separate integration hardening wave | process | Task table, waves, graph | Keep migration and existing workflow hardening distinct | Add T4/V3 |

## Post-Change Fixes

| Finding | Category | Target sections | Applied change | Checklist impact |
|---------|----------|-----------------|----------------|------------------|
| IMAGE_TAG ordered after V2 | bug | T3, V2, T4 | Moved Make image-tag support into T3 before the unique build | IDs unchanged |
| Temporary module versions unproven | bug | T3, V2 | Required exact go-list assertions inside copied CoreDNS setup and logs | V2 criteria revised |
| Evidence records not executable | process | Telemetry, F1 | Added exact PowerShell JSONL append and schema validation commands | Final evidence runnable |
| Compose project collision risk | bug | T4, V3 | Added persisted unique project and explicit project name on every operation | Mutation flow materially changed |
| Integration image leak | bug | T4, V3 | Added persisted unique Compose image and scoped removal | V3 criteria revised |
| Tester status not authoritative | bug | T4 | Required `--exit-code-from dns-tester` and status truth table | T4 criteria revised |
| Cleanup/inventory commands vague | bug | T4, V3 | Added trap requirements, exact run wrapper, project-label checks, and recovery rule | V3 criteria revised |
| Unknown timeout vague | bug | T4 | Added reserved name and exact captured timeout/status assertions | T4 criteria revised |
| Broad all-daemon inventories | theater | V3 | Replaced with project-label and fixed-name postconditions | Scope reduced |
| Tag collision/failure evidence | hardening | V2 | Persist tag before build, add PID/timestamp/commit entropy, reject collision | V2 criteria revised |
| Allowed path check vague | process | F2 | Added explicit allowlist and unexpected-path evidence | F2 revised |
| Hardcoded completion date | process | F5 | Use actual UTC completion date | F5 revised |
| Closed Messages assumption | bug | T2 | Focus compatibility tests on closed Err; label Messages handling defensive | T2 criteria revised |
| EdgeRouter outcome ambiguity | clarity | Success | Mark local timeout as precondition and deployment outcome deferred | No new gate |

These post-change fixes alter the Docker project/image mutation boundary. Per the review state machine, the plan is blocked for a fresh `/review-it` rather than recursively launching another panel.

## Intentional Omissions

- Separate disposable Docker daemon: disproportionate for a personal local repository after explicit capability/risk, source inspection, inventory, and scoped cleanup.
- Aligning unrelated memberlist/fsnotify versions: outside the Moby migration; production Docker build validates its own graph.
- Removing telemetry: prohibited by the plan workflow's evidence contract.

No checklist item is marked complete by this review.
