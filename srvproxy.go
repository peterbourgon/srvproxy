package srvproxy

import (
	stdhttp "net/http"
	"time"

	"github.com/peterbourgon/srvproxy/http"
	"github.com/peterbourgon/srvproxy/pool"
	"github.com/peterbourgon/srvproxy/resolve"
)

// New returns a client capable of executing HTTP requests against a set of
// backends continuously resolved from the passed name.
func New(name string, options Options) http.Client {
	options = options.defaults()

	var p pool.Pool
	p = pool.Streaming(options.Resolver, name, options.PoolFactory)
	p = pool.Instrumented(p)

	var d http.Director
	d = http.DirectorFunc(pool.Director(p, options.PoolSuccess))

	var c http.Client
	c = options.Client
	c = http.Directed(d, c)
	c = http.Retrying(options.MaxAttempts, options.Cutoff, options.ResponseValidator, c)
	c = http.Instrumented(c)

	return c
}

// Options configures the client.
type Options struct {
	// Resolver determines how the name is resolved to a set of hosts.
	// If Resolver is nil, a DNS SRV resolver is used.
	Resolver resolve.Resolver

	// PoolFactory determines how a set of hosts is transformed into a pool,
	// which provides get and put semantics. If PoolFactory is nil, a naïve
	// round-robin pool is used.
	PoolFactory pool.Factory

	// PoolSuccess maps the result of an HTTP request against a specific host
	// to success or failure. That result may optionally be used by the pool
	// to influence how it yields hosts in the future. If PoolSuccess is nil,
	// any valid HTTP response (any status code) is considered a success.
	PoolSuccess pool.SuccessFunc

	// Client is the base http.Client used to send requests. If Client is nil,
	// http.DefaultClient is used.
	Client http.Client

	// MaxAttempts is how many times to attempt each HTTP request against
	// sequential hosts from the Pool. The default value is 3.
	MaxAttempts int

	// Cutoff determines the deadline for an individual request, after which
	// no more attempts will be made, and the request aborted. The default is
	// no cutoff.
	Cutoff time.Duration

	// ResponseValidator determines if an HTTP response is considered valid
	// and can be returned to the calling context. That is, if
	// ResponseValidator returns a non-nil error for an HTTP response, the
	// request may be retried.
	//
	// If ResponseValidator is nil, any HTTP response with a 1xx, 2xx, 3xx, or
	// 4xx status code will be considered valid, and won't be retried.
	ResponseValidator http.ValidateFunc
}

func (o Options) defaults() Options {
	if o.Resolver == nil {
		o.Resolver = resolve.ResolverFunc(resolve.DNSSRV)
	}

	if o.PoolFactory == nil {
		o.PoolFactory = pool.RoundRobin
	}

	if o.PoolSuccess == nil {
		o.PoolSuccess = pool.SimpleSuccess
	}

	if o.Client == nil {
		o.Client = stdhttp.DefaultClient
	}

	if o.MaxAttempts <= 0 {
		o.MaxAttempts = 3
	}

	if o.ResponseValidator == nil {
		o.ResponseValidator = http.SimpleValidator
	}

	return o
}
