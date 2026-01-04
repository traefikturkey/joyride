package traefikexternals

import (
	"testing"
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
