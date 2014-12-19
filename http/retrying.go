package http

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Retrying implements request retry logic. Requests must be idempotent.
func Retrying(max int, cutoff time.Duration, ok func(*http.Response) error, next Client) Client {
	return &retrying{max, cutoff, ok, next}
}

// ValidateFunc determines if an HTTP response is invalid and should be
// retried. A nil error indicates a valid response, which can be returned to
// the caller.
type ValidateFunc func(*http.Response) error

// SimpleValidator considers a response with a 1xx, 2xx, 3xx, or 4xx response
// code valid, and returns an error for any other (higher) response code.
func SimpleValidator(resp *http.Response) error {
	if resp.StatusCode <= 499 {
		return nil
	}
	return fmt.Errorf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
}

type retrying struct {
	max    int
	cutoff time.Duration
	ok     ValidateFunc
	Client
}

func (r retrying) Do(req *http.Request) (*http.Response, error) {
	var (
		deadline time.Time
		attempt  = 0
		errs     = []string{}
	)

	if r.cutoff > 0 {
		deadline = time.Now().Add(r.cutoff)
	}

	for {
		if !deadline.IsZero() && time.Now().After(deadline) {
			errs = append(errs, "deadline reached")
			break
		}

		attempt++
		if attempt > r.max {
			errs = append(errs, "all attempts exhausted")
			break
		}

		resp, err := r.Client.Do(req)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}

		if err = r.ok(resp); err != nil {
			errs = append(errs, err.Error())
			continue
		}

		return resp, nil
	}

	return nil, errors.New(strings.Join(errs, "; "))
}
