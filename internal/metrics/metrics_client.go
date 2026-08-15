package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	runtimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Metrics subsystem and all of the keys used by the metrics client.
const (
	MetricsClientSubsystem                  = "metrics_client"
	MetricsClientFailTotalKey               = "fail_total"
	MetricsClientDeletedNodeSkippedTotalKey = "deleted_node_skipped_total"
)

func init() {
	registerMetricsClientMetrics()
}

type metricsClientFailTotalAdapter struct {
	metric prometheus.Counter
}

func (a *metricsClientFailTotalAdapter) Increment() {
	a.metric.Inc()
}

type metricsClientDeletedNodeSkippedTotalAdapter struct {
	metric prometheus.Counter
}

func (a *metricsClientDeletedNodeSkippedTotalAdapter) Increment() {
	a.metric.Inc()
}

var (
	metricsClientFailTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Subsystem: MetricsClientSubsystem,
		Name:      MetricsClientFailTotalKey,
		Help:      "counter that indicates how many API requests to metrics server(e.g. prometheus) are failed.",
	})
	metricsClientDeletedNodeSkippedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Subsystem: MetricsClientSubsystem,
		Name:      MetricsClientDeletedNodeSkippedTotalKey,
		Help:      "counter that indicates how many kubelet metrics errors were skipped after Kubernetes confirmed that the node was deleted.",
	})

	MetricsClientFailTotal               = &metricsClientFailTotalAdapter{metric: metricsClientFailTotal}
	MetricsClientDeletedNodeSkippedTotal = &metricsClientDeletedNodeSkippedTotalAdapter{
		metric: metricsClientDeletedNodeSkippedTotal,
	}
)

func registerMetricsClientMetrics() {
	runtimemetrics.Registry.MustRegister(metricsClientFailTotal)
	runtimemetrics.Registry.MustRegister(metricsClientDeletedNodeSkippedTotal)
}
