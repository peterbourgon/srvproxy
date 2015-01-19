package roundtrip

import (
	"fmt"
	"io"
	"net/http"

	"github.com/peterbourgon/srvproxy/pool"
	"github.com/peterbourgon/srvproxy/resolve"
)

// Proxy yields a proxying RoundTripper.
// Pass it to http.Transport.RegisterProtocol.
func Proxy(options ...ProxyOption) http.RoundTripper {
	p := &proxy{
		next:         http.DefaultTransport,
		scheme:       "http",
		resolver:     resolve.ResolverFunc(resolve.DNSSRV),
		poolReporter: nil,
		factory:      pool.RoundRobin,
		registry:     nil,
	}
	p.setOptions(options...)
	p.registry = newRegistry(p.resolver, p.poolReporter, p.factory)
	return p
}

type proxy struct {
	next         http.RoundTripper
	scheme       string
	resolver     resolve.Resolver
	poolReporter io.Writer
	factory      pool.Factory
	registry     *registry
}

func (p *proxy) RoundTrip(req *http.Request) (*http.Response, error) {
	host, err := p.registry.get(req.URL.Host).Get()
	if err != nil {
		return nil, fmt.Errorf("couldn't send request: %v", err)
	}

	// "RoundTrip should not modify the request, except for
	// consuming and closing the Body, including on errors."
	// -- http://golang.org/pkg/net/http/#RoundTripper
	newurl := (*req.URL)
	newurl.Scheme = p.scheme
	newurl.Host = host
	newreq := (*req)
	newreq.URL = &newurl

	return p.next.RoundTrip(&newreq)
}

func (p *proxy) setOptions(options ...ProxyOption) {
	for _, f := range options {
		f(p)
	}
}

// ProxyOption sets a specific option for the Proxy. This is the functional
// options idiom. See https://www.youtube.com/watch?v=24lFtGHWxAQ for more
// information.
type ProxyOption func(*proxy)

// Scheme sets the protocol scheme, probably "http" or "https". If no scheme
// is provided, "http" is used.
func Scheme(scheme string) ProxyOption {
	return func(p *proxy) { p.scheme = scheme }
}

// ProxyNext sets the http.RoundTripper that's used to transport reconstructed
// HTTP requests. If Next isn't provided, http.DefaultTransport is used.
func ProxyNext(rt http.RoundTripper) ProxyOption {
	return func(p *proxy) { p.next = rt }
}

// Resolver sets which name resolver will be used. If Resolver isn't provided,
// a DNS SRV resolver is used.
func Resolver(r resolve.Resolver) ProxyOption {
	return func(p *proxy) { p.resolver = r }
}

// PoolReporter sets the destination where the pool will report each
// invocation as JSON-encoded events. If PoolReporter isn't provided, the pool
// won't report any information.
func PoolReporter(w io.Writer) ProxyOption {
	return func(p *proxy) { p.poolReporter = w }
}

// Factory sets which type of pool will be used. If Factory isn't provided,
// the RoundRobin pool is used.
func Factory(f pool.Factory) ProxyOption {
	return func(p *proxy) { p.factory = f }
}
