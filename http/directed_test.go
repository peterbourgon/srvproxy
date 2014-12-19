package http_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	srvhttp "github.com/peterbourgon/srvproxy/http"
)

func TestDirected(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		status, _ := strconv.ParseInt(strings.Trim(r.URL.Path, "/"), 10, 32)
		w.WriteHeader(int(status))
	}

	var (
		server     = httptest.NewServer(http.HandlerFunc(handler))
		director   = &pathDirector{"/418", 0}
		client     = srvhttp.Directed(director, http.DefaultClient)
		request, _ = http.NewRequest("GET", server.URL+"/200", nil)
	)

	defer server.Close()

	resp, err := client.Do(request)
	if err != nil {
		t.Errorf("GET: %s", err)
	}

	want := http.StatusTeapot
	if have := resp.StatusCode; want != have {
		t.Errorf("response: want %d, have %d", want, have)
	}
	if have := int(atomic.LoadInt32(&director.result)); want != have {
		t.Errorf("director: want %d, have %d", want, have)
	}
}

type pathDirector struct {
	path   string
	result int32
}

func (d *pathDirector) Direct(req *http.Request) func(*http.Response, error) {
	req.URL.Path = d.path
	return func(resp *http.Response, err error) {
		atomic.StoreInt32(&d.result, int32(resp.StatusCode))
	}
}
