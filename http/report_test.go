package http_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/peterbourgon/srvproxy/http"
)

func TestReport(t *testing.T) {
	code := stdhttp.StatusOK
	contentLength := 123

	responseBody := make([]byte, contentLength)
	for i := range responseBody {
		responseBody[i] = 'A'
	}

	s := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(code)
		w.Write(responseBody)
	}))
	defer s.Close()

	req, err := stdhttp.NewRequest("GET", s.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	buf := bytes.Buffer{}
	rep := http.Report(&buf, stdhttp.DefaultClient)
	if _, err := rep.Do(req); err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", buf.String())

	var m map[string]interface{}
	if err := json.NewDecoder(&buf).Decode(&m); err != nil {
		t.Fatal(err)
	}

	if want, have := fmt.Sprint(code), fmt.Sprint(m["status"]); want != have {
		t.Errorf("want %v, have %v", want, have)
	}

	if want, have := fmt.Sprint(contentLength), fmt.Sprint(m["content_length"]); want != have {
		t.Errorf("want %v, have %v", want, have)
	}
}
