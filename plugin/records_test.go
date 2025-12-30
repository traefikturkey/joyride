package dockercluster

import (
	"sync"
	"testing"
)

func TestNewRecords(t *testing.T) {
	r := NewRecords()
	if r == nil {
		t.Fatal("NewRecords returned nil")
	}
	if r.Count() != 0 {
		t.Errorf("expected 0 records, got %d", r.Count())
	}
}

func TestRecordsAdd(t *testing.T) {
	r := NewRecords()

	r.Add("example.com", "192.168.1.1")
	ip, found := r.Lookup("example.com")
	if !found {
		t.Error("expected to find example.com")
	}
	if ip != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", ip)
	}
}

func TestRecordsAddNormalizesHostname(t *testing.T) {
	r := NewRecords()

	r.Add("EXAMPLE.COM", "192.168.1.1")
	ip, found := r.Lookup("example.com")
	if !found {
		t.Error("expected to find example.com (lowercase)")
	}
	if ip != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", ip)
	}
}

func TestRecordsLookupNormalizesHostname(t *testing.T) {
	r := NewRecords()

	r.Add("example.com", "192.168.1.1")
	ip, found := r.Lookup("EXAMPLE.COM")
	if !found {
		t.Error("expected to find EXAMPLE.COM")
	}
	if ip != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", ip)
	}
}

func TestRecordsRemove(t *testing.T) {
	r := NewRecords()

	r.Add("example.com", "192.168.1.1")
	r.Remove("example.com")

	_, found := r.Lookup("example.com")
	if found {
		t.Error("expected example.com to be removed")
	}
}

func TestRecordsRemoveNormalizesHostname(t *testing.T) {
	r := NewRecords()

	r.Add("example.com", "192.168.1.1")
	r.Remove("EXAMPLE.COM")

	_, found := r.Lookup("example.com")
	if found {
		t.Error("expected example.com to be removed via uppercase key")
	}
}

func TestRecordsRemoveNonExistent(t *testing.T) {
	r := NewRecords()

	// Should not panic
	r.Remove("nonexistent.com")
}

func TestRecordsUpdate(t *testing.T) {
	r := NewRecords()

	r.Add("example.com", "192.168.1.1")
	r.Add("example.com", "192.168.1.2")

	ip, found := r.Lookup("example.com")
	if !found {
		t.Error("expected to find example.com")
	}
	if ip != "192.168.1.2" {
		t.Errorf("expected 192.168.1.2, got %s", ip)
	}
}

func TestRecordsGetAll(t *testing.T) {
	r := NewRecords()

	r.Add("a.com", "1.1.1.1")
	r.Add("b.com", "2.2.2.2")

	all := r.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2 records, got %d", len(all))
	}
	if all["a.com"] != "1.1.1.1" {
		t.Errorf("expected 1.1.1.1 for a.com, got %s", all["a.com"])
	}
	if all["b.com"] != "2.2.2.2" {
		t.Errorf("expected 2.2.2.2 for b.com, got %s", all["b.com"])
	}
}

func TestRecordsGetAllReturnsCopy(t *testing.T) {
	r := NewRecords()

	r.Add("example.com", "192.168.1.1")
	all := r.GetAll()

	// Modify the returned map
	all["example.com"] = "modified"

	// Original should be unchanged
	ip, _ := r.Lookup("example.com")
	if ip != "192.168.1.1" {
		t.Error("GetAll did not return a copy - original data was modified")
	}
}

func TestRecordsCount(t *testing.T) {
	r := NewRecords()

	if r.Count() != 0 {
		t.Errorf("expected 0, got %d", r.Count())
	}

	r.Add("a.com", "1.1.1.1")
	if r.Count() != 1 {
		t.Errorf("expected 1, got %d", r.Count())
	}

	r.Add("b.com", "2.2.2.2")
	if r.Count() != 2 {
		t.Errorf("expected 2, got %d", r.Count())
	}

	r.Remove("a.com")
	if r.Count() != 1 {
		t.Errorf("expected 1 after removal, got %d", r.Count())
	}
}

func TestRecordsConcurrentAccess(t *testing.T) {
	r := NewRecords()
	var wg sync.WaitGroup
	iterations := 1000

	// Concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				r.Add("test.com", "192.168.1.1")
				r.Remove("test.com")
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				r.Lookup("test.com")
				r.Count()
				r.GetAll()
			}
		}()
	}

	wg.Wait()
}

func BenchmarkRecordsLookup(b *testing.B) {
	r := NewRecords()
	r.Add("example.com", "192.168.1.1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Lookup("example.com")
	}
}

func BenchmarkRecordsAdd(b *testing.B) {
	r := NewRecords()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Add("example.com", "192.168.1.1")
	}
}

// Tests for metadata methods

func TestRecordsAddWithMeta(t *testing.T) {
	r := NewRecords()

	added := r.AddWithMeta("example.com", "192.168.1.1", 1000, "node1")
	if !added {
		t.Error("expected AddWithMeta to return true for new record")
	}

	ip, found := r.Lookup("example.com")
	if !found {
		t.Error("expected to find example.com")
	}
	if ip != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", ip)
	}

	meta, ok := r.GetMeta("example.com")
	if !ok {
		t.Error("expected to find metadata for example.com")
	}
	if meta.Timestamp != 1000 {
		t.Errorf("expected timestamp 1000, got %d", meta.Timestamp)
	}
	if meta.NodeID != "node1" {
		t.Errorf("expected nodeID 'node1', got %s", meta.NodeID)
	}
}

func TestRecordsLWWNewerWins(t *testing.T) {
	r := NewRecords()

	// Add initial record
	r.AddWithMeta("example.com", "192.168.1.1", 1000, "node1")

	// Update with newer timestamp
	added := r.AddWithMeta("example.com", "192.168.1.2", 2000, "node2")
	if !added {
		t.Error("expected newer timestamp to win")
	}

	ip, _ := r.Lookup("example.com")
	if ip != "192.168.1.2" {
		t.Errorf("expected 192.168.1.2, got %s", ip)
	}

	meta, _ := r.GetMeta("example.com")
	if meta.Timestamp != 2000 {
		t.Errorf("expected timestamp 2000, got %d", meta.Timestamp)
	}
	if meta.NodeID != "node2" {
		t.Errorf("expected nodeID 'node2', got %s", meta.NodeID)
	}
}

func TestRecordsLWWOlderRejected(t *testing.T) {
	r := NewRecords()

	// Add initial record with higher timestamp
	r.AddWithMeta("example.com", "192.168.1.1", 2000, "node1")

	// Try to update with older timestamp
	added := r.AddWithMeta("example.com", "192.168.1.2", 1000, "node2")
	if added {
		t.Error("expected older timestamp to be rejected")
	}

	ip, _ := r.Lookup("example.com")
	if ip != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", ip)
	}

	meta, _ := r.GetMeta("example.com")
	if meta.Timestamp != 2000 {
		t.Errorf("expected timestamp 2000, got %d", meta.Timestamp)
	}
	if meta.NodeID != "node1" {
		t.Errorf("expected nodeID 'node1', got %s", meta.NodeID)
	}
}

func TestRecordsLWWSameTimestampTieBreak(t *testing.T) {
	r := NewRecords()

	// Add with lower nodeID
	r.AddWithMeta("example.com", "192.168.1.1", 1000, "node-a")

	// Try to update with same timestamp but lower nodeID - should be rejected
	added := r.AddWithMeta("example.com", "192.168.1.2", 1000, "node-a")
	if added {
		t.Error("expected same timestamp and same nodeID to be rejected")
	}

	// Try with higher nodeID - should win
	added = r.AddWithMeta("example.com", "192.168.1.3", 1000, "node-b")
	if !added {
		t.Error("expected higher nodeID to win with same timestamp")
	}

	ip, _ := r.Lookup("example.com")
	if ip != "192.168.1.3" {
		t.Errorf("expected 192.168.1.3, got %s", ip)
	}

	meta, _ := r.GetMeta("example.com")
	if meta.NodeID != "node-b" {
		t.Errorf("expected nodeID 'node-b', got %s", meta.NodeID)
	}
}

func TestRecordsRemoveWithMeta(t *testing.T) {
	r := NewRecords()

	// Add a record
	r.AddWithMeta("example.com", "192.168.1.1", 1000, "node1")

	// Remove with newer timestamp
	removed := r.RemoveWithMeta("example.com", 2000, "node2")
	if !removed {
		t.Error("expected RemoveWithMeta to return true")
	}

	_, found := r.Lookup("example.com")
	if found {
		t.Error("expected example.com to be removed")
	}

	_, ok := r.GetMeta("example.com")
	if ok {
		t.Error("expected metadata for example.com to be removed")
	}
}

func TestRecordsRemoveWithMetaNonExistent(t *testing.T) {
	r := NewRecords()

	// Try to remove a non-existent record
	removed := r.RemoveWithMeta("nonexistent.com", 1000, "node1")
	if removed {
		t.Error("expected RemoveWithMeta to return false for non-existent record")
	}
}

func TestRecordsRemoveWithMetaLWW(t *testing.T) {
	r := NewRecords()

	// Add a record with newer timestamp
	r.AddWithMeta("example.com", "192.168.1.1", 2000, "node1")

	// Try to remove with older timestamp
	removed := r.RemoveWithMeta("example.com", 1000, "node2")
	if removed {
		t.Error("expected older remove to be rejected")
	}

	ip, found := r.Lookup("example.com")
	if !found {
		t.Error("expected example.com to still exist")
	}
	if ip != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", ip)
	}

	// Remove with same timestamp but lower nodeID should be rejected
	removed = r.RemoveWithMeta("example.com", 2000, "node0")
	if removed {
		t.Error("expected lower nodeID remove to be rejected with same timestamp")
	}

	// Remove with same timestamp but higher nodeID should succeed
	removed = r.RemoveWithMeta("example.com", 2000, "node2")
	if !removed {
		t.Error("expected higher nodeID remove to succeed with same timestamp")
	}

	_, found = r.Lookup("example.com")
	if found {
		t.Error("expected example.com to be removed")
	}
}

func TestRecordsGetAllWithMeta(t *testing.T) {
	r := NewRecords()

	r.AddWithMeta("a.com", "1.1.1.1", 1000, "node1")
	r.AddWithMeta("b.com", "2.2.2.2", 2000, "node2")

	all := r.GetAllWithMeta()
	if len(all) != 2 {
		t.Errorf("expected 2 records, got %d", len(all))
	}

	entryA := all["a.com"]
	if entryA.IP != "1.1.1.1" {
		t.Errorf("expected 1.1.1.1 for a.com, got %s", entryA.IP)
	}
	if entryA.Timestamp != 1000 {
		t.Errorf("expected timestamp 1000 for a.com, got %d", entryA.Timestamp)
	}
	if entryA.NodeID != "node1" {
		t.Errorf("expected nodeID 'node1' for a.com, got %s", entryA.NodeID)
	}

	entryB := all["b.com"]
	if entryB.IP != "2.2.2.2" {
		t.Errorf("expected 2.2.2.2 for b.com, got %s", entryB.IP)
	}
	if entryB.Timestamp != 2000 {
		t.Errorf("expected timestamp 2000 for b.com, got %d", entryB.Timestamp)
	}
	if entryB.NodeID != "node2" {
		t.Errorf("expected nodeID 'node2' for b.com, got %s", entryB.NodeID)
	}
}

func TestRecordsGetAllWithMetaReturnsCopy(t *testing.T) {
	r := NewRecords()

	r.AddWithMeta("example.com", "192.168.1.1", 1000, "node1")
	all := r.GetAllWithMeta()

	// Modify the returned map
	all["example.com"] = RecordEntry{IP: "modified", Timestamp: 9999, NodeID: "modified"}

	// Original should be unchanged
	ip, _ := r.Lookup("example.com")
	if ip != "192.168.1.1" {
		t.Error("GetAllWithMeta did not return a copy - original data was modified")
	}

	meta, _ := r.GetMeta("example.com")
	if meta.Timestamp != 1000 {
		t.Error("GetAllWithMeta did not return a copy - original metadata was modified")
	}
}

func TestRecordsApplyMessageAdd(t *testing.T) {
	r := NewRecords()

	msg := &RecordMessage{
		Hostname:  "example.com",
		IP:        "192.168.1.1",
		Action:    RecordActionAdd,
		Timestamp: 1000,
		NodeID:    "node1",
	}

	applied := r.ApplyMessage(msg)
	if !applied {
		t.Error("expected ApplyMessage to return true for add")
	}

	ip, found := r.Lookup("example.com")
	if !found {
		t.Error("expected to find example.com")
	}
	if ip != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", ip)
	}
}

func TestRecordsApplyMessageRemove(t *testing.T) {
	r := NewRecords()

	// First add a record
	r.AddWithMeta("example.com", "192.168.1.1", 1000, "node1")

	// Apply remove message with newer timestamp
	msg := &RecordMessage{
		Hostname:  "example.com",
		Action:    RecordActionRemove,
		Timestamp: 2000,
		NodeID:    "node2",
	}

	applied := r.ApplyMessage(msg)
	if !applied {
		t.Error("expected ApplyMessage to return true for remove")
	}

	_, found := r.Lookup("example.com")
	if found {
		t.Error("expected example.com to be removed")
	}
}

func TestRecordsApplyMessageUnknownAction(t *testing.T) {
	r := NewRecords()

	msg := &RecordMessage{
		Hostname:  "example.com",
		IP:        "192.168.1.1",
		Action:    RecordAction(99), // Unknown action
		Timestamp: 1000,
		NodeID:    "node1",
	}

	applied := r.ApplyMessage(msg)
	if applied {
		t.Error("expected ApplyMessage to return false for unknown action")
	}
}

func TestRecordsGetMeta(t *testing.T) {
	r := NewRecords()

	// Test non-existent hostname
	_, ok := r.GetMeta("nonexistent.com")
	if ok {
		t.Error("expected GetMeta to return false for non-existent hostname")
	}

	// Add a record
	r.AddWithMeta("example.com", "192.168.1.1", 1000, "node1")

	// Test existing hostname
	meta, ok := r.GetMeta("example.com")
	if !ok {
		t.Error("expected GetMeta to return true for existing hostname")
	}
	if meta.Timestamp != 1000 {
		t.Errorf("expected timestamp 1000, got %d", meta.Timestamp)
	}
	if meta.NodeID != "node1" {
		t.Errorf("expected nodeID 'node1', got %s", meta.NodeID)
	}

	// Test case normalization
	meta, ok = r.GetMeta("EXAMPLE.COM")
	if !ok {
		t.Error("expected GetMeta to find EXAMPLE.COM (case insensitive)")
	}
	if meta.Timestamp != 1000 {
		t.Errorf("expected timestamp 1000 for uppercase lookup, got %d", meta.Timestamp)
	}
}

func TestRecordsConcurrentWithMeta(t *testing.T) {
	r := NewRecords()
	var wg sync.WaitGroup
	iterations := 1000

	// Concurrent writers with metadata
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			nodeID := "node" + string(rune('a'+id))
			for j := 0; j < iterations; j++ {
				timestamp := int64(j * 10)
				r.AddWithMeta("test.com", "192.168.1.1", timestamp, nodeID)
				r.RemoveWithMeta("test.com", timestamp+5, nodeID)
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				r.Lookup("test.com")
				r.GetMeta("test.com")
				r.GetAllWithMeta()
				r.Count()
			}
		}()
	}

	// Concurrent message appliers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			nodeID := "applier" + string(rune('a'+id))
			for j := 0; j < iterations; j++ {
				msg := &RecordMessage{
					Hostname:  "apply-test.com",
					IP:        "10.0.0.1",
					Action:    RecordActionAdd,
					Timestamp: int64(j),
					NodeID:    nodeID,
				}
				r.ApplyMessage(msg)

				msg.Action = RecordActionRemove
				msg.Timestamp = int64(j + 1)
				r.ApplyMessage(msg)
			}
		}(i)
	}

	wg.Wait()
}

func TestRecordsLWWConcurrentUpdates(t *testing.T) {
	t.Parallel()

	r := NewRecords()
	var wg sync.WaitGroup
	hostname := "concurrent-lww.com"
	goroutines := 20
	iterations := 500

	// All goroutines try to update the same hostname with different timestamps
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			nodeID := "node-" + string(rune('a'+id))
			for j := 0; j < iterations; j++ {
				// Use different timestamp strategies to stress LWW logic
				timestamp := int64((id * iterations) + j)
				ip := "10." + string(rune('0'+id%10)) + ".0." + string(rune('1'+j%9))
				r.AddWithMeta(hostname, ip, timestamp, nodeID)
			}
		}(i)
	}

	wg.Wait()

	// Verify final state is deterministic: highest timestamp wins
	ip, found := r.Lookup(hostname)
	if !found {
		t.Fatal("expected hostname to exist after concurrent updates")
	}

	meta, ok := r.GetMeta(hostname)
	if !ok {
		t.Fatal("expected metadata to exist after concurrent updates")
	}

	// The highest timestamp should be (goroutines-1) * iterations + (iterations-1)
	// = 19 * 500 + 499 = 9999
	expectedMaxTimestamp := int64((goroutines-1)*iterations + (iterations - 1))
	if meta.Timestamp != expectedMaxTimestamp {
		t.Errorf("expected timestamp %d, got %d", expectedMaxTimestamp, meta.Timestamp)
	}

	// With same max timestamp, we expect nodeID "node-t" (id=19, 'a'+19='t')
	expectedNodeID := "node-t"
	if meta.NodeID != expectedNodeID {
		t.Errorf("expected nodeID %s, got %s", expectedNodeID, meta.NodeID)
	}

	// Verify the IP is non-empty (specific value depends on timing, but must exist)
	if ip == "" {
		t.Error("expected IP to be non-empty")
	}
}

func TestRecordsLargeStateSync(t *testing.T) {
	t.Parallel()

	r := NewRecords()
	recordCount := 150 // More than 100 as specified

	// Add many records with metadata
	for i := 0; i < recordCount; i++ {
		hostname := "host-" + string(rune('a'+i/26)) + string(rune('a'+i%26)) + ".example.com"
		ip := "10." + string(rune('0'+i/100)) + "." + string(rune('0'+(i/10)%10)) + "." + string(rune('0'+i%10))
		timestamp := int64(1000 + i)
		nodeID := "node-" + string(rune('a'+i%5))
		r.AddWithMeta(hostname, ip, timestamp, nodeID)
	}

	// Verify count
	if r.Count() != recordCount {
		t.Errorf("expected %d records, got %d", recordCount, r.Count())
	}

	// Get all with meta and verify each record
	allWithMeta := r.GetAllWithMeta()
	if len(allWithMeta) != recordCount {
		t.Errorf("expected GetAllWithMeta to return %d records, got %d", recordCount, len(allWithMeta))
	}

	// Verify a sample of records
	for i := 0; i < recordCount; i += 10 {
		hostname := "host-" + string(rune('a'+i/26)) + string(rune('a'+i%26)) + ".example.com"
		entry, ok := allWithMeta[hostname]
		if !ok {
			t.Errorf("expected to find %s in GetAllWithMeta", hostname)
			continue
		}

		expectedTimestamp := int64(1000 + i)
		if entry.Timestamp != expectedTimestamp {
			t.Errorf("expected timestamp %d for %s, got %d", expectedTimestamp, hostname, entry.Timestamp)
		}

		expectedNodeID := "node-" + string(rune('a'+i%5))
		if entry.NodeID != expectedNodeID {
			t.Errorf("expected nodeID %s for %s, got %s", expectedNodeID, hostname, entry.NodeID)
		}
	}

	// Verify GetAllWithMeta returns a copy (modifying it doesn't affect original)
	delete(allWithMeta, "host-aa.example.com")
	if r.Count() != recordCount {
		t.Error("modifying GetAllWithMeta result affected original records")
	}
}
