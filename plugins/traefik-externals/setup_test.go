package traefikexternals

import (
	"os"
	"testing"

	"github.com/coredns/caddy"
)

func TestSetup_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()

	input := `traefik-externals {
		directory ` + tmpDir + `
		host_ip 192.168.1.100
		ttl 120
		fallthrough
	}`

	c := caddy.NewTestController("dns", input)
	te, err := parseConfig(c)

	if err != nil {
		t.Fatalf("parseConfig failed: %v", err)
	}

	if te.Watcher.directory != tmpDir {
		t.Errorf("expected directory %s, got %s", tmpDir, te.Watcher.directory)
	}
	if te.Watcher.hostIP != "192.168.1.100" {
		t.Errorf("expected host_ip 192.168.1.100, got %s", te.Watcher.hostIP)
	}
	if te.TTL != 120 {
		t.Errorf("expected TTL 120, got %d", te.TTL)
	}
}

func TestSetup_InvalidIP(t *testing.T) {
	tmpDir := t.TempDir()

	input := `traefik-externals {
		directory ` + tmpDir + `
		host_ip notanip
	}`

	c := caddy.NewTestController("dns", input)
	_, err := parseConfig(c)

	if err == nil {
		t.Error("expected error for invalid IP, got nil")
	}
}

func TestSetup_MissingHostIP(t *testing.T) {
	tmpDir := t.TempDir()

	input := `traefik-externals {
		directory ` + tmpDir + `
	}`

	c := caddy.NewTestController("dns", input)
	_, err := parseConfig(c)

	if err == nil {
		t.Error("expected error for missing host_ip, got nil")
	}
}

func TestSetup_InvalidTTL(t *testing.T) {
	tmpDir := t.TempDir()

	input := `traefik-externals {
		directory ` + tmpDir + `
		host_ip 192.168.1.100
		ttl notanumber
	}`

	c := caddy.NewTestController("dns", input)
	_, err := parseConfig(c)

	if err == nil {
		t.Error("expected error for invalid TTL, got nil")
	}
}

func TestSetup_UnknownProperty(t *testing.T) {
	tmpDir := t.TempDir()

	input := `traefik-externals {
		directory ` + tmpDir + `
		host_ip 192.168.1.100
		unknown_property value
	}`

	c := caddy.NewTestController("dns", input)
	_, err := parseConfig(c)

	if err == nil {
		t.Error("expected error for unknown property, got nil")
	}
}

func TestSetup_EnvVarOverrides(t *testing.T) {
	tmpDir := t.TempDir()

	// Set environment variables
	os.Setenv("HOSTIP", "10.0.0.1")
	os.Setenv("TRAEFIK_EXTERNALS_DIRECTORY", tmpDir)
	os.Setenv("TRAEFIK_EXTERNALS_TTL", "300")
	defer func() {
		os.Unsetenv("HOSTIP")
		os.Unsetenv("TRAEFIK_EXTERNALS_DIRECTORY")
		os.Unsetenv("TRAEFIK_EXTERNALS_TTL")
	}()

	input := `traefik-externals {
		host_ip 192.168.1.100
		ttl 60
	}`

	c := caddy.NewTestController("dns", input)
	te, err := parseConfig(c)

	if err != nil {
		t.Fatalf("parseConfig failed: %v", err)
	}

	// Environment variables should override config
	if te.Watcher.hostIP != "10.0.0.1" {
		t.Errorf("expected HOSTIP env override, got %s", te.Watcher.hostIP)
	}
	if te.Watcher.directory != tmpDir {
		t.Errorf("expected TRAEFIK_EXTERNALS_DIRECTORY env override, got %s", te.Watcher.directory)
	}
	if te.TTL != 300 {
		t.Errorf("expected TRAEFIK_EXTERNALS_TTL env override, got %d", te.TTL)
	}
}

func TestSetup_DefaultValues(t *testing.T) {
	os.Setenv("HOSTIP", "192.168.1.100")
	defer os.Unsetenv("HOSTIP")

	input := `traefik-externals`

	c := caddy.NewTestController("dns", input)
	te, err := parseConfig(c)

	if err != nil {
		t.Fatalf("parseConfig failed: %v", err)
	}

	// Check default values
	if te.Watcher.directory != "/etc/traefik/external-enabled" {
		t.Errorf("expected default directory, got %s", te.Watcher.directory)
	}
	if te.TTL != 60 {
		t.Errorf("expected default TTL 60, got %d", te.TTL)
	}
}

func TestIsDisabled(t *testing.T) {
	tests := []struct {
		envValue string
		expected bool
	}{
		{"", false},          // Empty = enabled
		{"true", false},      // true = enabled
		{"yes", false},       // yes = enabled
		{"1", false},         // 1 = enabled
		{"false", true},      // false = disabled
		{"no", true},         // no = disabled
		{"0", true},          // 0 = disabled
		{"FALSE", true},      // Case insensitive
		{"False", true},      // Case insensitive
		{"NO", true},         // Case insensitive
		{"random", false},    // Unknown value = enabled
	}

	for _, tc := range tests {
		os.Setenv("TRAEFIK_EXTERNALS_ENABLED", tc.envValue)
		result := isDisabled()
		os.Unsetenv("TRAEFIK_EXTERNALS_ENABLED")

		if result != tc.expected {
			t.Errorf("isDisabled() with TRAEFIK_EXTERNALS_ENABLED=%q: expected %v, got %v",
				tc.envValue, tc.expected, result)
		}
	}
}

// Regression test for GitHub issue: nil pointer panic when plugin disabled
// When the plugin is disabled (via env var or missing directory), setup() must
// return nil WITHOUT calling AddPlugin(). Previously, it called AddPlugin with
// a pass-through handler that returned `next`, but if `next` was nil, CoreDNS
// would panic during server initialization.
//
// This test verifies setup() returns nil (success) when disabled, which means
// the plugin is gracefully skipped without registering any handler.

func TestSetup_DisabledViaEnvVar_NoError(t *testing.T) {
	// Disable plugin via environment variable
	os.Setenv("TRAEFIK_EXTERNALS_ENABLED", "false")
	os.Setenv("HOSTIP", "192.168.1.100")
	defer func() {
		os.Unsetenv("TRAEFIK_EXTERNALS_ENABLED")
		os.Unsetenv("HOSTIP")
	}()

	input := `traefik-externals`
	c := caddy.NewTestController("dns", input)

	// setup() should return nil (no error) when disabled
	// Previously this would register a nil handler that caused panic
	err := setup(c)
	if err != nil {
		t.Errorf("setup() should return nil when disabled, got: %v", err)
	}
}

func TestSetup_MissingDirectory_NoError(t *testing.T) {
	// Use a directory that definitely doesn't exist
	nonExistentDir := "/this/directory/does/not/exist/ever"

	os.Setenv("HOSTIP", "192.168.1.100")
	defer os.Unsetenv("HOSTIP")

	input := `traefik-externals {
		directory ` + nonExistentDir + `
	}`
	c := caddy.NewTestController("dns", input)

	// setup() should return nil (no error) when directory doesn't exist
	// Previously this would register a nil handler that caused panic
	err := setup(c)
	if err != nil {
		t.Errorf("setup() should return nil for missing directory, got: %v", err)
	}
}

func TestSetup_ValidDirectory_StartsWatcher(t *testing.T) {
	// Create a real temporary directory
	tmpDir := t.TempDir()

	os.Setenv("HOSTIP", "192.168.1.100")
	defer os.Unsetenv("HOSTIP")

	input := `traefik-externals {
		directory ` + tmpDir + `
	}`
	c := caddy.NewTestController("dns", input)

	// setup() should succeed and start the watcher
	err := setup(c)
	if err != nil {
		t.Errorf("setup() should succeed with valid directory, got: %v", err)
	}
}
