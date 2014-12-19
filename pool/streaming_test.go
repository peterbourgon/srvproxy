package pool_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/peterbourgon/srvproxy/pool"
)

func TestStreaming(t *testing.T) {
	a := "≠≠≠≠≠"
	b := "•••••"
	d := time.Millisecond
	r := &fixedResolver{[]string{a}, d}
	p := pool.Streaming(r, "irrelevant", pool.RoundRobin)

	if err := waitGet(p, time.Millisecond); err != nil {
		t.Fatal(err)
	}

	have, err := p.Get()
	if err != nil {
		t.Fatalf("Get (1): %s", err)
	}

	if want := a; want != have {
		t.Errorf("want %q, have %q", want, have)
	}

	r.hosts = []string{b}
	time.Sleep(3 * (d / 2))

	have, err = p.Get()
	if err != nil {
		t.Fatalf("Get (2): %s", err)
	}

	if want := b; want != have {
		t.Errorf("want %q, have %q", want, have)
	}
}

type fixedResolver struct {
	hosts []string
	ttl   time.Duration
}

func (r *fixedResolver) Resolve(_ string) ([]string, time.Duration, error) {
	return r.hosts, r.ttl, nil
}

func waitGet(p pool.Pool, max time.Duration) error {
	deadline := time.Now().Add(max)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout")
		}
		host, err := p.Get()
		if err != nil {
			time.Sleep(max / 10)
			continue
		}
		p.Put(host, true)
		return nil
	}
}
