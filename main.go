package main

import (
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"net/http"
	"os"
	"ping_exporter/collector"
	"ping_exporter/config"
	"ping_exporter/ping"
	"time"
)

func main() {
	var configFilePath string
	flag.StringVar(&configFilePath, "config-file", "config.yml", "配置文件")
	flag.Parse()

	if configFilePath == "" {
		fmt.Fprintf(os.Stderr, "config-file must be specified")
		os.Exit(-1)
	}

	file, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read config-file failed: %v", err)
		os.Exit(-1)
	}

	cf, err := config.PhraseWithYaml(string(file))
	if err != nil {
		fmt.Fprintf(os.Stderr, "phrase config-file failed: %v", err)
		os.Exit(-1)
	}

	var bucket []float64
	if cf.RttHistogramBucket != nil {
		bucket = cf.RttHistogramBucket
	} else {
		bucket = config.DefaultRttHistogramBucket
	}

	pingSucceed, err := collector.NewPingRttHistogram(cf.Endpoints, prometheus.HistogramOpts{
		Name:    "ping_rtt",
		Buckets: bucket,
	})

	pingTimeout, err := collector.NewPingTimeoutCounter(cf.Endpoints, prometheus.CounterOpts{
		Name: "ping_timeout_count",
	})

	pingFailed, err := collector.NewPingFailedCounter(cf.Endpoints, prometheus.CounterOpts{
		Name: "ping_failed_count",
	})
	prometheus.MustRegister(pingSucceed, pingTimeout, pingFailed)

	var network ping.Network
	if cf.Network != "" {
		network = ping.Network(cf.Network)
	} else {
		network = config.DefaultNetwork
	}

	pinger, err := ping.New(network, "", cf.Endpoints, []ping.ReplyHandler{pingTimeout, pingSucceed, pingFailed})
	if err != nil {
		fmt.Fprintf(os.Stderr, "create pinger failed: %v", err)
		os.Exit(-1)
	}

	var timeout time.Duration
	if cf.Ping.Timeout != 0 {
		timeout = cf.Ping.Timeout
	} else {
		timeout = config.DefaultPingTimeout
	}

	var interval time.Duration
	if cf.Ping.Timeout != 0 {
		interval = cf.Ping.Interval
	} else {
		interval = config.DefaultPingInterval
	}

	go func() {
		pinger.Start(timeout, interval)
	}()

	http.Handle("/metrics", promhttp.Handler())

	var addr string
	if cf.Addr != "" {
		addr = cf.Addr
	} else {
		addr = config.DefaultAddr
	}

	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintf(os.Stderr, "start http server failed: %v", err)
		os.Exit(-1)
	}
}
