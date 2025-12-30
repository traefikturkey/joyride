package dockercluster

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewClusterDelegate(t *testing.T) {
	records := NewRecords()
	numNodes := func() int { return 3 }

	d := NewClusterDelegate("node1", records, numNodes)

	if d == nil {
		t.Fatal("NewClusterDelegate returned nil")
	}
	if d.nodeID != "node1" {
		t.Errorf("expected nodeID 'node1', got %s", d.nodeID)
	}
	if d.records != records {
		t.Error("expected records to be set")
	}
	if d.broadcasts == nil {
		t.Error("expected broadcasts queue to be initialized")
	}
	if d.msgChan == nil {
		t.Error("expected msgChan to be initialized")
	}
	if cap(d.msgChan) != msgChanBufferSize {
		t.Errorf("expected msgChan buffer size %d, got %d", msgChanBufferSize, cap(d.msgChan))
	}
}

func TestDelegateNodeMeta(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	meta := d.NodeMeta(1024)

	if meta != nil {
		t.Errorf("expected NodeMeta to return nil, got %v", meta)
	}
}

func TestDelegateNotifyMsg(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	msg := &RecordMessage{
		Hostname:  "example.com",
		IP:        "192.168.1.1",
		Action:    RecordActionAdd,
		Timestamp: 1000,
		NodeID:    "node2",
	}
	data, err := msg.Encode()
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	d.NotifyMsg(data)

	// Message should be in the channel
	select {
	case received := <-d.msgChan:
		if received.Hostname != msg.Hostname {
			t.Errorf("expected hostname %s, got %s", msg.Hostname, received.Hostname)
		}
		if received.IP != msg.IP {
			t.Errorf("expected IP %s, got %s", msg.IP, received.IP)
		}
		if received.Action != msg.Action {
			t.Errorf("expected action %d, got %d", msg.Action, received.Action)
		}
		if received.Timestamp != msg.Timestamp {
			t.Errorf("expected timestamp %d, got %d", msg.Timestamp, received.Timestamp)
		}
		if received.NodeID != msg.NodeID {
			t.Errorf("expected nodeID %s, got %s", msg.NodeID, received.NodeID)
		}
	default:
		t.Error("expected message to be queued in msgChan")
	}
}

func TestDelegateNotifyMsgEmptyData(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	// Empty data should not panic or add to channel
	d.NotifyMsg([]byte{})

	select {
	case <-d.msgChan:
		t.Error("expected empty data to not produce a message")
	default:
		// Expected
	}
}

func TestDelegateNotifyMsgInvalidJSON(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	// Invalid JSON should be handled gracefully
	d.NotifyMsg([]byte("not valid json"))

	select {
	case <-d.msgChan:
		t.Error("expected invalid JSON to not produce a message")
	default:
		// Expected - invalid JSON is silently dropped
	}
}

func TestDelegateNotifyMsgChannelFull(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	msg := &RecordMessage{
		Hostname:  "example.com",
		IP:        "192.168.1.1",
		Action:    RecordActionAdd,
		Timestamp: 1000,
		NodeID:    "node2",
	}
	data, err := msg.Encode()
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	// Fill the channel
	for i := 0; i < msgChanBufferSize; i++ {
		d.NotifyMsg(data)
	}

	// This should not block - message is dropped when channel is full
	done := make(chan bool)
	go func() {
		d.NotifyMsg(data)
		done <- true
	}()

	select {
	case <-done:
		// Expected - should not block
	case <-time.After(100 * time.Millisecond):
		t.Error("NotifyMsg blocked when channel was full")
	}
}

func TestDelegateGetBroadcasts(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	// Initially empty
	broadcasts := d.GetBroadcasts(0, 1024)
	if len(broadcasts) != 0 {
		t.Errorf("expected 0 broadcasts initially, got %d", len(broadcasts))
	}

	// Queue a broadcast
	msg := &RecordMessage{
		Hostname:  "example.com",
		IP:        "192.168.1.1",
		Action:    RecordActionAdd,
		Timestamp: 1000,
		NodeID:    "node1",
	}
	d.BroadcastRecord(msg)

	// Should have one broadcast
	broadcasts = d.GetBroadcasts(0, 1024)
	if len(broadcasts) != 1 {
		t.Errorf("expected 1 broadcast, got %d", len(broadcasts))
	}

	// TransmitLimitedQueue retransmits based on RetransmitMult * log(numNodes)
	// With RetransmitMult=3 and 1 node, messages persist for multiple retrievals
	// Just verify we can retrieve broadcasts multiple times (gossip reliability)
	for i := 0; i < 5; i++ {
		d.GetBroadcasts(0, 1024)
	}
	// After enough retrievals, the message should be pruned
	broadcasts = d.GetBroadcasts(0, 1024)
	if len(broadcasts) != 0 {
		t.Errorf("expected 0 broadcasts after multiple retrievals, got %d", len(broadcasts))
	}
}

func TestDelegateLocalState(t *testing.T) {
	records := NewRecords()
	records.AddWithMeta("a.com", "1.1.1.1", 1000, "node1")
	records.AddWithMeta("b.com", "2.2.2.2", 2000, "node2")

	d := NewClusterDelegate("node1", records, func() int { return 1 })

	// Get local state
	data := d.LocalState(false)
	if data == nil {
		t.Fatal("LocalState returned nil")
	}

	// Decode and verify
	state, err := DecodeFullState(data)
	if err != nil {
		t.Fatalf("failed to decode local state: %v", err)
	}

	if state.NodeID != "node1" {
		t.Errorf("expected nodeID 'node1', got %s", state.NodeID)
	}
	if len(state.Records) != 2 {
		t.Errorf("expected 2 records, got %d", len(state.Records))
	}

	entryA := state.Records["a.com"]
	if entryA.IP != "1.1.1.1" {
		t.Errorf("expected IP 1.1.1.1 for a.com, got %s", entryA.IP)
	}
	if entryA.Timestamp != 1000 {
		t.Errorf("expected timestamp 1000 for a.com, got %d", entryA.Timestamp)
	}
	if entryA.NodeID != "node1" {
		t.Errorf("expected nodeID 'node1' for a.com, got %s", entryA.NodeID)
	}

	entryB := state.Records["b.com"]
	if entryB.IP != "2.2.2.2" {
		t.Errorf("expected IP 2.2.2.2 for b.com, got %s", entryB.IP)
	}
	if entryB.Timestamp != 2000 {
		t.Errorf("expected timestamp 2000 for b.com, got %d", entryB.Timestamp)
	}
	if entryB.NodeID != "node2" {
		t.Errorf("expected nodeID 'node2' for b.com, got %s", entryB.NodeID)
	}
}

func TestDelegateLocalStateJoin(t *testing.T) {
	records := NewRecords()
	records.AddWithMeta("test.com", "10.0.0.1", 5000, "node1")

	d := NewClusterDelegate("node1", records, func() int { return 1 })

	// LocalState with join=true should work the same
	data := d.LocalState(true)
	if data == nil {
		t.Fatal("LocalState(true) returned nil")
	}

	state, err := DecodeFullState(data)
	if err != nil {
		t.Fatalf("failed to decode local state: %v", err)
	}

	if len(state.Records) != 1 {
		t.Errorf("expected 1 record, got %d", len(state.Records))
	}
}

func TestDelegateLocalStateEmpty(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	data := d.LocalState(false)
	if data == nil {
		t.Fatal("LocalState returned nil for empty records")
	}

	state, err := DecodeFullState(data)
	if err != nil {
		t.Fatalf("failed to decode empty local state: %v", err)
	}

	if state.NodeID != "node1" {
		t.Errorf("expected nodeID 'node1', got %s", state.NodeID)
	}
	if len(state.Records) != 0 {
		t.Errorf("expected 0 records, got %d", len(state.Records))
	}
}

func TestDelegateMergeRemoteState(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	// Create remote state
	remoteState := &FullState{
		NodeID: "node2",
		Records: map[string]RecordEntry{
			"remote.com": {IP: "10.0.0.1", Timestamp: 1000, NodeID: "node2"},
		},
	}
	data, err := remoteState.Encode()
	if err != nil {
		t.Fatalf("failed to encode remote state: %v", err)
	}

	// Merge remote state
	d.MergeRemoteState(data, false)

	// Verify record was added
	ip, found := records.Lookup("remote.com")
	if !found {
		t.Error("expected remote.com to be added")
	}
	if ip != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %s", ip)
	}

	meta, _ := records.GetMeta("remote.com")
	if meta.Timestamp != 1000 {
		t.Errorf("expected timestamp 1000, got %d", meta.Timestamp)
	}
	if meta.NodeID != "node2" {
		t.Errorf("expected nodeID 'node2', got %s", meta.NodeID)
	}
}

func TestDelegateMergeRemoteStateLWW(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	// Add existing record with newer timestamp
	records.AddWithMeta("existing.com", "1.1.1.1", 2000, "node1")

	// Create remote state with older timestamp
	remoteState := &FullState{
		NodeID: "node2",
		Records: map[string]RecordEntry{
			"existing.com": {IP: "2.2.2.2", Timestamp: 1000, NodeID: "node2"},
			"new.com":      {IP: "3.3.3.3", Timestamp: 3000, NodeID: "node2"},
		},
	}
	data, err := remoteState.Encode()
	if err != nil {
		t.Fatalf("failed to encode remote state: %v", err)
	}

	// Merge remote state
	d.MergeRemoteState(data, false)

	// Existing record should not be overwritten (local is newer)
	ip, _ := records.Lookup("existing.com")
	if ip != "1.1.1.1" {
		t.Errorf("expected existing.com to keep local IP 1.1.1.1, got %s", ip)
	}

	// New record should be added
	ip, found := records.Lookup("new.com")
	if !found {
		t.Error("expected new.com to be added")
	}
	if ip != "3.3.3.3" {
		t.Errorf("expected IP 3.3.3.3 for new.com, got %s", ip)
	}
}

func TestDelegateMergeRemoteStateEmpty(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	// Empty data should not panic
	d.MergeRemoteState([]byte{}, false)
	d.MergeRemoteState(nil, false)

	if records.Count() != 0 {
		t.Error("expected no records after merging empty data")
	}
}

func TestDelegateMergeRemoteStateInvalidJSON(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	// Invalid JSON should not panic
	d.MergeRemoteState([]byte("not valid json"), false)

	if records.Count() != 0 {
		t.Error("expected no records after merging invalid JSON")
	}
}

func TestDelegateMergeRemoteStateJoin(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	remoteState := &FullState{
		NodeID: "node2",
		Records: map[string]RecordEntry{
			"join.com": {IP: "10.0.0.1", Timestamp: 1000, NodeID: "node2"},
		},
	}
	data, _ := remoteState.Encode()

	// MergeRemoteState with join=true should work the same
	d.MergeRemoteState(data, true)

	ip, found := records.Lookup("join.com")
	if !found {
		t.Error("expected join.com to be added on join")
	}
	if ip != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %s", ip)
	}
}

func TestDelegateBroadcastRecord(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	msg := &RecordMessage{
		Hostname:  "broadcast.com",
		IP:        "192.168.1.1",
		Action:    RecordActionAdd,
		Timestamp: 1000,
		NodeID:    "node1",
	}

	d.BroadcastRecord(msg)

	// Should be able to retrieve via GetBroadcasts
	broadcasts := d.GetBroadcasts(0, 1024)
	if len(broadcasts) != 1 {
		t.Fatalf("expected 1 broadcast, got %d", len(broadcasts))
	}

	// Decode and verify
	decoded, err := DecodeRecordMessage(broadcasts[0])
	if err != nil {
		t.Fatalf("failed to decode broadcast: %v", err)
	}
	if decoded.Hostname != msg.Hostname {
		t.Errorf("expected hostname %s, got %s", msg.Hostname, decoded.Hostname)
	}
	if decoded.IP != msg.IP {
		t.Errorf("expected IP %s, got %s", msg.IP, decoded.IP)
	}
	if decoded.Action != msg.Action {
		t.Errorf("expected action %d, got %d", msg.Action, decoded.Action)
	}
}

func TestDelegateBroadcastMultipleRecords(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 3 })

	// Queue multiple broadcasts
	for i := 0; i < 5; i++ {
		msg := &RecordMessage{
			Hostname:  "host" + string(rune('a'+i)) + ".com",
			IP:        "192.168.1." + string(rune('1'+i)),
			Action:    RecordActionAdd,
			Timestamp: int64(1000 + i),
			NodeID:    "node1",
		}
		d.BroadcastRecord(msg)
	}

	// Should retrieve all broadcasts
	broadcasts := d.GetBroadcasts(0, 4096)
	if len(broadcasts) != 5 {
		t.Errorf("expected 5 broadcasts, got %d", len(broadcasts))
	}
}

func TestDelegateStartStop(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	ctx := context.Background()
	d.Start(ctx)

	// Should be running
	if d.ctx == nil {
		t.Error("expected context to be set after Start")
	}
	if d.cancel == nil {
		t.Error("expected cancel to be set after Start")
	}

	// Stop should complete without hanging
	done := make(chan bool)
	go func() {
		d.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Expected
	case <-time.After(1 * time.Second):
		t.Error("Stop did not complete within timeout")
	}
}

func TestDelegateStartStopIdempotent(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	// Stop before Start should not panic
	d.Stop()

	// Start and stop normally
	d.Start(context.Background())
	d.Stop()

	// Stop again should not panic
	d.Stop()
}

func TestDelegateMessageProcessing(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	ctx := context.Background()
	d.Start(ctx)
	defer d.Stop()

	// Send a message via NotifyMsg
	msg := &RecordMessage{
		Hostname:  "processed.com",
		IP:        "192.168.1.100",
		Action:    RecordActionAdd,
		Timestamp: time.Now().UnixNano(),
		NodeID:    "node2",
	}
	data, err := msg.Encode()
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	d.NotifyMsg(data)

	// Wait for message processing
	time.Sleep(50 * time.Millisecond)

	// Record should have been applied
	ip, found := records.Lookup("processed.com")
	if !found {
		t.Error("expected processed.com to be added after message processing")
	}
	if ip != "192.168.1.100" {
		t.Errorf("expected IP 192.168.1.100, got %s", ip)
	}
}

func TestDelegateMessageProcessingMultiple(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	ctx := context.Background()
	d.Start(ctx)
	defer d.Stop()

	// Send multiple messages
	baseTimestamp := time.Now().UnixNano()
	for i := 0; i < 10; i++ {
		msg := &RecordMessage{
			Hostname:  "multi" + string(rune('a'+i)) + ".com",
			IP:        "10.0.0." + string(rune('1'+i)),
			Action:    RecordActionAdd,
			Timestamp: baseTimestamp + int64(i),
			NodeID:    "node2",
		}
		data, _ := msg.Encode()
		d.NotifyMsg(data)
	}

	// Wait for message processing
	time.Sleep(100 * time.Millisecond)

	// All records should be added
	for i := 0; i < 10; i++ {
		hostname := "multi" + string(rune('a'+i)) + ".com"
		_, found := records.Lookup(hostname)
		if !found {
			t.Errorf("expected %s to be added", hostname)
		}
	}
}

func TestDelegateMessageProcessingRemove(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	// Add a record first
	records.AddWithMeta("toremove.com", "1.1.1.1", 1000, "node1")

	ctx := context.Background()
	d.Start(ctx)
	defer d.Stop()

	// Send remove message
	msg := &RecordMessage{
		Hostname:  "toremove.com",
		Action:    RecordActionRemove,
		Timestamp: 2000,
		NodeID:    "node2",
	}
	data, _ := msg.Encode()
	d.NotifyMsg(data)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Record should be removed
	_, found := records.Lookup("toremove.com")
	if found {
		t.Error("expected toremove.com to be removed")
	}
}

func TestDelegateContextCancellation(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 1 })

	ctx, cancel := context.WithCancel(context.Background())
	d.Start(ctx)

	// Cancel context
	cancel()

	// Stop should complete quickly
	done := make(chan bool)
	go func() {
		d.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Expected
	case <-time.After(500 * time.Millisecond):
		t.Error("Stop did not complete after context cancellation")
	}
}

func TestBroadcastImplementation(t *testing.T) {
	data := []byte("test data")
	b := &broadcast{data: data}

	// Test Message
	if string(b.Message()) != string(data) {
		t.Errorf("expected message %s, got %s", data, b.Message())
	}

	// Test Invalidates (always returns false)
	other := &broadcast{data: []byte("other")}
	if b.Invalidates(other) {
		t.Error("expected Invalidates to return false")
	}

	// Test Finished (should not panic)
	b.Finished()
}

func TestDelegateConcurrentOperations(t *testing.T) {
	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 3 })

	ctx := context.Background()
	d.Start(ctx)
	defer d.Stop()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent NotifyMsg calls
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			msg := &RecordMessage{
				Hostname:  "concurrent" + string(rune('a'+(i%26))) + ".com",
				IP:        "192.168.1.1",
				Action:    RecordActionAdd,
				Timestamp: int64(i),
				NodeID:    "node2",
			}
			data, _ := msg.Encode()
			d.NotifyMsg(data)
		}
	}()

	// Concurrent BroadcastRecord calls
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			msg := &RecordMessage{
				Hostname:  "broadcast" + string(rune('a'+(i%26))) + ".com",
				IP:        "10.0.0.1",
				Action:    RecordActionAdd,
				Timestamp: int64(i),
				NodeID:    "node1",
			}
			d.BroadcastRecord(msg)
		}
	}()

	// Concurrent GetBroadcasts calls
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			d.GetBroadcasts(0, 1024)
		}
	}()

	// Concurrent LocalState calls
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			d.LocalState(i%2 == 0)
		}
	}()

	// Concurrent MergeRemoteState calls
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			state := &FullState{
				NodeID: "remote",
				Records: map[string]RecordEntry{
					"merge.com": {IP: "5.5.5.5", Timestamp: int64(i), NodeID: "remote"},
				},
			}
			data, _ := state.Encode()
			d.MergeRemoteState(data, i%2 == 0)
		}
	}()

	// Concurrent NodeMeta calls
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			d.NodeMeta(1024)
		}
	}()

	wg.Wait()
}

func TestDelegateConcurrentBroadcasts(t *testing.T) {
	t.Parallel()

	records := NewRecords()
	d := NewClusterDelegate("node1", records, func() int { return 5 })

	ctx := context.Background()
	d.Start(ctx)
	defer d.Stop()

	var wg sync.WaitGroup
	goroutines := 10
	broadcastsPerGoroutine := 10 // 10 * 10 = 100 concurrent broadcasts

	// Launch concurrent BroadcastRecord calls
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < broadcastsPerGoroutine; i++ {
				msg := &RecordMessage{
					Hostname:  "stress-" + string(rune('a'+id)) + "-" + string(rune('0'+i)) + ".com",
					IP:        "192.168." + string(rune('0'+id)) + "." + string(rune('0'+i)),
					Action:    RecordActionAdd,
					Timestamp: time.Now().UnixNano(),
					NodeID:    "node1",
				}
				d.BroadcastRecord(msg)
			}
		}(g)
	}

	wg.Wait()

	// Retrieve all broadcasts - should complete without panic
	// TransmitLimitedQueue handles concurrent access internally
	allBroadcasts := make([][]byte, 0)
	for {
		broadcasts := d.GetBroadcasts(0, 4096)
		if len(broadcasts) == 0 {
			break
		}
		allBroadcasts = append(allBroadcasts, broadcasts...)
	}

	// Verify we received some broadcasts (exact count depends on retransmit logic)
	if len(allBroadcasts) == 0 {
		t.Error("expected to receive at least some broadcasts")
	}

	// Decode a few to verify they're valid
	decodedCount := 0
	for _, data := range allBroadcasts {
		msg, err := DecodeRecordMessage(data)
		if err != nil {
			t.Errorf("failed to decode broadcast: %v", err)
			continue
		}
		if msg.Action != RecordActionAdd {
			t.Errorf("expected action Add, got %d", msg.Action)
		}
		decodedCount++
	}

	if decodedCount == 0 {
		t.Error("expected at least one broadcast to decode successfully")
	}
}

func TestDelegateLargeStateSync(t *testing.T) {
	t.Parallel()

	records := NewRecords()
	recordCount := 150 // 100+ records as specified

	// Pre-populate records store with many records
	for i := 0; i < recordCount; i++ {
		hostname := "large-" + string(rune('a'+i/26)) + string(rune('a'+i%26)) + ".example.com"
		ip := "10." + string(rune('0'+i/100)) + "." + string(rune('0'+(i/10)%10)) + "." + string(rune('0'+i%10))
		timestamp := int64(1000 + i)
		nodeID := "node-" + string(rune('a'+i%3))
		records.AddWithMeta(hostname, ip, timestamp, nodeID)
	}

	d := NewClusterDelegate("node1", records, func() int { return 3 })

	// Get local state (serializes all records)
	localState := d.LocalState(false)
	if localState == nil {
		t.Fatal("LocalState returned nil")
	}

	// Verify the state can be decoded
	state, err := DecodeFullState(localState)
	if err != nil {
		t.Fatalf("failed to decode local state: %v", err)
	}

	if len(state.Records) != recordCount {
		t.Errorf("expected %d records in state, got %d", recordCount, len(state.Records))
	}

	// Create a new delegate to merge this state into
	records2 := NewRecords()
	d2 := NewClusterDelegate("node2", records2, func() int { return 3 })

	// Merge the large state
	d2.MergeRemoteState(localState, false)

	// Verify all records were merged
	if records2.Count() != recordCount {
		t.Errorf("expected %d records after merge, got %d", recordCount, records2.Count())
	}

	// Spot-check a few records
	for i := 0; i < recordCount; i += 25 {
		hostname := "large-" + string(rune('a'+i/26)) + string(rune('a'+i%26)) + ".example.com"
		ip, found := records2.Lookup(hostname)
		if !found {
			t.Errorf("expected to find %s after merge", hostname)
			continue
		}

		expectedIP := "10." + string(rune('0'+i/100)) + "." + string(rune('0'+(i/10)%10)) + "." + string(rune('0'+i%10))
		if ip != expectedIP {
			t.Errorf("expected IP %s for %s, got %s", expectedIP, hostname, ip)
		}

		meta, ok := records2.GetMeta(hostname)
		if !ok {
			t.Errorf("expected metadata for %s", hostname)
			continue
		}

		expectedTimestamp := int64(1000 + i)
		if meta.Timestamp != expectedTimestamp {
			t.Errorf("expected timestamp %d for %s, got %d", expectedTimestamp, hostname, meta.Timestamp)
		}
	}
}

func TestDelegateLargeStateSyncWithLWW(t *testing.T) {
	t.Parallel()

	// Test that large state sync respects LWW when records already exist
	records := NewRecords()

	// Add some records with high timestamps (these should "win")
	for i := 0; i < 50; i++ {
		hostname := "conflict-" + string(rune('a'+i/26)) + string(rune('a'+i%26)) + ".example.com"
		ip := "192.168.1." + string(rune('0'+i%10))
		timestamp := int64(9999) // High timestamp
		records.AddWithMeta(hostname, ip, timestamp, "local-node")
	}

	d := NewClusterDelegate("local-node", records, func() int { return 2 })

	// Create remote state with lower timestamps for same hostnames
	remoteState := &FullState{
		NodeID:  "remote-node",
		Records: make(map[string]RecordEntry),
	}

	for i := 0; i < 100; i++ {
		hostname := "conflict-" + string(rune('a'+i/26)) + string(rune('a'+i%26)) + ".example.com"
		remoteState.Records[hostname] = RecordEntry{
			IP:        "10.0.0." + string(rune('0'+i%10)),
			Timestamp: int64(1000 + i), // Lower timestamps
			NodeID:    "remote-node",
		}
	}

	data, err := remoteState.Encode()
	if err != nil {
		t.Fatalf("failed to encode remote state: %v", err)
	}

	// Merge remote state
	d.MergeRemoteState(data, false)

	// Verify LWW was respected:
	// - First 50 should keep local values (higher timestamp 9999)
	// - Last 50 should have remote values (new records)

	for i := 0; i < 50; i++ {
		hostname := "conflict-" + string(rune('a'+i/26)) + string(rune('a'+i%26)) + ".example.com"
		ip, found := records.Lookup(hostname)
		if !found {
			t.Errorf("expected to find %s", hostname)
			continue
		}

		expectedIP := "192.168.1." + string(rune('0'+i%10))
		if ip != expectedIP {
			t.Errorf("LWW failed for %s: expected local IP %s, got %s", hostname, expectedIP, ip)
		}

		meta, _ := records.GetMeta(hostname)
		if meta.Timestamp != 9999 {
			t.Errorf("LWW failed for %s: expected local timestamp 9999, got %d", hostname, meta.Timestamp)
		}
	}

	for i := 50; i < 100; i++ {
		hostname := "conflict-" + string(rune('a'+i/26)) + string(rune('a'+i%26)) + ".example.com"
		ip, found := records.Lookup(hostname)
		if !found {
			t.Errorf("expected new record %s to be added", hostname)
			continue
		}

		expectedIP := "10.0.0." + string(rune('0'+i%10))
		if ip != expectedIP {
			t.Errorf("expected remote IP %s for %s, got %s", expectedIP, hostname, ip)
		}
	}

	// Total count should be 100
	if records.Count() != 100 {
		t.Errorf("expected 100 records, got %d", records.Count())
	}
}
