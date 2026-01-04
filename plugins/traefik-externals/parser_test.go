package traefikexternals

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParser_ParseContent_SimpleHost(t *testing.T) {
	p := NewParser()

	content := `
http:
  routers:
    homeassistant:
      rule: "Host(` + "`" + `hass.example.com` + "`" + `)"
      service: homeassistant
`
	hosts := p.ParseContent(content)

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0] != "hass.example.com" {
		t.Errorf("expected hass.example.com, got %s", hosts[0])
	}
}

func TestParser_ParseContent_EnvTemplate(t *testing.T) {
	p := NewParserWithEnv(map[string]string{
		"HOMEASSISTANT_HOST_NAME": "hass",
		"HOST_DOMAIN":             "example.com",
	})

	content := `
http:
  routers:
    homeassistant:
      rule: "Host(` + "`" + `{{env "HOMEASSISTANT_HOST_NAME"}}.{{env "HOST_DOMAIN"}}` + "`" + `)"
      service: homeassistant
`
	hosts := p.ParseContent(content)

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0] != "hass.example.com" {
		t.Errorf("expected hass.example.com, got %s", hosts[0])
	}
}

func TestParser_ParseContent_MissingEnvVar(t *testing.T) {
	p := NewParser()
	// Don't set MISSING_VAR

	content := `
http:
  routers:
    test:
      rule: "Host(` + "`" + `{{env "MISSING_VAR"}}.example.com` + "`" + `)"
`
	hosts := p.ParseContent(content)

	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts for missing env var, got %d", len(hosts))
	}
}

func TestParser_ParseContent_MultipleHosts(t *testing.T) {
	p := NewParserWithEnv(map[string]string{
		"WEB_HOST":    "web",
		"API_HOST":    "api",
		"HOST_DOMAIN": "example.com",
	})

	content := `
http:
  routers:
    web:
      rule: "Host(` + "`" + `{{env "WEB_HOST"}}.{{env "HOST_DOMAIN"}}` + "`" + `)"
    api:
      rule: "Host(` + "`" + `{{env "API_HOST"}}.{{env "HOST_DOMAIN"}}` + "`" + `)"
`
	hosts := p.ParseContent(content)

	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}

	hostMap := make(map[string]bool)
	for _, h := range hosts {
		hostMap[h] = true
	}

	if !hostMap["web.example.com"] {
		t.Error("expected web.example.com in hosts")
	}
	if !hostMap["api.example.com"] {
		t.Error("expected api.example.com in hosts")
	}
}

func TestParser_ParseContent_HostWithPathPrefix(t *testing.T) {
	p := NewParser()

	content := `
http:
  routers:
    api:
      rule: "Host(` + "`" + `api.example.com` + "`" + `) && PathPrefix(` + "`" + `/v1` + "`" + `)"
`
	hosts := p.ParseContent(content)

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0] != "api.example.com" {
		t.Errorf("expected api.example.com, got %s", hosts[0])
	}
}

func TestParser_ParseContent_DeduplicatesHosts(t *testing.T) {
	p := NewParser()

	content := `
http:
  routers:
    web1:
      rule: "Host(` + "`" + `example.com` + "`" + `)"
    web2:
      rule: "Host(` + "`" + `example.com` + "`" + `)"
`
	hosts := p.ParseContent(content)

	if len(hosts) != 1 {
		t.Errorf("expected 1 deduplicated host, got %d", len(hosts))
	}
}

func TestParser_ParseContent_NormalizesToLowercase(t *testing.T) {
	p := NewParser()

	content := `
http:
  routers:
    web:
      rule: "Host(` + "`" + `EXAMPLE.COM` + "`" + `)"
`
	hosts := p.ParseContent(content)

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0] != "example.com" {
		t.Errorf("expected lowercase example.com, got %s", hosts[0])
	}
}

func TestParser_IsMiddlewareOnly(t *testing.T) {
	p := NewParser()

	middlewareFiles := []string{
		"middleware.yml",
		"authentik_middleware.yml",
		"authelia_middlewares.yml",
		"crowdsec-bouncer.yml",
	}

	for _, f := range middlewareFiles {
		if !p.IsMiddlewareOnly(f) {
			t.Errorf("expected %s to be middleware-only", f)
		}
	}

	nonMiddlewareFiles := []string{
		"homeassistant.yml",
		"proxmox.yml",
		"service.yml",
	}

	for _, f := range nonMiddlewareFiles {
		if p.IsMiddlewareOnly(f) {
			t.Errorf("expected %s to NOT be middleware-only", f)
		}
	}
}

func TestParser_ParseFile(t *testing.T) {
	p := NewParserWithEnv(map[string]string{
		"TEST_HOST":   "test",
		"HOST_DOMAIN": "example.com",
	})

	// Create temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.yml")

	content := `
http:
  routers:
    test:
      rule: "Host(` + "`" + `{{env "TEST_HOST"}}.{{env "HOST_DOMAIN"}}` + "`" + `)"
      service: test
  services:
    test:
      loadBalancer:
        servers:
          - url: "http://192.168.1.100:8080/"
`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	hosts, err := p.ParseFile(filePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0] != "test.example.com" {
		t.Errorf("expected test.example.com, got %s", hosts[0])
	}
}

func TestParser_ImmutableEnvVars(t *testing.T) {
	// Test that the envVars map is truly copied and immutable
	originalEnv := map[string]string{
		"TEST_KEY": "original",
	}

	p := NewParserWithEnv(originalEnv)

	// Modify the original map
	originalEnv["TEST_KEY"] = "modified"
	originalEnv["NEW_KEY"] = "new"

	// Parser should still have the original values
	val, ok := p.GetEnvVar("TEST_KEY")
	if !ok || val != "original" {
		t.Errorf("expected TEST_KEY=original, got %s", val)
	}

	_, ok = p.GetEnvVar("NEW_KEY")
	if ok {
		t.Error("NEW_KEY should not exist in parser")
	}
}

func TestParser_ParseContent_MultiHostSyntax(t *testing.T) {
	p := NewParser()

	// Test Traefik multi-host syntax: Host(`a.com`, `b.com`)
	content := `
http:
  routers:
    multi:
      rule: "Host(` + "`" + `app.example.com` + "`" + `, ` + "`" + `www.example.com` + "`" + `)"
`
	hosts := p.ParseContent(content)

	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts from multi-host syntax, got %d", len(hosts))
	}

	hostMap := make(map[string]bool)
	for _, h := range hosts {
		hostMap[h] = true
	}

	if !hostMap["app.example.com"] {
		t.Error("expected app.example.com in hosts")
	}
	if !hostMap["www.example.com"] {
		t.Error("expected www.example.com in hosts")
	}
}

func TestParser_ParseContent_HostSNI(t *testing.T) {
	p := NewParser()

	// Test HostSNI for TCP/TLS services
	content := `
tcp:
  routers:
    db:
      rule: "HostSNI(` + "`" + `db.example.com` + "`" + `)"
      service: postgres
      tls: {}
`
	hosts := p.ParseContent(content)

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host from HostSNI, got %d", len(hosts))
	}
	if hosts[0] != "db.example.com" {
		t.Errorf("expected db.example.com, got %s", hosts[0])
	}
}

func TestParser_ParseContent_SkipsComments(t *testing.T) {
	p := NewParser()

	content := `
http:
  routers:
    # This is commented out
    # old:
    #   rule: "Host(` + "`" + `old.example.com` + "`" + `)"
    active:
      rule: "Host(` + "`" + `active.example.com` + "`" + `)"
`
	hosts := p.ParseContent(content)

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host (comments skipped), got %d", len(hosts))
	}
	if hosts[0] != "active.example.com" {
		t.Errorf("expected active.example.com, got %s", hosts[0])
	}
}

func TestParser_ParseContent_SkipsInlineComments(t *testing.T) {
	p := NewParser()

	content := `
http:
  routers:
    web:
      rule: "Host(` + "`" + `web.example.com` + "`" + `)" # inline comment with Host(` + "`" + `fake.example.com` + "`" + `)
`
	hosts := p.ParseContent(content)

	if len(hosts) != 1 {
		t.Fatalf("expected 1 host (inline comment skipped), got %d", len(hosts))
	}
	if hosts[0] != "web.example.com" {
		t.Errorf("expected web.example.com, got %s", hosts[0])
	}
}

func TestParser_ParseFile_LargeFileRejected(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "large.yml")

	// Create a file larger than maxFileSize (1MB)
	largeContent := make([]byte, maxFileSize+1)
	for i := range largeContent {
		largeContent[i] = 'a'
	}

	if err := os.WriteFile(filePath, largeContent, 0644); err != nil {
		t.Fatalf("failed to write large file: %v", err)
	}

	p := NewParser()
	_, err := p.ParseFile(filePath)

	if err == nil {
		t.Error("expected error for large file, got nil")
	}
}

func TestParser_ParseContent_CombinedHostAndHostSNI(t *testing.T) {
	p := NewParser()

	// Test file with both HTTP Host() and TCP HostSNI()
	content := `
http:
  routers:
    web:
      rule: "Host(` + "`" + `web.example.com` + "`" + `)"

tcp:
  routers:
    db:
      rule: "HostSNI(` + "`" + `db.example.com` + "`" + `)"
`
	hosts := p.ParseContent(content)

	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}

	hostMap := make(map[string]bool)
	for _, h := range hosts {
		hostMap[h] = true
	}

	if !hostMap["web.example.com"] {
		t.Error("expected web.example.com from Host()")
	}
	if !hostMap["db.example.com"] {
		t.Error("expected db.example.com from HostSNI()")
	}
}
