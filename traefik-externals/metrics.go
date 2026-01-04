package traefikexternals

import (
	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for traefik-externals plugin
var (
	// recordsTotal tracks the current number of DNS records loaded
	recordsTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: plugin.Namespace,
		Subsystem: "traefik_externals",
		Name:      "records_total",
		Help:      "Number of DNS records currently loaded from Traefik external configs.",
	})

	// reloadsTotal counts successful config reloads
	reloadsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: "traefik_externals",
		Name:      "reloads_total",
		Help:      "Total number of successful config reloads.",
	})

	// parseErrorsTotal counts file parse errors
	parseErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: "traefik_externals",
		Name:      "parse_errors_total",
		Help:      "Total number of config file parse errors.",
	})

	// queriesTotal counts DNS queries handled
	queriesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: "traefik_externals",
		Name:      "queries_total",
		Help:      "Total number of DNS queries handled by traefik-externals.",
	}, []string{"type", "result"})
)
