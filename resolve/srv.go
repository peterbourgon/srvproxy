package resolve

import (
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ResolveSRV resolves the name via a DNS SRV lookup.
func ResolveSRV(name string) ([]string, time.Duration, error) {
	_, addrs, err := net.LookupSRV("", "", name)
	if err != nil {
		return []string{}, 0, err
	}

	hosts := make([]string, len(addrs))
	for i := 0; i < len(addrs); i++ {
		host := strings.TrimRight(addrs[i].Target, ".")
		port := strconv.FormatUint(uint64(addrs[i].Port), 10)
		hosts[i] = host + ":" + port
	}

	sort.Strings(hosts)
	ttl := 5 * time.Second // TODO: get DNS TTL, probably via github.com/miekg/dns
	return hosts, ttl, nil
}
