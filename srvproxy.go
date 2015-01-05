package srvproxy

import (
	"io"
	"io/ioutil"
	stdhttp "net/http"
	"time"

	"github.com/peterbourgon/srvproxy/http"
	"github.com/peterbourgon/srvproxy/pool"
	"github.com/peterbourgon/srvproxy/resolve"
)

// New returns a client capable of executing HTTP requests against a set of
// backends continuously resolved from the passed name.
func New(name string, opts ...OptionFunc) http.Client {
	o := makeOptions(opts...)

	var p pool.Pool
	p = pool.Streaming(o.resolver, name, o.poolFactory)
	p = pool.Instrumented(p)

	var d http.Director
	d = http.DirectorFunc(pool.Director(p, o.poolSuccess))

	var c http.Client
	c = o.client
	c = http.Directed(d, c)
	c = http.Retrying(o.maxAttempts, o.timeout, o.responseValidator, c)
	c = http.Instrumented(c)
	c = http.Report(o.reportWriter, c)

	return c
}

// OptionFunc sets a specific option for the srvproxy. This is the functional
// options idiom. See https://www.youtube.com/watch?v=24lFtGHWxAQ for more
// information.
type OptionFunc func(*options)

// Resolver sets the resolver, used to translate the name to a set of hosts.
//
// If Resolver isn't provided, a DNS SRV resolver is used.
func Resolver(r resolve.Resolver) OptionFunc {
	return func(o *options) { o.resolver = r }
}

// PoolFactory sets the pool factory function, which converts a set of hosts
// to a single pool.
//
// If PoolFactory isn't provided, a pool factory that generates priority-queue
// based pools is used.
func PoolFactory(f pool.Factory) OptionFunc {
	return func(o *options) { o.poolFactory = f }
}

// PoolSuccess sets the function that determines if a HTTP response against a
// specific host in a pool should be considered successful. That result may
// optionally be used by the pool to influence how it yields hosts in the
// future.
//
// If PoolSuccess isn't provided, any valid HTTP response (any status code) is
// considered successful.
func PoolSuccess(f pool.SuccessFunc) OptionFunc {
	return func(o *options) { o.poolSuccess = f }
}

// Client sets the default, underlying HTTP client used to send requests.
//
// If Client isn't provided, http.DefaultClient is used.
func Client(c http.Client) OptionFunc {
	return func(o *options) { o.client = c }
}

// MaxAttempts sets the number of attempts to complete a single HTTP request.
// If MaxAttempts isn't provided, the default value is 3.
func MaxAttempts(n int) OptionFunc {
	return func(o *options) { o.maxAttempts = n }
}

// Timeout sets the maximum time to complete a single HTTP request. If Timeout
// isn't provided, the default value is zero, i.e. no timeout
func Timeout(d time.Duration) OptionFunc {
	return func(o *options) { o.timeout = d }
}

// ResponseValidator sets the function that determines if a HTTP response is
// valid and may be returned to the client. That is, if ResponseValidator
// returns a non-nil error, the request may be retried.
//
// If ResponseValidator isn't provided, http.SimpleValidator is used.
func ResponseValidator(f http.ValidateFunc) OptionFunc {
	return func(o *options) { o.responseValidator = f }
}

// ReportWriter sets the io.Writer which will receive JSON reports for each
// completed HTTP request. If ReportWriter isn't provided, ioutil.Discard is
// used.
func ReportWriter(w io.Writer) OptionFunc {
	return func(o *options) { o.reportWriter = w }
}

type options struct {
	resolver          resolve.Resolver
	poolFactory       pool.Factory
	poolSuccess       pool.SuccessFunc
	client            http.Client
	maxAttempts       int
	timeout           time.Duration
	responseValidator http.ValidateFunc
	reportWriter      io.Writer
}

func makeOptions(opts ...OptionFunc) options {
	o := options{
		resolver:          resolve.ResolverFunc(resolve.DNSSRV),
		poolFactory:       pool.RoundRobin,
		poolSuccess:       pool.SimpleSuccess,
		client:            stdhttp.DefaultClient,
		maxAttempts:       3,
		timeout:           0,
		responseValidator: http.SimpleValidator,
		reportWriter:      ioutil.Discard,
	}

	for _, f := range opts {
		f(&o)
	}

	return o
}
