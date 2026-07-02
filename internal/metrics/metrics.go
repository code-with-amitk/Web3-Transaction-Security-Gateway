package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	DecisionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_decisions_total",
			Help: "Transaction policy decisions by action and policy name.",
		},
		[]string{"action", "policy"},
	)

	SubmitLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gateway_submit_latency_seconds",
			Help:    "End-to-end latency for POST /transactions.",
			Buckets: prometheus.DefBuckets,
		},
	)

	PendingQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gateway_pending_approvals",
			Help: "Number of INSPECT transactions awaiting approval.",
		},
	)
)

func Handler() http.Handler {
	return promhttp.Handler()
}

func ObserveSubmit(start time.Time) {
	SubmitLatency.Observe(time.Since(start).Seconds())
}

func RecordDecision(action, policyName string) {
	DecisionsTotal.WithLabelValues(action, policyName).Inc()
}

func SetPendingDepth(n float64) {
	PendingQueueDepth.Set(n)
}
