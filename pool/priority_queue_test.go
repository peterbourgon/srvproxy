package pool_test

import (
	"github.com/peterbourgon/srvproxy/pool"

	"testing"
	"time"
)

func TestPriorityQueueNoFailure(t *testing.T) {
	var (
		h = []string{"a", "b", "c"}
		p = pool.PriorityQueue(time.Second)(h)
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
		f = map[string]bool{"a": true, "c": true} // fail hosts
		p = pool.PriorityQueue(time.Second)(h)
		m = map[string]int{}
		n = 100 * len(h)
	)

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
		var (
			chosenPercent = (float64(m[host]) / float64(n)) * 100.0
			maxPercent    = float64(n) / 100.0
		)
		t.Logf("%q: %.2f%% (max %.2f%%)", host, chosenPercent, maxPercent)
		if _, badHost := f[host]; badHost && chosenPercent > maxPercent {
			t.Errorf("consistently-bad host %q was chosen too often (%.2f%% > %.2f%%)", host, chosenPercent, maxPercent)
		}
	}
}

func TestPriorityQueueRecycling(t *testing.T) {
	var (
		d = time.Millisecond
		h = []string{"a", "b", "c"}
		p = pool.PriorityQueue(d)(h)
		m = map[string]int{}
		n = 1000 * len(h)
	)

	for i := 0; i < n; i++ {
		host, _ := p.Get()
		success := i > 0 // first Get fails, all others succeed
		m[host]++
		p.Put(host, success)

		if i == n/10 {
			// At some point, after some Get/Put cycles, let the timer fire
			// and recycle the failed host.
			time.Sleep(2 * d)
		}
	}

	for _, host := range h {
		var (
			chosenPercent = (float64(m[host]) / float64(n)) * 100.0
			minPercent    = (100.0 / float64(len(h))) - 5.0
		)
		t.Logf("%q: %.2f%% (min %.2f%%)", host, chosenPercent, minPercent)
		if chosenPercent < minPercent {
			t.Errorf("host %q was chosen too infrequently (%.2f%% < %.2f%%)", host, chosenPercent, minPercent)
		}
	}
}
