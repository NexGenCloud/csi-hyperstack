package metrics

import (
	"sync"

	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

var (
	CSIOperationMetrics = &OpenstackMetrics{
		Duration: metrics.NewHistogramVec(
			&metrics.HistogramOpts{
				Name:    "csi_operation_duration_seconds",
				Help:    "Latency of a CSI driver operation",
				Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300, 600, 1200},
			}, []string{"operation"}),
		Total: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Name: "csi_operations_total",
				Help: "Total number of CSI driver operations",
			}, []string{"operation"}),
		Errors: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Name: "csi_operation_errors_total",
				Help: "Total number of errors for a CSI driver operation",
			}, []string{"operation"}),
	}

	VolumeAttachmentsGauge = metrics.NewGaugeVec(
		&metrics.GaugeOpts{
			Name: "csi_volume_attachments_total",
			Help: "Number of volumes currently published (bind-mounted) to pods by this CSI node plugin",
		},
		[]string{"node_id"},
	)
)

// ObserveCSIOperation records the operation latency and counts errors.
func (mc *MetricContext) ObserveCSIOperation(err error) error {
	return mc.Observe(CSIOperationMetrics, err)
}

var registerCSIMetrics sync.Once

func doRegisterCSIMetrics() {
	registerCSIMetrics.Do(func() {
		legacyregistry.MustRegister(
			CSIOperationMetrics.Duration,
			CSIOperationMetrics.Total,
			CSIOperationMetrics.Errors,
			VolumeAttachmentsGauge,
		)
	})
}
