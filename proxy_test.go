package srvproxy_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/peterbourgon/srvproxy"
)

func TestSingleProxy(t *testing.T) {
	n := 5
	s := makeTestServers(n, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	defer s.close()
	interval := 5 * time.Second // irrelevant
	proxy, err := srvproxy.NewProxy("irrelevant", s.resolver, interval)
	if err != nil {
		t.Fatal(err)
	}
	m := map[srvproxy.Endpoint]bool{}
	for i := 0; i < 100*n; i++ {
		endpoint, err := proxy.Endpoint()
		if err != nil {
			t.Fatal(err)
		}
		m[endpoint] = true
	}
	if len(m) != n {
		t.Errorf("expected to get %d unique Endpoints, but got %d", n, len(m))
	}
}

type testServers []*httptest.Server

func makeTestServers(n int, h http.HandlerFunc) testServers {
	s := make([]*httptest.Server, n)
	for i := 0; i < n; i++ {
		s[i] = httptest.NewServer(h)
	}
	return testServers(s)
}

func (s testServers) resolver(name string) ([]srvproxy.Endpoint, error) {
	endpoints := make([]srvproxy.Endpoint, len(s))
	for i, server := range s {
		url, err := url.Parse(server.URL)
		if err != nil {
			return []srvproxy.Endpoint{}, err
		}
		host, portStr := strings.Split(url.Host, ":")[0], strings.Split(url.Host, ":")[1]
		port, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			return []srvproxy.Endpoint{}, err
		}
		endpoints[i] = srvproxy.Endpoint{IP: host, Port: uint16(port)}
	}
	return endpoints, nil
}

func (s testServers) close() {
	for _, server := range s {
		server.Close()
	}
}
