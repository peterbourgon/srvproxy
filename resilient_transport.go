package srvproxy

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/streadway/handy/breaker"
	proxypkg "github.com/streadway/handy/proxy"
)

var (
	// ErrNoTransportAvailable is returned by the choosing
	// transport when no underlying transport will allow the request.
	ErrNoTransportAvailable = errors.New("no successful transport available")
)

// retryTransport forwards the request to each next RoundTripper in random
// sequence. It will return the first validated response it receives. It will
// abort when it exceeds the max attempt count, or the cutoff deadline, or it
// exhausts the next RoundTrippers, whichever occurs first.
type retryTransport struct {
	next     []http.RoundTripper
	validate breaker.ResponseValidator
	cutoff   time.Duration
	max      int
}

func (t retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	deadline := time.Now().Add(t.cutoff)
	errs := []string{}
	for attempt, index := range rand.Perm(len(t.next)) {
		if attempt >= t.max {
			errs = append(errs, "too many attempts")
			break
		}
		if time.Now().After(deadline) {
			errs = append(errs, "deadline exceeded")
			break
		}
		resp, err := t.next[index].RoundTrip(req)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if resp == nil {
			errs = append(errs, "nil response")
			continue
		}
		if !t.validate(resp) {
			errs = append(errs, "response failed validation")
			continue
		}
		return resp, err
	}
	return nil, fmt.Errorf("request failed (%s)", strings.Join(errs, "; "))
}

// updatingTransport uses a generator function to make and cache a new
// RoundTripper from Endpoints that arrive via a StreamingProxy.
type updatingTransport struct {
	requests chan http.RoundTripper
}

func newUpdatingTransport(proxy StreamingProxy, generator func([]Endpoint) http.RoundTripper) *updatingTransport {
	t := &updatingTransport{
		requests: make(chan http.RoundTripper),
	}
	go t.run(proxy, generator, generator(<-proxy.Stream())) // get one set before returning
	return t
}

func (t *updatingTransport) run(proxy StreamingProxy, generator func([]Endpoint) http.RoundTripper, next http.RoundTripper) {
	for {
		select {
		case t.requests <- next:
		case endpoints := <-proxy.Stream():
			next = generator(endpoints)
		}
	}
}

func (t *updatingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return (<-t.requests).RoundTrip(req)
}

// makeResilientTransportGenerator returns a generator function that creates a
// retrying, round-robining, circuit-breaking http.RoundTripper around a slice
// of endpoints. All of the caveats described by NewResilientTransport apply
// here.
func makeResilientTransportGenerator(
	readTimeout time.Duration,
	retryValidator breaker.ResponseValidator,
	retryCutoff time.Duration,
	maxRetries int,
	breakerFailureRatio float64,
	maxIdleConnsPerEndpoint int,
) func([]Endpoint) http.RoundTripper {
	return func(endpoints []Endpoint) http.RoundTripper {
		candidates := make([]http.RoundTripper, len(endpoints))
		for i, endpoint := range endpoints {
			baseTransport := &http.Transport{
				ResponseHeaderTimeout: readTimeout,
				MaxIdleConnsPerHost:   maxIdleConnsPerEndpoint,
			}
			proxyHost := fmt.Sprintf("%s:%d", endpoint.IP, endpoint.Port) // capture
			rewritingTransport := proxypkg.Transport{
				Proxy: func(req *http.Request) (*url.URL, error) {
					return &url.URL{
						Scheme:   req.URL.Scheme,
						Opaque:   req.URL.Opaque,
						User:     req.URL.User,
						Host:     proxyHost,
						Path:     req.URL.Path,
						RawQuery: req.URL.RawQuery,
						Fragment: req.URL.Fragment,
					}, nil
				},
				Next: baseTransport,
			}
			candidates[i] = breaker.Transport(
				breaker.NewBreaker(breakerFailureRatio),
				breaker.DefaultResponseValidator,
				rewritingTransport,
			)
		}
		return retryTransport{
			next:     candidates,
			validate: retryValidator,
			cutoff:   retryCutoff,
			max:      maxRetries + 1, // attempts = retries + 1
		}
	}
}

// NewResilientTransport creates an http.RoundTripper that acts as a resilient
// round-robining proxy over endpoints yielded by the proxy. Resiliency is
// achieved by combining a circuit breaker per endpoint with introspective
// retry logic over all endpoints. Endpoints are assumed to be functionally
// identical. Requests are assumed to be idempotent.
//
// Retry validator determines if a given non-error response is valid and can
// be returned to the client, or should be retried. Retry cutoff is a hard
// deadline after which no more retries will be attempted, and the most recent
// error will be returned to the client. Max retries is the maximum number of
// retries that will be attempted within the deadline. Breaker failure ratio
// sets how many failures per success are required to trigger the circuit
// breaker and open the circuit.
func NewResilientTransport(
	proxy StreamingProxy,
	readTimeout time.Duration,
	retryValidator breaker.ResponseValidator,
	retryCutoff time.Duration,
	maxRetries int,
	breakerFailureRatio float64,
	maxIdleConnsPerEndpoint int,
) http.RoundTripper {
	return newUpdatingTransport(
		proxy,
		makeResilientTransportGenerator(
			readTimeout,
			retryValidator,
			retryCutoff,
			maxRetries,
			breakerFailureRatio,
			maxIdleConnsPerEndpoint,
		),
	)
}
