---
reviewer: reviewer
status: complete
finding_count: 5
---

# Findings

- severity: high
  category: "process defect"
  confidence: high
  evidence: "Severity rationale: high because /do-it is explicitly required to resume without conversation context, but the plan has no `## Execution Status` section. It only has per-checkbox `Status: pending` fields and a draft frontmatter status, with no current phase, next action, blocker, or completion state. Evidence: `.specs/migrate-moby-client/plan.md` frontmatter and `## Execution Checklist` (lines 1-3, 106-137)."
  required_fix: "Add an explicit Execution Status section containing current overall status, active phase/task, next command, blockers, and the rule for updating it after each mutation or validation gate. Keep it consistent with the checklist and evidence ledger."
- severity: high
  category: "substantive defect"
  confidence: high
  evidence: "Severity rationale: high because the branch mutation can start from the wrong base or collide remotely while still appearing to satisfy the listed checks. T1 says to confirm current `main` and local/remote collisions, but its only Verify command checks status, HEAD, and current branch; it never checks current branch is `main`, local branch existence, or `git ls-remote` for the remote branch. The plan also hardcodes HEAD `8c91d64` without a command establishing that this is the current main tip."
  required_fix: "Add exact read-only preflight commands for current branch/base (`git branch --show-current`, `git rev-parse main`, and the intended relationship), local collision, and remote collision (`git show-ref`/`git ls-remote`). Define the pass/fail behavior when main has moved or the target exists."
- severity: high
  category: "false positive"
  confidence: high
  evidence: "Severity rationale: high because the required cleanup guarantee is false on the primary failure path. `Makefile:test-integration` runs `docker compose ... up --build --abort-on-container-exit` followed by `down -v` as separate recipe lines; a nonzero `up` causes make to stop before cleanup. V2 nevertheless requires cleanup after success or failure and says to run cleanup after a failed Compose mutation."
  required_fix: "Make cleanup unconditional, for example with a shell trap or a make recipe that captures the `up` status, always runs `docker compose ... down -v`, then returns the original status. Also specify the exact direct cleanup command for recovery if the recipe cannot be changed."
- severity: medium
  category: "substantive defect"
  confidence: high
  evidence: "Severity rationale: medium because the stated cross-entry-point version proof is not actually performed. `Makefile:test-unit` and `test-race` resolve `COREDNS_VERSION` from the GitHub latest-release API, while Dockerfile and CI pin `v1.14.3`; the plan claims V2 proves compilation with exactly the versions installed by local tests, CI, and Dockerfile. A future CoreDNS release can therefore make local validation pass while the pinned image/CI path differs."
  required_fix: "Make local test targets use the pinned `v1.14.3` (or a documented shared source) and add an exact check that Dockerfile, CI, and Make targets agree. If latest is intentional, remove the exact-version claim and add separate pinned-path validation."
- severity: medium
  category: "process defect"
  confidence: high
  evidence: "Severity rationale: medium because archive completion can be reported without a deterministic final procedure. F1-F5 are checklist labels but F1-F4 have no individual commands, evidence paths, or explicit pass transitions; F5 says only `Archive preflight complete`, while the archive rule says merely that `/do-it` may archive. No archive command, destination, status update, or verification of the archived artifact is defined."
  required_fix: "Map each final gate to one exact command and evidence record, define when its checkbox/status changes, and document the archive operation, destination, required final status, and post-archive verification. Add the missing gate evidence to the JSONL contract."
