package dockercluster

import (
	"os"
	"strings"
	"testing"

	"github.com/coredns/caddy"
)

func TestSetupMinimalConfig(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.Watcher == nil {
		t.Error("expected watcher to be created")
	}
	if dc.Records == nil {
		t.Error("expected records to be created")
	}
	if dc.TTL != 60 {
		t.Errorf("expected default TTL 60, got %d", dc.TTL)
	}
}

func TestSetupMissingHostIP(t *testing.T) {
	input := `docker-cluster {
	}`

	c := caddy.NewTestController("dns", input)
	_, err := parseConfig(c)

	if err == nil {
		t.Error("expected error for missing host_ip")
	}
}

func TestSetupWithDockerSocket(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		docker_socket tcp://docker:2375
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.Watcher.dockerSocket != "tcp://docker:2375" {
		t.Errorf("expected tcp://docker:2375, got %s", dc.Watcher.dockerSocket)
	}
}

func TestSetupWithCustomLabels(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		label custom.label.one custom.label.two
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dc.Watcher.labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(dc.Watcher.labels))
	}
	if dc.Watcher.labels[0] != "custom.label.one" {
		t.Errorf("expected custom.label.one, got %s", dc.Watcher.labels[0])
	}
	if dc.Watcher.labels[1] != "custom.label.two" {
		t.Errorf("expected custom.label.two, got %s", dc.Watcher.labels[1])
	}
}

func TestSetupWithDefaultLabels(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dc.Watcher.labels) != 2 {
		t.Fatalf("expected 2 default labels, got %d", len(dc.Watcher.labels))
	}
	if dc.Watcher.labels[0] != "coredns.host.name" {
		t.Errorf("expected coredns.host.name, got %s", dc.Watcher.labels[0])
	}
	if dc.Watcher.labels[1] != "joyride.host.name" {
		t.Errorf("expected joyride.host.name, got %s", dc.Watcher.labels[1])
	}
}

func TestSetupWithTTL(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		ttl 300
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.TTL != 300 {
		t.Errorf("expected TTL 300, got %d", dc.TTL)
	}
}

func TestSetupWithInvalidTTL(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		ttl notanumber
	}`

	c := caddy.NewTestController("dns", input)
	_, err := parseConfig(c)

	if err == nil {
		t.Error("expected error for invalid TTL")
	}
}

func TestSetupWithFallthrough(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		fallthrough
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// fallthrough with no zones means fall through for all
	if !dc.Fall.Through("anything.") {
		t.Error("expected fallthrough to be enabled for all zones")
	}
}

func TestSetupWithUnknownProperty(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		unknown_property value
	}`

	c := caddy.NewTestController("dns", input)
	_, err := parseConfig(c)

	if err == nil {
		t.Error("expected error for unknown property")
	}
}

func TestSetupFullConfig(t *testing.T) {
	input := `docker-cluster {
		docker_socket unix:///custom/docker.sock
		host_ip 10.0.0.1
		label my.custom.label
		ttl 120
		fallthrough
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.Watcher.dockerSocket != "unix:///custom/docker.sock" {
		t.Errorf("wrong docker_socket: %s", dc.Watcher.dockerSocket)
	}
	if dc.Watcher.hostIP != "10.0.0.1" {
		t.Errorf("wrong host_ip: %s", dc.Watcher.hostIP)
	}
	if len(dc.Watcher.labels) != 1 || dc.Watcher.labels[0] != "my.custom.label" {
		t.Errorf("wrong labels: %v", dc.Watcher.labels)
	}
	if dc.TTL != 120 {
		t.Errorf("wrong TTL: %d", dc.TTL)
	}
	if !dc.Fall.Through("test.") {
		t.Error("expected fallthrough to be enabled")
	}
}

func TestSetupWithUnknownActionDrop(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		unknown_action drop
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.UnknownAction != ActionDrop {
		t.Errorf("expected ActionDrop, got %d", dc.UnknownAction)
	}
}

func TestSetupWithUnknownActionNXDomain(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		unknown_action nxdomain
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.UnknownAction != ActionNXDomain {
		t.Errorf("expected ActionNXDomain, got %d", dc.UnknownAction)
	}
}

func TestSetupWithInvalidUnknownAction(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		unknown_action invalid
	}`

	c := caddy.NewTestController("dns", input)
	_, err := parseConfig(c)

	if err == nil {
		t.Error("expected error for invalid unknown_action")
	}
}

func TestSetupDefaultUnknownAction(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Default should be ActionDrop for split DNS setups
	if dc.UnknownAction != ActionDrop {
		t.Errorf("expected default ActionDrop, got %d", dc.UnknownAction)
	}
}

func TestSetupClusterDisabledByDefault(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.ClusterConfig == nil {
		t.Fatal("expected ClusterConfig to be initialized")
	}
	if dc.ClusterConfig.Enabled {
		t.Error("expected cluster to be disabled by default")
	}
	if dc.ClusterConfig.Port != 7946 {
		t.Errorf("expected default cluster port 7946, got %d", dc.ClusterConfig.Port)
	}
	if dc.ClusterConfig.BindAddr != "0.0.0.0" {
		t.Errorf("expected default bind addr 0.0.0.0, got %s", dc.ClusterConfig.BindAddr)
	}
}

func TestSetupWithClusterEnabled(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		cluster_enabled true
		node_name node1
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dc.ClusterConfig.Enabled {
		t.Error("expected cluster to be enabled")
	}
	if dc.ClusterConfig.NodeName != "node1" {
		t.Errorf("expected node_name 'node1', got %s", dc.ClusterConfig.NodeName)
	}
}

func TestSetupWithClusterSeeds(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		cluster_enabled true
		node_name node1
		cluster_seeds 192.168.1.2:7946,192.168.1.3:7946
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dc.ClusterConfig.Seeds) != 2 {
		t.Fatalf("expected 2 seeds, got %d", len(dc.ClusterConfig.Seeds))
	}
	if dc.ClusterConfig.Seeds[0] != "192.168.1.2:7946" {
		t.Errorf("expected seed '192.168.1.2:7946', got %s", dc.ClusterConfig.Seeds[0])
	}
	if dc.ClusterConfig.Seeds[1] != "192.168.1.3:7946" {
		t.Errorf("expected seed '192.168.1.3:7946', got %s", dc.ClusterConfig.Seeds[1])
	}
}

func TestSetupWithClusterPort(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		cluster_enabled true
		node_name node1
		cluster_port 8000
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.ClusterConfig.Port != 8000 {
		t.Errorf("expected cluster port 8000, got %d", dc.ClusterConfig.Port)
	}
}

func TestSetupWithNodeName(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		cluster_enabled true
		node_name my-unique-node
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.ClusterConfig.NodeName != "my-unique-node" {
		t.Errorf("expected node_name 'my-unique-node', got %s", dc.ClusterConfig.NodeName)
	}
}

func TestSetupWithClusterBindAddr(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		cluster_enabled true
		node_name node1
		cluster_bind_addr 10.0.0.1
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.ClusterConfig.BindAddr != "10.0.0.1" {
		t.Errorf("expected cluster_bind_addr '10.0.0.1', got %s", dc.ClusterConfig.BindAddr)
	}
}

func TestSetupWithClusterSecret(t *testing.T) {
	// Use a 16-byte key for AES-128 (memberlist requires valid AES key sizes)
	input := `docker-cluster {
		host_ip 192.168.1.1
		cluster_enabled true
		node_name node1
		cluster_secret mysecretkey12345
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(dc.ClusterConfig.SecretKey) != "mysecretkey12345" {
		t.Errorf("expected cluster_secret 'mysecretkey12345', got %s", string(dc.ClusterConfig.SecretKey))
	}
}

func TestSetupClusterEnabledAutoGeneratesNodeName(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		cluster_enabled true
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.ClusterConfig.NodeName == "" {
		t.Error("expected NodeName to be auto-generated from hostname")
	}
}

func TestSetupWithInvalidClusterPort(t *testing.T) {
	input := `docker-cluster {
		host_ip 192.168.1.1
		cluster_port notanumber
	}`

	c := caddy.NewTestController("dns", input)
	_, err := parseConfig(c)

	if err == nil {
		t.Error("expected error for invalid cluster_port")
	}
}

func TestSetupWithInvalidClusterSecretLength(t *testing.T) {
	// 11 bytes is not a valid AES key size (must be 16, 24, or 32)
	input := `docker-cluster {
		host_ip 192.168.1.1
		cluster_enabled true
		node_name node1
		cluster_secret short_key
	}`

	c := caddy.NewTestController("dns", input)
	_, err := parseConfig(c)

	if err == nil {
		t.Error("expected error for invalid cluster_secret length")
	}
	if err != nil && !strings.Contains(err.Error(), "16, 24, or 32 bytes") {
		t.Errorf("expected AES key size error, got: %v", err)
	}
}

// Environment variable override tests

func TestSetupHOSTIPEnvOverride(t *testing.T) {
	// Set HOSTIP env var
	os.Setenv("HOSTIP", "10.20.30.40")
	defer os.Unsetenv("HOSTIP")

	// Config has a placeholder host_ip that should be overridden
	input := `docker-cluster {
		host_ip 0.0.0.0
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.Watcher.hostIP != "10.20.30.40" {
		t.Errorf("expected HOSTIP env override '10.20.30.40', got %s", dc.Watcher.hostIP)
	}
}

func TestSetupHOSTIPEnvProvidesMissingHostIP(t *testing.T) {
	// Set HOSTIP env var to provide missing host_ip
	os.Setenv("HOSTIP", "192.168.100.1")
	defer os.Unsetenv("HOSTIP")

	// Config has no host_ip - env var should provide it
	input := `docker-cluster {
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.Watcher.hostIP != "192.168.100.1" {
		t.Errorf("expected HOSTIP env '192.168.100.1', got %s", dc.Watcher.hostIP)
	}
}

func TestSetupDOCKER_SOCKETEnvOverride(t *testing.T) {
	os.Setenv("DOCKER_SOCKET", "tcp://remote-docker:2375")
	defer os.Unsetenv("DOCKER_SOCKET")

	input := `docker-cluster {
		host_ip 192.168.1.1
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.Watcher.dockerSocket != "tcp://remote-docker:2375" {
		t.Errorf("expected DOCKER_SOCKET env override, got %s", dc.Watcher.dockerSocket)
	}
}

func TestSetupDNS_UNKNOWN_ACTIONEnvOverride(t *testing.T) {
	os.Setenv("DNS_UNKNOWN_ACTION", "nxdomain")
	defer os.Unsetenv("DNS_UNKNOWN_ACTION")

	input := `docker-cluster {
		host_ip 192.168.1.1
		unknown_action drop
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.UnknownAction != ActionNXDomain {
		t.Errorf("expected DNS_UNKNOWN_ACTION env override to nxdomain, got %d", dc.UnknownAction)
	}
}

func TestSetupCLUSTER_ENABLEDEnvOverride(t *testing.T) {
	os.Setenv("CLUSTER_ENABLED", "true")
	os.Setenv("NODE_NAME", "test-node")
	defer os.Unsetenv("CLUSTER_ENABLED")
	defer os.Unsetenv("NODE_NAME")

	input := `docker-cluster {
		host_ip 192.168.1.1
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dc.ClusterConfig.Enabled {
		t.Error("expected CLUSTER_ENABLED env to enable clustering")
	}
	if dc.ClusterConfig.NodeName != "test-node" {
		t.Errorf("expected NODE_NAME 'test-node', got %s", dc.ClusterConfig.NodeName)
	}
}

func TestSetupCLUSTER_PORTEnvOverride(t *testing.T) {
	os.Setenv("CLUSTER_PORT", "9999")
	defer os.Unsetenv("CLUSTER_PORT")

	input := `docker-cluster {
		host_ip 192.168.1.1
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.ClusterConfig.Port != 9999 {
		t.Errorf("expected CLUSTER_PORT env override 9999, got %d", dc.ClusterConfig.Port)
	}
}

func TestSetupCLUSTER_SEEDSEnvOverride(t *testing.T) {
	os.Setenv("CLUSTER_SEEDS", "10.0.0.1:7946,10.0.0.2:7946")
	defer os.Unsetenv("CLUSTER_SEEDS")

	input := `docker-cluster {
		host_ip 192.168.1.1
	}`

	c := caddy.NewTestController("dns", input)
	dc, err := parseConfig(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dc.ClusterConfig.Seeds) != 2 {
		t.Fatalf("expected 2 seeds from env, got %d", len(dc.ClusterConfig.Seeds))
	}
	if dc.ClusterConfig.Seeds[0] != "10.0.0.1:7946" {
		t.Errorf("expected first seed '10.0.0.1:7946', got %s", dc.ClusterConfig.Seeds[0])
	}
}
