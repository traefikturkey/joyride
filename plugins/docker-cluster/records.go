// Package dockercluster provides a CoreDNS plugin for Docker container DNS resolution.
package dockercluster

import (
	"strings"
	"sync"
	"sync/atomic"
)

// RecordMeta stores metadata for a DNS record, used for LWW conflict resolution.
type RecordMeta struct {
	Timestamp int64  // Unix nanosecond timestamp of last update
	NodeID    string // Node that created/updated this record
}

// Records provides thread-safe storage for DNS hostname-to-IP mappings.
// It uses atomic.Value for lock-free reads on the hot path (DNS queries),
// with copy-on-write semantics for updates.
//
// For cluster synchronization, it also maintains metadata (timestamps, node IDs)
// to support last-write-wins (LWW) conflict resolution.
type Records struct {
	// data holds the current immutable snapshot of records.
	// Read operations access this atomically without locks.
	data atomic.Value // holds map[string]string

	// meta holds metadata for each record (timestamp, nodeID).
	// Used for cluster LWW conflict resolution.
	meta atomic.Value // holds map[string]RecordMeta

	// mu protects write operations (Add/Remove) to ensure
	// atomic copy-on-write updates.
	mu sync.Mutex
}

// NewRecords creates a new empty Records store.
func NewRecords() *Records {
	r := &Records{}
	r.data.Store(make(map[string]string))
	r.meta.Store(make(map[string]RecordMeta))
	return r
}

// Add adds or updates a DNS record mapping hostname to ip.
// The hostname is normalized to lowercase.
// This operation uses copy-on-write for thread safety.
func (r *Records) Add(hostname, ip string) {
	hostname = strings.ToLower(hostname)

	r.mu.Lock()
	defer r.mu.Unlock()

	// Load current data
	current := r.data.Load().(map[string]string)

	// Create a new map with all existing entries plus the new one
	newData := make(map[string]string, len(current)+1)
	for k, v := range current {
		newData[k] = v
	}
	newData[hostname] = ip

	// Atomically swap in the new map
	r.data.Store(newData)
}

// Remove removes a DNS record for the given hostname.
// The hostname is normalized to lowercase.
// This operation uses copy-on-write for thread safety.
// If the hostname doesn't exist, this is a no-op.
func (r *Records) Remove(hostname string) {
	hostname = strings.ToLower(hostname)

	r.mu.Lock()
	defer r.mu.Unlock()

	// Load current data
	current := r.data.Load().(map[string]string)

	// Check if the key exists; if not, nothing to do
	if _, exists := current[hostname]; !exists {
		return
	}

	// Create a new map without the removed entry
	newData := make(map[string]string, len(current)-1)
	for k, v := range current {
		if k != hostname {
			newData[k] = v
		}
	}

	// Atomically swap in the new map
	r.data.Store(newData)
}

// Lookup retrieves the IP address for a hostname.
// The hostname is normalized to lowercase.
// This is a lock-free operation optimized for the DNS query hot path.
// Returns the IP and true if found, or empty string and false if not found.
func (r *Records) Lookup(hostname string) (ip string, found bool) {
	hostname = strings.ToLower(hostname)
	current := r.data.Load().(map[string]string)
	ip, found = current[hostname]
	return
}

// GetAll returns a copy of all current DNS records.
// The returned map is safe to modify without affecting the stored records.
func (r *Records) GetAll() map[string]string {
	current := r.data.Load().(map[string]string)

	// Return a copy to prevent external modification
	result := make(map[string]string, len(current))
	for k, v := range current {
		result[k] = v
	}
	return result
}

// Count returns the number of DNS records currently stored.
func (r *Records) Count() int {
	current := r.data.Load().(map[string]string)
	return len(current)
}

// AddWithMeta adds or updates a DNS record with metadata for LWW conflict resolution.
// Returns true if the record was added/updated, false if an existing record has a newer timestamp.
// The hostname is normalized to lowercase.
func (r *Records) AddWithMeta(hostname, ip string, timestamp int64, nodeID string) bool {
	hostname = strings.ToLower(hostname)

	r.mu.Lock()
	defer r.mu.Unlock()

	// Load current metadata
	currentMeta := r.meta.Load().(map[string]RecordMeta)

	// Check if existing record is newer (LWW)
	if existing, ok := currentMeta[hostname]; ok {
		if existing.Timestamp > timestamp {
			return false // Existing record is newer
		}
		if existing.Timestamp == timestamp && existing.NodeID >= nodeID {
			return false // Same timestamp, tie-break by nodeID (higher wins)
		}
	}

	// Load current data
	currentData := r.data.Load().(map[string]string)

	// Create new data map
	newData := make(map[string]string, len(currentData)+1)
	for k, v := range currentData {
		newData[k] = v
	}
	newData[hostname] = ip

	// Create new meta map
	newMeta := make(map[string]RecordMeta, len(currentMeta)+1)
	for k, v := range currentMeta {
		newMeta[k] = v
	}
	newMeta[hostname] = RecordMeta{
		Timestamp: timestamp,
		NodeID:    nodeID,
	}

	// Atomically swap in new maps
	r.data.Store(newData)
	r.meta.Store(newMeta)

	return true
}

// RemoveWithMeta removes a DNS record with metadata for LWW conflict resolution.
// Returns true if the record was removed, false if an existing record has a newer timestamp.
// The hostname is normalized to lowercase.
func (r *Records) RemoveWithMeta(hostname string, timestamp int64, nodeID string) bool {
	hostname = strings.ToLower(hostname)

	r.mu.Lock()
	defer r.mu.Unlock()

	// Load current data
	currentData := r.data.Load().(map[string]string)

	// Check if the key exists
	if _, exists := currentData[hostname]; !exists {
		return false // Nothing to remove
	}

	// Load current metadata
	currentMeta := r.meta.Load().(map[string]RecordMeta)

	// Check if existing record is newer (LWW)
	if existing, ok := currentMeta[hostname]; ok {
		if existing.Timestamp > timestamp {
			return false // Existing record is newer
		}
		if existing.Timestamp == timestamp && existing.NodeID >= nodeID {
			return false // Same timestamp, tie-break by nodeID (higher wins)
		}
	}

	// Create new data map without the removed entry
	newData := make(map[string]string, len(currentData)-1)
	for k, v := range currentData {
		if k != hostname {
			newData[k] = v
		}
	}

	// Create new meta map without the removed entry
	newMeta := make(map[string]RecordMeta, len(currentMeta)-1)
	for k, v := range currentMeta {
		if k != hostname {
			newMeta[k] = v
		}
	}

	// Atomically swap in new maps
	r.data.Store(newData)
	r.meta.Store(newMeta)

	return true
}

// GetAllWithMeta returns a copy of all current DNS records with metadata.
// The returned map is safe to modify without affecting the stored records.
func (r *Records) GetAllWithMeta() map[string]RecordEntry {
	currentData := r.data.Load().(map[string]string)
	currentMeta := r.meta.Load().(map[string]RecordMeta)

	result := make(map[string]RecordEntry, len(currentData))
	for hostname, ip := range currentData {
		meta := currentMeta[hostname]
		result[hostname] = RecordEntry{
			IP:        ip,
			Timestamp: meta.Timestamp,
			NodeID:    meta.NodeID,
		}
	}
	return result
}

// ApplyMessage applies a gossip message to the records store.
// Returns true if the message was applied, false if it was rejected (e.g., stale).
func (r *Records) ApplyMessage(msg *RecordMessage) bool {
	switch msg.Action {
	case RecordActionAdd:
		return r.AddWithMeta(msg.Hostname, msg.IP, msg.Timestamp, msg.NodeID)
	case RecordActionRemove:
		return r.RemoveWithMeta(msg.Hostname, msg.Timestamp, msg.NodeID)
	default:
		return false
	}
}

// GetMeta returns the metadata for a hostname, or zero value if not found.
func (r *Records) GetMeta(hostname string) (RecordMeta, bool) {
	hostname = strings.ToLower(hostname)
	currentMeta := r.meta.Load().(map[string]RecordMeta)
	meta, ok := currentMeta[hostname]
	return meta, ok
}
