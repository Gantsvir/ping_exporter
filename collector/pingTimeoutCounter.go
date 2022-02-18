package collector

import (
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

func NewPingTimeoutCounter(endpoints []string, opts prometheus.CounterOpts) (PingCollector, error) {
	if _, ok := opts.ConstLabels[endpointLabelKey]; ok {
		return nil, errors.Errorf("can not use %s as label key", endpointLabelKey)
	}

	collector := &pingTimeoutCounter{}

	collector.counterVec = prometheus.NewCounterVec(
		opts,
		[]string{endpointLabelKey},
	)
	for _, ep := range endpoints {
		collector.counterVec.With(map[string]string{endpointLabelKey: ep})
	}

	return collector, nil
}

type pingTimeoutCounter struct {
	counterVec *prometheus.CounterVec
}

func (p *pingTimeoutCounter) OnSucceed(endpoint string, time time.Duration, id int) {

}

func (p *pingTimeoutCounter) OnTimeout(endpoint string, id int) {
	p.counterVec.With(map[string]string{endpointLabelKey: endpoint}).Inc()
}

func (p *pingTimeoutCounter) OnFailed(endpoint string, err error) {

}

func (p *pingTimeoutCounter) Describe(descC chan<- *prometheus.Desc) {
	p.counterVec.Describe(descC)
}

func (p *pingTimeoutCounter) Collect(metricC chan<- prometheus.Metric) {
	p.counterVec.Collect(metricC)
}
