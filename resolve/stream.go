package resolve

import "time"

// Stream continuously resolves the given name, and forwards the results to
// the hosts or errs chan. Stream blocks until cancel is closed.
func Stream(r Resolver, name string, hosts chan<- []string, errs chan<- error, cancel chan struct{}) {
	refresh := time.After(0)
	for {
		select {
		case <-refresh:
			if h, ttl, err := r.Resolve(name); err != nil {
				errs <- err
				refresh = time.After(time.Second)
			} else {
				hosts <- h
				refresh = time.After(ttl)
			}

		case <-cancel:
			return
		}
	}
}
