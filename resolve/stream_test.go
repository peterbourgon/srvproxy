package resolve

import (
	"testing"
	"time"
)

func TestStream(t *testing.T) {
	var (
		hosts   = []string{"foo", "bar"}
		ttl     = 5 * time.Second
		r       = fixedResolver{hosts, ttl}
		hostc   = make(chan []string)
		errc    = make(chan error)
		cancelc = make(chan struct{})
		afterc  = make(chan time.Time)
		donec   = make(chan struct{})
	)

	after = func(time.Duration) <-chan time.Time { return afterc }
	defer func() { after = time.After }()

	go func() {
		defer close(donec)
		Stream(r, "irrelevant", hostc, errc, cancelc)
	}()

	afterc <- time.Now()

	select {
	case <-hostc:
	case <-time.After(time.Millisecond):
		t.Fatalf("didn't get first broadcast in time")
	}

	afterc <- time.Now()

	select {
	case <-hostc:
	case <-time.After(time.Millisecond):
		t.Fatalf("didn't get second broadcast in time")
	}

	close(cancelc)

	select {
	case <-donec:
	case <-time.After(time.Millisecond):
		t.Fatalf("didn't get done signal in time")
	}
}

type fixedResolver struct {
	hosts []string
	ttl   time.Duration
}

func (r fixedResolver) Resolve(_ string) ([]string, time.Duration, error) {
	return r.hosts, r.ttl, nil
}
