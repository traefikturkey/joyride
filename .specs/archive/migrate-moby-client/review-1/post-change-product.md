# Post-change product review

## Finding 1
- Category: substantive defect
- Evidence: Severity rationale: high -- the plan validates only local DNS timeout behavior while explicitly deferring the stated user outcome: EdgeRouter must fall back to upstream DNS and resolve external names such as `www.ilude.com`. The Objective and Success Criteria call out dynamic labels and unknown-query timeouts, but do not state that local timeout validation is only a proxy for the end-user fallback outcome or identify the deferred outcome as remaining unverified.
- Confidence: high -- the plan's Explicit Deferrals excludes host/EdgeRouter DNS checks, while the project goal depends on that behavior.
- Required fix: Add an explicit user-outcome criterion: unknown split-DNS queries permit EdgeRouter upstream resolution of external names without an EdgeRouter configuration change. Mark it deployment-environment deferred, and state that V3 proves only the necessary local no-response precondition, not the final EdgeRouter outcome.

## Finding 2
- Category: duplicate
- Evidence: Severity rationale: medium -- T4 requires `make test-integration` to always execute `docker compose ... down -v --remove-orphans`; V3 then unconditionally executes the same down operation after that target returns. The second teardown adds Docker mutation and failure paths without testing a different user-visible behavior. It can also make it unclear whether the target's cleanup guarantee actually worked.
- Confidence: high -- both cleanup operations are explicitly required in T4 and V3.
- Required fix: Retain cleanup inside `make test-integration` as the workflow under test. In V3, verify cleanup from the target using targeted postconditions, and invoke an external recovery teardown only if the target fails before its cleanup guarantee can run.

## Finding 3
- Category: low-value/theater
- Evidence: Severity rationale: medium -- V3 records and requires equality of every local Docker container, network, and volume ID before and after the test. This broad inventory can fail from unrelated local Docker activity and does not provide a more direct proof than the fixed-name absence checks plus project-scoped resource checks. It increases evidence collection without improving the migration's DNS outcome.
- Confidence: high -- the preflight and postflight inventory requirements cover all local Docker resources rather than resources owned by the Compose test project.
- Required fix: Use a unique Compose project name or project label, snapshot and compare only that project's resources, and retain the fixed-name collision/absence checks where fixed names remain necessary.

## Finding 4
- Category: process defect
- Evidence: Severity rationale: low -- F5 requires changing frontmatter to `completed: 2026-07-15` before archive regardless of when execution actually completes. A fixed completion date makes the archive evidence inaccurate and undermines the plan's stated evidence contract.
- Confidence: high -- the required F5 value is hard-coded.
- Required fix: Require the executor to set `completed` to the actual archive completion date at F5, or omit the field until that date is known.
