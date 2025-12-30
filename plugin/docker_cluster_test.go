package dockercluster

import (
	"context"
	"net"
	"testing"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
)

func TestDockerClusterName(t *testing.T) {
	dc := &DockerCluster{}
	if dc.Name() != "docker-cluster" {
		t.Errorf("expected 'docker-cluster', got '%s'", dc.Name())
	}
}

func TestServeDNSFound(t *testing.T) {
	records := NewRecords()
	records.Add("test.example.com", "192.168.1.100")

	dc := &DockerCluster{
		Records: records,
		TTL:     60,
		Fall:    fall.F{},
	}

	req := new(dns.Msg)
	req.SetQuestion("test.example.com.", dns.TypeA)

	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	code, err := dc.ServeDNS(context.Background(), rec, req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if code != dns.RcodeSuccess {
		t.Errorf("expected RcodeSuccess, got %d", code)
	}
	if rec.Msg == nil {
		t.Fatal("expected response message")
	}
	if len(rec.Msg.Answer) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(rec.Msg.Answer))
	}

	a, ok := rec.Msg.Answer[0].(*dns.A)
	if !ok {
		t.Fatal("expected A record")
	}
	if !a.A.Equal(net.ParseIP("192.168.1.100")) {
		t.Errorf("expected 192.168.1.100, got %s", a.A)
	}
	if a.Hdr.Ttl != 60 {
		t.Errorf("expected TTL 60, got %d", a.Hdr.Ttl)
	}
}

func TestServeDNSNotFoundDrop(t *testing.T) {
	records := NewRecords()

	dc := &DockerCluster{
		Records:       records,
		TTL:           60,
		Fall:          fall.F{},
		UnknownAction: ActionDrop, // default - no response
	}

	req := new(dns.Msg)
	req.SetQuestion("unknown.example.com.", dns.TypeA)

	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	code, err := dc.ServeDNS(context.Background(), rec, req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// ActionDrop returns success but doesn't write a response (timeout behavior)
	if code != dns.RcodeSuccess {
		t.Errorf("expected RcodeSuccess for drop action, got %d", code)
	}
	if rec.Msg != nil {
		t.Error("expected no response for drop action")
	}
}

func TestServeDNSNotFoundNXDomain(t *testing.T) {
	records := NewRecords()

	dc := &DockerCluster{
		Records:       records,
		TTL:           60,
		Fall:          fall.F{},
		UnknownAction: ActionNXDomain,
	}

	req := new(dns.Msg)
	req.SetQuestion("unknown.example.com.", dns.TypeA)

	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	code, err := dc.ServeDNS(context.Background(), rec, req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if code != dns.RcodeNameError {
		t.Errorf("expected RcodeNameError, got %d", code)
	}
}

func TestServeDNSFallthrough(t *testing.T) {
	records := NewRecords()

	var f fall.F
	f.SetZonesFromArgs([]string{})

	nextCalled := false
	next := test.HandlerFunc(func(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		nextCalled = true
		return dns.RcodeSuccess, nil
	})

	dc := &DockerCluster{
		Records: records,
		TTL:     60,
		Fall:    f,
		Next:    next,
	}

	req := new(dns.Msg)
	req.SetQuestion("unknown.example.com.", dns.TypeA)

	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	dc.ServeDNS(context.Background(), rec, req)

	if !nextCalled {
		t.Error("expected next handler to be called with fallthrough")
	}
}

func TestServeDNSAAAAForKnownHost(t *testing.T) {
	// IPv6 is dumb - AAAA queries for known hosts return empty success
	records := NewRecords()
	records.Add("test.example.com", "192.168.1.100")

	dc := &DockerCluster{
		Records: records,
		TTL:     60,
		Fall:    fall.F{},
	}

	req := new(dns.Msg)
	req.SetQuestion("test.example.com.", dns.TypeAAAA)

	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	code, err := dc.ServeDNS(context.Background(), rec, req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if code != dns.RcodeSuccess {
		t.Errorf("expected RcodeSuccess for AAAA query, got %d", code)
	}
	if rec.Msg == nil {
		t.Fatal("expected response message")
	}
	if len(rec.Msg.Answer) != 0 {
		t.Errorf("expected empty answer for AAAA query, got %d answers", len(rec.Msg.Answer))
	}
}

func TestServeDNSAAAAForUnknownHost(t *testing.T) {
	// AAAA for unknown host should use unknown_action (drop by default)
	records := NewRecords()

	dc := &DockerCluster{
		Records:       records,
		TTL:           60,
		Fall:          fall.F{},
		UnknownAction: ActionDrop,
	}

	req := new(dns.Msg)
	req.SetQuestion("unknown.example.com.", dns.TypeAAAA)

	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	code, err := dc.ServeDNS(context.Background(), rec, req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if code != dns.RcodeSuccess {
		t.Errorf("expected RcodeSuccess for drop action, got %d", code)
	}
	if rec.Msg != nil {
		t.Error("expected no response for drop action")
	}
}

func TestServeDNSCaseInsensitive(t *testing.T) {
	records := NewRecords()
	records.Add("test.example.com", "192.168.1.100")

	dc := &DockerCluster{
		Records: records,
		TTL:     60,
		Fall:    fall.F{},
	}

	// Query with uppercase
	req := new(dns.Msg)
	req.SetQuestion("TEST.EXAMPLE.COM.", dns.TypeA)

	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	code, err := dc.ServeDNS(context.Background(), rec, req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if code != dns.RcodeSuccess {
		t.Errorf("expected RcodeSuccess for case-insensitive lookup, got %d", code)
	}
}

func BenchmarkServeDNS(b *testing.B) {
	records := NewRecords()
	records.Add("test.example.com", "192.168.1.100")

	dc := &DockerCluster{
		Records: records,
		TTL:     60,
		Fall:    fall.F{},
	}

	req := new(dns.Msg)
	req.SetQuestion("test.example.com.", dns.TypeA)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := dnstest.NewRecorder(&test.ResponseWriter{})
		dc.ServeDNS(context.Background(), rec, req)
	}
}
