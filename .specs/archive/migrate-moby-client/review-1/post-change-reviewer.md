---
reviewer: post-change:reviewer
status: complete
finding_count: 5
---

# Findings

- severity: high
  category: "substantive defect"
  confidence: high
  evidence: "Severity rationale: V2 cannot meet its required unique-image pass criterion in its scheduled order. V2 runs `make build IMAGE_TAG=\"$VALIDATION_TAG\"` (plan.md:181), but the current Makefile build target unconditionally tags `coredns-docker-cluster` (Makefile:7-8). The plan assigns IMAGE_TAG support to T4 (plan.md:193), which is blocked by V2 (plan.md:188). Thus V2 overwrites the default tag and `docker image inspect \"$VALIDATION_TAG\"` fails before the task intended to add tag support can run."
  required_fix: "Move the IMAGE_TAG Makefile change into T3 or a prerequisite task before V2, and update mutation boundaries/dependencies. Then retain V2's unique-tag build and inspect as the proof."
- severity: high
  category: "process defect"
  confidence: high
  evidence: "Severity rationale: required evidence cannot be produced deterministically, so F1's pass condition is not runnable. The telemetry contract requires a JSONL record with nine named fields for every T/V/F task and says a checkbox is checked only after its passing record (plan.md:276-284). No task command or reusable writer command creates/appends a valid record. F1 merely says to assert passing JSONL records (plan.md:255), leaving its own record and all preceding records undefined."
  required_fix: "Specify one exact, safe JSONL append/validation command or wrapper, invoke it for each task/gate including failures, and give F1 an exact schema/status query plus an evidence path before allowing checklist transitions."
- severity: high
  category: "substantive defect"
  confidence: high
  evidence: "Severity rationale: the plan claims exact Moby selection in temporary copied-CoreDNS modules but only executes root-module queries. T3 requires assertions after temporary-module setup (plan.md:157), yet its listed `go list -m` commands run from the repository root (plan.md:161-162); V2 only reruns those checks (plan.md:176). `make test-unit` and `make test-race` create and modify `/tmp/coredns` inside containers, where MVS can select different versions, without any required `go list -m` assertion there."
  required_fix: "Add exact API/client `go list -m` checks in each copied-CoreDNS setup shell after the `go get` operations and before tests, or provide a dedicated reproducible temporary-module validation command. Capture its output as V2 evidence."
- severity: medium
  category: "process defect"
  confidence: high
  evidence: "Severity rationale: V3's local-resource safety decision is not executable or auditable. Its pre/postflight says to save sorted Docker IDs and compare inventories (plan.md:210-222), but supplies neither the `docker ps/network ls/volume ls` commands nor the comparison command. Its Run instruction also says to capture primary and cleanup status without a shell wrapper that defines the status truth table. Consequently an executor can claim matching inventories or preserve the wrong exit code without violating a concrete command contract."
  required_fix: "Provide exact before/after inventory commands (including deterministic ID-only sorting), exact diff commands, and one shell wrapper that runs integration and cleanup, records both statuses, and returns the specified primary-or-cleanup result. Name the resulting log/evidence files."
- severity: medium
  category: "process defect"
  confidence: high
  evidence: "Severity rationale: F2 cannot deterministically enforce its planned-path isolation requirement. It runs `git status --short` and `git diff --name-only` (plan.md:256) but defines no allowed-path predicate, even though success criterion 5 requires only planned implementation, plan, review, and evidence paths (plan.md:241). A status with an unrelated changed path therefore has command exit 0 and no stated machine-checkable failure behavior."
  required_fix: "Add an exact allowed-path validation command for both tracked and untracked status entries, define the permitted prefixes/files, and make F2 fail with the offending paths recorded in evidence."
