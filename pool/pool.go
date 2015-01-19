package pool

import "errors"

var (
	// ErrNoHosts indicates a pool is empty.
	ErrNoHosts = errors.New("no hosts available")
)

// Pool describes anything which can yield hosts for transactions. Pools don't
// grant exclusive use of hosts.
type Pool interface {
	Get() (host string, err error)
	Close()
}

// Factory converts a slice of hosts to a Pool.
type Factory func([]string) Pool
