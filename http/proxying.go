package http

import "net/http"

// HostProvider describes something which can yield hosts for transactions,
// and record a given host's success/failure.
type HostProvider interface {
	Get() (host string, err error)
	Put(host string, success bool)
}

// Proxying implements host proxying logic.
func Proxying(p HostProvider, next Client) Client {
	return &proxying{
		HostProvider: p,
		Client:       next,
	}
}

type proxying struct {
	HostProvider
	Client
}

func (p proxying) Do(req *http.Request) (*http.Response, error) {
	host, err := p.HostProvider.Get()
	if err != nil {
		return nil, err
	}

	req.Host = host
	resp, err := p.Client.Do(req)

	if err == nil {
		p.HostProvider.Put(host, true)
	} else {
		p.HostProvider.Put(host, false)
	}

	return resp, err
}
