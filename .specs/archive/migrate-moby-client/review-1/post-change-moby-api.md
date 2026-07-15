---
reviewer: post-change:moby-api
status: complete
finding_count: 1
---

# Findings

- severity: medium
  category: "substantive defect"
  confidence: high
  evidence: "Severity rationale: medium because T2 requires a generic 'closed channel behavior' test for the event result (plan.md:142-143), but stable Moby client v0.5.0 never closes `EventsResult.Messages`. `Events` allocates `messages := make(chan events.Message)`, only defers `close(errs)`, and returns those channels (`C:/Users/mglenn/go/pkg/mod/github.com/moby/moby/client@v0.5.0/system_events.go:29-35,87-90`). Stream completion is signaled by a non-nil error, including `io.EOF`, on `EventsResult.Err`, followed by closure of that error channel (lines 24-27,35-36,67-71). A test that injects a closed messages channel validates an invented behavior rather than the client contract; it can also obscure the necessary behavior for a closed Err channel, whose receive otherwise produces nil and causes the nil-error reconnect loop T2 is meant to prevent."
  required_fix: "Split the requirement by channel. Require deterministic tests for message dispatch, non-nil Err including io.EOF, context cancellation, and a closed Err channel. Require the channel consumer to treat `ok == false` on Err as a non-nil terminal/reconnect error (or an explicitly defined equivalent), never as nil. Do not describe a closed Messages channel as a Moby event-stream behavior; if defensive handling of a synthetic closed messages channel is retained, label it separately as defensive helper behavior rather than compatibility coverage."
