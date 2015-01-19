package pool

import (
	"encoding/json"
	"io"
)

// Report logs JSON-encoded pool operations for the wrapped pool to the passed
// io.Writer. If w is nil, Report is a no-op.
func Report(w io.Writer, next Pool) Pool {
	var enc encoder = nopEncoder{}
	if w != nil {
		enc = json.NewEncoder(w)
	}
	return &report{
		enc:  enc,
		next: next,
	}
}

type report struct {
	enc  encoder
	next Pool
}

func (r *report) Get() (string, error) {
	host, err := r.next.Get()
	r.enc.Encode(poolGet{host, err})
	return host, err
}

func (r *report) Put(host string, success bool) {
	r.next.Put(host, success)
	r.enc.Encode(poolPut{host, success})
}

func (r *report) Close() {
	r.next.Close()
}

type encoder interface {
	Encode(interface{}) error
}

type nopEncoder struct{}

func (e nopEncoder) Encode(interface{}) error { return nil }

type poolGet struct {
	Host string `json:"host"`
	Err  error  `json:"error,omitempty"`
}

type poolPut struct {
	Host    string `json:"host"`
	Success bool   `json:"success"`
}
