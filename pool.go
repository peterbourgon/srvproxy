package srvproxy

import (
	"net/http"
	"net/url"
)

// Pool describes anything which can yield hosts.
type Pool interface {
	Get() (string, error)
}

// HTTPProxy adapts a Pool into a http.Client.Proxy.
func HTTPProxy(p Pool) func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		host, err := p.Get()
		if err != nil {
			return nil, err
		}

		req.URL.Host = host
		return req.URL, nil
	}
}

// TODO: Pool based on Resolver
