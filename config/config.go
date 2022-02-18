package config

import (
	"gopkg.in/yaml.v2"
	"ping_exporter/ping"
	"time"
)

var (
	DefaultPingTimeout        = time.Second
	DefaultPingInterval       = time.Second
	DefaultAddr               = ":2112"
	DefaultNetwork            = ping.Ipv4
	DefaultRttHistogramBucket = []float64{5, 10, 20, 50, 100, 200, 300, 400, 500, 700, 1000}
)

type Config struct {
	Endpoints          []string
	Ping               Ping
	Addr               string
	Network            string
	RttHistogramBucket []float64 `yaml:"rtt_histogram_bucket"`
}

type Ping struct {
	Timeout  time.Duration
	Interval time.Duration
}

func PhraseWithYaml(file string) (*Config, error) {
	config := Config{}
	err := yaml.Unmarshal([]byte(file), &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
