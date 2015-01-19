# srvproxy [![GoDoc](https://godoc.org/github.com/peterbourgon/srvproxy?status.svg)](http://godoc.org/github.com/peterbourgon/srvproxy) [![Build Status](https://travis-ci.org/peterbourgon/srvproxy.svg)](https://travis-ci.org/peterbourgon/srvproxy)

Proxy for DNS SRV records.

## Usage

```go
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
```

For more examples, see examples.go.