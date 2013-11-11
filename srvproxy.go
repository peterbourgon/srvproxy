package srvproxy

import (
	"errors"
	"log"
	"time"
)

var (
	errNoEndpoints = errors.New("no endpoints available")
)

// Proxy does regular resolve lookups against a defined name string, and caches
// the list of returned endpoints. The Endpoint method round-robins through the
// currently active set of endpoints.
type Proxy struct {
	endpoints        []Endpoint
	endpointRequests chan endpointRequest
	quit             chan chan struct{}
}

// New returns a new SRV Proxy, using the resolver to regularly transform the
// name string on the poll interval.
func New(resolve Resolver, name string, pollInterval time.Duration) (*Proxy, error) {
	endpoints, err := resolve(name)
	if err != nil {
		return nil, err
	}
	p := &Proxy{
		endpoints:        endpoints,
		endpointRequests: make(chan endpointRequest),
		quit:             make(chan chan struct{}),
	}
	go p.loop(resolve, name, pollInterval)
	return p, nil
}

// Stop terminates the Proxy polling goroutine. Stopped proxies can no longer
// serve URL requests.
func (p *Proxy) Stop() {
	q := make(chan struct{})
	p.quit <- q
	<-q
}

// Endpoint returns the next endpoint (round-robin) from the cached set of
// endpoints. The client must expect that the endpoint can be non-responsive.
func (p *Proxy) Endpoint() (Endpoint, error) {
	req := endpointRequest{make(chan Endpoint), make(chan error)}
	p.endpointRequests <- req
	select {
	case endpoint := <-req.endpoint:
		return endpoint, nil
	case err := <-req.err:
		return Endpoint{}, err
	}
}

func (p *Proxy) loop(resolve Resolver, name string, interval time.Duration) {
	tick := time.Tick(interval)
	index := 0
	for {
		select {
		case <-tick:
			endpoints, err := resolve(name)
			if err != nil {
				log.Printf("srvproxy: poll: %s", err)
			}
			p.endpoints = endpoints

		case req := <-p.endpointRequests:
			if len(p.endpoints) <= 0 {
				req.err <- errNoEndpoints
				continue
			}
			index = (index + 1) % len(p.endpoints)
			req.endpoint <- p.endpoints[index]

		case q := <-p.quit:
			close(q)
			return
		}
	}
}

type endpointRequest struct {
	endpoint chan Endpoint
	err      chan error
}
