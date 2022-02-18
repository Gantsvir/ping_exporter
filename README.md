# ping_exporter
A Prometheus export that uses ping to monitor network quality

## usage
- **build**: `go build`
- **run**: `sudo ./ping_exporter`  
send icmp packet must have root privileges.  
  

## configuration
See config.yml

## metrics
- **ping_rtt**: round-trip time of icmp packets, in milliseconds  
- **ping_failed_count**: total number of ping failures  
- **ping_timeout_ocunt**: total number of ping timeout  

## http url
The default URL is http://{host}:2112/metrics

## prometheus example
- **packet loss rate**:   
```rate(ping_timeout_count[1m]) / (rate(ping_rtt_count[1m]) + rate(ping_timeout_count[1m]))```
- **rtt over 30ms rate**:   
```(rate(ping_rtt_count[1m]) - sum without(le) (rate(ping_rtt_bucket{le="30"}[1m]))) / rate(ping_rtt_count[1m])```
