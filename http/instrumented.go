package http

import (
	"expvar"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	requestCount = expvar.NewInt("srvproxy_http_request_count")
	successCount = expvar.NewInt("srvproxy_http_success_count")
	failCount    = expvar.NewInt("srvproxy_http_failed_count")
)

var (
	requestTime = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: "srvproxy",
			Subsystem: "http",
			Name:      "request_time_nanoseconds",
			Help:      "Total time spent making HTTP requests.",
			MaxAge:    10 * time.Second, // like statsd
		},
		[]string{"status_code"},
	)
)

// Instrumented records request metrics.
func Instrumented(next Client) Client {
	return &instrumented{next}
}

type instrumented struct{ Client }

func (i instrumented) Do(req *http.Request) (resp *http.Response, err error) {
	defer func(begin time.Time) {
		requestCount.Add(1)
		if err == nil {
			successCount.Add(1)
		} else {
			failCount.Add(1)
		}

		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}

		labelValues := strconv.FormatInt(int64(statusCode), 10)
		observation := float64(time.Since(begin).Nanoseconds())
		requestTime.WithLabelValues(labelValues).Observe(observation)
	}(time.Now())

	return i.Client.Do(req)
}
