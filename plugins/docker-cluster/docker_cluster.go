package dockercluster

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/plugin/docker-cluster/version"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

// UnknownAction defines what to do when a hostname is not found.
type UnknownAction int

const (
	// ActionDrop - don't respond at all (let it timeout) - for split DNS
	ActionDrop UnknownAction = iota
	// ActionNXDomain - return NXDOMAIN
	ActionNXDomain
)

// DockerCluster implements the plugin.Handler interface for Docker container DNS resolution.
type DockerCluster struct {
	Records        *Records
	Watcher        *DockerWatcher
	TTL            uint32
	Fall           fall.F
	Next           plugin.Handler
	UnknownAction  UnknownAction
	ClusterConfig  *ClusterConfig
	ClusterManager *ClusterManager
}

// Name returns the plugin name.
func (dc *DockerCluster) Name() string {
	return "docker-cluster"
}

// ServeDNS implements the plugin.Handler interface.
func (dc *DockerCluster) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	// Normalize the query name (lowercase, remove trailing dot)
	qname := strings.ToLower(state.Name())
	qname = strings.TrimSuffix(qname, ".")

	// Check if we know this hostname
	ip, found := dc.Records.Lookup(qname)

	// Handle AAAA queries for known hostnames
	// IPv6 is dumb - return empty response so dual-stack clients don't wait
	if state.QType() == dns.TypeAAAA && found {
		m := new(dns.Msg)
		m.SetReply(r)
		w.WriteMsg(m)
		return dns.RcodeSuccess, nil
	}

	// Only handle A record queries
	if state.QType() != dns.TypeA {
		return dc.handleUnknown(ctx, w, r, state)
	}

	if !found {
		return dc.handleUnknown(ctx, w, r, state)
	}

	// Build the response
	a := &dns.A{
		Hdr: dns.RR_Header{
			Name:   state.QName(),
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    dc.TTL,
		},
		A: net.ParseIP(ip).To4(),
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.Answer = append(m.Answer, a)

	w.WriteMsg(m)
	return dns.RcodeSuccess, nil
}

// handleUnknown handles queries for hostnames we don't know about.
func (dc *DockerCluster) handleUnknown(ctx context.Context, w dns.ResponseWriter, r *dns.Msg, state request.Request) (int, error) {
	// Check if fallthrough is enabled for this zone (passes to next plugin in chain)
	if dc.Fall.Through(state.Name()) {
		return plugin.NextOrFailure(dc.Name(), dc.Next, ctx, w, r)
	}

	// Handle based on configured action
	switch dc.UnknownAction {
	case ActionNXDomain:
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeNameError)
		w.WriteMsg(m)
		return dns.RcodeNameError, nil

	default:
		// ActionDrop: Don't respond at all (let it timeout)
		// This is for split DNS setups where the upstream DNS server
		// (e.g., EdgeRouter with all-servers) queries other sources in parallel.
		log.Debugf("no record found for %s, dropping query", state.Name())
		return dns.RcodeSuccess, nil
	}
}

// versionHandler handles GET /version requests
func (dc *DockerCluster) versionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(version.GetInfo()); err != nil {
		log.Errorf("docker-cluster: failed to encode version info: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ServeVersionHTTP starts the version HTTP endpoint and returns the server for shutdown
func (dc *DockerCluster) ServeVersionHTTP(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/version", dc.versionHandler)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Infof("docker-cluster: version endpoint listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("docker-cluster: version endpoint failed: %v", err)
		}
	}()

	return server
}
