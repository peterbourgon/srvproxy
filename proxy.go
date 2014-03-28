package srvproxy

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"
)

// Stopper is anything that should be stopped.
type Stopper interface {
	Stop()
}

// SingleProxy is anything that yields endpoints, one-by-one.
type SingleProxy interface {
	Endpoint() (Endpoint, error)
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

// ErrNoEndpointAvailable is returned by SingleProxy when the underlying
// resource yields no endpoints.
var ErrNoEndpointAvailable = errors.New("no endpoint available")

// NewProxy returns a proxy satisfying both single and bulk proxy interfaces.
// It regularly resolves the name using the resolver, on the poll interval.
func NewProxy(name string, resolver Resolver, pollInterval time.Duration) (*proxy, error) {
	endpoints, err := resolver(name)
	if err != nil {
		return nil, err
	}
	p := &proxy{
		requests: make(chan []Endpoint),
		stream:   make(chan []Endpoint, 1), // pre-seed with initial endpoints
		quit:     make(chan chan struct{}),
	}
	p.stream <- endpoints
	go p.run(name, resolver, pollInterval, endpoints)
	return p, nil
}

func (p *proxy) run(name string, resolver Resolver, pollInterval time.Duration, endpoints []Endpoint) {
	tick := time.Tick(pollInterval)
	for {
		select {
		case <-tick:
			newEndpoints, err := resolver(name)
			if err != nil {
				log.Printf("srvproxy: %s", err)
				continue // don't replace potentially good endpoints with bad
			}
			if same(newEndpoints, endpoints) {
				continue // no need to update
			}
			endpoints = newEndpoints
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
func (p *proxy) Endpoint() (Endpoint, error) {
	endpoints := p.Endpoints()
	if len(endpoints) <= 0 {
		return Endpoint{}, ErrNoEndpointAvailable
	}
	return endpoints[rand.Intn(len(endpoints))], nil
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

func same(a, b []Endpoint) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Sort(endpointSlice(a))
	sort.Sort(endpointSlice(b))
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type endpointSlice []Endpoint

func (a endpointSlice) Len() int           { return len(a) }
func (a endpointSlice) Less(i, j int) bool { return fmt.Sprint(a[i]) < fmt.Sprint(a[j]) }
func (a endpointSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
