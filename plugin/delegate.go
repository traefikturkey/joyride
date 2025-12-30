package dockercluster

import (
	"context"
	"sync"

	"github.com/hashicorp/memberlist"
)

// ClusterDelegate implements the memberlist.Delegate interface for DNS record gossip.
// It handles broadcasting record changes to cluster peers and merging remote state.
type ClusterDelegate struct {
	nodeID     string
	records    *Records
	broadcasts *memberlist.TransmitLimitedQueue
	msgChan    chan *RecordMessage
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// msgChanBufferSize is the buffer size for the async message processing channel.
const msgChanBufferSize = 256

// NewClusterDelegate creates a new ClusterDelegate for gossip protocol.
// The numNodes function should return the current cluster size for broadcast retransmit calculation.
func NewClusterDelegate(nodeID string, records *Records, numNodes func() int) *ClusterDelegate {
	d := &ClusterDelegate{
		nodeID:  nodeID,
		records: records,
		msgChan: make(chan *RecordMessage, msgChanBufferSize),
	}

	d.broadcasts = &memberlist.TransmitLimitedQueue{
		NumNodes:       numNodes,
		RetransmitMult: 3,
	}

	return d
}

// NodeMeta returns metadata to send to other nodes during push/pull sync.
// We don't use custom node metadata, so return empty slice.
func (d *ClusterDelegate) NodeMeta(limit int) []byte {
	return nil
}

// NotifyMsg is called when a user-data message is received from another node.
// Messages are pushed to msgChan for async processing to avoid blocking memberlist.
func (d *ClusterDelegate) NotifyMsg(data []byte) {
	if len(data) == 0 {
		return
	}

	msg, err := DecodeRecordMessage(data)
	if err != nil {
		return
	}

	// Non-blocking send - drop message if channel is full
	select {
	case d.msgChan <- msg:
	default:
		// Channel full, drop message (will be recovered via full state sync)
	}
}

// GetBroadcasts returns queued messages to broadcast to other nodes.
// Called by memberlist when it has capacity to send more messages.
func (d *ClusterDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.broadcasts.GetBroadcasts(overhead, limit)
}

// LocalState returns the full local state to send during TCP push/pull sync.
// Called when joining a cluster or during anti-entropy sync.
func (d *ClusterDelegate) LocalState(join bool) []byte {
	allRecords := d.records.GetAllWithMeta()

	state := &FullState{
		NodeID:  d.nodeID,
		Records: allRecords,
	}

	data, err := state.Encode()
	if err != nil {
		return nil
	}

	return data
}

// MergeRemoteState merges state received from another node during TCP push/pull sync.
// Each record is applied using LWW conflict resolution.
func (d *ClusterDelegate) MergeRemoteState(buf []byte, join bool) {
	if len(buf) == 0 {
		return
	}

	state, err := DecodeFullState(buf)
	if err != nil {
		return
	}

	// Apply each record using LWW conflict resolution
	for hostname, entry := range state.Records {
		msg := &RecordMessage{
			Hostname:  hostname,
			IP:        entry.IP,
			Action:    RecordActionAdd,
			Timestamp: entry.Timestamp,
			NodeID:    entry.NodeID,
		}
		d.records.ApplyMessage(msg)
	}
}

// BroadcastRecord queues a record message for broadcast to cluster peers.
func (d *ClusterDelegate) BroadcastRecord(msg *RecordMessage) {
	data, err := msg.Encode()
	if err != nil {
		return
	}

	d.broadcasts.QueueBroadcast(&broadcast{data: data})
}

// Start begins the background goroutine that processes incoming messages.
func (d *ClusterDelegate) Start(ctx context.Context) {
	d.ctx, d.cancel = context.WithCancel(ctx)

	d.wg.Add(1)
	go d.processMessages()
}

// Stop gracefully shuts down the delegate, waiting for the background goroutine to finish.
func (d *ClusterDelegate) Stop() {
	if d.cancel != nil {
		d.cancel()
	}
	d.wg.Wait()
}

// processMessages handles incoming gossip messages from the msgChan.
func (d *ClusterDelegate) processMessages() {
	defer d.wg.Done()

	for {
		select {
		case <-d.ctx.Done():
			return
		case msg := <-d.msgChan:
			if msg != nil {
				d.records.ApplyMessage(msg)
			}
		}
	}
}

// broadcast implements memberlist.Broadcast for the TransmitLimitedQueue.
type broadcast struct {
	data []byte
}

// Invalidates returns a flag indicating if this broadcast should invalidate another.
// We don't use message invalidation, so always return false.
func (b *broadcast) Invalidates(other memberlist.Broadcast) bool {
	return false
}

// Message returns the broadcast data to send.
func (b *broadcast) Message() []byte {
	return b.data
}

// Finished is called when the broadcast is complete (transmitted to all nodes).
// We don't need to take any action on completion.
func (b *broadcast) Finished() {
}
