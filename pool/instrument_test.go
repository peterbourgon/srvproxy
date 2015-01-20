package pool_test

import (
	"expvar"
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/peterbourgon/srvproxy/pool"
)

func TestGets(t *testing.T) {
	var p pool.Pool
	p = pool.RoundRobin([]string{"≠"})
	p = pool.Instrument(p)
	p = pool.Report(ioutil.Discard, p)

	n := 123
	for i := 0; i < n; i++ {
		p.Get()
	}

	want := strconv.FormatInt(int64(n), 10)
	have := expvar.Get(pool.ExpvarKeyGets).String()
	if want != have {
		t.Errorf("want %q, have %q", want, have)
	}
}
