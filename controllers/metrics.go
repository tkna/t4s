package controllers

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	RemovedRowsVec = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "removed_rows_total",
			Help:    "Number of removed rows",
			Buckets: prometheus.LinearBuckets(1, 1, 4),
		}, []string{"namespace"})
)

func init() {
	metrics.Registry.MustRegister(RemovedRowsVec)
}
