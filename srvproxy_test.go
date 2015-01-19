package srvproxy_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/peterbourgon/srvproxy"
)

func TestRoundTripper(t *testing.T) {
	count := 0
	code := http.StatusTeapot
	handler := func(w http.ResponseWriter, _ *http.Request) { count++; w.WriteHeader(code) }
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	resolver := fixedResolver{[]string{u.Host}, time.Minute}
	roundTripper := srvproxy.RoundTripper(srvproxy.Resolver(resolver))
	transport := &http.Transport{}
	transport.RegisterProtocol("dummy", roundTripper)
	client := &http.Client{}
	client.Transport = transport

	resp, err := client.Get("dummy://foo.bar.net/path/query?key=value")
	if err != nil {
		t.Fatal(err)
	}

	if want, have := 1, count; want != have {
		t.Errorf("want count %d, have %d", want, have)
	}

	if want, have := code, resp.StatusCode; want != have {
		t.Errorf("want HTTP %d, have %d", want, have)
	}
}

type fixedResolver struct {
	hosts []string
	ttl   time.Duration
}

func (r fixedResolver) Resolve(_ string) ([]string, time.Duration, error) {
	return r.hosts, r.ttl, nil
}
