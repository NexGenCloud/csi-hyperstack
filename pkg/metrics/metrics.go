package metrics

import (
	"time"

	"k8s.io/component-base/metrics"
)

type OpenstackMetrics struct {
	Duration *metrics.HistogramVec
	Total    *metrics.CounterVec
	Errors   *metrics.CounterVec
}

// MetricContext indicates the context for OpenStack metrics.
type MetricContext struct {
	Start      time.Time
	Attributes []string
	Metrics    *OpenstackMetrics
}

// NewMetricContext creates a new MetricContext.
func NewMetricContext(resource string, request string) *MetricContext {
	return &MetricContext{
		Start:      time.Now(),
		Attributes: []string{resource + "_" + request},
	}
}

// ObserveRequest records the request latency and counts the errors.
func (mc *MetricContext) Observe(om *OpenstackMetrics, err error) error {
	if om == nil {
		// mc.RequestMetrics not set, ignore this request
		return nil
	}

	om.Duration.WithLabelValues(mc.Attributes...).Observe(
		time.Since(mc.Start).Seconds())
	om.Total.WithLabelValues(mc.Attributes...).Inc()
	if err != nil {
		om.Errors.WithLabelValues(mc.Attributes...).Inc()
	}
	return err
}

func RegisterMetrics(component string) {
	doRegisterAPIMetrics()
	if component == "occm" {
		doRegisterOccmMetrics()
	}
}
