package http

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// Report emits a JSON object containing significant fields from the request
// and/or response to the passed io.Writer upon every complete request.
func Report(w io.Writer, next Client) Client {
	return &report{
		w:    w,
		next: next,
	}
}

type report struct {
	w    io.Writer
	next Client
}

func (r *report) Do(req *http.Request) (*http.Response, error) {
	begin := time.Now()
	resp, err := r.next.Do(req)
	json.NewEncoder(r.w).Encode(struct {
		Time           time.Time `json:"time,omitempty"`
		Method         string    `json:"method,omitempty"`
		URL            string    `json:"url,omitempty"`
		Path           string    `json:"path,omitempty"`
		Proto          string    `json:"proto,omitempty"`
		Status         int       `json:"status,omitempty"`
		ContentLength  int64     `json:"content_length"`
		MS             int       `json:"ms"`
		RemoteAddr     string    `json:"remote_addr,omitempty"`
		ForwardedFor   string    `json:"forwarded_for,omitempty"`
		ForwardedProto string    `json:"forwarded_proto,omitempty"`
		Range          string    `json:"range,omitempty"`
		Host           string    `json:"host,omitempty"`
		Referrer       string    `json:"referrer,omitempty"`
		UserAgent      string    `json:"user_agent,omitempty"`
		Authorization  string    `json:"authorization,omitempty"`
		Region         string    `json:"region,omitempty"`
		Country        string    `json:"country,omitempty"`
		City           string    `json:"city,omitempty"`
		RequestID      string    `json:"request_id,omitempty"`
	}{
		Time:           begin.UTC(),
		Method:         req.Method,
		URL:            req.RequestURI,
		Path:           req.URL.Path,
		Proto:          req.Proto,
		Status:         resp.StatusCode,
		ContentLength:  resp.ContentLength,
		MS:             int(time.Since(begin) / time.Millisecond),
		Host:           req.Host,
		RemoteAddr:     req.RemoteAddr,
		ForwardedFor:   req.Header.Get("X-Forwarded-For"),
		ForwardedProto: req.Header.Get("X-Forwarded-Proto"),
		Authorization:  req.Header.Get("Authorization"),
		Referrer:       req.Header.Get("Referer"),
		UserAgent:      req.Header.Get("User-Agent"),
		Range:          req.Header.Get("Range"),
		RequestID:      req.Header.Get("X-Request-Id"),
		Region:         req.Header.Get("X-Region"),
		Country:        req.Header.Get("X-Country"),
		City:           req.Header.Get("X-City"),
	})
	return resp, err
}
