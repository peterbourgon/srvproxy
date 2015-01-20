package pool

import (
	"reflect"
	"time"

	"github.com/peterbourgon/srvproxy/resolve"
)

// Stream returns a Pool, created via the Factory, that's continuously updated
// with hosts resolved from the name.
func Stream(r resolve.Resolver, name string, f Factory) Pool {
	s := &stream{
		getc:   make(chan getRequest),
		closec: make(chan chan struct{}),
	}

	hosts, ttl := mustResolve(r, name, []string{})
	go s.loop(r, name, hosts, time.After(ttl), f)

	return s
}

type stream struct {
	getc   chan getRequest
	closec chan chan struct{}
}

func (s *stream) Get() (string, error) {
	req := getRequest{make(chan string), make(chan error)}
	s.getc <- req

	select {
	case host := <-req.hostc:
		return host, nil
	case err := <-req.errc:
		return "", err
	}
}

func (s *stream) Close() {
	q := make(chan struct{})
	s.closec <- q
	<-q
}

func (s *stream) loop(r resolve.Resolver, name string, hosts []string, refreshc <-chan time.Time, f Factory) {
	pool := f(hosts)
	for {
		select {
		case <-refreshc:
			newHosts, ttl := mustResolve(r, name, hosts)
			refreshc = time.After(ttl)

			// Only re-build the Pool if the hosts have changed.
			if reflect.DeepEqual(newHosts, hosts) {
				continue
			}

			pool.Close() // close the old
			hosts = newHosts
			pool = f(hosts) // create the new

		case req := <-s.getc:
			host, err := pool.Get()
			if err != nil {
				req.errc <- err
				continue
			}

			req.hostc <- host

		case q := <-s.closec:
			close(q)
			return
		}
	}
}

func mustResolve(r resolve.Resolver, name string, currentHosts []string) ([]string, time.Duration) {
	hosts, ttl, err := r.Resolve(name)
	if err != nil {
		hosts = currentHosts
		ttl = time.Second
	}
	return hosts, ttl
}

type getRequest struct {
	hostc chan string
	errc  chan error
}
