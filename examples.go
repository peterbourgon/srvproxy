package srvproxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/peterbourgon/srvproxy/pool"
	"github.com/peterbourgon/srvproxy/proxy"
	"github.com/peterbourgon/srvproxy/retry"
)

// ExampleDefaultUsage shows how to wire up a default DNS SRV proxy into
// http.DefaultClient.
func ExampleDefaultUsage() {
	var rt http.RoundTripper
	rt = http.DefaultTransport
	rt = proxy.Proxy(proxy.Next(rt))
	rt = retry.Retry(retry.Next(rt))

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

	rt = proxy.Proxy(
		proxy.Next(rt),
		proxy.Scheme("https"),
		proxy.PoolReporter(os.Stderr),
		proxy.Factory(pool.RoundRobin),
	)

	rt = retry.Retry(
		retry.Next(rt),
		retry.Pass(func(resp *http.Response, err error) error {
			if err != nil {
				return err
			}
			if resp.StatusCode >= 500 {
				return fmt.Errorf("HTTP %d %s", resp.StatusCode, resp.Status)
			}
			return nil
		}),
		retry.MaxAttempts(10),
		retry.Timeout(750*time.Millisecond),
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
