package resolve

import "time"

// Resolver represents anything that can resolve an abstract name to a set of
// hosts, inclusive ports when appropriate, and their TTL.
type Resolver interface {
	Resolve(name string) (hosts []string, ttl time.Duration, err error)
}

// ResolverFunc is an adapter that allows use of ordinary functions as
// Resolvers. If f is a function with the appropriate signature,
// ResolverFunc(f) is a Resolver object that calls f.
type ResolverFunc func(name string) (hosts []string, ttl time.Duration, err error)

// Resolve calls f(name).
func (f ResolverFunc) Resolve(name string) (hosts []string, ttl time.Duration, err error) {
	return f(name)
}
