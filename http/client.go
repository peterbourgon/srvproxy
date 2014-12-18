package http

import (
	"net/http"
)

// Client represents a http.Client.
type Client interface {
	Do(*http.Request) (*http.Response, error)
}
