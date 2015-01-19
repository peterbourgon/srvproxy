package srvproxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/peterbourgon/srvproxy/pool"

	"github.com/peterbourgon/srvproxy/roundtrip"
)

// ExampleDefaultUsage shows how to wire up a default DNS SRV proxy into
// http.DefaultClient.
func ExampleDefaultUsage() {
	var rt http.RoundTripper
	rt = http.DefaultTransport
	rt = roundtrip.Proxy(roundtrip.ProxyNext(rt))
	rt = roundtrip.Retry(roundtrip.RetryNext(rt))

	t := &http.Transport{}
	t.RegisterProtocol("dnssrv", rt)

	http.DefaultClient.Transport = t

	resp, err := http.Get("dnssrv://foo.bar.srv.internal.name/normal/path?key=value")
	if err != nil {
		log.Fatal(err)
	}

	io.Copy(os.Stdout, resp.Body)
	resp.Body.Close()
}

// ExampleCustomUsage shows how to wire up a customized DNS SRV proxy into
// http.DefaultClient.
func ExampleCustomUsage() {
	var rt http.RoundTripper
	rt = http.DefaultTransport

	rt = roundtrip.Proxy(
		roundtrip.ProxyNext(rt),
		roundtrip.Scheme("https"),
		roundtrip.PoolReporter(os.Stderr),
		roundtrip.Factory(pool.RoundRobin),
	)

	rt = roundtrip.Retry(
		roundtrip.RetryNext(rt),
		roundtrip.Pass(func(resp *http.Response, err error) error {
			if err != nil {
				return err
			}
			if resp.StatusCode >= 500 {
				return fmt.Errorf("HTTP %d %s", resp.StatusCode, resp.Status)
			}
			return nil
		}),
		roundtrip.MaxAttempts(10),
		roundtrip.Timeout(750*time.Millisecond),
	)

	t := &http.Transport{}
	t.RegisterProtocol("dnssrv", rt)

	http.DefaultClient.Transport = t

	resp, err := http.Get("dnssrv://foo.bar.srv.internal.name/normal/path?key=value")
	if err != nil {
		log.Fatal(err)
	}

	io.Copy(os.Stdout, resp.Body)
	resp.Body.Close()
}
