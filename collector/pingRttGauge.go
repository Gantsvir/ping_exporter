package collector

import (
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

func NewPingRttGauge(endpoints []string, opts prometheus.GaugeOpts) (PingCollector, error) {
	if _, ok := opts.ConstLabels[endpointLabelKey]; ok {
		return nil, errors.Errorf("can not use %s as label key", endpointLabelKey)
	}

	collector := &pingRttGauge{}

	collector.gaugeVec = prometheus.NewGaugeVec(
		opts,
		[]string{endpointLabelKey},
	)

	return collector, nil
}

type pingRttGauge struct {
	gaugeVec *prometheus.GaugeVec
}

func (p *pingRttGauge) OnSucceed(endpoint string, time time.Duration, id int) {
	p.gaugeVec.With(map[string]string{endpointLabelKey: endpoint}).Set(time.Seconds() * 1000)
}

func (p *pingRttGauge) OnTimeout(endpoint string, id int) {

}

func (p *pingRttGauge) OnFailed(endpoint string, err error) {

}

func (p *pingRttGauge) Describe(descC chan<- *prometheus.Desc) {
	p.gaugeVec.Describe(descC)
}

func (p *pingRttGauge) Collect(metrics chan<- prometheus.Metric) {
	p.gaugeVec.Collect(metrics)
}
