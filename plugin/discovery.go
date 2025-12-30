package dockercluster

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/coredns/coredns/plugin/pkg/log"
)

const (
	// DefaultDiscoveryPort is the UDP port for broadcast discovery.
	DefaultDiscoveryPort = 8889

	// discoveryInterval is how often to broadcast presence.
	discoveryInterval = 5 * time.Second

	// discoveryMessage is the magic prefix for discovery packets.
	discoveryMessage = "COREDNS-CLUSTER"
)

// discoveryPacket is broadcast to announce node presence.
type discoveryPacket struct {
	Magic    string `json:"m"`
	NodeID   string `json:"n"`
	ClusterPort int `json:"p"`
}

// PeerDiscovery handles UDP broadcast-based peer discovery.
type PeerDiscovery struct {
	nodeID      string
	clusterPort int
	discoveryPort int

	conn    *net.UDPConn
	peers   map[string]string // nodeID -> "ip:port"
	mu      sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewPeerDiscovery creates a new broadcast-based peer discovery service.
func NewPeerDiscovery(nodeID string, clusterPort, discoveryPort int) *PeerDiscovery {
	if discoveryPort == 0 {
		discoveryPort = DefaultDiscoveryPort
	}
	return &PeerDiscovery{
		nodeID:        nodeID,
		clusterPort:   clusterPort,
		discoveryPort: discoveryPort,
		peers:         make(map[string]string),
	}
}

// Start begins broadcasting presence and listening for peers.
func (pd *PeerDiscovery) Start(ctx context.Context) error {
	pd.ctx, pd.cancel = context.WithCancel(ctx)

	// Bind to discovery port for receiving broadcasts
	addr := &net.UDPAddr{Port: pd.discoveryPort}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return fmt.Errorf("failed to bind discovery port %d: %w", pd.discoveryPort, err)
	}
	pd.conn = conn

	// Enable broadcast
	if err := conn.SetReadBuffer(65535); err != nil {
		log.Warningf("discovery: failed to set read buffer: %v", err)
	}

	pd.wg.Add(2)
	go pd.broadcastLoop()
	go pd.listenLoop()

	log.Infof("discovery: started on port %d", pd.discoveryPort)
	return nil
}

// Stop shuts down the discovery service.
func (pd *PeerDiscovery) Stop() {
	if pd.cancel != nil {
		pd.cancel()
	}
	if pd.conn != nil {
		pd.conn.Close()
	}
	pd.wg.Wait()
	log.Info("discovery: stopped")
}

// GetPeers returns the currently discovered peer addresses (ip:clusterPort).
func (pd *PeerDiscovery) GetPeers() []string {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	peers := make([]string, 0, len(pd.peers))
	for _, addr := range pd.peers {
		peers = append(peers, addr)
	}
	return peers
}

// broadcastLoop periodically broadcasts this node's presence.
func (pd *PeerDiscovery) broadcastLoop() {
	defer pd.wg.Done()

	packet := discoveryPacket{
		Magic:       discoveryMessage,
		NodeID:      pd.nodeID,
		ClusterPort: pd.clusterPort,
	}
	data, err := json.Marshal(packet)
	if err != nil {
		log.Errorf("discovery: failed to marshal packet: %v", err)
		return
	}

	// Broadcast address
	broadcastAddr := &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: pd.discoveryPort,
	}

	ticker := time.NewTicker(discoveryInterval)
	defer ticker.Stop()

	// Broadcast immediately on start
	pd.broadcast(data, broadcastAddr)

	for {
		select {
		case <-pd.ctx.Done():
			return
		case <-ticker.C:
			pd.broadcast(data, broadcastAddr)
		}
	}
}

// broadcast sends a discovery packet to the broadcast address.
func (pd *PeerDiscovery) broadcast(data []byte, addr *net.UDPAddr) {
	// Create a separate socket for sending broadcasts
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		log.Debugf("discovery: failed to dial broadcast: %v", err)
		return
	}
	defer conn.Close()

	if _, err := conn.Write(data); err != nil {
		log.Debugf("discovery: failed to send broadcast: %v", err)
	}
}

// listenLoop receives discovery broadcasts from other nodes.
func (pd *PeerDiscovery) listenLoop() {
	defer pd.wg.Done()

	buf := make([]byte, 1024)
	for {
		select {
		case <-pd.ctx.Done():
			return
		default:
		}

		// Set read deadline so we can check for cancellation
		pd.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, remoteAddr, err := pd.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if pd.ctx.Err() != nil {
				return
			}
			log.Debugf("discovery: read error: %v", err)
			continue
		}

		pd.handlePacket(buf[:n], remoteAddr)
	}
}

// handlePacket processes a received discovery packet.
func (pd *PeerDiscovery) handlePacket(data []byte, remoteAddr *net.UDPAddr) {
	var packet discoveryPacket
	if err := json.Unmarshal(data, &packet); err != nil {
		return // Ignore malformed packets
	}

	// Verify magic
	if packet.Magic != discoveryMessage {
		return
	}

	// Ignore our own broadcasts
	if packet.NodeID == pd.nodeID {
		return
	}

	// Build peer address using sender's IP and their cluster port
	peerAddr := fmt.Sprintf("%s:%d", remoteAddr.IP.String(), packet.ClusterPort)

	pd.mu.Lock()
	if _, exists := pd.peers[packet.NodeID]; !exists {
		log.Infof("discovery: found peer %s at %s", packet.NodeID, peerAddr)
	}
	pd.peers[packet.NodeID] = peerAddr
	pd.mu.Unlock()
}
