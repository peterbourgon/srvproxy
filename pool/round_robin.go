package pool

import "sync"

// RoundRobin returns a plain round-robining Pool. Close is a no-op.
func RoundRobin(hosts []string) Pool {
	return &roundRobin{
		hosts: hosts,
	}
}

type roundRobin struct {
	sync.Mutex
	hosts []string
}

func (rr *roundRobin) Get() (string, error) {
	rr.Lock()
	defer rr.Unlock()

	if len(rr.hosts) <= 0 {
		return "", ErrNoHosts
	}

	host := rr.hosts[0]
	rr.hosts = append(rr.hosts[1:], rr.hosts[0])
	return host, nil
}

func (rr *roundRobin) Close() {}
