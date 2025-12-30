package dockercluster

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/pkg/log"
)

func init() {
	plugin.Register("docker-cluster", setup)
}

func setup(c *caddy.Controller) error {
	dc, err := parseConfig(c)
	if err != nil {
		return plugin.Error("docker-cluster", err)
	}

	// Log configuration at startup
	actionName := "drop"
	if dc.UnknownAction == ActionNXDomain {
		actionName = "nxdomain"
	}
	log.Infof("docker-cluster: host_ip=%s labels=%v ttl=%d unknown_action=%s",
		dc.Watcher.hostIP, dc.Watcher.labels, dc.TTL, actionName)
	log.Infof("docker-cluster: docker_socket=%s", dc.Watcher.dockerSocket)

	// Create ClusterManager if clustering is enabled
	if dc.ClusterConfig != nil && dc.ClusterConfig.Enabled {
		cm, err := NewClusterManager(dc.ClusterConfig, dc.Records)
		if err != nil {
			return plugin.Error("docker-cluster", err)
		}
		dc.ClusterManager = cm

		// Wire callback from DockerWatcher to ClusterManager
		dc.Watcher.SetCallback(func(hostname, ip string, added bool, timestamp int64) {
			if added {
				cm.NotifyRecordAdd(hostname, ip, timestamp)
			} else {
				cm.NotifyRecordRemove(hostname, timestamp)
			}
		})

		// Start cluster manager
		if err := cm.Start(context.Background()); err != nil {
			return plugin.Error("docker-cluster", err)
		}

		// Join cluster (non-blocking, logs warnings on failure)
		go func() {
			if err := cm.Join(); err != nil {
				log.Warningf("docker-cluster: failed to join cluster: %v", err)
			}
		}()

		log.Infof("docker-cluster: clustering enabled, node=%s", dc.ClusterConfig.NodeName)
	}

	// Start the Docker watcher
	if err := dc.Watcher.Start(context.Background()); err != nil {
		return plugin.Error("docker-cluster", err)
	}

	// Register shutdown handler
	c.OnShutdown(func() error {
		dc.Watcher.Stop()
		if dc.ClusterManager != nil {
			dc.ClusterManager.Stop()
		}
		return nil
	})

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		dc.Next = next
		return dc
	})

	return nil
}

func parseConfig(c *caddy.Controller) (*DockerCluster, error) {
	var (
		dockerSocket  = "unix:///var/run/docker.sock"
		hostIP        string
		labels        []string
		ttl           uint32 = 60
		f             fall.F
		unknownAction = ActionDrop // default: no response for split DNS
		clusterConfig = NewClusterConfig()
	)

	for c.Next() {
		// docker-cluster can have arguments (zones), but typically none
		_ = c.RemainingArgs()

		for c.NextBlock() {
			switch c.Val() {
			case "docker_socket":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				dockerSocket = c.Val()

			case "host_ip":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				hostIP = c.Val()

			case "label":
				args := c.RemainingArgs()
				if len(args) == 0 {
					return nil, c.ArgErr()
				}
				labels = append(labels, args...)

			case "ttl":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				parsed, err := strconv.ParseUint(c.Val(), 10, 32)
				if err != nil {
					return nil, c.Errf("invalid ttl: %s", c.Val())
				}
				ttl = uint32(parsed)

			case "fallthrough":
				f.SetZonesFromArgs(c.RemainingArgs())

			case "unknown_action":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				action, err := parseUnknownAction(c.Val())
				if err != nil {
					return nil, c.Errf("invalid unknown_action: %s (valid: drop, nxdomain)", c.Val())
				}
				unknownAction = action

			case "cluster_enabled":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				val := strings.ToLower(c.Val())
				clusterConfig.Enabled = val == "true" || val == "1" || val == "yes"

			case "cluster_port":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				port, err := strconv.Atoi(c.Val())
				if err != nil {
					return nil, c.Errf("invalid cluster_port: %s", c.Val())
				}
				clusterConfig.Port = port

			case "cluster_seeds":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				seeds := strings.Split(c.Val(), ",")
				for i, seed := range seeds {
					seeds[i] = strings.TrimSpace(seed)
				}
				clusterConfig.Seeds = seeds

			case "node_name":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				clusterConfig.NodeName = c.Val()

			case "cluster_bind_addr":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				clusterConfig.BindAddr = c.Val()

			case "cluster_secret":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				clusterConfig.SecretKey = []byte(c.Val())

			case "discovery_port":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				port, err := strconv.Atoi(c.Val())
				if err != nil {
					return nil, c.Errf("invalid discovery_port: %s", c.Val())
				}
				clusterConfig.DiscoveryPort = port

			default:
				return nil, c.Errf("unknown property '%s'", c.Val())
			}
		}
	}

	// Legacy Joyride backward compatibility (checked first, new vars override)
	if envLegacySerf := os.Getenv("JOYRIDE_ENABLE_SERF"); envLegacySerf != "" {
		val := strings.ToLower(envLegacySerf)
		clusterConfig.Enabled = val == "true" || val == "1" || val == "yes"
		log.Infof("JOYRIDE_ENABLE_SERF is deprecated, use CLUSTER_ENABLED")
	}
	if envLegacyNXDomain := os.Getenv("JOYRIDE_NXDOMAIN_ENABLED"); envLegacyNXDomain != "" {
		val := strings.ToLower(envLegacyNXDomain)
		if val == "true" || val == "1" || val == "yes" {
			unknownAction = ActionNXDomain
		}
		// false/empty = ActionDrop (already the default)
		log.Infof("JOYRIDE_NXDOMAIN_ENABLED is deprecated, use DNS_UNKNOWN_ACTION")
	}

	// Check for environment variable overrides
	if envHostIP := os.Getenv("HOST_IP"); envHostIP != "" {
		hostIP = envHostIP
	}
	if envDockerSocket := os.Getenv("DOCKER_SOCKET"); envDockerSocket != "" {
		dockerSocket = envDockerSocket
	}
	if envUnknownAction := os.Getenv("DNS_UNKNOWN_ACTION"); envUnknownAction != "" {
		action, err := parseUnknownAction(envUnknownAction)
		if err != nil {
			return nil, fmt.Errorf("invalid DNS_UNKNOWN_ACTION env var: %s (valid: drop, nxdomain)", envUnknownAction)
		}
		unknownAction = action
	}

	// Check for cluster environment variable overrides
	if envClusterEnabled := os.Getenv("CLUSTER_ENABLED"); envClusterEnabled != "" {
		val := strings.ToLower(envClusterEnabled)
		clusterConfig.Enabled = val == "true" || val == "1" || val == "yes"
	}
	if envClusterPort := os.Getenv("CLUSTER_PORT"); envClusterPort != "" {
		port, err := strconv.Atoi(envClusterPort)
		if err != nil {
			return nil, fmt.Errorf("invalid CLUSTER_PORT env var: %s", envClusterPort)
		}
		clusterConfig.Port = port
	}
	if envClusterSeeds := os.Getenv("CLUSTER_SEEDS"); envClusterSeeds != "" {
		seeds := strings.Split(envClusterSeeds, ",")
		for i, seed := range seeds {
			seeds[i] = strings.TrimSpace(seed)
		}
		clusterConfig.Seeds = seeds
	}
	if envNodeName := os.Getenv("NODE_NAME"); envNodeName != "" {
		clusterConfig.NodeName = envNodeName
	}
	if envClusterBindAddr := os.Getenv("CLUSTER_BIND_ADDR"); envClusterBindAddr != "" {
		clusterConfig.BindAddr = envClusterBindAddr
	}
	if envClusterSecret := os.Getenv("CLUSTER_SECRET"); envClusterSecret != "" {
		clusterConfig.SecretKey = []byte(envClusterSecret)
	}
	if envDiscoveryPort := os.Getenv("DISCOVERY_PORT"); envDiscoveryPort != "" {
		port, err := strconv.Atoi(envDiscoveryPort)
		if err != nil {
			return nil, fmt.Errorf("invalid DISCOVERY_PORT env var: %s", envDiscoveryPort)
		}
		clusterConfig.DiscoveryPort = port
	}

	// Validate required fields
	if hostIP == "" {
		return nil, c.Err("host_ip is required (set in config or HOST_IP env var)")
	}

	// Validate cluster config
	if err := clusterConfig.Validate(); err != nil {
		return nil, c.Errf("cluster configuration error: %v", err)
	}

	// Set default labels if none specified
	if len(labels) == 0 {
		labels = []string{"coredns.host.name", "joyride.host.name"}
	}

	// Create shared records store
	records := NewRecords()

	// Create Docker watcher
	watcher := NewDockerWatcher(dockerSocket, hostIP, labels, records)

	dc := &DockerCluster{
		Records:       records,
		Watcher:       watcher,
		TTL:           ttl,
		Fall:          f,
		UnknownAction: unknownAction,
		ClusterConfig: clusterConfig,
	}

	return dc, nil
}

// parseUnknownAction converts a string to UnknownAction.
func parseUnknownAction(s string) (UnknownAction, error) {
	switch strings.ToLower(s) {
	case "drop", "timeout", "none":
		return ActionDrop, nil
	case "nxdomain":
		return ActionNXDomain, nil
	default:
		return ActionDrop, fmt.Errorf("unknown action: %s", s)
	}
}
