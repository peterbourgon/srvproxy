package pool

import (
	"errors"
	"sync"
)

// PriorityQueue returns a Pool that yields hosts according to their recent
// successes or failures.
func PriorityQueue(hosts []string) Pool {
	return &priorityQueue{
		hosts: hosts,
	}
}

type priorityQueue struct {
	sync.RWMutex
	hosts []string
}

func (pq *priorityQueue) Get() (string, error) {
	return "", errors.New("not implemented")
}

func (pq *priorityQueue) Put(string, bool) {
	// not implemented
}
