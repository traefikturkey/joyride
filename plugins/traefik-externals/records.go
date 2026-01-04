package traefikexternals

import (
	"strings"
	"sync"
	"sync/atomic"
)

// Records provides thread-safe storage for DNS hostname-to-IP mappings.
// It uses atomic.Value for lock-free reads on the hot path (DNS queries),
// with copy-on-write semantics for updates.
type Records struct {
	// data holds the current immutable snapshot of records.
	// Read operations access this atomically without locks.
	data atomic.Value // holds map[string]string

	// mu protects write operations to ensure atomic copy-on-write updates.
	mu sync.Mutex
}

// NewRecords creates a new empty Records store.
func NewRecords() *Records {
	r := &Records{}
	r.data.Store(make(map[string]string))
	return r
}

// Add adds or updates a DNS record mapping hostname to ip.
// The hostname is normalized to lowercase.
func (r *Records) Add(hostname, ip string) {
	hostname = strings.ToLower(hostname)

	r.mu.Lock()
	defer r.mu.Unlock()

	current := r.data.Load().(map[string]string)
	newData := make(map[string]string, len(current)+1)
	for k, v := range current {
		newData[k] = v
	}
	newData[hostname] = ip
	r.data.Store(newData)
}

// Remove removes a DNS record for the given hostname.
func (r *Records) Remove(hostname string) {
	hostname = strings.ToLower(hostname)

	r.mu.Lock()
	defer r.mu.Unlock()

	current := r.data.Load().(map[string]string)
	if _, exists := current[hostname]; !exists {
		return
	}

	newData := make(map[string]string, len(current)-1)
	for k, v := range current {
		if k != hostname {
			newData[k] = v
		}
	}
	r.data.Store(newData)
}

// Lookup retrieves the IP address for a hostname.
// This is a lock-free operation optimized for the DNS query hot path.
func (r *Records) Lookup(hostname string) (ip string, found bool) {
	hostname = strings.ToLower(hostname)
	current := r.data.Load().(map[string]string)
	ip, found = current[hostname]
	return
}

// GetAll returns a copy of all current DNS records.
func (r *Records) GetAll() map[string]string {
	current := r.data.Load().(map[string]string)
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

// ReplaceAll atomically replaces all records with the new set.
// This is used when reloading all external configs.
func (r *Records) ReplaceAll(newRecords map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Normalize all keys to lowercase
	normalized := make(map[string]string, len(newRecords))
	for k, v := range newRecords {
		normalized[strings.ToLower(k)] = v
	}
	r.data.Store(normalized)
}
