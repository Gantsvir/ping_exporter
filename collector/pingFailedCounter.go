package collector

import (
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

func NewPingFailedCounter(endpoints []string, opts prometheus.CounterOpts) (PingCollector, error) {
	if _, ok := opts.ConstLabels[endpointLabelKey]; ok {
		return nil, errors.Errorf("can not use %s as label key", endpointLabelKey)
	}

	collector := &pingFailedCounter{}

	collector.counterVec = prometheus.NewCounterVec(
		opts,
		[]string{endpointLabelKey},
	)
	for _, ep := range endpoints {
		collector.counterVec.With(map[string]string{endpointLabelKey: ep})
	}

	return collector, nil
}

type pingFailedCounter struct {
	counterVec *prometheus.CounterVec
}

func (p *pingFailedCounter) OnSucceed(endpoint string, time time.Duration, id int) {

}

func (p *pingFailedCounter) OnTimeout(endpoint string, id int) {

}

func (p *pingFailedCounter) OnFailed(endpoint string, err error) {
	p.counterVec.With(map[string]string{endpointLabelKey: endpoint}).Inc()
}

func (p *pingFailedCounter) Describe(descC chan<- *prometheus.Desc) {
	p.counterVec.Describe(descC)
}

func (p *pingFailedCounter) Collect(metricC chan<- prometheus.Metric) {
	p.counterVec.Collect(metricC)
}
