package version

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultVersionValues(t *testing.T) {
	// Test default values when not set by ldflags
	if Version != "dev" {
		t.Errorf("expected default Version to be 'dev', got '%s'", Version)
	}
	if GitCommit != "unknown" {
		t.Errorf("expected default GitCommit to be 'unknown', got '%s'", GitCommit)
	}
	if BuildTime != "unknown" {
		t.Errorf("expected default BuildTime to be 'unknown', got '%s'", BuildTime)
	}
}

func TestGetInfo(t *testing.T) {
	info := GetInfo()
	if info.Version != Version {
		t.Errorf("expected info.Version to match Version variable, got '%s' want '%s'", info.Version, Version)
	}
	if info.GitCommit != GitCommit {
		t.Errorf("expected info.GitCommit to match GitCommit variable, got '%s' want '%s'", info.GitCommit, GitCommit)
	}
	if info.BuildTime != BuildTime {
		t.Errorf("expected info.BuildTime to match BuildTime variable, got '%s' want '%s'", info.BuildTime, BuildTime)
	}
}

func TestInfoJSONSerialization(t *testing.T) {
	info := GetInfo()
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal Info to JSON: %v", err)
	}

	var decoded Info
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal Info from JSON: %v", err)
	}

	if decoded.Version != info.Version {
		t.Errorf("JSON round-trip failed for Version: got '%s' want '%s'", decoded.Version, info.Version)
	}
	if decoded.GitCommit != info.GitCommit {
		t.Errorf("JSON round-trip failed for GitCommit: got '%s' want '%s'", decoded.GitCommit, info.GitCommit)
	}
	if decoded.BuildTime != info.BuildTime {
		t.Errorf("JSON round-trip failed for BuildTime: got '%s' want '%s'", decoded.BuildTime, info.BuildTime)
	}
}

func TestVersionHandler_GET(t *testing.T) {
	// Create a request to the version endpoint
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	// Create handler and serve
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GetInfo())
	})
	handler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	// Verify JSON response
	var info Info
	if err := json.NewDecoder(w.Body).Decode(&info); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if info.Version != Version {
		t.Errorf("expected version %s, got %s", Version, info.Version)
	}
}

func TestVersionHandler_MethodNotAllowed(t *testing.T) {
	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GetInfo())
	})

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/version", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}
