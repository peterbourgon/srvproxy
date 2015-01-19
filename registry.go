package srvproxy

import (
	"sync"

	"github.com/peterbourgon/srvproxy/resolve"

	"github.com/peterbourgon/srvproxy/pool"
)

// registry is a map of DNS SRV name to corresponding pool of hosts. If the
// name doesn't yet have a pool, a new pool will be allocated via the factory
// function, and wrapped with pool.Stream to keep it up-to-date.
//
// The registry will grow with every unique host passed to get, so it's
// important to keep the set of input hosts bounded.
type registry struct {
	sync.Mutex
	r resolve.Resolver
	f pool.Factory
	m map[string]pool.Pool
}

func newRegistry(r resolve.Resolver, f pool.Factory) *registry {
	return &registry{
		r: r,
		f: f,
		m: map[string]pool.Pool{},
	}
}

func (r *registry) get(host string) pool.Pool {
	r.Lock()
	defer r.Unlock()

	p, ok := r.m[host]
	if !ok {
		p = pool.Stream(r.r, host, r.f)
		r.m[host] = p
	}
	return p
}
