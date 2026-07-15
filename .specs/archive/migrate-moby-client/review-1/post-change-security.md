---
reviewer: post-change:security
status: complete
finding_count: 3
---

# Findings

- severity: critical
  category: "substantive defect"
  confidence: high
  evidence: "CRITICAL rationale: T4 and V3 run `docker compose -f docker-compose.test.yml ... down -v --remove-orphans` without an explicit, validation-unique project name (plan.md:194, 214). Compose derives its project from the working directory/configuration; the production compose file is in the same repository, so a matching project can have its containers classified as orphans and stopped/removed. The fixed container-name check does not protect differently named production containers, and before/after inventories only detect the loss after it occurs."
  required_fix: "Generate and persist a validation project name before any Compose operation, pass `--project-name \"$VALIDATION_PROJECT\"` to every test `up` and `down`, and restrict cleanup to that project. Preflight must reject resources already bearing that project label; retain the inventory comparison as postcondition evidence, not as the safety control."
- severity: high
  category: "substantive defect"
  confidence: high
  evidence: "HIGH rationale: The plan treats a static check for new direct Docker client mutation methods plus before/after ID inventories as adequate protection for a container holding the daemon-admin socket (plan.md:34, 213, 342). Neither control constrains what code running in the socket-mounted container can submit to the daemon, and inventories cannot restore deleted containers, volumes, images, or altered daemon state. This makes the stated rollback both non-enforceable and insufficient on the personal primary daemon."
  required_fix: "Run the socket-mounted integration workflow against an isolated disposable Docker daemon/VM, or make a manual gate mandatory and explicitly accept the non-reversible primary-daemon risk. Do not represent static source inspection and postflight inventories as rollback controls."
- severity: medium
  category: "substantive defect"
  confidence: high
  evidence: "MEDIUM rationale: V2 calls a timestamp-to-the-second image tag \"unique\" and writes its evidence only after `make build` returns (plan.md:181). Concurrent/repeated runs in one second can retag an existing image; later V3 removes that tag (plan.md:218), deleting another run's reference. A failed build after creating the tag leaves no recorded tag for failure cleanup."
  required_fix: "Generate a collision-resistant tag before building, persist it before the build command, and preflight with `docker image inspect` to reject an existing tag. On every V2 exit path, use the persisted tag for scoped cleanup; V3 should remove it only after confirming it belongs to this run (for example, a run-specific OCI label)."
