package dockercluster

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
)

func TestExtractHostnamesValidation(t *testing.T) {
	dw := &DockerWatcher{labels: []string{"coredns.host.name"}}

	tests := []struct {
		name   string
		labels map[string]string
		want   []string
	}{
		{
			name:   "single valid hostname",
			labels: map[string]string{"coredns.host.name": "app.example.com"},
			want:   []string{"app.example.com"},
		},
		{
			name:   "comma-separated valid hostnames",
			labels: map[string]string{"coredns.host.name": "a.example.com, b.example.com"},
			want:   []string{"a.example.com", "b.example.com"},
		},
		{
			name:   "duplicates deduplicated",
			labels: map[string]string{"coredns.host.name": "a.example.com,a.example.com"},
			want:   []string{"a.example.com"},
		},
		{
			name:   "empty entries skipped",
			labels: map[string]string{"coredns.host.name": ",, ,a.example.com,"},
			want:   []string{"a.example.com"},
		},
		{
			name:   "label absent",
			labels: map[string]string{"other.label": "a.example.com"},
			want:   nil,
		},
		{
			name:   "invalid hostname dropped (whitespace)",
			labels: map[string]string{"coredns.host.name": "bad host.example.com"},
			want:   nil,
		},
		{
			name:   "invalid hostname dropped (NUL byte)",
			labels: map[string]string{"coredns.host.name": "bad\x00.example.com"},
			want:   nil,
		},
		{
			name:   "oversized hostname dropped",
			labels: map[string]string{"coredns.host.name": strings.Repeat("a", 254)},
			want:   nil,
		},
		{
			name:   "valid + invalid mixed",
			labels: map[string]string{"coredns.host.name": "good.example.com,bad host"},
			want:   []string{"good.example.com"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dw.extractHostnames(tc.labels)
			if !equalSlice(got, tc.want) {
				t.Errorf("extractHostnames(%v) = %v, want %v", tc.labels, got, tc.want)
			}
		})
	}
}

func TestApplyContainerResults(t *testing.T) {
	dw := &DockerWatcher{
		hostIP:     "192.168.1.100",
		labels:     []string{"coredns.host.name"},
		records:    NewRecords(),
		containers: make(map[string][]string),
	}

	if !dw.applyContainerSummary(container.Summary{
		ID:     "summary-id",
		Labels: map[string]string{"coredns.host.name": "summary.example.com"},
	}) {
		t.Fatal("applyContainerSummary returned false for a labeled container")
	}
	if got, ok := dw.records.Lookup("summary.example.com"); !ok || got != dw.hostIP {
		t.Fatalf("summary record = %q, %v; want %q, true", got, ok, dw.hostIP)
	}

	hostnames := dw.applyContainerInspect("inspect-id", container.InspectResponse{
		Config: &container.Config{Labels: map[string]string{"coredns.host.name": "inspect.example.com"}},
	})
	if !equalSlice(hostnames, []string{"inspect.example.com"}) {
		t.Fatalf("inspect hostnames = %v, want [inspect.example.com]", hostnames)
	}
	if got, ok := dw.records.Lookup("inspect.example.com"); !ok || got != dw.hostIP {
		t.Fatalf("inspect record = %q, %v; want %q, true", got, ok, dw.hostIP)
	}
}

func TestConsumeEventsDispatchesMessage(t *testing.T) {
	messages := make(chan events.Message)
	errs := make(chan error)
	dispatched := make(chan events.Message, 1)

	go func() {
		messages <- events.Message{Action: "stop", Actor: events.Actor{ID: "container-id"}}
		errs <- io.EOF
	}()

	err := consumeEvents(context.Background(), messages, errs, func(event events.Message) {
		dispatched <- event
	})
	if !errors.Is(err, io.EOF) {
		t.Fatalf("consumeEvents error = %v, want EOF", err)
	}
	if got := <-dispatched; got.Actor.ID != "container-id" || got.Action != "stop" {
		t.Fatalf("dispatched event = %#v", got)
	}
}

func TestConsumeEventsReturnsStreamErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "non-nil error", err: errors.New("stream failed")},
		{name: "EOF", err: io.EOF},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := make(chan error, 1)
			errs <- tc.err
			if got := consumeEvents(context.Background(), make(chan events.Message), errs, func(events.Message) {}); !errors.Is(got, tc.err) {
				t.Fatalf("consumeEvents error = %v, want %v", got, tc.err)
			}
		})
	}
}

func TestConsumeEventsReturnsEOFWhenErrorChannelCloses(t *testing.T) {
	errs := make(chan error)
	close(errs)

	if err := consumeEvents(context.Background(), make(chan events.Message), errs, func(events.Message) {}); !errors.Is(err, io.EOF) {
		t.Fatalf("consumeEvents error = %v, want EOF", err)
	}
}

func TestConsumeEventsStopsOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := consumeEvents(ctx, make(chan events.Message), make(chan error), func(events.Message) {}); err != nil {
		t.Fatalf("consumeEvents error = %v, want nil", err)
	}
}

func TestHandleEventRemovesStoppedContainers(t *testing.T) {
	for _, action := range []events.Action{"stop", "die"} {
		t.Run(string(action), func(t *testing.T) {
			dw := &DockerWatcher{
				hostIP:     "192.168.1.100",
				records:    NewRecords(),
				containers: map[string][]string{"container-id": {"app.example.com"}},
			}
			dw.records.Add("app.example.com", dw.hostIP)

			dw.handleEvent(events.Message{Action: action, Actor: events.Actor{ID: "container-id"}})

			if _, ok := dw.records.Lookup("app.example.com"); ok {
				t.Fatal("record still exists after stop event")
			}
			if _, ok := dw.containers["container-id"]; ok {
				t.Fatal("container still tracked after stop event")
			}
		})
	}
}

func equalSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
