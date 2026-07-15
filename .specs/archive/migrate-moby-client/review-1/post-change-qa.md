---
reviewer: post-change:qa
status: complete
finding_count: 4
---

# Findings

## 1. [substantive defect] Compose does not select the tester exit code

- Severity: high
- Category: substantive defect
- Confidence: high
- Evidence: Severity rationale: high because V3 can report a passing integration run when a non-tester service exits early, or when Compose itself finishes without propagating the DNS tester's assertion result. The current command is `docker compose -f docker-compose.test.yml up --build --abort-on-container-exit` (`Makefile`, `test-integration`). `--abort-on-container-exit` stops the project but does not name the assertion container whose exit code is the test result. The updated T4 requirement to preserve the `up` status therefore does not make that status a trustworthy DNS verdict. The only test assertions execute in `dns-tester` (`docker-compose.test.yml`, `dns-tester.command`).
- Required fix: run Compose with `--exit-code-from dns-tester` (which also implies abort-on-container-exit), capture that command's status before cleanup, and make V3 explicitly require the captured tester status to be zero. Add the truth-table case where `dns-tester` exits nonzero and verify that `make test-integration` returns nonzero after `down -v --remove-orphans`.

## 2. [substantive defect] The unknown-name timeout assertion is required but not operationally defined

- Severity: high
- Category: substantive defect
- Confidence: high
- Evidence: Severity rationale: high because the existing Compose checks turn any `dig` failure into success with `|| echo`, and the updated plan replaces that with a claim of an "expected no-response timeout" without specifying the command result, output, or timing condition that distinguishes timeout from NXDOMAIN, REFUSED, a CoreDNS crash, name-resolution failure, or a missing `dig` binary. `docker-compose.test.yml` currently applies this false-positive form to both `www.google.com` and `nonexistent.example.com`. T4 and V3 have no exact assertion truth table to prove the stated distinction.
- Required fix: define one deterministic unknown-name assertion that runs `dig @coredns -p 54 <unknown> +tries=1 +time=2`, requires its expected timeout exit/result and timeout diagnostic, and rejects a DNS response status such as NXDOMAIN or REFUSED. Capture the command output and status before evaluating it, and include an explicit negative assertion that an empty successful `+short` result is not accepted. Use a reserved test name rather than an externally meaningful public hostname.

## 3. [substantive defect] Focused event tests do not prove list, inspect, or stop-path behavior

- Severity: medium
- Category: substantive defect
- Confidence: high
- Evidence: Severity rationale: medium because the dynamic Compose topology can exercise a start event but cannot reliably prove initial `ContainerList`, `ContainerInspect`, or record removal on stop. The actual watcher calls list during `syncContainers`, inspect on a start event, and removes records only on `stop` or `die` (`plugins/docker-cluster/docker_watcher.go`). Existing tests only cover hostname parsing (`plugins/docker-cluster/docker_watcher_test.go`). Updated T2 requires message dispatch, error/EOF, closed channels, and cancellation, but does not require assertions that a start message invokes inspect and adds a record, that a stop/die message removes it, or that initial list `.Items` populates records. Thus tests of a narrow channel helper can pass while the migrated wrapper and state paths are wrong.
- Required fix: make the seam testable at the watcher boundary and add deterministic focused cases for (1) list result `.Items` creating a labeled record, (2) start message causing inspect `.Container` label extraction and record addition, and (3) stop and die messages removing the prior record. Keep the existing error, closed-channel, and cancellation cases, and require the focused test names in V2 evidence.

## 4. [process defect] V3's cleanup claim does not cover interruption exits or define comparison commands

- Severity: medium
- Category: process defect
- Confidence: high
- Evidence: Severity rationale: medium because "always run down" and "inventories match" are acceptance claims without a shell-level guarantee or executable comparison. T4 requires preservation of the `up` status and cleanup failure reporting, but does not require a trap for `INT`, `TERM`, or shell errors after resources are created. V3 says to save sorted IDs and compare before/after, but gives no commands and does not state that container inventory includes stopped containers. A recipe can therefore clean normal success/failure paths while leaving resources on interruption, or compare only running containers and falsely prove cleanup.
- Required fix: require one shell recipe that installs an idempotent cleanup trap for `EXIT INT TERM`, records the primary status, runs `docker compose -f docker-compose.test.yml down -v --remove-orphans`, and returns the documented truth-table result. Specify inventory commands using `docker ps -aq`, `docker network ls -q`, and `docker volume ls -q` with sorted outputs, then require `cmp` or `diff -u` for each before/after file. Add a controlled failure-path validation that demonstrates cleanup and inventory equality after a nonzero tester result.
