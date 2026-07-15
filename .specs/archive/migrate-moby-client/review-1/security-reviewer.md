---
reviewer: security-reviewer
status: complete
finding_count: 5
---

# Findings

- severity: high
  category: "substantive defect"
  confidence: high
  evidence: "Severity rationale: high because the validation container is granted a Docker-daemon control boundary, so compromise of the test image or plugin can affect every local Docker resource. `docker-compose.test.yml` runs `coredns` as `user: root` and mounts `/var/run/docker.sock:/var/run/docker.sock:ro`; a read-only filesystem mount does not make the Docker API read-only. The plan therefore understates its stated \"personal-local-repo and local Docker resources\" blast radius and \"no destructive data operation\" claim."
  required_fix: "Run validation against an isolated Docker context/VM or a dedicated disposable daemon, or remove the socket-backed integration from the automated gate. If the socket is unavoidable, document that it is effectively daemon-admin access, narrow the test boundary, and add explicit post-run checks for containers, networks, volumes, and images outside the test project."
- severity: high
  category: "substantive defect"
  confidence: high
  evidence: "Severity rationale: high because a failed integration run can leave containers, networks, and volumes on the host, while the documented cleanup is unreachable on failure. `Makefile` target `test-integration` runs `docker compose ... up --build --abort-on-container-exit` followed by `down -v` as separate recipe lines; make stops at the first nonzero line. The plan's `make test-unit && make test-race && make build && make test-integration` likewise cannot reach its later cleanup command after a failed integration gate."
  required_fix: "Put cleanup in an unconditional shell trap or a make wrapper that records the primary exit status, runs `docker compose ... down -v --remove-orphans`, verifies no project resources remain, then returns the original failure. Specify the same failure-safe cleanup for every Compose workflow used by the plan."
- severity: medium
  category: "substantive defect"
  confidence: high
  evidence: "Severity rationale: medium because the stated easy rollback does not cover local Docker image state. `Makefile` target `build` always tags the result as `coredns-docker-cluster`; a pre-existing image with that tag is overwritten by the validation build. The plan only calls these \"temporary Docker images\" and says rollback is to preserve changes, but provides no preflight capture, unique validation tag, or restoration/removal rule."
  required_fix: "Build validation with a unique run-specific tag (and pass it through Compose), or capture the existing tag digest and restore it after validation. Add a gate that checks the original tag digest and reports/remediates any changed local image state before archive."
- severity: medium
  category: "process defect"
  confidence: high
  evidence: "Severity rationale: medium because branch creation does not isolate the working tree and the plan has no executable recovery path for uncommitted edits. `T1` uses `git switch -c` after checking only the current commit/status, then the rollback instruction is merely \"Stop with changes preserved on the branch\". Git branch refs do not own uncommitted changes; edits remain in the shared worktree and can follow a later switch to `main` or block it. The preflight also does not show a concrete local/remote collision check despite requiring one."
  required_fix: "Define and execute a reversible preflight snapshot (status, diff, untracked paths, current branch, and local/remote target refs), and provide an explicit failure procedure that preserves the plan/evidence while preventing edits from being carried onto `main`. Add commands that actually verify both local and remote branch-name collisions before `switch -c`."
- severity: low
  category: "process defect"
  confidence: high
  evidence: "Severity rationale: low because it weakens reproducibility and can execute unreviewed upstream code during a security-sensitive validation gate. `Makefile` `test-unit` and `test-race` fetch the GitHub API's latest CoreDNS release into the test container, while the plan says CoreDNS v1.14.3 is pinned and the Dockerfile/CI use that pin. The required validation therefore does not exercise the same pinned source across local entry points."
  required_fix: "Pin the CoreDNS tag in the Make test and audit commands to v1.14.3 (or one explicit variable shared by those commands), verify the fetched ref, and make archive eligibility require that all local, Dockerfile, and CI validation paths use the same pinned source."
