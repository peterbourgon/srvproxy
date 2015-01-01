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
		h.quit()
	}
}

type host struct {
	sometimesc chan<- string
	alwaysc    chan<- string
	signalc    chan bool
	quitc      chan chan struct{}
}

func newHost(hoststr string, sometimes, always chan<- string, recycle time.Duration) *host {
	h := &host{
		sometimesc: sometimes,
		alwaysc:    always,
		signalc:    make(chan bool),
		quitc:      make(chan chan struct{}),
	}
	go h.loop(hoststr, recycle)
	return h
}

func (h *host) loop(hoststr string, recycle time.Duration) {
	var (
		sometimesc = h.sometimesc
		resetc     <-chan time.Time
	)

	for {
		select {
		case sometimesc <- hoststr:

		case h.alwaysc <- hoststr:

		case success := <-h.signalc:
			if !success && resetc == nil {
				resetc = time.After(recycle)
				sometimesc = nil
			}

		case <-resetc:
			resetc = nil
			sometimesc = h.sometimesc

		case q := <-h.quitc:
			close(q)
			return
		}
	}
}

func (h *host) put(success bool) {
	h.signalc <- success
}

func (h *host) quit() {
	q := make(chan struct{})
	h.quitc <- q
	<-q
}
