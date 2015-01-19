package srvproxy

import (
	"testing"
	"time"

	"github.com/peterbourgon/srvproxy/pool"
)

func TestRegistry(t *testing.T) {
	registry := newRegistry(&doublingResolver{time.Millisecond}, pool.RoundRobin)

	// A new registry should have no pools.
	if want, have := 0, len(registry.m); want != have {
		t.Errorf("want %d, have %d", want, have)
	}

	have, err := registry.get("foo").Get()
	if err != nil {
		t.Fatal(err)
	}

	if want := "foofoo"; want != have {
		t.Errorf("want %q, have %q", want, have)
	}

	// After the first unique name, the registry should have one pool.
	if want, have := 1, len(registry.m); want != have {
		t.Errorf("want %d, have %d", want, have)
	}

	registry.get("foo").Get()

	// It should reuse the pool for the same name.
	if want, have := 1, len(registry.m); want != have {
		t.Errorf("want %d, have %d", want, have)
	}

	registry.get("bar").Get()

	// It should create a new pool for a new name.
	if want, have := 2, len(registry.m); want != have {
		t.Errorf("want %d, have %d", want, have)
	}
}

type doublingResolver struct {
	ttl time.Duration
}

func (r doublingResolver) Resolve(name string) ([]string, time.Duration, error) {
	return []string{name + name}, r.ttl, nil
}
