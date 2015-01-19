package srvproxy

import (
	"fmt"
	"io"
	"net/http"

	"github.com/peterbourgon/srvproxy/pool"
	"github.com/peterbourgon/srvproxy/resolve"
)

// RoundTripper yields a proxying RoundTripper.
// Pass it to http.Transport.RegisterProtocol.
func RoundTripper(opts ...ProxyOption) http.RoundTripper {
	t := &transport{
		next:         http.DefaultTransport,
		resolver:     resolve.ResolverFunc(resolve.DNSSRV),
		poolReporter: nil,
		poolFactory:  pool.RoundRobin,
		registry:     nil,
	}
	t.setOptions(opts...)
	t.registry = newRegistry(t.resolver, t.poolReporter, t.poolFactory)
	return t
}

type transport struct {
	next         http.RoundTripper
	resolver     resolve.Resolver
	poolReporter io.Writer
	poolFactory  pool.Factory
	registry     *registry
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	pool := t.registry.get(req.URL.Host)
	host, err := pool.Get()
	if err != nil {
		return nil, fmt.Errorf("couldn't send request: %v", err)
	}

	// "RoundTrip should not modify the request, except for
	// consuming and closing the Body, including on errors."
	// -- http://golang.org/pkg/net/http/#RoundTripper
	newurl := (*req.URL)
	newurl.Scheme = "http"
	newurl.Host = host
	newreq := (*req)
	newreq.URL = &newurl

	return t.next.RoundTrip(&newreq)
}

// ProxyOption sets a specific option for the RoundTripper. This is the
// functional options idiom. See https://www.youtube.com/watch?v=24lFtGHWxAQ
// for more information.
type ProxyOption func(*transport)

// ProxyNext sets the http.RoundTripper that's used to transport reconstructed
// HTTP requests. If Next isn't provided, http.DefaultTransport is used.
func ProxyNext(rt http.RoundTripper) ProxyOption {
	return func(t *transport) { t.next = rt }
}

// Resolver sets which name resolver will be used. If Resolver isn't provided,
// a DNS SRV resolver is used.
func Resolver(r resolve.Resolver) ProxyOption {
	return func(t *transport) { t.resolver = r }
}

// PoolReporter sets the destination where the pool will report each
// invocation as JSON-encoded events. If PoolReporter isn't provided, the pool
// won't report any information.
func PoolReporter(w io.Writer) ProxyOption {
	return func(t *transport) { t.poolReporter = w }
}

// PoolFactory sets which type of pool will be used. If PoolFactory isn't
// provided, the RoundRobin pool is used.
func PoolFactory(f pool.Factory) ProxyOption {
	return func(t *transport) { t.poolFactory = f }
}

func (t *transport) setOptions(opts ...ProxyOption) {
	for _, f := range opts {
		f(t)
	}
}
