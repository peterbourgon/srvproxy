package pool

import "expvar"

var (
	getCount       = expvar.NewInt("srvproxy_pool_get_count")
	putOKCount     = expvar.NewInt("srvproxy_pool_put_ok_count")
	putFailedCount = expvar.NewInt("srvproxy_pool_put_failed_count")
	outstanding    = expvar.NewInt("srvproxy_pool_outstanding")
)

// Instrumented records metrics for operations against the wrapped Pool.
func Instrumented(next Pool) Pool {
	return instrumented{next}
}

type instrumented struct{ Pool }

func (i instrumented) Get() (string, error) {
	getCount.Add(1)
	outstanding.Add(1)

	return i.Pool.Get()
}

func (i instrumented) Put(s string, b bool) {
	outstanding.Add(-1)
	if b {
		putOKCount.Add(1)
	} else {
		putFailedCount.Add(1)
	}

	i.Pool.Put(s, b)
}