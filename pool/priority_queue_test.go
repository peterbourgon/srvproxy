package pool_test

import (
	"math"
	"testing"
	"time"

	"github.com/peterbourgon/srvproxy/pool"
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

func TestPriorityQueueSingleFailure(t *testing.T) {
	var (
		h = []string{"a", "b", "c", "d"}
		p = pool.PriorityQueue(time.Second)(h)
		m = map[string]int{}
		n = 100 * len(h)
	)

	for i := 0; i < n; i++ {
		host, _ := p.Get()
		m[host]++
		success := i > 0 // first host has 1 failure
		//t.Logf("i=%d: %q (success %v)", i, host, success)
		p.Put(host, success)
	}

	for i, host := range h {
		var (
			want      = ((1.0 / float64(len(h)-1)) * 100.0)     // %
			have      = (float64(m[host]) / float64(n)) * 100.0 // %
			tolerance = 1.0                                     // %
			printf    = t.Logf
		)

		if i == 0 {
			want = 0.0 // ideally
		}

		if math.Abs(want-have) > tolerance {
			printf = t.Errorf
		}

		printf("%q: want %.2f%%, have %.2f%%", host, want, have)
	}
}

func TestPriorityQueueCompleteFailure(t *testing.T) {
	var (
		h = []string{"a", "b", "c", "d", "e", "f"}
		f = map[string]bool{"a": true, "c": true} // fail hosts
		p = pool.PriorityQueue(time.Second)(h)
		m = map[string]int{}
		n = 100 * len(h)
	)

	for i := 0; i < n; i++ {
		host, _ := p.Get()
		m[host]++
		success := !f[host]
		//t.Logf("i=%d: %q (success %v)", i, host, success)
		p.Put(host, success)
	}

	for _, host := range h {
		var (
			want      = 100.0 / float64(len(m)-len(f))          // %
			have      = (float64(m[host]) / float64(n)) * 100.0 // %
			tolerance = 5.0                                     // %
			printf    = t.Logf
		)

		if _, ok := f[host]; ok {
			want = 0.0
		}

		if math.Abs(want-have) > tolerance {
			printf = t.Errorf
		}

		printf("%q: want %.2f%%, have %.2f%%", host, want, have)
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
			want      = 100.0 / float64(len(h))                 // %
			have      = (float64(m[host]) / float64(n)) * 100.0 // %
			tolerance = 5.0                                     // %
			printf    = t.Logf
		)

		if math.Abs(want-have) > tolerance {
			printf = t.Errorf
		}

		printf("%q: want %.2f%%, have %.2f%%", host, want, have)
	}
}
