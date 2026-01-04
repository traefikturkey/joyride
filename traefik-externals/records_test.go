package traefikexternals

import (
	"testing"
)

func TestRecords_AddAndLookup(t *testing.T) {
	r := NewRecords()

	r.Add("example.com", "192.168.1.10")

	ip, found := r.Lookup("example.com")
	if !found {
		t.Error("expected to find example.com")
	}
	if ip != "192.168.1.10" {
		t.Errorf("expected 192.168.1.10, got %s", ip)
	}
}

func TestRecords_LookupNormalizesCase(t *testing.T) {
	r := NewRecords()

	r.Add("EXAMPLE.COM", "192.168.1.10")

	ip, found := r.Lookup("example.com")
	if !found {
		t.Error("expected to find example.com (lowercase lookup)")
	}
	if ip != "192.168.1.10" {
		t.Errorf("expected 192.168.1.10, got %s", ip)
	}

	ip, found = r.Lookup("EXAMPLE.COM")
	if !found {
		t.Error("expected to find EXAMPLE.COM (uppercase lookup)")
	}
	if ip != "192.168.1.10" {
		t.Errorf("expected 192.168.1.10, got %s", ip)
	}
}

func TestRecords_Remove(t *testing.T) {
	r := NewRecords()

	r.Add("example.com", "192.168.1.10")
	r.Remove("example.com")

	_, found := r.Lookup("example.com")
	if found {
		t.Error("expected example.com to be removed")
	}
}

func TestRecords_RemoveNonExistent(t *testing.T) {
	r := NewRecords()

	// Should not panic
	r.Remove("nonexistent.com")

	if r.Count() != 0 {
		t.Error("expected count to be 0")
	}
}

func TestRecords_GetAll(t *testing.T) {
	r := NewRecords()

	r.Add("a.example.com", "192.168.1.1")
	r.Add("b.example.com", "192.168.1.2")

	all := r.GetAll()

	if len(all) != 2 {
		t.Errorf("expected 2 records, got %d", len(all))
	}
	if all["a.example.com"] != "192.168.1.1" {
		t.Error("wrong IP for a.example.com")
	}
	if all["b.example.com"] != "192.168.1.2" {
		t.Error("wrong IP for b.example.com")
	}

	// Verify returned map is a copy
	all["c.example.com"] = "192.168.1.3"
	if r.Count() != 2 {
		t.Error("GetAll should return a copy, not the actual map")
	}
}

func TestRecords_Count(t *testing.T) {
	r := NewRecords()

	if r.Count() != 0 {
		t.Error("expected count to be 0 initially")
	}

	r.Add("a.example.com", "192.168.1.1")
	if r.Count() != 1 {
		t.Error("expected count to be 1")
	}

	r.Add("b.example.com", "192.168.1.2")
	if r.Count() != 2 {
		t.Error("expected count to be 2")
	}

	r.Remove("a.example.com")
	if r.Count() != 1 {
		t.Error("expected count to be 1 after removal")
	}
}

func TestRecords_ReplaceAll(t *testing.T) {
	r := NewRecords()

	// Add initial records
	r.Add("old.example.com", "192.168.1.1")
	r.Add("keep.example.com", "192.168.1.2")

	// Replace all
	newRecords := map[string]string{
		"new.example.com":  "192.168.1.10",
		"keep.example.com": "192.168.1.20", // Updated IP
	}
	r.ReplaceAll(newRecords)

	// Old record should be gone
	_, found := r.Lookup("old.example.com")
	if found {
		t.Error("old.example.com should have been removed")
	}

	// New record should exist
	ip, found := r.Lookup("new.example.com")
	if !found || ip != "192.168.1.10" {
		t.Error("new.example.com should exist with 192.168.1.10")
	}

	// Kept record should have new IP
	ip, found = r.Lookup("keep.example.com")
	if !found || ip != "192.168.1.20" {
		t.Error("keep.example.com should have updated IP 192.168.1.20")
	}

	if r.Count() != 2 {
		t.Errorf("expected 2 records, got %d", r.Count())
	}
}

func TestRecords_ReplaceAllNormalizesCase(t *testing.T) {
	r := NewRecords()

	newRecords := map[string]string{
		"UPPERCASE.EXAMPLE.COM": "192.168.1.1",
		"MixedCase.Example.Com": "192.168.1.2",
	}
	r.ReplaceAll(newRecords)

	// Should be able to lookup with lowercase
	ip, found := r.Lookup("uppercase.example.com")
	if !found || ip != "192.168.1.1" {
		t.Error("should find uppercase.example.com")
	}

	ip, found = r.Lookup("mixedcase.example.com")
	if !found || ip != "192.168.1.2" {
		t.Error("should find mixedcase.example.com")
	}
}

func TestRecords_UpdateExisting(t *testing.T) {
	r := NewRecords()

	r.Add("example.com", "192.168.1.1")
	r.Add("example.com", "192.168.1.2") // Update

	ip, found := r.Lookup("example.com")
	if !found || ip != "192.168.1.2" {
		t.Error("expected IP to be updated to 192.168.1.2")
	}

	if r.Count() != 1 {
		t.Error("updating should not increase count")
	}
}
