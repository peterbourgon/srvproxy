package http

import "net/http"

// Director receives and may modify an http.Request, and returns a function
// which will be called with the result of that request.
type Director interface {
	Direct(*http.Request) func(*http.Response, error)
}

// DirectorFunc is an adapter that allows use of ordinary functions as
// Directors. If f is a function with the appropriate signature,
// DirectorFunc(f) is a Director object that calls f.
type DirectorFunc func(*http.Request) func(*http.Response, error)

// Direct calls f(req).
func (f DirectorFunc) Direct(req *http.Request) func(*http.Response, error) {
	return f(req)
}

// Directed wraps a Client with a Director.
func Directed(d Director, next Client) Client {
	return directed{d, next}
}

type directed struct {
	Director
	Client
}

func (d directed) Do(req *http.Request) (*http.Response, error) {
	result := d.Director.Direct(req)
	resp, err := d.Client.Do(req)
	result(resp, err)
	return resp, err
}
