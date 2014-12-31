package pool

import (
	"time"

	"sync"
)

// PriorityQueue returns a Pool that yields hosts according to their recent
// successes or failures.
func PriorityQueue(recycle time.Duration) func([]string) Pool {
	return func(hosts []string) Pool {
		return &priorityQueue{
			good:     newHosts(hosts),
			bad:      newHosts([]string{}),
			recycle:  recycle,
			deadline: time.Now().Add(recycle),
		}
	}
}

type priorityQueue struct {
	good     *hosts
	bad      *hosts
	recycle  time.Duration
	deadline time.Time
}

func (pq *priorityQueue) Get() (string, error) {
	first, second := pq.good, pq.bad

	if time.Now().After(pq.deadline) {
		first, second = second, first
		pq.deadline = time.Now().Add(pq.recycle)
	}

	if host, err := first.get(); err == nil {
		return host, nil
	}

	host, err := second.get()
	return host, err
}

func (pq *priorityQueue) Put(host string, success bool) {
	// When a host transitions from one set to the other, there's a brief
	// period when it's present in both. That's fine.
	if success {
		pq.good.add(host)
		pq.bad.remove(host)
	} else {
		pq.bad.add(host)
		pq.good.remove(host)
	}
}

type hosts struct {
	sync.RWMutex
	a []string
	p int
}

func newHosts(a []string) *hosts {
	h := &hosts{
		a: []string{},
		p: 0,
	}
	for _, host := range a {
		h.add(host)
	}
	return h
}

func (h *hosts) add(host string) {
	h.Lock()
	defer h.Unlock()

	for _, s := range h.a {
		if s == host {
			return
		}
	}

	h.a = append(h.a, host)
}

func (h *hosts) remove(host string) {
	h.Lock()
	defer h.Unlock()

	for i, s := range h.a {
		if s == host {
			h.a = append(h.a[:i], h.a[i+1:]...)
			return
		}
	}
}

func (h *hosts) get() (string, error) {
	h.Lock()
	defer h.Unlock()

	if len(h.a) <= 0 {
		return "", ErrNoHosts
	}

	host := h.a[h.p%len(h.a)]
	h.p = (h.p + 1) % len(h.a)
	return host, nil
}
