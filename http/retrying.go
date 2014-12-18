package http

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

// Retrying implements request retry logic.
func Retrying(max int, cutoff time.Duration, next Client) Client {
	return &retrying{max, cutoff, next}
}

type retrying struct {
	max    int
	cutoff time.Duration
	Client
}

func (r retrying) Do(req *http.Request) (*http.Response, error) {
	var (
		deadline = time.Now().Add(r.cutoff)
		attempt  = 0
		errs     = []string{}
	)

	for {
		if time.Now().After(deadline) {
			errs = append(errs, "deadline reached")
			break
		}

		attempt++
		if attempt > r.max {
			errs = append(errs, "all attempts exhausted")
			break
		}

		resp, err := r.Client.Do(req)
		if err == nil {
			return resp, nil
		}

		errs = append(errs, err.Error())
	}

	return nil, errors.New(strings.Join(errs, "; "))
}
