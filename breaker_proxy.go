package srvproxy

import (
	"log"
	"time"
)

// BreakerProxy acts as a SRV proxy with a built-in circuit breaker.
type BreakerProxy struct {
	c chan Endpoint
	f chan Endpoint
	q chan chan struct{}
}

// NewBreakerProxy constructs and returns a BreakerProxy, performing regular
// resolve lookups against a defined name string, and caching the list of
// returned endpoints.
func NewBreakerProxy(resolve Resolver, name string, pollInterval time.Duration) (*BreakerProxy, error) {
	endpoints, err := resolve(name)
	if err != nil {
		return nil, err
	}
	p := &BreakerProxy{
		c: make(chan Endpoint),
		f: make(chan Endpoint),
		q: make(chan chan struct{}),
	}
	go p.loop(resolve, name, updateControllers(endpoints, map[Endpoint]*controller{}, p.c), pollInterval)
	return p, nil
}

func (p *BreakerProxy) loop(resolve Resolver, name string, controllers map[Endpoint]*controller, pollInterval time.Duration) {
	tick := time.Tick(pollInterval)
	for {
		select {
		case q := <-p.q:
			for _, controller := range controllers {
				controller.stop()
			}
			close(q)
			return

		case endpoint := <-p.f:
			if controller, ok := controllers[endpoint]; ok {
				controller.failure()
			}

		case <-tick:
			endpoints, err := resolve(name)
			if err != nil {
				log.Printf("srvproxy: poll: %s", err)
				continue // don't replace good endpoints with bad
			}
			controllers = updateControllers(endpoints, controllers, p.c)
		}
	}
}

// Close stops all goroutines associated with this BreakerProxy. Closed
// proxies must not be accessed.
func (p *BreakerProxy) Close() {
	q := make(chan struct{})
	p.q <- q
	<-q
}

// With chooses an endpoint and yields it to the passed closure. If the
// closure returns a non-nil error, With reports it as a failure to the
// associated endpoint, which will trip that endpoint's circuit breaker,
// rendering it unavailable for one second.
func (p BreakerProxy) With(f func(Endpoint) error) error {
	var endpoint Endpoint
	select {
	case endpoint = <-p.c:
	default:
		return errNoEndpoints
	}
	err := f(endpoint)
	if err != nil {
		p.f <- endpoint
	}
	return err
}

type controller struct {
	out  chan Endpoint
	fail chan struct{}
	quit chan chan struct{}
}

type states int

const (
	closed states = iota
	tripped
	open
)

func newController(out chan Endpoint, endpoint Endpoint) *controller {
	c := &controller{
		out:  make(chan Endpoint),
		fail: make(chan struct{}),
		quit: make(chan chan struct{}),
	}
	go c.loop(endpoint)
	return c
}

func (c *controller) loop(endpoint Endpoint) {
	var (
		state   states
		timeout <-chan time.Time
	)
	for {
		switch state {
		case closed:
			select {
			case q := <-c.quit:
				close(q)
				return
			case c.out <- endpoint:
				// continue
			case <-c.fail:
				log.Printf("%v: failed, tripping circuit", endpoint)
				state = tripped
			}

		case tripped:
			timeout = time.After(1 * time.Second)
			state = open

		case open:
			select {
			case q := <-c.quit:
				close(q)
				return
			case <-c.fail:
				// ignore
			case <-timeout:
				log.Printf("%v: cooldown expired, closing circuit", endpoint)
				state = closed
			}
		}
	}
}

func (c *controller) failure() {
	c.fail <- struct{}{}
}

func (c *controller) stop() {
	q := make(chan struct{})
	c.quit <- q
	<-q
}

func updateControllers(endpoints []Endpoint, existing map[Endpoint]*controller, c chan Endpoint) map[Endpoint]*controller {
	updated := map[Endpoint]*controller{}
	for _, endpoint := range endpoints {
		controller, ok := existing[endpoint]
		if ok {
			delete(existing, endpoint) // mv
		} else {
			controller = newController(c, endpoint)
		}
		updated[endpoint] = controller
	}
	for _, controller := range existing {
		controller.stop()
	}
	return updated
}
