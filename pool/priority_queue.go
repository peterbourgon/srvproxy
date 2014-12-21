package pool

import "sync/atomic"

// PriorityQueue returns a Pool that yields hosts according to their recent
// successes or failures.
func PriorityQueue(hosts []string) Pool {
	return &priorityQueue{
		hosts:  hosts,
		scores: make([]int64, len(hosts)),
		robin:  -1,
	}
}

type priorityQueue struct {
	hosts  []string
	scores []int64
	robin  int64
}

func (pq *priorityQueue) Get() (string, error) {
	if len(pq.hosts) <= 0 {
		return "", ErrNoHosts
	}

	var robin int
	for {
		var (
			vold = atomic.LoadInt64(&pq.robin)
			vnew = vold + 1
		)
		if atomic.CompareAndSwapInt64(&pq.robin, vold, vnew) {
			robin = int(vnew)
			break
		}
	}

	for { // find-a-zero
		for i := range pq.scores { // check all the scores
			index := (i + robin) % len(pq.scores)
			for { // individual score CAS
				if v := atomic.LoadInt64(&pq.scores[index]); v == 0 {
					return pq.hosts[index], nil
				} else if v < 0 {
					panic("impossible")
				} else if atomic.CompareAndSwapInt64(&pq.scores[index], v, v-1) {
					break
				}
			}
		}
	}
}

func (pq *priorityQueue) Put(host string, success bool) {
	for i, candidate := range pq.hosts {
		if candidate != host {
			continue
		}

		for {
			vold := atomic.LoadInt64(&pq.scores[i])

			vnew := vold + 1
			if success {
				vnew = 0
			}

			if atomic.CompareAndSwapInt64(&pq.scores[i], vold, vnew) {
				return
			}
		}
	}

	panic("bad Put")
}
