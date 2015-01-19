package pool

import (
	"errors"
	"net/http"
)

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
	Close()
}

// Factory converts a slice of hosts to a Pool.
type Factory func([]string) Pool

// SuccessFunc determines if an HTTP transaction is considered successful.
// The result is noted by the pool.
type SuccessFunc func(*http.Response, error) bool

// SimpleSuccess is a SuccessFunc which returns true if there was no error.
// The response status code is ignored.
func SimpleSuccess(_ *http.Response, err error) bool { return err == nil }
