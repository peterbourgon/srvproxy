package pool

import "net/http"

// SuccessFunc determines if an HTTP transaction is considered successful.
// The result is noted by the pool.
type SuccessFunc func(*http.Response, error) bool

// SimpleSuccess is a SuccessFunc which returns true if there was no error.
// The response status code is ignored.
func SimpleSuccess(_ *http.Response, err error) bool { return err == nil }

// Director provides request director functionality over a pool. The host
// portion of the request URL is rewritten to a host taken from the pool. The
// host is returned to the pool when the request has completed, with success
// determined by the SuccessFunc.
func Director(p Pool, s SuccessFunc) func(*http.Request) func(*http.Response, error) {
	return func(req *http.Request) func(*http.Response, error) {
		host, err := p.Get()
		if err != nil {
			return func(*http.Response, error) {}
		}

		req.URL.Host = host

		return func(resp *http.Response, err error) {
			p.Put(host, s(resp, err))
		}
	}
}
