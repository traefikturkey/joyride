package dockercluster

import (
	"context"
	"testing"
	"time"
)

func TestNewClusterManagerDisabled(t *testing.T) {
	records := NewRecords()

	// Test nil config
	cm, err := NewClusterManager(nil, records)
	if err != nil {
		t.Errorf("expected no error for nil config, got %v", err)
	}
	if cm != nil {
		t.Error("expected nil ClusterManager for nil config")
	}

	// Test Enabled=false
	config := NewClusterConfig()
	config.Enabled = false

	cm, err = NewClusterManager(config, records)
	if err != nil {
		t.Errorf("expected no error for disabled config, got %v", err)
	}
	if cm != nil {
		t.Error("expected nil ClusterManager for disabled config")
	}
}

func TestNewClusterManagerEnabled(t *testing.T) {
	records := NewRecords()
	config := NewClusterConfig()
	config.Enabled = true
	config.NodeName = "test-node"
	config.Port = 7950

	cm, err := NewClusterManager(config, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cm == nil {
		t.Fatal("expected non-nil ClusterManager")
	}
	if cm.config != config {
		t.Error("expected config to be set")
	}
	if cm.records != records {
		t.Error("expected records to be set")
	}
	if cm.delegate == nil {
		t.Error("expected delegate to be created")
	}
}

func TestNewClusterManagerInvalid(t *testing.T) {
	records := NewRecords()

	// Test missing NodeName
	config := NewClusterConfig()
	config.Enabled = true
	config.NodeName = "" // Missing required NodeName

	cm, err := NewClusterManager(config, records)
	if err == nil {
		t.Error("expected error for missing NodeName")
	}
	if cm != nil {
		t.Error("expected nil ClusterManager for invalid config")
	}

	// Test invalid port
	config2 := NewClusterConfig()
	config2.Enabled = true
	config2.NodeName = "test-node"
	config2.Port = -1 // Invalid port

	cm, err = NewClusterManager(config2, records)
	if err == nil {
		t.Error("expected error for invalid port")
	}
	if cm != nil {
		t.Error("expected nil ClusterManager for invalid config")
	}
}

func TestClusterManagerStartStop(t *testing.T) {
	records := NewRecords()
	config := NewClusterConfig()
	config.Enabled = true
	config.NodeName = "test-node-startstop"
	config.Port = 7951
	config.BindAddr = "127.0.0.1"

	cm, err := NewClusterManager(config, records)
	if err != nil {
		t.Fatalf("failed to create ClusterManager: %v", err)
	}

	ctx := context.Background()
	err = cm.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give memberlist time to initialize
	time.Sleep(50 * time.Millisecond)

	// Stop should complete without error
	done := make(chan error)
	go func() {
		done <- cm.Stop()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Stop returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("Stop did not complete within timeout")
	}
}

func TestClusterManagerIsHealthy(t *testing.T) {
	records := NewRecords()
	config := NewClusterConfig()
	config.Enabled = true
	config.NodeName = "test-node-healthy"
	config.Port = 7952
	config.BindAddr = "127.0.0.1"

	cm, err := NewClusterManager(config, records)
	if err != nil {
		t.Fatalf("failed to create ClusterManager: %v", err)
	}

	// Before Start, should not be healthy
	if cm.IsHealthy() {
		t.Error("expected IsHealthy to return false before Start")
	}

	ctx := context.Background()
	err = cm.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer cm.Stop()

	// After Start, should be healthy (at least one member - itself)
	time.Sleep(50 * time.Millisecond)
	if !cm.IsHealthy() {
		t.Error("expected IsHealthy to return true after Start")
	}
}

func TestClusterManagerMembers(t *testing.T) {
	records := NewRecords()
	config := NewClusterConfig()
	config.Enabled = true
	config.NodeName = "test-node-members"
	config.Port = 7953
	config.BindAddr = "127.0.0.1"

	cm, err := NewClusterManager(config, records)
	if err != nil {
		t.Fatalf("failed to create ClusterManager: %v", err)
	}

	// Before Start, Members should return nil
	members := cm.Members()
	if members != nil {
		t.Errorf("expected nil Members before Start, got %d members", len(members))
	}

	ctx := context.Background()
	err = cm.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer cm.Stop()

	// After Start, should have at least one member (itself)
	time.Sleep(50 * time.Millisecond)
	members = cm.Members()
	if members == nil {
		t.Error("expected non-nil Members after Start")
	}
	if len(members) < 1 {
		t.Errorf("expected at least 1 member, got %d", len(members))
	}

	// Verify the node name matches
	foundSelf := false
	for _, m := range members {
		if m.Name == config.NodeName {
			foundSelf = true
			break
		}
	}
	if !foundSelf {
		t.Error("expected to find self in Members list")
	}
}

func TestClusterManagerNotifyRecordAdd(t *testing.T) {
	records := NewRecords()
	config := NewClusterConfig()
	config.Enabled = true
	config.NodeName = "test-node-add"
	config.Port = 7954
	config.BindAddr = "127.0.0.1"

	cm, err := NewClusterManager(config, records)
	if err != nil {
		t.Fatalf("failed to create ClusterManager: %v", err)
	}

	ctx := context.Background()
	err = cm.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer cm.Stop()

	// Notify record add
	timestamp := time.Now().UnixNano()
	cm.NotifyRecordAdd("example.com", "192.168.1.100", timestamp)

	// Record should be applied locally
	ip, found := records.Lookup("example.com")
	if !found {
		t.Error("expected example.com to be added locally")
	}
	if ip != "192.168.1.100" {
		t.Errorf("expected IP 192.168.1.100, got %s", ip)
	}

	// Metadata should be set
	meta, ok := records.GetMeta("example.com")
	if !ok {
		t.Error("expected metadata to be set")
	}
	if meta.Timestamp != timestamp {
		t.Errorf("expected timestamp %d, got %d", timestamp, meta.Timestamp)
	}
	if meta.NodeID != config.NodeName {
		t.Errorf("expected nodeID %s, got %s", config.NodeName, meta.NodeID)
	}
}

func TestClusterManagerNotifyRecordRemove(t *testing.T) {
	records := NewRecords()
	config := NewClusterConfig()
	config.Enabled = true
	config.NodeName = "test-node-remove"
	config.Port = 7955
	config.BindAddr = "127.0.0.1"

	cm, err := NewClusterManager(config, records)
	if err != nil {
		t.Fatalf("failed to create ClusterManager: %v", err)
	}

	ctx := context.Background()
	err = cm.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer cm.Stop()

	// Add a record first
	addTimestamp := time.Now().UnixNano()
	records.AddWithMeta("toremove.com", "10.0.0.1", addTimestamp, config.NodeName)

	// Verify it exists
	_, found := records.Lookup("toremove.com")
	if !found {
		t.Fatal("expected toremove.com to exist before removal")
	}

	// Notify record remove with newer timestamp
	removeTimestamp := addTimestamp + 1000
	cm.NotifyRecordRemove("toremove.com", removeTimestamp)

	// Record should be removed locally
	_, found = records.Lookup("toremove.com")
	if found {
		t.Error("expected toremove.com to be removed")
	}
}

func TestClusterManagerJoinNoSeeds(t *testing.T) {
	records := NewRecords()
	config := NewClusterConfig()
	config.Enabled = true
	config.NodeName = "test-node-noseeds"
	config.Port = 7956
	config.BindAddr = "127.0.0.1"
	config.Seeds = nil // No seeds

	cm, err := NewClusterManager(config, records)
	if err != nil {
		t.Fatalf("failed to create ClusterManager: %v", err)
	}

	ctx := context.Background()
	err = cm.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer cm.Stop()

	// Join with no seeds should return nil (standalone operation)
	err = cm.Join()
	if err != nil {
		t.Errorf("expected Join with no seeds to return nil, got %v", err)
	}
}

func TestClusterManagerDoubleStart(t *testing.T) {
	records := NewRecords()
	config := NewClusterConfig()
	config.Enabled = true
	config.NodeName = "test-node-double"
	config.Port = 7957
	config.BindAddr = "127.0.0.1"

	cm, err := NewClusterManager(config, records)
	if err != nil {
		t.Fatalf("failed to create ClusterManager: %v", err)
	}

	ctx := context.Background()
	err = cm.Start(ctx)
	if err != nil {
		t.Fatalf("first Start failed: %v", err)
	}
	defer cm.Stop()

	// Second Start should fail (port already bound)
	err = cm.Start(ctx)
	if err == nil {
		t.Error("expected second Start to fail")
	}
}

func TestClusterManagerNotifyBeforeStart(t *testing.T) {
	records := NewRecords()
	config := NewClusterConfig()
	config.Enabled = true
	config.NodeName = "test-node-beforestart"
	config.Port = 7958
	config.BindAddr = "127.0.0.1"

	cm, err := NewClusterManager(config, records)
	if err != nil {
		t.Fatalf("failed to create ClusterManager: %v", err)
	}

	// NotifyRecordAdd before Start should not panic
	// The delegate exists but memberlist is nil
	cm.NotifyRecordAdd("test.com", "1.1.1.1", time.Now().UnixNano())

	// Record should still be applied locally via ApplyMessage
	ip, found := records.Lookup("test.com")
	if !found {
		t.Error("expected record to be applied locally even before Start")
	}
	if ip != "1.1.1.1" {
		t.Errorf("expected IP 1.1.1.1, got %s", ip)
	}

	// NotifyRecordRemove before Start should not panic
	cm.NotifyRecordRemove("test.com", time.Now().UnixNano()+1)
}

func TestClusterManagerStopWithoutStart(t *testing.T) {
	records := NewRecords()
	config := NewClusterConfig()
	config.Enabled = true
	config.NodeName = "test-node-nostart"
	config.Port = 7959
	config.BindAddr = "127.0.0.1"

	cm, err := NewClusterManager(config, records)
	if err != nil {
		t.Fatalf("failed to create ClusterManager: %v", err)
	}

	// Stop without Start should not panic and should complete quickly
	done := make(chan error)
	go func() {
		done <- cm.Stop()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Stop without Start returned error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Stop without Start did not complete within timeout")
	}
}

func TestClusterManagerWithSecretKey(t *testing.T) {
	records := NewRecords()
	config := NewClusterConfig()
	config.Enabled = true
	config.NodeName = "test-node-secret"
	config.Port = 7960
	config.BindAddr = "127.0.0.1"
	config.SecretKey = []byte("0123456789abcdef") // 16-byte key for AES-128

	cm, err := NewClusterManager(config, records)
	if err != nil {
		t.Fatalf("failed to create ClusterManager: %v", err)
	}

	ctx := context.Background()
	err = cm.Start(ctx)
	if err != nil {
		t.Fatalf("Start with secret key failed: %v", err)
	}
	defer cm.Stop()

	// Should be healthy with encryption enabled
	time.Sleep(50 * time.Millisecond)
	if !cm.IsHealthy() {
		t.Error("expected cluster to be healthy with encryption")
	}
}

func TestClusterManagerJoinInvalidSeeds(t *testing.T) {
	records := NewRecords()
	config := NewClusterConfig()
	config.Enabled = true
	config.NodeName = "test-node-badseeds"
	config.Port = 7961
	config.BindAddr = "127.0.0.1"
	config.Seeds = []string{"192.0.2.1:7946"} // Invalid/unreachable seed (TEST-NET-1)

	cm, err := NewClusterManager(config, records)
	if err != nil {
		t.Fatalf("failed to create ClusterManager: %v", err)
	}

	ctx := context.Background()
	err = cm.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer cm.Stop()

	// Join with unreachable seeds should return error
	err = cm.Join()
	if err == nil {
		t.Error("expected Join with unreachable seeds to return error")
	}
}

func TestClusterManagerTwoNodeCluster(t *testing.T) {
	// Skip in short mode or CI environments where UDP may not work reliably
	if testing.Short() {
		t.Skip("Skipping two-node cluster test in short mode")
	}

	// Create shared records for both nodes (simulates distributed setup)
	records1 := NewRecords()
	records2 := NewRecords()

	// Configure node1
	config1 := NewClusterConfig()
	config1.Enabled = true
	config1.NodeName = "test-node1-twonode"
	config1.Port = 7970
	config1.BindAddr = "127.0.0.1"

	// Configure node2 - will join node1
	config2 := NewClusterConfig()
	config2.Enabled = true
	config2.NodeName = "test-node2-twonode"
	config2.Port = 7971
	config2.BindAddr = "127.0.0.1"
	config2.Seeds = []string{"127.0.0.1:7970"}

	// Create managers
	cm1, err := NewClusterManager(config1, records1)
	if err != nil {
		t.Fatalf("failed to create ClusterManager 1: %v", err)
	}

	cm2, err := NewClusterManager(config2, records2)
	if err != nil {
		t.Fatalf("failed to create ClusterManager 2: %v", err)
	}

	ctx := context.Background()

	// Start node1 first
	err = cm1.Start(ctx)
	if err != nil {
		t.Fatalf("Node1 Start failed: %v", err)
	}

	// Give node1 time to initialize
	time.Sleep(100 * time.Millisecond)

	// Start node2
	err = cm2.Start(ctx)
	if err != nil {
		cm1.Stop()
		t.Fatalf("Node2 Start failed: %v", err)
	}

	// Node2 joins node1
	err = cm2.Join()
	if err != nil {
		cm2.Stop()
		cm1.Stop()
		t.Fatalf("Node2 failed to join: %v", err)
	}

	// Give time for cluster to stabilize
	time.Sleep(500 * time.Millisecond)

	// Both nodes should see 2 members
	members1 := cm1.Members()
	if len(members1) != 2 {
		t.Errorf("Node1 expected 2 members, got %d", len(members1))
	}

	members2 := cm2.Members()
	if len(members2) != 2 {
		t.Errorf("Node2 expected 2 members, got %d", len(members2))
	}

	// Add record on node1
	timestamp1 := time.Now().UnixNano()
	cm1.NotifyRecordAdd("shared.example.com", "192.168.1.100", timestamp1)

	// Verify record exists on node1
	ip1, found1 := records1.Lookup("shared.example.com")
	if !found1 {
		t.Error("Node1: expected record to exist locally")
	}
	if ip1 != "192.168.1.100" {
		t.Errorf("Node1: expected IP 192.168.1.100, got %s", ip1)
	}

	// Wait for gossip propagation
	time.Sleep(500 * time.Millisecond)

	// Verify record propagated to node2
	ip2, found2 := records2.Lookup("shared.example.com")
	if !found2 {
		t.Error("Node2: expected record to be propagated from Node1")
	}
	if ip2 != "192.168.1.100" {
		t.Errorf("Node2: expected IP 192.168.1.100, got %s", ip2)
	}

	// Add record on node2
	timestamp2 := time.Now().UnixNano()
	cm2.NotifyRecordAdd("node2.example.com", "10.0.0.50", timestamp2)

	// Wait for gossip propagation
	time.Sleep(500 * time.Millisecond)

	// Verify record propagated to node1
	ip1b, found1b := records1.Lookup("node2.example.com")
	if !found1b {
		t.Error("Node1: expected record to be propagated from Node2")
	}
	if ip1b != "10.0.0.50" {
		t.Errorf("Node1: expected IP 10.0.0.50, got %s", ip1b)
	}

	// Both nodes should be healthy
	if !cm1.IsHealthy() {
		t.Error("Node1 should be healthy")
	}
	if !cm2.IsHealthy() {
		t.Error("Node2 should be healthy")
	}

	// Stop both nodes (node2 first, then node1)
	cm2.Stop()
	cm1.Stop()
}
