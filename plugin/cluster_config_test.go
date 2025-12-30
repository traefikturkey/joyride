package dockercluster

import (
	"testing"
)

func TestClusterConfigDefaults(t *testing.T) {
	cfg := NewClusterConfig()

	if cfg.Enabled {
		t.Error("expected Enabled to be false by default")
	}

	if cfg.Port != 7946 {
		t.Errorf("expected Port to be 7946, got %d", cfg.Port)
	}

	if cfg.Seeds != nil {
		t.Errorf("expected Seeds to be nil, got %v", cfg.Seeds)
	}

	if cfg.NodeName != "" {
		t.Errorf("expected NodeName to be empty, got %q", cfg.NodeName)
	}

	if cfg.BindAddr != "0.0.0.0" {
		t.Errorf("expected BindAddr to be '0.0.0.0', got %q", cfg.BindAddr)
	}

	if cfg.SecretKey != nil {
		t.Error("expected SecretKey to be nil by default")
	}
}

func TestClusterConfigValidateEnabled(t *testing.T) {
	cfg := NewClusterConfig()
	cfg.Enabled = true
	cfg.NodeName = "node1"

	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected no error for valid enabled config, got %v", err)
	}

	// Test with seeds
	cfg.Seeds = []string{"192.168.1.1:7946", "192.168.1.2:7946"}
	err = cfg.Validate()
	if err != nil {
		t.Errorf("expected no error for valid enabled config with seeds, got %v", err)
	}
}

func TestClusterConfigValidateDisabled(t *testing.T) {
	cfg := NewClusterConfig()
	cfg.Enabled = false

	// Should pass validation even without NodeName when disabled
	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected no error for disabled config, got %v", err)
	}

	// Should still pass with empty NodeName
	cfg.NodeName = ""
	err = cfg.Validate()
	if err != nil {
		t.Errorf("expected no error for disabled config with empty NodeName, got %v", err)
	}
}

func TestClusterConfigValidateInvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero port", 0},
		{"negative port", -1},
		{"port too high", 65536},
		{"port way too high", 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewClusterConfig()
			cfg.Port = tt.port

			err := cfg.Validate()
			if err == nil {
				t.Errorf("expected error for port %d, got nil", tt.port)
			}
		})
	}

	// Test valid edge cases
	validPorts := []int{1, 7946, 65535}
	for _, port := range validPorts {
		cfg := NewClusterConfig()
		cfg.Port = port

		err := cfg.Validate()
		if err != nil {
			t.Errorf("expected no error for valid port %d, got %v", port, err)
		}
	}
}

func TestClusterConfigValidateMissingNodeName(t *testing.T) {
	cfg := NewClusterConfig()
	cfg.Enabled = true
	cfg.NodeName = ""

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for enabled config with missing NodeName")
	}

	// Verify the error message mentions node_name
	expectedSubstr := "node_name"
	if err != nil && !containsSubstring(err.Error(), expectedSubstr) {
		t.Errorf("expected error message to contain %q, got %q", expectedSubstr, err.Error())
	}
}

// containsSubstring checks if s contains substr.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
