package srvproxy

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Retry wraps a http.RoundTripper with basic retry logic. Requests are
// assumed to be idempotent.
func Retry(options ...RetryOption) http.RoundTripper {
	r := &retry{
		max:     3,
		timeout: time.Second,
		pass:    func(_ *http.Response, err error) error { return err },
		next:    http.DefaultTransport,
	}
	r.setOptions(options...)
	return r
}

type retry struct {
	max     int
	timeout time.Duration
	pass    func(*http.Response, error) error
	next    http.RoundTripper
}

func (r *retry) setOptions(options ...RetryOption) {
	for _, option := range options {
		option(r)
	}
}

// RetryOption sets a specific option for the Retry. This is the functional
// options idiom. See https://www.youtube.com/watch?v=24lFtGHWxAQ for more
// information.
type RetryOption func(*retry)

// MaxAttempts sets how many attempts will be made to complete the request.
// The attempt is aborted when this value or Timeout is reached, whichever
// comes first. A value of zero implies unlimited attempts. If MaxAttempts
// isn't provided, a default value of 3 is used.
func MaxAttempts(n int) RetryOption {
	return func(r *retry) { r.max = n }
}

// Timeout sets the maximum time the Retry will spend trying to complete the
// request. The attempt is aborted when this value or MaxAttempts is reached,
// whichever comes first. A value of zero implies no timeout. If Timeout isn't
// provided, a default value of 1 second is used.
func Timeout(d time.Duration) RetryOption {
	return func(r *retry) { r.timeout = d }
}

// Pass sets the function that determines if a (http.Response, error) return
// pair shall be considered valid and forwarded to the calling context, or if
// it should be retried. If the Pass function returns a non-nil error, the
// request will be retried. If Pass isn't provided, a default function that
// ignores the http.Response and returns the provided error is used.
func Pass(f func(*http.Response, error) error) RetryOption {
	return func(r *retry) { r.pass = f }
}

// RetryNext sets the http.RoundTripper that will be used to execute the
// requests. If RetryNext isn't provided, http.DefaultTransport is used.
func RetryNext(rt http.RoundTripper) RetryOption {
	return func(r *retry) { r.next = rt }
}

func (r *retry) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		haveDeadline = r.timeout > 0
		deadline     = time.Now().Add(r.timeout)
		attempt      = 0
		errs         = []string{}
	)

	for {
		attempt++
		if r.max > 0 && attempt > r.max {
			return nil, fmt.Errorf("request failed, max attempts (%d) exceeded%s", r.max, suffix(errs))
		}

		if haveDeadline && time.Now().After(deadline) {
			return nil, fmt.Errorf("request failed, timeout reached%s", suffix(errs))
		}

		resp, err := r.next.RoundTrip(req)

		if passErr := r.pass(resp, err); passErr != nil {
			errs = append(errs, passErr.Error())
			continue
		}

		return resp, err
	}
}

func suffix(errs []string) string {
	if len(errs) > 0 {
		return fmt.Sprintf(" (%s)", strings.Join(errs, "; "))
	}
	return ""
}
