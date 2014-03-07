package srvproxy

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/streadway/handy/breaker"
	proxypkg "github.com/streadway/handy/proxy"
)

// NewResilientClient constructs a retrying, round-robining, circuit-breaking
// http.Client over each of the endpoints yielded by the proxy. The client
// assumes all endpoints are functionally identical, and all requests are
// idempotent. The client will rewrite the host section of requested URLs to
// one of the endpoints.
//
// Retry validator determines if a given response is valid and can be returned
// to the client, or should be retried. Retry cutoff is a hard deadline after
// which no more retries will be attempted, and the most recent error will be
// returned to the client. Max retries is the maximum number of retries that
// will be attempted within the deadline.
func NewResilientClient(
	proxy StreamingProxy,
	retryValidator breaker.ResponseValidator,
	retryCutoff time.Duration,
	maxRetries int,
	breakerFailureRatio float64,
	maxIdleConnsPerEndpoint int,
) *http.Client {
	return &http.Client{
		Transport: newUpdatingTransport(
			proxy,
			resilientTransportGenerator(
				retryValidator,
				retryCutoff,
				maxRetries,
				breakerFailureRatio,
				maxIdleConnsPerEndpoint,
			),
		),
	}
}

// updatingTransport builds a resilient and up-to-date http.RoundTripper
// around endpoints yielded by the proxy.
type updatingTransport struct {
	proxy      StreamingProxy
	transports chan http.RoundTripper
	quit       chan chan struct{}
}

func newUpdatingTransport(proxy StreamingProxy, generator transportGenerator) *updatingTransport {
	t := &updatingTransport{
		proxy:      proxy,
		transports: make(chan http.RoundTripper),
		quit:       make(chan chan struct{}),
	}
	go t.run(generator, generator(<-t.proxy.Stream()))
	return t
}

func (t *updatingTransport) run(generator transportGenerator, transport http.RoundTripper) {
	for {
		select {
		case t.transports <- transport:
		case endpoints := <-t.proxy.Stream():
			transport = generator(endpoints)
		case q := <-t.quit:
			close(q)
			return
		}
	}
}

func (t *updatingTransport) stop() {
	q := make(chan struct{})
	t.quit <- q
	<-q
}

func (t *updatingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return (<-t.transports).RoundTrip(req)
}

type transportGenerator func([]Endpoint) http.RoundTripper

func resilientTransportGenerator(
	retryValidator breaker.ResponseValidator,
	retryCutoff time.Duration,
	maxRetries int,
	breakerFailureRatio float64,
	maxIdleConnsPerEndpoint int,
) func([]Endpoint) http.RoundTripper {
	return func(endpoints []Endpoint) http.RoundTripper {
		roundRobinTransport := make(allowingTransports, len(endpoints))
		for i, endpoint := range endpoints {
			myBreaker := breaker.NewBreaker(breakerFailureRatio)
			roundRobinTransport[i] = allowingTransport{
				allow: myBreaker,
				next: breaker.Transport(
					myBreaker,
					breaker.DefaultResponseValidator,
					proxypkg.Transport{
						Proxy: func(*http.Request) (*url.URL, error) {
							return url.Parse(fmt.Sprintf("http://%s:%d", endpoint.IP, endpoint.Port))
						},
						Next: &http.Transport{
							MaxIdleConnsPerHost: maxIdleConnsPerEndpoint,
						},
					},
				),
			}
		}
		return retryingTransport{
			validate: retryValidator,
			cutoff:   retryCutoff,
			max:      maxRetries,
			next:     roundRobinTransport,
		}
	}
}

// allowingTransports performs a sort of round-robin behavior on top of the
// underlying transports, checking the Allow state before forwarding the
// request.
type allowingTransports []allowingTransport

func (t allowingTransports) RoundTrip(req *http.Request) (*http.Response, error) {
	for _, index := range rand.Perm(len(t)) {
		if t[index].allow.Allow() {
			return t[index].next.RoundTrip(req)
		}
	}
	return nil, fmt.Errorf("no transports available")
}

// allowingTransport combines an allower (typically a circuit breaker) and an
// underlying RoundTripper (typically a circuit breaker transport). It enables
// you to check if a request submitted to the RoundTripper would be allowed to
// proceed, or would immediately fail. When combined with functionally
// equivalent peers, it enables more intelligent request routing.
type allowingTransport struct {
	allow allower
	next  http.RoundTripper
}

type allower interface {
	Allow() bool
}

// retryingTransport retries requests against the underlying transport until
// it achieves success, the retryCutoff is elapsed, or maxRetries is reached.
// retryingTransport assumes the request is idempotent.
type retryingTransport struct {
	validate breaker.ResponseValidator
	cutoff   time.Duration
	max      int
	next     http.RoundTripper
}

func (t retryingTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	deadline := time.Now().Add(t.cutoff)
	for retry := 0; retry < t.max && time.Now().Before(deadline); retry++ {
		resp, err = t.next.RoundTrip(req)
		if err != nil || !t.validate(resp) {
			continue
		}
	}
	return
}
