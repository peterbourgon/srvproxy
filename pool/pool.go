package pool

import "errors"

var (
	// ErrNoHosts indicates a pool is empty.
	ErrNoHosts = errors.New("no hosts available")
)

// Pool describes anything which can yield hosts for transactions, and accept
// reports on their transactional success. While every Get should be followed
// by a Put, Pool implementations don't grant exclusive use of a host, unless
// explicitly noted.
type Pool interface {
	Get() (host string, err error)
	Put(host string, success bool)
}

// Factory converts a slice of hosts to a Pool.
type Factory func([]string) Pool
