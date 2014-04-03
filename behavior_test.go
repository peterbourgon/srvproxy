package srvproxy_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/peterbourgon/srvproxy"
	"github.com/streadway/handy/breaker"
)

func handleGood(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }
func handleSlow(w http.ResponseWriter, _ *http.Request) { time.Sleep(slowDelay); w.WriteHeader(200) }
func handleBad(w http.ResponseWriter, _ *http.Request)  { w.WriteHeader(503) }

func handleDegrade(start, increment time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(start)
		start += increment
		w.WriteHeader(200)
	}
}

var (
	readTimeout = 10 * time.Millisecond
	slowDelay   = 100 * time.Millisecond
	retryCutoff = 1000 * time.Millisecond

	maxRetries              = 100
	breakerFailureRatio     = 0.05
	maxIdleConnsPerEndpoint = 10
)

func TestBadServer(t *testing.T) {
	// Even with 1 bad server returning non-200s, the retrying component of
	// the resilient proxy should ensure we only receive 200s. runTestServers
	// will t.Fatal on a non-200, so all we care about here is that we don't
	// do that.
	runTestServers(t, 1000, handleGood, handleGood, handleBad)
}

func TestSlowServer(t *testing.T) {
	// If read timeout isn't working, avg will be ~slowDelay/2.
	min, avg, max := runTestServers(t, 1000, handleGood, handleSlow)
	t.Logf("min %s", min)
	t.Logf("avg %s", avg)
	t.Logf("max %s", max)
	if avg > slowDelay/3 {
		t.Error("read timeout doesn't seem to be working properly")
	}
}

func TestDegradingServer(t *testing.T) {
	// If read timeout isn't working, max will be ~n/2 ms
	n := 1000
	min, avg, max := runTestServers(t, n, handleGood, handleDegrade(0, time.Millisecond))
	t.Logf("min %s", min)
	t.Logf("avg %s", avg)
	t.Logf("max %s", max)
	if max > time.Duration(n/3)*time.Millisecond {
		t.Error("read timeout doesn't seem to be working properly")
	}
}

func runTestServers(t *testing.T, numRequests int, handlers ...http.HandlerFunc) (min, avg, max time.Duration) {
	servers := make(testServers, len(handlers))
	for i, handler := range handlers {
		servers[i] = httptest.NewServer(handler)
	}
	defer servers.close()

	proxy, err := srvproxy.NewProxy("irrelevant", servers.resolver, 100*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{
		Transport: srvproxy.NewResilientTransport(
			proxy,
			readTimeout,
			breaker.DefaultResponseValidator,
			retryCutoff,
			maxRetries,
			breakerFailureRatio,
			maxIdleConnsPerEndpoint,
		),
	}

	// One bad apple don't spoil the whole bunch, girl... ooh, give it several
	// tries, before you give up on HTTP requests through this resilient
	// transport altogether.

	d := make(chan time.Duration, numRequests)
	s := make(chan struct{}, 10) // semaphore for concurrent HTTP requests
	for i := 0; i < numRequests; i++ {
		go func() {
			s <- struct{}{}
			defer func() { <-s }()

			begin := time.Now()
			resp, err := client.Get("http://will.be.rewritten.anyway")
			took := time.Since(begin)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != 200 {
				t.Fatalf("final status code %d", resp.StatusCode)
			}
			d <- took
		}()
	}

	var sum time.Duration
	for i := 0; i < numRequests; i++ {
		took := <-d
		if i == 0 || took < min {
			min = took
		}
		if i == 0 || took > max {
			max = took
		}
		sum += took
	}
	avg = sum / time.Duration(numRequests)
	return min, avg, max
}
