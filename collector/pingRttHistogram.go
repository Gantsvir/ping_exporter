package collector

import (
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

func NewPingRttHistogram(endpoints []string, opts prometheus.HistogramOpts) (PingCollector, error) {
	if _, ok := opts.ConstLabels[endpointLabelKey]; ok {
		return nil, errors.Errorf("can not use %s as label key", endpointLabelKey)
	}

	collector := &pingRttHistogram{}

	collector.histogramVec = prometheus.NewHistogramVec(
		opts,
		[]string{endpointLabelKey},
	)
	for _, ep := range endpoints {
		collector.histogramVec.With(map[string]string{endpointLabelKey: ep})
	}

	return collector, nil
}

type pingRttHistogram struct {
	histogramVec *prometheus.HistogramVec
}

func (p *pingRttHistogram) OnSucceed(endpoint string, time time.Duration, id int) {
	p.histogramVec.With(map[string]string{endpointLabelKey: endpoint}).Observe(time.Seconds() * 1000)
}

func (p *pingRttHistogram) OnTimeout(endpoint string, id int) {

}

func (p *pingRttHistogram) OnFailed(endpoint string, err error) {

}

func (p *pingRttHistogram) Describe(descC chan<- *prometheus.Desc) {
	p.histogramVec.Describe(descC)
}

func (p *pingRttHistogram) Collect(metricC chan<- prometheus.Metric) {
	p.histogramVec.Collect(metricC)
}
