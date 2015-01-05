package http

import (
	"net/http/httptest"
	"net/http"

	"testing"
)

func TestInstrumented(t *testing.T) {
	codeWriter := func(code int) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			t.Logf("%s %d %s", r.Method, code, http.StatusText(code))
			w.WriteHeader(code)
		}
	}

	pass := httptest.NewServer(codeWriter(http.StatusOK))
	defer pass.Close()

	fail := httptest.NewServer(codeWriter(http.StatusTeapot))
	fail.Close() // immediately, to generate errors

	ins := Instrumented(http.DefaultClient)
	ins.Do(mustNewRequest("GET", pass.URL))
	ins.Do(mustNewRequest("GET", fail.URL))

	if want, have := "2", requestCount.String(); want != have {
		t.Errorf("want %v, have %v", want, have)
	}

	if want, have := "1", successCount.String(); want != have {
		t.Errorf("want %v, have %v", want, have)
	}

	if want, have := "1", failCount.String(); want != have {
		t.Errorf("want %v, have %v", want, have)
	}
}

func mustNewRequest(method, urlStr string) *http.Request {
	req, err := http.NewRequest(method, urlStr, nil)
	if err != nil {
		panic(err)
	}
	return req
}