package dockercluster

import (
	"errors"
	"fmt"
)

// ClusterConfig holds configuration for memberlist-based clustering.
type ClusterConfig struct {
	// Enabled indicates whether clustering is enabled.
	Enabled bool

	// Port is the memberlist port (default 7946).
	Port int

	// Seeds is a list of seed node addresses (host:port) to join.
	// Optional - if empty, broadcast discovery is used.
	Seeds []string

	// NodeName is the unique identifier for this node in the cluster.
	NodeName string

	// BindAddr is the address to bind to (default "0.0.0.0").
	BindAddr string

	// SecretKey is an optional encryption key for cluster communication.
	SecretKey []byte

	// DiscoveryPort is the UDP port for broadcast peer discovery (default 8889).
	DiscoveryPort int
}

// NewClusterConfig returns a ClusterConfig with default values.
func NewClusterConfig() *ClusterConfig {
	return &ClusterConfig{
		Enabled:       false,
		Port:          7946,
		Seeds:         nil,
		NodeName:      "",
		BindAddr:      "0.0.0.0",
		DiscoveryPort: 8889,
	}
}

// Validate checks that the cluster configuration is valid.
// Returns an error if validation fails.
func (c *ClusterConfig) Validate() error {
	if c.Port <= 0 || c.Port >= 65536 {
		return fmt.Errorf("cluster port must be between 1 and 65535, got %d", c.Port)
	}

	if c.Enabled && c.NodeName == "" {
		return errors.New("cluster node_name is required when clustering is enabled")
	}

	return nil
}
