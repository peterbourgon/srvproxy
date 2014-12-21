package pool_test

import (
	"github.com/peterbourgon/srvproxy/pool"

	"testing"
)

func TestPriorityQueueNoFailure(t *testing.T) {
	var (
		h = []string{"a", "b", "c"}
		p = pool.PriorityQueue(h)
	)

	for i := 0; i < 10*len(h); i++ {
		have, err := p.Get()
		if err != nil {
			t.Fatal(err)
		}

		if want := h[i%len(h)]; want != have {
			t.Errorf("i=%d: want %q, have %q", i, want, have)
			continue
		}

		p.Put(have, true)
	}
}

func TestPriorityQueueDistribution(t *testing.T) {
	var (
		h = []string{"a", "b", "c", "d", "e", "f"}
		f = map[string]bool{"a": true, "c": true} // fail
		p = pool.PriorityQueue(h)
		m = map[string]int{}
	)

	n := 10 * len(h)
	for i := 0; i < n; i++ {
		host, err := p.Get()
		if err != nil {
			t.Errorf("i=%d: %s", i, err)
			continue
		}

		m[host]++
		success := !f[host]
		//t.Logf("i=%d: %q (success %v)", i, host, success)
		p.Put(host, success)
	}

	for _, host := range h {
		t.Logf("%q: %.0f%%", host, (float64(m[host])/float64(n))*100.0)
	}
}
