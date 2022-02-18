package ping

import (
	"context"
	"errors"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"math"
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"
)

const (
	ProtocolICMP   = 1
	ProtocolICMPv6 = 58
)

type Network string

const (
	Ipv4 Network = "ip4:icmp"
	Ipv6 Network = "ip6:ipv6-icmp"
)

type ReplyHandler interface {
	OnSucceed(endpoint string, time time.Duration, id int)

	OnTimeout(endpoint string, id int)

	OnFailed(endpoint string, err error)
}

type Pinger struct {
	conn       net.PacketConn
	writeMutex sync.Mutex

	network              Network
	handlers             []ReplyHandler
	endpoints            []string
	endpointsIdMap       map[string]uint16
	endpointsSequenceMap map[string]uint16
	endpointMtx          sync.Mutex
	responseMap          map[responseKey]response
	responseMtx          sync.RWMutex
}

type responseKey struct {
	id       uint16
	sequence uint16
}

type response struct {
	ch  chan struct{}
	err error
}

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

func New(network Network, address string, endpoints []string, handlers []ReplyHandler) (*Pinger, error) {
	if network != Ipv4 && network != Ipv6 {
		return nil, errors.New("illegal network")
	}

	if len(endpoints) > math.MaxUint16 {
		return nil, errors.New("endpoints number out of limit")
	}

	conn, err := icmp.ListenPacket(string(network), address)
	if err != nil {
		return nil, err
	}

	epIdMap := make(map[string]uint16, len(endpoints))
	epSeMap := make(map[string]uint16, len(endpoints))

	for i, endpoint := range endpoints {
		epIdMap[endpoint] = uint16(i)
		epSeMap[endpoint] = 0
	}

	var protocol int
	if network == Ipv4 {
		protocol = ProtocolICMP
	} else {
		protocol = ProtocolICMPv6
	}

	pinger := Pinger{
		network:              network,
		conn:                 conn,
		handlers:             handlers,
		endpoints:            endpoints,
		endpointsIdMap:       epIdMap,
		endpointsSequenceMap: epSeMap,
		responseMap:          make(map[responseKey]response),
	}

	go pinger.receiver(protocol, conn)

	return &pinger, nil
}

func (pinger *Pinger) Start(timeout, interval time.Duration) {
	for range time.NewTicker(interval).C {
		go pinger.pingAll(timeout)
	}
}

func (pinger *Pinger) Close() error {
	if err := pinger.conn.Close(); err != nil {
		return err
	}
	return nil
}

func (pinger *Pinger) pingAll(timeout time.Duration) {
	for _, endpoint := range pinger.endpoints {
		go func(ep string) {
			start := time.Now()
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
			defer cancel()
			id, err := pinger.request(ctx, ep)
			duration := time.Now().Sub(start)

			for _, handler := range pinger.handlers {
				if err != nil {
					if _, ok := err.(*timeoutError); ok {
						handler.OnTimeout(ep, id)
					} else {
						handler.OnFailed(ep, err)
					}
				} else {
					handler.OnSucceed(ep, duration, id)
				}
			}

		}(endpoint)
	}
}

func (pinger *Pinger) request(ctx context.Context, endpoint string) (int, error) {
	pinger.endpointMtx.Lock()

	id := pinger.endpointsIdMap[endpoint]
	sequence := pinger.endpointsSequenceMap[endpoint]
	sequence++
	pinger.endpointsSequenceMap[endpoint] = sequence

	pinger.endpointMtx.Unlock()

	var icmpType icmp.Type
	if pinger.network == Ipv4 {
		icmpType = ipv4.ICMPTypeEcho
	} else {
		icmpType = ipv6.ICMPTypeEchoRequest
	}

	wm := icmp.Message{
		Type: icmpType,
		Code: 0,
		Body: &icmp.Echo{
			ID:   int(id),
			Seq:  int(sequence),
			Data: make([]byte, 0),
		},
	}

	destination, err := net.ResolveIPAddr("ip", endpoint)
	if err != nil {
		return int(id), err
	}

	wb, err := wm.Marshal(nil)
	if err != nil {
		return int(sequence), err
	}

	pinger.responseMtx.Lock()
	response := response{
		ch: make(chan struct{}),
	}
	responseKey := responseKey{id: id, sequence: sequence}
	pinger.responseMap[responseKey] = response
	pinger.responseMtx.Unlock()

	defer func() {
		pinger.responseMtx.Lock()
		delete(pinger.responseMap, responseKey)
		pinger.responseMtx.Unlock()
	}()

	pinger.writeMutex.Lock()
	if _, err := pinger.conn.WriteTo(wb, destination); err != nil {
		pinger.writeMutex.Unlock()
		return int(sequence), err
	}
	pinger.writeMutex.Unlock()

	select {
	case <-response.ch:
		if response.err != nil {
			return int(id), err
		}
		return int(sequence), nil
	case <-ctx.Done():
		return int(sequence), &timeoutError{}
	}
}

func (pinger *Pinger) receiver(proto int, conn net.PacketConn) {
	rb := make([]byte, 1500)

	for {
		if n, _, err := conn.ReadFrom(rb); err != nil {
			if netErr, ok := err.(net.Error); !ok || !netErr.Temporary() {
				break
			}
		} else {
			pinger.receive(proto, rb[:n])
		}
	}
}

func (pinger *Pinger) receive(proto int, bytes []byte) {
	// parse message
	m, err := icmp.ParseMessage(proto, bytes)
	if err != nil {
		return
	}

	// evaluate message
	switch m.Type {
	case ipv4.ICMPTypeEchoReply, ipv6.ICMPTypeEchoReply:
		pinger.process(m.Body)

	case ipv4.ICMPTypeDestinationUnreachable, ipv6.ICMPTypeDestinationUnreachable:
		body := m.Body.(*icmp.DstUnreach)
		if body == nil {
			return
		}

		var bodyData []byte
		switch proto {
		case ProtocolICMP:
			hdr, err := ipv4.ParseHeader(body.Data)
			if err != nil {
				return
			}
			bodyData = body.Data[hdr.Len:]
		case ProtocolICMPv6:
			_, err := ipv6.ParseHeader(body.Data)
			if err != nil {
				return
			}
			bodyData = body.Data[ipv6.HeaderLen:]
		default:
			return
		}

		msg, err := icmp.ParseMessage(proto, bodyData)
		if err != nil {
			return
		}
		pinger.process(msg.Body)
	}
}

func (pinger *Pinger) process(body icmp.MessageBody) {
	echo, ok := body.(*icmp.Echo)
	if !ok || echo == nil {
		return
	}
	pinger.responseMtx.Lock()
	responseKey := responseKey{id: uint16(echo.ID), sequence: uint16(echo.Seq)}
	if response, ok := pinger.responseMap[responseKey]; ok {
		response.ch <- struct{}{}
	}
	pinger.responseMtx.Unlock()
}
