package srvproxy

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/streadway/handy/breaker"
	proxypkg "github.com/streadway/handy/proxy"
)

func TestChoosingTransport(t *testing.T) {
	n := 5

	a := make([]countingTransport, n)
	r := make(choosingTransport, n)
	for i := 0; i < n; i++ {
		r[i] = &a[i]
	}

	x := 10000
	for i := 0; i < x; i++ {
		r.RoundTrip(&http.Request{})
	}

	tolerance := 0.01
	for i, c := range a {
		expected := float64(x) / float64(n)
		got := float64(c)
		if skew := math.Abs(expected-got) / float64(x); skew > tolerance {
			t.Errorf("transport %d/%d had bad distribution: got %d, skew %.3f > %.3f", i+1, n, c, skew, tolerance)
		}
	}
}

func TestRetryingTransportMaxViaError(t *testing.T) {
	var f failingTransport
	r := retryingTransport{
		next:     &f,
		validate: func(*http.Response) bool { return true },
		cutoff:   5 * time.Second, // very high
		max:      3,
	}
	if _, err := r.RoundTrip(&http.Request{}); err == nil {
		t.Errorf("expected an error, got nil")
	}
	if expected, got := r.max, int(f); expected != got {
		t.Errorf("expected %d attempts, got %d", expected, got)
	}
}

func TestRetryingTransportMaxViaValidate(t *testing.T) {
	var c countingTransport
	r := retryingTransport{
		next:     &c,
		validate: func(*http.Response) bool { return false },
		cutoff:   5 * time.Second, // very high
		max:      4,
	}
	if _, err := r.RoundTrip(&http.Request{}); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if expected, got := r.max, int(c); expected != got {
		t.Errorf("expected %d attempts, got %d", expected, got)
	}
}

func TestRetryingTransportCutoff(t *testing.T) {
	d := 1 * time.Millisecond
	var f failingTransport
	r := retryingTransport{
		next:     &f,
		validate: func(*http.Response) bool { time.Sleep(d); return false },
		cutoff:   10 * d,
		max:      math.MaxInt32,
	}

	e := make(chan error, 1)
	go func() { _, err := r.RoundTrip(&http.Request{}); e <- err }()

	var err error
	select {
	case err = <-e:
		break
	case <-time.After(2 * r.cutoff):
		t.Fatal("timeout")
	}

	if err == nil {
		t.Errorf("expected an error, got nil")
	}
}

func TestUpdatingTransport(t *testing.T) {
	var foo int
	fooServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		foo++
		w.WriteHeader(501)
	}))
	defer fooServer.Close()

	var bar int
	barServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bar++
		w.WriteHeader(200)

	}))
	defer barServer.Close()

	pointer := url2endpoint(t, fooServer.URL)
	resolver := func(string) ([]Endpoint, error) { return []Endpoint{pointer}, nil }
	interval := 10 * time.Millisecond
	proxy, _ := NewProxy("", resolver, interval)

	generator := func(endpoints []Endpoint) http.RoundTripper {
		return proxypkg.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				if len(endpoints) != 1 {
					t.Fatal("in generator, not precisely one endpoint")
				}
				proxyHost := fmt.Sprintf("%s:%d", endpoints[0].IP, endpoints[0].Port)
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
		}
	}

	transport := newUpdatingTransport(proxy, generator)

	if int(foo) != 0 || int(bar) != 0 {
		t.Errorf("expected foo=0, bar=0; got foo=%d bar=%d", foo, bar)
	}

	dummyRequest, _ := http.NewRequest("GET", "http://irrelevant", &bytes.Buffer{})
	transport.RoundTrip(dummyRequest)
	if int(foo) != 1 || int(bar) != 0 {
		t.Errorf("expected foo=1, bar=0; got foo=%d bar=%d", foo, bar)
	}

	transport.RoundTrip(dummyRequest)
	if int(foo) != 2 || int(bar) != 0 {
		t.Errorf("expected foo=2, bar=0; got foo=%d bar=%d", foo, bar)
	}

	pointer = url2endpoint(t, barServer.URL)
	time.Sleep(2 * interval) // allow time for resolver to resolve

	transport.RoundTrip(dummyRequest)
	if int(foo) != 2 || int(bar) != 1 {
		t.Errorf("expected foo=2, bar=1; got foo=%d bar=%d", foo, bar)
	}

	transport.RoundTrip(dummyRequest)
	if int(foo) != 2 || int(bar) != 2 {
		t.Errorf("expected foo=2, bar=2; got foo=%d bar=%d", foo, bar)
	}
}

func TestResilientTransport(t *testing.T) {
	out, _ := exec.Command("ulimit", "-n").CombinedOutput()
	ulimit, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if ulimit < 10000 {
		t.Logf("To run this test, set ulimit -n to at least 10000 (currently %s, %d)", out, ulimit)
		return
	}

	var badCount int
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		badCount++
		w.WriteHeader(501)
	}))
	defer badServer.Close()

	var goodCount int
	goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		goodCount++
		w.WriteHeader(200)

	}))
	defer goodServer.Close()

	resolver := func(string) ([]Endpoint, error) {
		return []Endpoint{
			url2endpoint(t, goodServer.URL),
			url2endpoint(t, badServer.URL),
		}, nil
	}

	seed := time.Now().UnixNano()
	//t.Logf("seed %d", seed)
	rand.Seed(seed)

	run := func(maxRetries int, breakerFailureRatio float64, requestCount int) int {
		proxy, err := NewProxy("", resolver, 50*time.Millisecond)
		if err != nil {
			t.Fatal(err)
		}
		validate := breaker.DefaultResponseValidator
		transport := NewResilientTransport(
			proxy,               // StreamingProxy
			validate,            // retryValidator
			10*time.Second,      // retryCutoff
			maxRetries,          // maxRetries
			breakerFailureRatio, // breakerFailureRatio
			1,                   // maxIdleConnsPerEndpoint
		)
		var errCount int
		for i := 0; i < requestCount; i++ {
			req, _ := http.NewRequest("GET", "http://nonesuch", &bytes.Buffer{})
			resp, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatal(err) // should get 500s, no straight-up errors
			}
			if resp.StatusCode != 200 {
				errCount++
			}
		}
		proxy.Stop()
		return errCount
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	w.Write([]byte("requests\tmax retries\tBFR\terrors\terror ratio\n"))
	for _, requestCount := range []int{10000} {
		for _, maxRetries := range []int{3} {
			for _, breakerFailureRatio := range []float64{0.05} {
				errorCount := run(maxRetries, breakerFailureRatio, requestCount)
				errorRatio := float64(errorCount) / float64(requestCount)
				w.Write([]byte(fmt.Sprintf("%d\t%d\t%f\t%d\t%f\n", requestCount, maxRetries, breakerFailureRatio, errorCount, errorRatio)))
				if errorRatio > 0.01 {
					t.Errorf("error ratio %f > %f", errorRatio, 0.01)
				}
			}
		}
	}
	//w.Flush()
}

type countingTransport int

func (t *countingTransport) Allow() bool { return true }

func (t *countingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	(*t)++
	return &http.Response{}, nil
}

type failingTransport int

func (t *failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	(*t)++
	return nil, fmt.Errorf("fail")
}

func url2endpoint(t *testing.T, rawurl string) Endpoint {
	u, err := url.Parse(rawurl)
	if err != nil {
		t.Fatal(err)
	}
	host, portStr := strings.Split(u.Host, ":")[0], strings.Split(u.Host, ":")[1]
	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	return Endpoint{host, uint16(port)}
}
