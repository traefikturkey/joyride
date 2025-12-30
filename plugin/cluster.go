package dockercluster

import (
	"context"
	"sync"
	"time"

	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/hashicorp/memberlist"
)

// ClusterManager orchestrates memberlist-based clustering for DNS record replication.
// It manages the cluster lifecycle, handles peer discovery, and coordinates
// record synchronization across cluster nodes.
type ClusterManager struct {
	config     *ClusterConfig
	records    *Records
	delegate   *ClusterDelegate
	discovery  *PeerDiscovery
	memberlist *memberlist.Memberlist

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

// NewClusterManager creates a new ClusterManager for DNS record replication.
// Returns nil, nil if config is nil or clustering is disabled.
// Returns an error if the configuration is invalid.
func NewClusterManager(config *ClusterConfig, records *Records) (*ClusterManager, error) {
	// Clustering disabled - return nil without error
	if config == nil || !config.Enabled {
		return nil, nil
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	cm := &ClusterManager{
		config:  config,
		records: records,
	}

	// Create delegate with numNodes function that returns current cluster size
	cm.delegate = NewClusterDelegate(config.NodeName, records, func() int {
		cm.mu.RLock()
		defer cm.mu.RUnlock()
		if cm.memberlist == nil {
			return 1
		}
		return cm.memberlist.NumMembers()
	})

	return cm, nil
}

// Start initializes the memberlist and begins cluster operations.
// It creates the memberlist configuration, starts the memberlist,
// and begins the delegate's background processing.
func (cm *ClusterManager) Start(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.ctx, cm.cancel = context.WithCancel(ctx)

	// Create memberlist configuration
	mlConfig := memberlist.DefaultLANConfig()
	mlConfig.Name = cm.config.NodeName
	mlConfig.BindAddr = cm.config.BindAddr
	mlConfig.BindPort = cm.config.Port
	mlConfig.AdvertisePort = cm.config.Port
	mlConfig.Delegate = cm.delegate

	// Set encryption key if configured
	if len(cm.config.SecretKey) > 0 {
		mlConfig.SecretKey = cm.config.SecretKey
	}

	// Tune intervals for DNS use case - faster gossip for quicker convergence
	mlConfig.GossipInterval = 200 * time.Millisecond
	mlConfig.ProbeInterval = 1 * time.Second
	mlConfig.ProbeTimeout = 500 * time.Millisecond
	mlConfig.SuspicionMult = 4

	// Disable memberlist logging (we use CoreDNS logging)
	mlConfig.LogOutput = nil
	mlConfig.Logger = nil

	// Create memberlist
	ml, err := memberlist.Create(mlConfig)
	if err != nil {
		return err
	}
	cm.memberlist = ml

	// Start delegate background processing
	cm.delegate.Start(cm.ctx)

	// Start broadcast discovery if no static seeds configured
	if len(cm.config.Seeds) == 0 {
		cm.discovery = NewPeerDiscovery(cm.config.NodeName, cm.config.Port, cm.config.DiscoveryPort)
		if err := cm.discovery.Start(cm.ctx); err != nil {
			log.Warningf("Cluster discovery failed to start: %v", err)
		}
	}

	log.Infof("Cluster started: node=%s addr=%s:%d", cm.config.NodeName, cm.config.BindAddr, cm.config.Port)

	return nil
}

// Stop gracefully shuts down the cluster manager.
// It stops the delegate, leaves the cluster, and shuts down memberlist.
func (cm *ClusterManager) Stop() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Cancel context
	if cm.cancel != nil {
		cm.cancel()
	}

	// Stop discovery
	if cm.discovery != nil {
		cm.discovery.Stop()
	}

	// Stop delegate
	if cm.delegate != nil {
		cm.delegate.Stop()
	}

	// Leave and shutdown memberlist
	if cm.memberlist != nil {
		// Leave with timeout
		if err := cm.memberlist.Leave(5 * time.Second); err != nil {
			log.Warningf("Error leaving cluster: %v", err)
		}

		if err := cm.memberlist.Shutdown(); err != nil {
			log.Warningf("Error shutting down memberlist: %v", err)
			return err
		}
	}

	log.Info("Cluster stopped")
	return nil
}

// Join attempts to join the cluster by contacting seed nodes or discovered peers.
// If using broadcast discovery, it will retry periodically until peers are found.
func (cm *ClusterManager) Join() error {
	cm.mu.RLock()
	seeds := cm.config.Seeds
	ml := cm.memberlist
	discovery := cm.discovery
	cm.mu.RUnlock()

	// If static seeds are configured, use them directly
	if len(seeds) > 0 {
		n, err := ml.Join(seeds)
		if err != nil {
			log.Warningf("Failed to join cluster seeds %v: %v", seeds, err)
			return err
		}
		log.Infof("Joined cluster via %d seed node(s)", n)
		return nil
	}

	// No static seeds - use broadcast discovery with retry loop
	if discovery == nil {
		log.Info("No seed nodes configured and discovery not running, operating standalone")
		return nil
	}

	log.Info("Using broadcast discovery to find cluster peers...")
	go cm.discoveryJoinLoop()
	return nil
}

// discoveryJoinLoop periodically attempts to join discovered peers.
func (cm *ClusterManager) discoveryJoinLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.mu.RLock()
			discovery := cm.discovery
			ml := cm.memberlist
			cm.mu.RUnlock()

			if discovery == nil || ml == nil {
				return
			}

			peers := discovery.GetPeers()
			if len(peers) == 0 {
				continue
			}

			// Only try to join if we're alone
			if ml.NumMembers() > 1 {
				continue
			}

			n, err := ml.Join(peers)
			if err != nil {
				log.Debugf("Failed to join discovered peers: %v", err)
				continue
			}

			if n > 0 {
				log.Infof("Joined cluster via %d discovered peer(s)", n)
			}
		}
	}
}

// NotifyRecordAdd broadcasts a record addition to cluster peers.
// The message is first applied locally, then broadcast to other nodes.
func (cm *ClusterManager) NotifyRecordAdd(hostname, ip string, timestamp int64) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.delegate == nil {
		return
	}

	msg := &RecordMessage{
		Hostname:  hostname,
		IP:        ip,
		Action:    RecordActionAdd,
		Timestamp: timestamp,
		NodeID:    cm.config.NodeName,
	}

	// Apply locally first
	cm.records.ApplyMessage(msg)

	// Broadcast to cluster
	cm.delegate.BroadcastRecord(msg)
}

// NotifyRecordRemove broadcasts a record removal to cluster peers.
// The message is first applied locally, then broadcast to other nodes.
func (cm *ClusterManager) NotifyRecordRemove(hostname string, timestamp int64) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.delegate == nil {
		return
	}

	msg := &RecordMessage{
		Hostname:  hostname,
		Action:    RecordActionRemove,
		Timestamp: timestamp,
		NodeID:    cm.config.NodeName,
	}

	// Apply locally first
	cm.records.ApplyMessage(msg)

	// Broadcast to cluster
	cm.delegate.BroadcastRecord(msg)
}

// Members returns the current list of cluster members.
// Returns nil if memberlist is not initialized.
func (cm *ClusterManager) Members() []*memberlist.Node {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.memberlist == nil {
		return nil
	}
	return cm.memberlist.Members()
}

// IsHealthy returns true if the cluster is operational.
// A cluster is considered healthy if it has at least one member (itself).
func (cm *ClusterManager) IsHealthy() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.memberlist == nil {
		return false
	}
	return cm.memberlist.NumMembers() > 0
}
