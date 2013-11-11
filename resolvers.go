package srvproxy

import (
	"github.com/miekg/dns"
	"github.com/soundcloud/go-dns-resolver/resolv"
)

// Endpoint combines a host IP and port number.
type Endpoint struct {
	IP   string
	Port uint16
}

// Resolver converts a symbolic/opaque name string to a set of Endpoints.
type Resolver func(name string) ([]Endpoint, error)

// DNSResolver implements Resolver by querying the system's configured DNS
// server for SRV records.
func DNSResolver(name string) ([]Endpoint, error) {
	msg, err := resolv.LookupString("SRV", name)
	if err != nil {
		return []Endpoint{}, err
	}

	endpoints := []Endpoint{}
	for _, rr := range msg.Answer {
		if srv, ok := rr.(*dns.SRV); ok {
			endpoints = append(endpoints, Endpoint{srv.Target, srv.Port})
		}
	}
	return endpoints, nil
}
