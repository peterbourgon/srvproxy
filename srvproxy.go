package srvproxy

import (
	"fmt"
	"net/http"

	"github.com/peterbourgon/srvproxy/pool"
	"github.com/peterbourgon/srvproxy/resolve"
)

// RoundTripper yields a proxying RoundTripper.
// Pass it to http.Transport.RegisterProtocol.
func RoundTripper(opts ...OptionFunc) http.RoundTripper {
	t := &transport{
		next:        http.DefaultTransport,
		resolver:    resolve.ResolverFunc(resolve.DNSSRV),
		poolFactory: pool.RoundRobin,
		poolSuccess: pool.SimpleSuccess,
		registry:    nil,
	}
	t.setOptions(opts...)
	t.registry = newRegistry(t.resolver, t.poolFactory)
	return t
}

type transport struct {
	next        http.RoundTripper
	resolver    resolve.Resolver
	poolFactory pool.Factory
	poolSuccess pool.SuccessFunc
	registry    *registry
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

	resp, err := t.next.RoundTrip(&newreq)
	pool.Put(host, t.poolSuccess(resp, err))
	return resp, err
}

// OptionFunc sets a specific option for the transport. This is the functional
// options idiom. See https://www.youtube.com/watch?v=24lFtGHWxAQ for more
// information.
type OptionFunc func(*transport)

// Next sets the http.RoundTripper that's used to transport reconstructed HTTP
// requests. If Next isn't provided, http.DefaultTransport is used.
func Next(rt http.RoundTripper) OptionFunc {
	return func(t *transport) { t.next = rt }
}

// Resolver sets which name resolver will be used. If Resolver isn't provided,
// a DNS SRV resolver is used.
func Resolver(r resolve.Resolver) OptionFunc {
	return func(t *transport) { t.resolver = r }
}

// PoolFactory sets which type of pool will be used. If PoolFactory isn't
// provided, the RoundRobin pool is used.
func PoolFactory(f pool.Factory) OptionFunc {
	return func(t *transport) { t.poolFactory = f }
}

// PoolSuccess sets the function that determines if a HTTP response against a
// specific host in a pool should be considered successful. That result may
// optionally be used by the pool to influence how it yields hosts in the
// future. If PoolSuccess isn't provided, pool.SimpleSuccess is used.
func PoolSuccess(f pool.SuccessFunc) OptionFunc {
	return func(t *transport) { t.poolSuccess = f }
}

func (t *transport) setOptions(opts ...OptionFunc) {
	for _, f := range opts {
		f(t)
	}
}
