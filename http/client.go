package http

import (
	"net/http"
)

// Client represents an http.Client.
type Client interface {
	Do(*http.Request) (*http.Response, error)
}
