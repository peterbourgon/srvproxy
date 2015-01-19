# srvproxy [![GoDoc](https://godoc.org/github.com/peterbourgon/srvproxy?status.svg)](http://godoc.org/github.com/peterbourgon/srvproxy) [![Build Status](https://travis-ci.org/peterbourgon/srvproxy.svg)](https://travis-ci.org/peterbourgon/srvproxy)

Proxy for DNS SRV records.

## Usage

```go
var rt http.RoundTripper
rt = http.DefaultTransport
rt = srvproxy.RoundTripper(srvproxy.ProxyNext(rt))
rt = srvproxy.Retry(srvproxy.RetryNext(rt))

t := &http.Transport{}
t.RegisterProtocol("dnssrv", rt)

c := http.Client{}
c.Transport = t

resp, err := c.Get("dnssrv://foo.bar.baz.internal.net/normal/path?key=value")
if err != nil {
	log.Fatal(err)
}

io.Copy(os.Stdout, resp.Body)
```
