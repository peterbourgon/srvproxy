package roundtrip

import (
	"io"
	"sync"

	"github.com/peterbourgon/srvproxy/pool"
	"github.com/peterbourgon/srvproxy/resolve"
)

// registry is a map of DNS SRV name to corresponding pool of hosts. If the
// name doesn't yet have a pool, a new pool will be allocated via the factory
// function, and wrapped with pool.Stream to keep it up-to-date.
//
// The registry will grow with every unique host passed to get, so it's
// important to keep the set of input hosts bounded.
type registry struct {
	sync.Mutex
	resolver     resolve.Resolver
	reportWriter io.Writer
	factory      pool.Factory
	m            map[string]pool.Pool
}

func newRegistry(r resolve.Resolver, reportWriter io.Writer, f pool.Factory) *registry {
	return &registry{
		resolver:     r,
		reportWriter: reportWriter,
		factory:      f,
		m:            map[string]pool.Pool{},
	}
}

func (r *registry) get(host string) pool.Pool {
	r.Lock()
	defer r.Unlock()
	p, ok := r.m[host]
	if !ok {
		p = pool.Stream(r.resolver, host, r.factory)
		p = pool.Report(r.reportWriter, p)
		p = pool.Instrument(p)
		r.m[host] = p
	}
	return p
}
