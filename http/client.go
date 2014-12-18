package http

import (
	stdhttp "net/http"
)

// Client represents a http.Client.
type Client interface {
	Do(*stdhttp.Request) (*stdhttp.Response, error)
}
