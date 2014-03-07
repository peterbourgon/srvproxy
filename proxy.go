package srvproxy

import (
	"log"
	"math/rand"
	"time"
)

// Stopper is anything that should be stopped.
type Stopper interface {
	Stop()
}

// SingleProxy is anything that yields endpoints, one-by-one.
type SingleProxy interface {
	Endpoint() Endpoint
	Stopper
}

// BulkProxy is anything that can yield the latest set of endpoints.
type BulkProxy interface {
	Endpoints() []Endpoint
	Stopper
}

// StreamingProxy is anything that yields endpoints as they become available.
// Only one client should receive from a streaming channel simultaneously.
type StreamingProxy interface {
	Stream() <-chan []Endpoint
	Stopper
}

// proxy implements SingleProxy, BulkProxy, StreamingProxy, and Stopper.
type proxy struct {
	requests chan []Endpoint
	stream   chan []Endpoint
	quit     chan chan struct{}
}

// NewProxy returns a proxy satisfying both single and bulk proxy interfaces.
// It regularly resolves the name using the resolver, on the poll interval.
func NewProxy(name string, resolver Resolver, pollInterval time.Duration) *proxy {
	p := &proxy{
		requests: make(chan []Endpoint),
		stream:   make(chan []Endpoint),
		quit:     make(chan chan struct{}),
	}
	go p.run(name, resolver, pollInterval)
	return p
}

func (p *proxy) run(name string, resolver Resolver, pollInterval time.Duration) {
	tick := time.Tick(pollInterval)
	var endpoints []Endpoint
	for {
		select {
		case <-tick:
			ep, err := resolver(name)
			if err != nil {
				log.Printf("srvproxy: %s", err)
				continue // don't replace potentially good endpoints with bad
			}
			// TODO check it's different
			endpoints = ep
			select {
			case p.stream <- endpoints:
			default: // best effort
			}

		case p.requests <- endpoints:

		case q := <-p.quit:
			close(q)
			return
		}
	}
}

// Endpoint implements SingleProxy.
func (p *proxy) Endpoint() Endpoint {
	endpoints := p.Endpoints()
	return endpoints[rand.Intn(len(endpoints))]
}

// Endpoints implements BulkProxy.
func (p *proxy) Endpoints() []Endpoint {
	return <-p.requests
}

// Stream implements StreamingProxy.
func (p *proxy) Stream() <-chan []Endpoint {
	return p.stream
}

// Stop implements Stopper.
func (p *proxy) Stop() {
	q := make(chan struct{})
	p.quit <- q
	<-q
}
