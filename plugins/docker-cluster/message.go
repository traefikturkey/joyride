package dockercluster

import (
	"encoding/json"
)

// RecordAction defines the type of record operation for gossip messages.
type RecordAction uint8

const (
	// RecordActionAdd indicates a hostname should be added or updated.
	RecordActionAdd RecordAction = iota
	// RecordActionRemove indicates a hostname should be removed.
	RecordActionRemove
)

// RecordMessage represents a DNS record update for gossip protocol.
// These messages are broadcast to cluster peers when local Docker
// containers start or stop.
type RecordMessage struct {
	Hostname  string       `json:"h"` // Hostname (lowercase, without trailing dot)
	IP        string       `json:"i"` // IP address to resolve to
	Action    RecordAction `json:"a"` // Add or Remove
	Timestamp int64        `json:"t"` // Unix nanosecond timestamp for LWW
	NodeID    string       `json:"n"` // Source node identifier
}

// RecordEntry stores a DNS record with metadata for conflict resolution.
// Used in both local storage and full state synchronization.
type RecordEntry struct {
	IP        string `json:"i"` // IP address
	Timestamp int64  `json:"t"` // Unix nanosecond timestamp
	NodeID    string `json:"n"` // Node that created/updated this record
}

// FullState represents the complete DNS record state of a node.
// Used for TCP-based full state synchronization during cluster joins
// and periodic anti-entropy syncs.
type FullState struct {
	NodeID  string                 `json:"node"`    // Source node identifier
	Records map[string]RecordEntry `json:"records"` // hostname -> record entry
}

// Encode serializes a RecordMessage to JSON bytes for gossip transmission.
func (m *RecordMessage) Encode() ([]byte, error) {
	return json.Marshal(m)
}

// DecodeRecordMessage deserializes JSON bytes into a RecordMessage.
func DecodeRecordMessage(data []byte) (*RecordMessage, error) {
	var m RecordMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Encode serializes a FullState to JSON bytes for TCP state sync.
func (s *FullState) Encode() ([]byte, error) {
	return json.Marshal(s)
}

// DecodeFullState deserializes JSON bytes into a FullState.
func DecodeFullState(data []byte) (*FullState, error) {
	var s FullState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// NewFullState creates a new FullState with the given node ID and empty records.
func NewFullState(nodeID string) *FullState {
	return &FullState{
		NodeID:  nodeID,
		Records: make(map[string]RecordEntry),
	}
}
