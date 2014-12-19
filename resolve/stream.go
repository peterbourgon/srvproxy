package resolve

import "time"

var after = time.After

// Stream continuously resolves the given name, and forwards the results to
// the hosts or errs chan. Stream blocks until cancel is closed.
func Stream(r Resolver, name string, hosts chan<- []string, errs chan<- error, cancel chan struct{}) {
	refresh := after(0)
	for {
		select {
		case <-refresh:
			if h, ttl, err := r.Resolve(name); err != nil {
				errs <- err
				refresh = after(time.Second)
			} else {
				hosts <- h
				refresh = after(ttl)
			}

		case <-cancel:
			return
		}
	}
}
