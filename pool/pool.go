package pool

import "errors"

var (
	// ErrNoHosts indicates a pool is empty.
	ErrNoHosts = errors.New("no hosts available")
)

// Pool describes anything which can yield hosts, and report on their
// transactional success.
type Pool interface {
	Get() (host string, err error)
	Put(host string, success bool)
}
