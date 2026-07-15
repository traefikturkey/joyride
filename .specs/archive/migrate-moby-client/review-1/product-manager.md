# Product Manager Review

## Finding 1
- Category: low-value/theater
- Severity: medium
- Confidence: high
- Evidence: Severity rationale: this adds process and mutation overhead without increasing confidence in the migration outcome. The plan requires one JSONL telemetry record per task and gate, an evidence directory, archive-status bookkeeping, and F5 gating, while the checklist and command results already provide the completion record for this small local dependency change. The telemetry contract also says it is needed only "until runtime records these fields automatically," but no runtime integration is specified.
- Required fix: Remove the per-task telemetry and archive-status contract. Keep a single concise validation record, or rely on the execution checklist and command output; retain only evidence needed to reproduce a failure.

## Finding 2
- Category: duplicate
- Severity: medium
- Confidence: high
- Evidence: Severity rationale: repeated validation increases execution cost without adding a distinct acceptance check. T2 runs `make test-unit`; V2 says to rerun every T2 and T3 acceptance command and separately runs `make test-unit` and `make test-race`; the Success Criteria and Validation Contract repeat the same commands again. F1-F5 then add generic gates without distinct observable criteria.
- Required fix: Make V2 the single post-change validation gate. Have T2/T3 use compile or focused inspection checks only, and reference V2 from the success and archive sections instead of repeating commands. Remove generic final gates that have no unique verification.

## Finding 3
- Category: substantive defect
- Severity: medium
- Confidence: high
- Evidence: Severity rationale: the stated "completely removed" outcome can pass while a tracked build or source path still references the frozen module. The removal command is restricted to `go.mod go.sum Dockerfile Makefile .github/workflows/docker-publish.yml plugins`; it does not inspect all tracked files. The objective says there is no dependency or import on the old path, but its verification does not establish that repository-wide claim.
- Required fix: Verify the old module with `git grep -n 'github.com/docker/docker' -- ':!.specs/**'` (and explicitly exclude only intentional review/evidence text if needed), then narrow the acceptance wording to the checked product paths if repository-wide removal is not required.

## Finding 4
- Category: substantive defect
- Severity: medium
- Confidence: medium
- Evidence: Severity rationale: passing a build-oriented integration command does not by itself make the preserved user outcome falsifiable. The plan's success criterion for local DNS is only `make test-integration` exit 0, with no asserted labeled-container hostname resolution, container update/removal behavior, or unknown-query no-response behavior. Those are the behavior being protected by the watcher migration.
- Required fix: State the expected DNS assertions in the integration acceptance criterion and verify the exact workflow: a labeled container resolves to its Docker address, updates/removal are reflected, and an unknown name receives no authoritative answer. If the existing integration test already asserts these, name the test or assertion explicitly.
