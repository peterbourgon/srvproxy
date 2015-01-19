package pool_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/peterbourgon/srvproxy/pool"
)

func TestReport(t *testing.T) {
	buf := &bytes.Buffer{}
	resolver := &fixedResolver{[]string{"foo"}, time.Millisecond}
	pool := pool.Report(buf, pool.Stream(resolver, "irrelevant", pool.RoundRobin))
	if _, err := pool.Get(); err != nil {
		t.Fatal(err)
	}
	if want, have := `{"host":"foo"}`+"\n", buf.String(); want != have {
		t.Errorf("want %q, have %q", want, have)
	}
}
