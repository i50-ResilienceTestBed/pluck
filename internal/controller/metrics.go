package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	TestRunJobReconcileTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "pluck_controller_testrunjob_reconcile_total",
		Help: "Total Test Run Job reconciliations per controller",
	}, []string{"job"})
)

func init() {

	metrics.Registry.MustRegister(TestRunJobReconcileTotal)
}
