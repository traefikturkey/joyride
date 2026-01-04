package dockercluster

import (
	"testing"
)

func TestRecordMessageEncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		msg  RecordMessage
	}{
		{
			name: "add record",
			msg: RecordMessage{
				Hostname:  "app.example.com",
				IP:        "192.168.1.100",
				Action:    RecordActionAdd,
				Timestamp: 1234567890123456789,
				NodeID:    "node1",
			},
		},
		{
			name: "remove record",
			msg: RecordMessage{
				Hostname:  "api.example.com",
				IP:        "",
				Action:    RecordActionRemove,
				Timestamp: 1735500000000000000, // Valid Unix nano timestamp
				NodeID:    "node2",
			},
		},
		{
			name: "empty hostname",
			msg: RecordMessage{
				Hostname:  "",
				IP:        "10.0.0.1",
				Action:    RecordActionAdd,
				Timestamp: 0,
				NodeID:    "",
			},
		},
		{
			name: "special characters in hostname",
			msg: RecordMessage{
				Hostname:  "my-app_v2.sub.example.com",
				IP:        "172.16.0.50",
				Action:    RecordActionAdd,
				Timestamp: 1000000000000000000,
				NodeID:    "node-with-dashes",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.msg.Encode()
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			decoded, err := DecodeRecordMessage(encoded)
			if err != nil {
				t.Fatalf("DecodeRecordMessage() error = %v", err)
			}

			if decoded.Hostname != tt.msg.Hostname {
				t.Errorf("Hostname = %q, want %q", decoded.Hostname, tt.msg.Hostname)
			}
			if decoded.IP != tt.msg.IP {
				t.Errorf("IP = %q, want %q", decoded.IP, tt.msg.IP)
			}
			if decoded.Action != tt.msg.Action {
				t.Errorf("Action = %d, want %d", decoded.Action, tt.msg.Action)
			}
			if decoded.Timestamp != tt.msg.Timestamp {
				t.Errorf("Timestamp = %d, want %d", decoded.Timestamp, tt.msg.Timestamp)
			}
			if decoded.NodeID != tt.msg.NodeID {
				t.Errorf("NodeID = %q, want %q", decoded.NodeID, tt.msg.NodeID)
			}
		})
	}
}

func TestDecodeRecordMessageInvalidJSON(t *testing.T) {
	invalidInputs := [][]byte{
		[]byte("not json"),
		[]byte("{invalid}"),
		[]byte(""),
		nil,
	}

	for _, input := range invalidInputs {
		_, err := DecodeRecordMessage(input)
		if err == nil {
			t.Errorf("DecodeRecordMessage(%q) expected error, got nil", input)
		}
	}
}

func TestFullStateEncodeDecode(t *testing.T) {
	tests := []struct {
		name  string
		state FullState
	}{
		{
			name: "empty state",
			state: FullState{
				NodeID:  "node1",
				Records: map[string]RecordEntry{},
			},
		},
		{
			name: "single record",
			state: FullState{
				NodeID: "node1",
				Records: map[string]RecordEntry{
					"app.example.com": {
						IP:        "192.168.1.100",
						Timestamp: 1234567890123456789,
						NodeID:    "node1",
					},
				},
			},
		},
		{
			name: "multiple records",
			state: FullState{
				NodeID: "node2",
				Records: map[string]RecordEntry{
					"app.example.com": {
						IP:        "192.168.1.100",
						Timestamp: 1000000000000000000,
						NodeID:    "node1",
					},
					"api.example.com": {
						IP:        "192.168.1.101",
						Timestamp: 2000000000000000000,
						NodeID:    "node2",
					},
					"web.example.com": {
						IP:        "192.168.1.102",
						Timestamp: 3000000000000000000,
						NodeID:    "node3",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.state.Encode()
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			decoded, err := DecodeFullState(encoded)
			if err != nil {
				t.Fatalf("DecodeFullState() error = %v", err)
			}

			if decoded.NodeID != tt.state.NodeID {
				t.Errorf("NodeID = %q, want %q", decoded.NodeID, tt.state.NodeID)
			}

			if len(decoded.Records) != len(tt.state.Records) {
				t.Fatalf("Records len = %d, want %d", len(decoded.Records), len(tt.state.Records))
			}

			for hostname, want := range tt.state.Records {
				got, ok := decoded.Records[hostname]
				if !ok {
					t.Errorf("Record %q not found", hostname)
					continue
				}
				if got.IP != want.IP {
					t.Errorf("Record[%q].IP = %q, want %q", hostname, got.IP, want.IP)
				}
				if got.Timestamp != want.Timestamp {
					t.Errorf("Record[%q].Timestamp = %d, want %d", hostname, got.Timestamp, want.Timestamp)
				}
				if got.NodeID != want.NodeID {
					t.Errorf("Record[%q].NodeID = %q, want %q", hostname, got.NodeID, want.NodeID)
				}
			}
		})
	}
}

func TestDecodeFullStateInvalidJSON(t *testing.T) {
	invalidInputs := [][]byte{
		[]byte("not json"),
		[]byte("{invalid}"),
		[]byte(""),
		nil,
	}

	for _, input := range invalidInputs {
		_, err := DecodeFullState(input)
		if err == nil {
			t.Errorf("DecodeFullState(%q) expected error, got nil", input)
		}
	}
}

func TestNewFullState(t *testing.T) {
	nodeID := "test-node"
	state := NewFullState(nodeID)

	if state.NodeID != nodeID {
		t.Errorf("NodeID = %q, want %q", state.NodeID, nodeID)
	}
	if state.Records == nil {
		t.Error("Records is nil, want empty map")
	}
	if len(state.Records) != 0 {
		t.Errorf("Records len = %d, want 0", len(state.Records))
	}
}

func TestRecordActionValues(t *testing.T) {
	// Ensure action values are stable (important for serialization compatibility)
	if RecordActionAdd != 0 {
		t.Errorf("RecordActionAdd = %d, want 0", RecordActionAdd)
	}
	if RecordActionRemove != 1 {
		t.Errorf("RecordActionRemove = %d, want 1", RecordActionRemove)
	}
}

func BenchmarkRecordMessageEncode(b *testing.B) {
	msg := RecordMessage{
		Hostname:  "app.example.com",
		IP:        "192.168.1.100",
		Action:    RecordActionAdd,
		Timestamp: 1234567890123456789,
		NodeID:    "node1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.Encode()
	}
}

func BenchmarkRecordMessageDecode(b *testing.B) {
	msg := RecordMessage{
		Hostname:  "app.example.com",
		IP:        "192.168.1.100",
		Action:    RecordActionAdd,
		Timestamp: 1234567890123456789,
		NodeID:    "node1",
	}
	encoded, _ := msg.Encode()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeRecordMessage(encoded)
	}
}

func BenchmarkFullStateEncode(b *testing.B) {
	state := FullState{
		NodeID:  "node1",
		Records: make(map[string]RecordEntry),
	}
	// Add 100 records
	for i := 0; i < 100; i++ {
		hostname := "app" + string(rune('0'+i%10)) + ".example.com"
		state.Records[hostname] = RecordEntry{
			IP:        "192.168.1.100",
			Timestamp: int64(i) * 1000000000,
			NodeID:    "node1",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = state.Encode()
	}
}
