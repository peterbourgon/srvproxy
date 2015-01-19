package pool

import (
	"runtime"
	"time"
)

// PriorityQueue returns a Pool that yields hosts according to their recent
// successes or failures.
//
// The pool implements a two-priority queue. It always tries to return hosts
// from the high-prio queue first. If no hosts are available, the pool falls
// back to the low-prio queue. When a high-prio host fails, it moves to the
// low-prio queue, and is moved back to the high-prio queue only after the
// recycle time.
func PriorityQueue(recycle time.Duration) func([]string) Pool {
	return func(hosts []string) Pool {
		var (
			m  = map[string]*host{}
			hi = make(chan string)
			lo = make(chan string)
		)

		for _, host := range hosts {
			m[host] = newHost(host, hi, lo, recycle)
		}

		runtime.Gosched() // get the hosts ready

		return &priorityQueue{
			hosts: m,
			hi:    hi,
			lo:    lo,
		}
	}
}

type priorityQueue struct {
	hosts map[string]*host
	hi    chan string
	lo    chan string
}

func (pq *priorityQueue) Get() (string, error) {
	select {
	case host := <-pq.hi:
		return host, nil
	default:
		select {
		case host := <-pq.lo:
			return host, nil
		default:
			return "", ErrNoHosts
		}
	}
}

func (pq *priorityQueue) Put(host string, success bool) {
	pq.hosts[host].put(success)
}

func (pq *priorityQueue) Close() {
	for _, h := range pq.hosts {
		h.close()
	}
}

type host struct {
	signal chan bool
	quit   chan chan struct{}
}

func newHost(hoststr string, sometimes, always chan<- string, recycle time.Duration) *host {
	h := &host{
		signal: make(chan bool),
		quit:   make(chan chan struct{}),
	}
	go h.loop(hoststr, sometimes, always, recycle)
	return h
}

func (h *host) loop(hoststr string, sometimes, always chan<- string, recycle time.Duration) {
	var (
		indirect = sometimes
		reset    <-chan time.Time
	)

	for {
		select {
		case indirect <- hoststr:

		case always <- hoststr:

		case success := <-h.signal:
			if !success && reset == nil {
				reset = time.After(recycle)
				indirect = nil
			}

		case <-reset:
			reset = nil
			indirect = sometimes

		case q := <-h.quit:
			close(q)
			return
		}
	}
}

func (h *host) put(success bool) {
	h.signal <- success
}

func (h *host) close() {
	q := make(chan struct{})
	h.quit <- q
	<-q
}
