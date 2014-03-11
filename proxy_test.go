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

func DontTestSingleProxy(t *testing.T) {
	n := 5
	servers := newTestServers(n)
	defer servers.close()
	interval := 5 * time.Second // irrelevant
	proxy, err := srvproxy.NewProxy("irrelevant", servers.resolver, interval)
	if err != nil {
		t.Fatal(err)
	}
	m := map[srvproxy.Endpoint]bool{}
	for i := 0; i < 100*n; i++ {
		m[proxy.Endpoint()] = true
	}
	if len(m) != n {
		t.Errorf("expected to get %d unique Endpoints, but got %d", n, len(m))
	}
}

type testServers struct {
	servers []*httptest.Server
}

func newTestServers(n int) *testServers {
	h := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
	servers := make([]*httptest.Server, n)
	for i := 0; i < n; i++ {
		servers[i] = httptest.NewServer(http.HandlerFunc(h))
	}
	return &testServers{
		servers: servers,
	}
}

func (s *testServers) resolver(name string) ([]srvproxy.Endpoint, error) {
	endpoints := make([]srvproxy.Endpoint, len(s.servers))
	for i, server := range s.servers {
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

func (s *testServers) close() {
	for _, server := range s.servers {
		server.Close()
	}
}
