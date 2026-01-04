package traefikexternals

import (
	"context"
	"testing"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
)

func TestTraefikExternals_Name(t *testing.T) {
	te := &TraefikExternals{}
	if te.Name() != "traefik-externals" {
		t.Errorf("expected name 'traefik-externals', got '%s'", te.Name())
	}
}

func TestRecords_Integration(t *testing.T) {
	records := NewRecords()

	// Test Add
	records.Add("test.example.com", "192.168.1.100")
	ip, found := records.Lookup("test.example.com")
	if !found || ip != "192.168.1.100" {
		t.Errorf("expected 192.168.1.100, got %s (found=%v)", ip, found)
	}

	// Test case insensitivity
	ip, found = records.Lookup("TEST.EXAMPLE.COM")
	if !found || ip != "192.168.1.100" {
		t.Error("expected case-insensitive lookup to work")
	}

	// Test Count
	if records.Count() != 1 {
		t.Errorf("expected count 1, got %d", records.Count())
	}

	// Test ReplaceAll
	records.ReplaceAll(map[string]string{
		"new.example.com": "10.0.0.1",
		"api.example.com": "10.0.0.2",
	})

	if records.Count() != 2 {
		t.Errorf("expected count 2 after ReplaceAll, got %d", records.Count())
	}

	_, found = records.Lookup("test.example.com")
	if found {
		t.Error("old record should be gone after ReplaceAll")
	}

	ip, found = records.Lookup("new.example.com")
	if !found || ip != "10.0.0.1" {
		t.Errorf("expected new.example.com -> 10.0.0.1")
	}
}

// TestServeDNS_UnknownWithoutFallthrough verifies that when fallthrough is NOT
// configured, unknown hostnames are dropped (no response) rather than passed
// to the next plugin.
// Regression test: Previously both branches called NextOrFailure, ignoring
// the fallthrough setting entirely.
func TestServeDNS_UnknownWithoutFallthrough(t *testing.T) {
	records := NewRecords()
	// Don't add any records - query will be for unknown hostname

	// Fall.F without zones set means fallthrough is DISABLED
	var f fall.F
	// Don't call f.SetZonesFromArgs() - fallthrough remains disabled

	nextCalled := false
	next := plugin.HandlerFunc(func(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		nextCalled = true
		return dns.RcodeSuccess, nil
	})

	te := &TraefikExternals{
		Records: records,
		TTL:     60,
		Fall:    f,
		Next:    next,
	}

	req := new(dns.Msg)
	req.SetQuestion("unknown.example.com.", dns.TypeA)

	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	te.ServeDNS(context.Background(), rec, req)

	// BUG: If this fails, Next was called even though fallthrough is disabled
	if nextCalled {
		t.Error("Next handler should NOT be called when fallthrough is disabled for unknown hostnames")
	}
}

// TestServeDNS_UnknownWithFallthrough verifies that when fallthrough IS
// configured, unknown hostnames are passed to the next plugin.
func TestServeDNS_UnknownWithFallthrough(t *testing.T) {
	records := NewRecords()

	// Enable fallthrough for all zones
	var f fall.F
	f.SetZonesFromArgs([]string{}) // Empty zones = all zones

	nextCalled := false
	next := plugin.HandlerFunc(func(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		nextCalled = true
		return dns.RcodeSuccess, nil
	})

	te := &TraefikExternals{
		Records: records,
		TTL:     60,
		Fall:    f,
		Next:    next,
	}

	req := new(dns.Msg)
	req.SetQuestion("unknown.example.com.", dns.TypeA)

	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	te.ServeDNS(context.Background(), rec, req)

	if !nextCalled {
		t.Error("Next handler SHOULD be called when fallthrough is enabled")
	}
}

// TestServeDNS_KnownHostname verifies known hostnames return proper A records.
func TestServeDNS_KnownHostname(t *testing.T) {
	records := NewRecords()
	records.Add("web.example.com", "192.168.1.100")

	te := &TraefikExternals{
		Records: records,
		TTL:     60,
	}

	req := new(dns.Msg)
	req.SetQuestion("web.example.com.", dns.TypeA)

	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	code, err := te.ServeDNS(context.Background(), rec, req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if code != dns.RcodeSuccess {
		t.Errorf("expected RcodeSuccess, got %d", code)
	}
	if len(rec.Msg.Answer) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(rec.Msg.Answer))
	}

	a, ok := rec.Msg.Answer[0].(*dns.A)
	if !ok {
		t.Fatal("expected A record")
	}
	if a.A.String() != "192.168.1.100" {
		t.Errorf("expected 192.168.1.100, got %s", a.A.String())
	}
}
