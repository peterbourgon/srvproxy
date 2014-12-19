package pool_test

import (
	"reflect"
	"testing"

	"github.com/peterbourgon/srvproxy/pool"
)

func TestRoundRobin(t *testing.T) {
	var (
		hosts = []string{"a", "b", "c"}
		p     = pool.RoundRobin(hosts)
		want  = []string{"a", "b", "c", "a", "b", "c", "a"}
		have  []string
	)

	for i := 0; i < 7; i++ {
		host, err := p.Get()
		if err != nil {
			t.Errorf("Get %d: %s", i, err)
			continue
		}
		have = append(have, host)
	}

	if !reflect.DeepEqual(want, have) {
		t.Errorf("want %v, have %v", want, have)
	}
}
