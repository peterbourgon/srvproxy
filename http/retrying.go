package http

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

var now = time.Now

// Retrying implements request retry logic.
func Retrying(max int, cutoff time.Duration, ok func(*http.Response) error, next Client) Client {
	return &retrying{max, cutoff, ok, next}
}

type retrying struct {
	max    int
	cutoff time.Duration
	ok     func(*http.Response) error
	Client
}

func (r retrying) Do(req *http.Request) (*http.Response, error) {
	var (
		deadline = now().Add(r.cutoff)
		attempt  = 0
		errs     = []string{}
	)

	for {
		if now().After(deadline) {
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
