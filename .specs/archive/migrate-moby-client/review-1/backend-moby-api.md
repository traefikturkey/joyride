## Findings

### 1. [substantive defect] T2 does not require a regression test for stable event-stream termination
- Severity: high
- Confidence: high
- Evidence: Severity rationale: the event loop is the highest-risk semantic boundary and the existing test file only covers hostname extraction. The downloaded `github.com/moby/moby/client v0.5.0` source documents that `Events` returns an `EventsResult`, sends `io.EOF` on `Err` when the stream is completely read, sends context/errors on `Err`, and then closes `Err` (`client/system_events.go:30-66`). T2 only accepts `make test-unit`, whose current docker-cluster tests do not exercise `watchEvents`, channel termination, or reconnect behavior. A migration can therefore compile while mishandling the result wrapper or a closed/error channel.
- Required fix: add a focused docker-cluster test (or an equivalent deterministic test seam) covering an event result that reports an error/EOF and a context cancellation, and require it in T2/V2. The implementation must consume `EventsResult.Messages` and `EventsResult.Err` without treating a closed channel as a real error or blocking forever.

### 2. [substantive defect] The T2 acceptance criteria are not precise enough about the new option and result signatures
- Severity: high
- Confidence: high
- Evidence: Severity rationale: using the old call shapes is a compile failure, while using the wrong result fields can become a runtime or data-flow defect. Stable client v0.5.0 defines `Ping(ctx, PingOptions) (PingResult, error)` (`client/ping.go:73`), `ContainerList(ctx, ContainerListOptions) (ContainerListResult, error)` with data in `ContainerListResult.Items` (`client/container_list.go:35-40`), `ContainerInspect(ctx, containerID, ContainerInspectOptions) (ContainerInspectResult, error)` with data in `.Container` (`client/container_inspect.go:24-30`), and `Events(ctx, EventsListOptions) EventsResult` (`client/system_events.go:16-32`). The plan lists these names but does not require exact option values, explicit result-field assertions, or a compile probe against the pinned modules.
- Required fix: make T2 require `PingOptions{NegotiateAPIVersion: true}` (or document and verify the intended lazy-negotiation behavior), `ContainerListOptions{Filters: ...}`, `ContainerInspectOptions{}`, and assertions/tests proving `.Items`, `.Container`, `.Messages`, and `.Err` are used. Compile the copied plugin against exactly `client v0.5.0` and `api v1.55.0`, not only whichever dependency resolution happens in the test image.

### 3. [substantive defect] T2 does not specify the required replacement for the old filter constructors
- Severity: medium
- Confidence: high
- Evidence: Severity rationale: a direct mechanical import rewrite cannot preserve this call site and will either fail compilation or invite an incorrect compatibility shim. The current source uses `filters.NewArgs(filters.Arg("status", "running"))` and `filters.NewArgs` plus repeated `Add` calls for events (`plugins/docker-cluster/docker_watcher.go:10-12,204-205,264-269`). Stable client v0.5.0 has no `filters` package in the client API and defines `client.Filters` as a map with `Add` (`client/filters.go:10-29`). T2 says to use `client.Filters` but does not state that filters must be initialized with `make(client.Filters)` and populated with `.Add(...)`, nor require verification that repeated event values remain ORed under the same term.
- Required fix: explicitly require replacing each constructor with `make(client.Filters).Add(...)`, preserving `status=running` and the three event values, and add a focused filter/query assertion or test so the serialized filter semantics are not changed.

### 4. [process defect] The plan's compilation gate is weaker than its claimed version-compatibility guarantee
- Severity: medium
- Confidence: high
- Evidence: Severity rationale: successful compilation is only meaningful if it uses the target modules and source layout. `make test-unit` and `make test-race` clone the latest CoreDNS release and run `go get` inside that temporary module (`Makefile:18-59`), while the plan's compatibility target is specifically pinned to `github.com/moby/moby/client v0.5.0` and `github.com/moby/moby/api v1.55.0`. The acceptance criteria do not capture `go list -m` output or fail if transitive resolution selects another Moby version. The current repository source still imports the old client (`plugins/docker-cluster/docker_watcher.go:10-13`), so no target-version compile evidence exists yet.
- Required fix: add a deterministic acceptance command after dependency installation, such as `go list -m github.com/moby/moby/client github.com/moby/moby/api` with exact-version assertions, and run the plugin compile/tests in that same prepared CoreDNS module. Record the module versions and compiler result as evidence for T2/V2.
