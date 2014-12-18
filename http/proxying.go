package http

import "net/http"

// HostProvider describes something which can yield hosts for transactions,
// and record a given host's success/failure.
type HostProvider interface {
	Get() (host string, err error)
	Put(host string, success bool)
}

// Proxying implements host proxying logic.
func Proxying(hp HostProvider, next Client) Client {
	return &proxying{
		hp:     hp,
		Client: next,
	}
}

type proxying struct {
	hp HostProvider
	Client
}

func (p proxying) Do(req *http.Request) (*http.Response, error) {
	host, err := p.hp.Get()
	if err != nil {
		return nil, err
	}

	req.Host = host
	resp, err := p.Client.Do(req)

	if err == nil {
		p.hp.Put(host, true)
	} else {
		p.hp.Put(host, false)
	}

	return resp, err
}
