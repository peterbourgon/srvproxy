package http

import stdhttp "net/http"

// HostProvider describes something which can yield hosts for transactions,
// and record a given host's success/failure.
type HostProvider interface {
	Get() (host string, err error)
	Put(host string, success bool)
}

// Proxying implements host proxying logic.
func Proxying(p HostProvider, next Client) Client {
	return &proxying{
		p:    p,
		next: next,
	}
}

type proxying struct {
	p    HostProvider
	next Client
}

func (p proxying) Do(req *stdhttp.Request) (*stdhttp.Response, error) {
	host, err := p.p.Get()
	if err != nil {
		return nil, err
	}

	req.Host = host
	resp, err := p.next.Do(req)

	if err == nil {
		p.p.Put(host, true)
	} else {
		p.p.Put(host, false)
	}

	return resp, err
}
