package pool

import (
	"expvar"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	getCount = expvar.NewInt("srvproxy_pool_get_count")
)

var (
	promGetCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "srvproxy",
			Subsystem: "pool",
			Name:      "get_count",
			Help:      "Number of get requests.",
		},
	)
)

// Instrument records metrics for operations against the wrapped Pool.
func Instrument(next Pool) Pool {
	return instrument{next}
}

type instrument struct{ Pool }

func (i instrument) Get() (string, error) {
	getCount.Add(1)
	promGetCount.Add(1)
	return i.Pool.Get()
}
