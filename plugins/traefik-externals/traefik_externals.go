// Package traefikexternals provides a CoreDNS plugin for DNS resolution
// based on Traefik external service configurations.
package traefikexternals

import (
	"context"
	"net"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

// TraefikExternals implements the plugin.Handler interface for Traefik external service DNS resolution.
type TraefikExternals struct {
	Records *Records
	Watcher *FileWatcher
	TTL     uint32
	Fall    fall.F
	Next    plugin.Handler
}

// Name returns the plugin name.
func (te *TraefikExternals) Name() string {
	return "traefik-externals"
}

// ServeDNS implements the plugin.Handler interface.
func (te *TraefikExternals) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	// Normalize the query name (lowercase, remove trailing dot)
	qname := strings.ToLower(state.Name())
	qname = strings.TrimSuffix(qname, ".")

	// Check if we know this hostname
	ip, found := te.Records.Lookup(qname)

	// Handle AAAA queries for known hostnames
	// Return empty authoritative response so dual-stack clients don't wait
	if state.QType() == dns.TypeAAAA && found {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		w.WriteMsg(m)
		queriesTotal.WithLabelValues("AAAA", "success").Inc()
		return dns.RcodeSuccess, nil
	}

	// Only handle A record queries
	if state.QType() != dns.TypeA {
		return plugin.NextOrFailure(te.Name(), te.Next, ctx, w, r)
	}

	if !found {
		// Pass to next plugin in chain
		queriesTotal.WithLabelValues("A", "miss").Inc()
		if te.Fall.Through(state.Name()) {
			return plugin.NextOrFailure(te.Name(), te.Next, ctx, w, r)
		}
		// Don't respond - let it fall through or timeout
		log.Debugf("traefik-externals: no record found for %s", state.Name())
		return plugin.NextOrFailure(te.Name(), te.Next, ctx, w, r)
	}

	// Build the response
	a := &dns.A{
		Hdr: dns.RR_Header{
			Name:   state.QName(),
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    te.TTL,
		},
		A: net.ParseIP(ip).To4(),
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.Answer = append(m.Answer, a)

	w.WriteMsg(m)
	queriesTotal.WithLabelValues("A", "success").Inc()
	return dns.RcodeSuccess, nil
}
