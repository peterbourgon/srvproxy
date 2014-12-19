package pool

import (
	"reflect"
	"time"

	"github.com/peterbourgon/srvproxy/resolve"
)

// Streaming returns a Pool, created from the passed Factory, that's
// continuously updated with hosts discovered via the Resolver.
func Streaming(r resolve.Resolver, name string, f Factory) Pool {
	s := &streaming{
		getc: make(chan getRequest),
		putc: make(chan putRequest),
	}

	hosts, ttl := mustResolve(r, name)
	go s.loop(r, name, hosts, time.After(ttl), f)

	return s
}

type streaming struct {
	getc chan getRequest
	putc chan putRequest
}

func (s *streaming) Get() (string, error) {
	req := getRequest{make(chan string), make(chan error)}
	s.getc <- req

	select {
	case host := <-req.hostc:
		return host, nil
	case err := <-req.errc:
		return "", err
	}
}

func (s *streaming) Put(host string, success bool) {
	s.putc <- putRequest{host, success}
}

func (s *streaming) loop(r resolve.Resolver, name string, hosts []string, refreshc <-chan time.Time, f Factory) {
	pool := f(hosts)
	for {
		select {
		case <-refreshc:
			newHosts, ttl := mustResolve(r, name)
			refreshc = time.After(ttl)

			// Only re-build the Pool if the hosts have changed.
			if reflect.DeepEqual(newHosts, hosts) {
				continue
			}

			hosts = newHosts
			pool = f(hosts)

		case req := <-s.getc:
			host, err := pool.Get()
			if err != nil {
				req.errc <- err
				continue
			}

			req.hostc <- host

		case req := <-s.putc:
			pool.Put(req.host, req.success)
		}
	}
}

func mustResolve(r resolve.Resolver, name string) ([]string, time.Duration) {
	hosts, ttl, err := r.Resolve(name)
	if err != nil {
		hosts = []string{}
		ttl = time.Second
	}
	return hosts, ttl
}

type getRequest struct {
	hostc chan string
	errc  chan error
}

type putRequest struct {
	host    string
	success bool
}
