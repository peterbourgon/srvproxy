package srvproxy

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/miekg/dns"
	"github.com/soundcloud/go-dns-resolver/resolv"
)

var (
	errNoEndpoints = errors.New("no endpoints available")
)

// Proxy does regular DNS SRV lookups against a defined name string, and caches
// the list of returned targets and ports as individual endpoints. The URL
// method round-robins through the currently active set of endpoints.
type Proxy struct {
	name        string // DNS lookup string
	endpoints   []*url.URL
	urlRequests chan urlRequest
	quit        chan chan struct{}
}

// New returns a new SRV Proxy, polling the given name string at the given
// interval.
func New(name string, pollInterval time.Duration) (*Proxy, error) {
	endpoints, err := pollSRV(name)
	if err != nil {
		return nil, err
	}
	p := &Proxy{
		name:        name,
		endpoints:   endpoints,
		urlRequests: make(chan urlRequest),
		quit:        make(chan chan struct{}),
	}
	go p.loop(pollInterval)
	return p, nil
}

// Stop terminates the Proxy polling goroutine. Stopped proxies can no longer
// serve URL requests.
func (p *Proxy) Stop() {
	q := make(chan struct{})
	p.quit <- q
	<-q
}

// URL returns the next endpoint (round-robin) from the cached set of endpoints.
func (p *Proxy) URL() (string, error) {
	req := urlRequest{make(chan string), make(chan error)}
	p.urlRequests <- req
	select {
	case rawurl := <-req.rawurl:
		return rawurl, nil
	case err := <-req.err:
		return "", err
	}
}

func (p *Proxy) loop(pollInterval time.Duration) {
	tick := time.Tick(pollInterval)
	index := 0
	for {
		select {
		case <-tick:
			endpoints, err := pollSRV(p.name)
			if err != nil {
				log.Printf("srvproxy: poll: %s", err)
			}
			p.endpoints = endpoints

		case req := <-p.urlRequests:
			if len(p.endpoints) <= 0 {
				req.err <- errNoEndpoints
				continue
			}
			index = (index + 1) % len(p.endpoints)
			req.rawurl <- p.endpoints[index].String()

		case q := <-p.quit:
			close(q)
			return
		}
	}
}

type urlRequest struct {
	rawurl chan string
	err    chan error
}

func pollSRV(name string) ([]*url.URL, error) {
	msg, err := resolv.LookupString("SRV", name)
	if err != nil {
		return []*url.URL{}, err
	}

	endpoints := []*url.URL{}
	for _, rr := range msg.Answer {
		if srv, ok := rr.(*dns.SRV); ok {
			endpoint, err := url.Parse(fmt.Sprintf("http://%s:%d", srv.Target, srv.Port))
			if err != nil {
				log.Printf("srvproxy: %s: %s", srv.String(), err)
				continue
			}
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints, nil
}
