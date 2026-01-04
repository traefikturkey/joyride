package traefikexternals

import (
	"context"
	"net"
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
	plugin.Register("traefik-externals", setup)
}

func setup(c *caddy.Controller) error {
	te, err := parseConfig(c)
	if err != nil {
		return plugin.Error("traefik-externals", err)
	}

	// Check if plugin is disabled via environment variable
	if isDisabled() {
		log.Info("traefik-externals: disabled via TRAEFIK_EXTERNALS_ENABLED=false")
		// Register a pass-through handler
		dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
			return next
		})
		return nil
	}

	// Phase 1.1: Check if directory exists - graceful handling
	if _, err := os.Stat(te.Watcher.directory); os.IsNotExist(err) {
		log.Warningf("traefik-externals: directory %s does not exist, plugin disabled", te.Watcher.directory)
		// Register a pass-through handler instead of failing
		dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
			return next
		})
		return nil
	}

	// Log configuration at startup
	log.Infof("traefik-externals: directory=%s host_ip=%s ttl=%d",
		te.Watcher.directory, te.Watcher.hostIP, te.TTL)

	// Start the file watcher
	if err := te.Watcher.Start(context.Background()); err != nil {
		return plugin.Error("traefik-externals", err)
	}

	// Register shutdown handler
	c.OnShutdown(func() error {
		te.Watcher.Stop()
		return nil
	})

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		te.Next = next
		return te
	})

	return nil
}

func parseConfig(c *caddy.Controller) (*TraefikExternals, error) {
	var (
		directory = "/etc/traefik/external-enabled"
		hostIP    string
		ttl       uint32 = 60
		f         fall.F
	)

	for c.Next() {
		// traefik-externals can have arguments (zones), but typically none
		_ = c.RemainingArgs()

		for c.NextBlock() {
			switch c.Val() {
			case "directory":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				directory = c.Val()

			case "host_ip":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				hostIP = c.Val()

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

			default:
				return nil, c.Errf("unknown property '%s'", c.Val())
			}
		}
	}

	// Check for environment variable overrides
	if envDir := os.Getenv("TRAEFIK_EXTERNALS_DIRECTORY"); envDir != "" {
		directory = envDir
	}
	if envHostIP := os.Getenv("HOSTIP"); envHostIP != "" {
		hostIP = envHostIP
	}
	if envTTL := os.Getenv("TRAEFIK_EXTERNALS_TTL"); envTTL != "" {
		parsed, err := strconv.ParseUint(envTTL, 10, 32)
		if err == nil {
			ttl = uint32(parsed)
		}
	}

	// Validate required fields
	if hostIP == "" {
		return nil, c.Err("host_ip is required (set in config or HOSTIP env var)")
	}

	// Validate IP format (Phase 1.2: prevent panic on invalid IP)
	if net.ParseIP(hostIP) == nil {
		return nil, c.Errf("invalid host_ip: %s", hostIP)
	}

	// Create shared records store
	records := NewRecords()

	// Create file watcher
	watcher := NewFileWatcher(directory, hostIP, records)

	te := &TraefikExternals{
		Records: records,
		Watcher: watcher,
		TTL:     ttl,
		Fall:    f,
	}

	return te, nil
}

// isDisabled checks if the plugin is disabled via environment variable.
// The plugin is enabled by default. Set TRAEFIK_EXTERNALS_ENABLED=false to disable.
func isDisabled() bool {
	val := os.Getenv("TRAEFIK_EXTERNALS_ENABLED")
	if val == "" {
		return false // Enabled by default
	}

	val = strings.ToLower(val)
	// Disabled if explicitly set to false/no/0
	return val == "false" || val == "no" || val == "0"
}
