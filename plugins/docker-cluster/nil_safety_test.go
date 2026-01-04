package dockercluster

import (
	"context"
	"net"
	"testing"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
)

// TestServeDNS_InvalidIPStored verifies that invalid IPs stored in records
// are handled gracefully without panicking.
// Regression test: net.ParseIP returns nil for invalid IPs, and calling
// .To4() on nil causes a panic.
func TestServeDNS_InvalidIPStored(t *testing.T) {
	records := NewRecords()
	records.Add("test.example.com", "not-a-valid-ip")

	dc := &DockerCluster{
		Records:       records,
		TTL:           60,
		UnknownAction: ActionDrop,
	}

	req := new(dns.Msg)
	req.SetQuestion("test.example.com.", dns.TypeA)

	rec := dnstest.NewRecorder(&test.ResponseWriter{})

	// This should NOT panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ServeDNS panicked on invalid IP: %v", r)
		}
	}()

	code, err := dc.ServeDNS(context.Background(), rec, req)

	// Should handle gracefully - either return error or treat as not found
	t.Logf("Result: code=%d, err=%v", code, err)
}

// TestServeDNS_EmptyIPStored verifies empty string IP is handled gracefully.
func TestServeDNS_EmptyIPStored(t *testing.T) {
	records := NewRecords()
	records.Add("test.example.com", "")

	dc := &DockerCluster{
		Records:       records,
		TTL:           60,
		UnknownAction: ActionDrop,
	}

	req := new(dns.Msg)
	req.SetQuestion("test.example.com.", dns.TypeA)

	rec := dnstest.NewRecorder(&test.ResponseWriter{})

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ServeDNS panicked on empty IP: %v", r)
		}
	}()

	dc.ServeDNS(context.Background(), rec, req)
}

// TestServeDNS_IPv6StoredForTypeA verifies IPv6 addresses don't panic when
// requesting A records (IPv4 only).
func TestServeDNS_IPv6StoredForTypeA(t *testing.T) {
	records := NewRecords()
	records.Add("test.example.com", "::1") // IPv6 loopback

	dc := &DockerCluster{
		Records:       records,
		TTL:           60,
		UnknownAction: ActionDrop,
	}

	req := new(dns.Msg)
	req.SetQuestion("test.example.com.", dns.TypeA)

	rec := dnstest.NewRecorder(&test.ResponseWriter{})

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ServeDNS panicked on IPv6 address: %v", r)
		}
	}()

	dc.ServeDNS(context.Background(), rec, req)
}

// TestIPValidation verifies our understanding of net.ParseIP behavior
func TestIPValidation(t *testing.T) {
	testCases := []struct {
		ip       string
		validV4  bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"0.0.0.0", true},
		{"255.255.255.255", true},
		{"::1", false},              // IPv6 - To4() returns nil
		{"2001:db8::1", false},      // IPv6
		{"not-an-ip", false},
		{"", false},
		{"192.168.1", false},
		{"192.168.1.256", false},
		{"192.168.1.1.1", false},
	}

	for _, tc := range testCases {
		t.Run(tc.ip, func(t *testing.T) {
			parsed := net.ParseIP(tc.ip)
			ipv4 := parsed != nil && parsed.To4() != nil

			if ipv4 != tc.validV4 {
				t.Errorf("IP %q: got validV4=%v, want validV4=%v", tc.ip, ipv4, tc.validV4)
			}
		})
	}
}
