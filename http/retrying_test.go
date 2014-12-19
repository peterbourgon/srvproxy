package http_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	srvhttp "github.com/peterbourgon/srvproxy/http"
)

func TestRetryingMax(t *testing.T) {
	var (
		server     = httptest.NewServer(failingHandler(2))
		client     = srvhttp.Retrying(3, time.Hour, ok, http.DefaultClient)
		request, _ = http.NewRequest("GET", server.URL, nil)
	)

	defer server.Close()

	if _, err := client.Do(request); err != nil {
		t.Error(err)
	}
}

func TestRetryingCutoff(t *testing.T) {
	var (
		server     = httptest.NewServer(slowHandler(time.Millisecond))
		client     = srvhttp.Retrying(999, time.Microsecond, ok, http.DefaultClient)
		request, _ = http.NewRequest("GET", server.URL, nil)
	)

	defer server.Close()

	_, err := client.Do(request)
	wantSuffix := "deadline reached"
	if err == nil || !strings.HasSuffix(err.Error(), wantSuffix) {
		t.Errorf("want %q, have %v", wantSuffix, err)
	}
}

func failingHandler(n int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		n--
		if n >= 0 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func slowHandler(d time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(d)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func ok(resp *http.Response) error {
	if resp.StatusCode >= 400 {
		return fmt.Errorf(http.StatusText(resp.StatusCode))
	}
	return nil
}
