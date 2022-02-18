package collector

import (
	"github.com/prometheus/client_golang/prometheus"
	"ping_exporter/ping"
)

const (
	endpointLabelKey = "endpoint"
)

type PingCollector interface {
	prometheus.Collector
	ping.ReplyHandler
}
