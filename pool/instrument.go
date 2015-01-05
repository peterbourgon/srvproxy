package pool

import (
	"expvar"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	getCount       = expvar.NewInt("srvproxy_pool_get_count")
	putOKCount     = expvar.NewInt("srvproxy_pool_put_ok_count")
	putFailedCount = expvar.NewInt("srvproxy_pool_put_failed_count")
	outstanding    = expvar.NewInt("srvproxy_pool_outstanding")
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
	promPutCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "srvproxy",
			Subsystem: "pool",
			Name:      "put_count",
			Help:      "Number of put requests.",
		},
		[]string{"success"},
	)
	promOutstanding = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "srvproxy",
			Subsystem: "pool",
			Name:      "outstanding",
			Help:      "Number of outstanding hosts.",
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
	outstanding.Add(1)
	promGetCount.Add(1)
	promOutstanding.Add(1)

	return i.Pool.Get()
}

func (i instrument) Put(s string, b bool) {
	i.Pool.Put(s, b)

	outstanding.Add(-1)
	if b {
		putOKCount.Add(1)
	} else {
		putFailedCount.Add(1)
	}
	promOutstanding.Sub(1)
	promPutCount.WithLabelValues(fmt.Sprintf("%v", b)).Add(1)
}
