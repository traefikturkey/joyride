package dockercluster

import (
	"strings"
	"testing"
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
