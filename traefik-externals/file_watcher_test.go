package traefikexternals

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileWatcher_LoadAllConfigs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test YAML files
	config1 := `
http:
  routers:
    web:
      rule: "Host(` + "`" + `web.example.com` + "`" + `)"
`
	config2 := `
http:
  routers:
    api:
      rule: "Host(` + "`" + `api.example.com` + "`" + `)"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "web.yml"), []byte(config1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "api.yml"), []byte(config2), 0644); err != nil {
		t.Fatal(err)
	}

	records := NewRecords()
	fw := NewFileWatcher(tmpDir, "192.168.1.10", records)

	if err := fw.loadAllConfigs(); err != nil {
		t.Fatalf("loadAllConfigs failed: %v", err)
	}

	if records.Count() != 2 {
		t.Errorf("expected 2 records, got %d", records.Count())
	}

	ip, found := records.Lookup("web.example.com")
	if !found || ip != "192.168.1.10" {
		t.Error("expected web.example.com with 192.168.1.10")
	}

	ip, found = records.Lookup("api.example.com")
	if !found || ip != "192.168.1.10" {
		t.Error("expected api.example.com with 192.168.1.10")
	}
}

func TestFileWatcher_SkipsMiddlewareFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a middleware-only file
	middleware := `
http:
  middlewares:
    auth:
      basicAuth:
        users:
          - "user:password"
`
	// Also has a Host rule but should be skipped
	middlewareWithHost := `
http:
  routers:
    should-skip:
      rule: "Host(` + "`" + `skip.example.com` + "`" + `)"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "middleware.yml"), []byte(middlewareWithHost), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "authentik_middleware.yml"), []byte(middleware), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a valid service file
	service := `
http:
  routers:
    web:
      rule: "Host(` + "`" + `web.example.com` + "`" + `)"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "web.yml"), []byte(service), 0644); err != nil {
		t.Fatal(err)
	}

	records := NewRecords()
	fw := NewFileWatcher(tmpDir, "192.168.1.10", records)

	if err := fw.loadAllConfigs(); err != nil {
		t.Fatalf("loadAllConfigs failed: %v", err)
	}

	// Should only have web.example.com, not skip.example.com
	if records.Count() != 1 {
		t.Errorf("expected 1 record (middleware files skipped), got %d", records.Count())
	}

	_, found := records.Lookup("skip.example.com")
	if found {
		t.Error("skip.example.com should not be loaded from middleware.yml")
	}

	_, found = records.Lookup("web.example.com")
	if !found {
		t.Error("web.example.com should be loaded")
	}
}

func TestFileWatcher_SkipsNonYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create non-YAML files
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("# README"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(`{"key": "value"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create valid YAML
	config := `
http:
  routers:
    web:
      rule: "Host(` + "`" + `web.example.com` + "`" + `)"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "web.yml"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	records := NewRecords()
	fw := NewFileWatcher(tmpDir, "192.168.1.10", records)

	if err := fw.loadAllConfigs(); err != nil {
		t.Fatalf("loadAllConfigs failed: %v", err)
	}

	if records.Count() != 1 {
		t.Errorf("expected 1 record (only YAML files), got %d", records.Count())
	}
}

func TestFileWatcher_StartStop(t *testing.T) {
	tmpDir := t.TempDir()

	records := NewRecords()
	fw := NewFileWatcher(tmpDir, "192.168.1.10", records)

	// Start
	if err := fw.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Start again should be no-op
	if err := fw.Start(context.Background()); err != nil {
		t.Fatalf("Second Start failed: %v", err)
	}

	// Stop
	fw.Stop()

	// Stop again should be safe
	fw.Stop()
}

func TestFileWatcher_DetectsFileChanges(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial config
	config := `
http:
  routers:
    web:
      rule: "Host(` + "`" + `web.example.com` + "`" + `)"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "web.yml"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	records := NewRecords()
	fw := NewFileWatcher(tmpDir, "192.168.1.10", records)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := fw.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer fw.Stop()

	// Wait for initial load
	time.Sleep(100 * time.Millisecond)

	if records.Count() != 1 {
		t.Errorf("expected 1 record after initial load, got %d", records.Count())
	}

	// Add a new file
	newConfig := `
http:
  routers:
    api:
      rule: "Host(` + "`" + `api.example.com` + "`" + `)"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "api.yml"), []byte(newConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for debounce + reload
	time.Sleep(800 * time.Millisecond)

	if records.Count() != 2 {
		t.Errorf("expected 2 records after adding file, got %d", records.Count())
	}
}

func TestFileWatcher_GetDirectory(t *testing.T) {
	records := NewRecords()
	fw := NewFileWatcher("/etc/traefik/external-enabled", "192.168.1.10", records)

	if fw.GetDirectory() != "/etc/traefik/external-enabled" {
		t.Errorf("unexpected directory: %s", fw.GetDirectory())
	}
}

func TestFileWatcher_HandlesEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	records := NewRecords()
	fw := NewFileWatcher(tmpDir, "192.168.1.10", records)

	if err := fw.loadAllConfigs(); err != nil {
		t.Fatalf("loadAllConfigs failed on empty dir: %v", err)
	}

	if records.Count() != 0 {
		t.Errorf("expected 0 records for empty directory, got %d", records.Count())
	}
}

func TestFileWatcher_HandlesMissingEnvVars(t *testing.T) {
	tmpDir := t.TempDir()

	// Config with unresolvable env var
	config := `
http:
  routers:
    web:
      rule: "Host(` + "`" + `{{env "UNDEFINED_VAR"}}.example.com` + "`" + `)"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "web.yml"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	records := NewRecords()
	fw := NewFileWatcher(tmpDir, "192.168.1.10", records)

	// Should not error, just skip the unresolvable host
	if err := fw.loadAllConfigs(); err != nil {
		t.Fatalf("loadAllConfigs failed: %v", err)
	}

	if records.Count() != 0 {
		t.Errorf("expected 0 records (unresolvable env var), got %d", records.Count())
	}
}
