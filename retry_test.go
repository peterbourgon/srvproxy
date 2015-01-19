package srvproxy_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/peterbourgon/srvproxy"
)

func TestRetryMaxAttempts(t *testing.T) {
	rt := &fixedRoundTripper{failFor: 2}
	c := http.Client{}
	c.Transport = srvproxy.Retry(srvproxy.MaxAttempts(3), srvproxy.RetryNext(rt))

	resp, err := c.Get("http://foo")
	if err != nil {
		t.Fatal(err)
	}

	if want, have := http.StatusTeapot, resp.StatusCode; want != have {
		t.Fatalf("want %v, have %v", want, have)
	}

	if want, have := 3, rt.count; want != have {
		t.Errorf("want %v, have %v", want, have)
	}
}

func TestRetryTimeout(t *testing.T) {
	d := time.Millisecond
	rt := &fixedRoundTripper{failUntil: time.Now().Add(d)}
	c := http.Client{}
	c.Transport = srvproxy.Retry(srvproxy.MaxAttempts(0), srvproxy.Timeout(2*d), srvproxy.RetryNext(rt))

	resp, err := c.Get("http://bar")
	if err != nil {
		t.Fatal(err)
	}

	if want, have := http.StatusTeapot, resp.StatusCode; want != have {
		t.Fatalf("want %v, have %v", want, have)
	}

	t.Logf("just FYI, it took %d attempt(s) to succeed", rt.count)
}

type fixedRoundTripper struct {
	failFor   int
	failUntil time.Time
	count     int
}

func (rt *fixedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.count++

	if rt.count <= rt.failFor {
		return nil, fmt.Errorf("attempt %d/%d fail", rt.count, rt.failFor)
	}

	if !rt.failUntil.IsZero() && time.Now().Before(rt.failUntil) {
		return nil, fmt.Errorf("attempt %d fail, will fail for another %s", rt.count, time.Now().Sub(rt.failUntil))
	}

	return &http.Response{StatusCode: http.StatusTeapot}, nil
}
