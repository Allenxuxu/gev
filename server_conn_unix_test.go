// +build !windows

package gev

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConnLoadBalanceLeastConnection(t *testing.T) {
	handler := new(example3)

	s, err := NewServer(handler,
		Network("tcp"),
		Address(":1840"),
		NumLoops(4),
		ReusePort(true),
		LoadBalance(LeastConnection()))
	if err != nil {
		t.Fatal(err)
	}

	go s.Start()

	for i := 0; i < 200; i++ {
		_, err := net.DialTimeout("tcp", "127.0.0.1:1840", time.Second*60)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Millisecond * 20)
	}

	time.Sleep(time.Millisecond * 20)

	for i := 0; i < len(s.workLoops); i++ {
		assert.Equal(t, int64(50), s.workLoops[i].ConnectionCount())
	}

	s.Stop()
}

func TestConnLoadBalanceRoundRobin(t *testing.T) {
	handler := new(example3)

	s, err := NewServer(handler,
		Network("tcp"),
		Address(":1841"),
		NumLoops(4),
		ReusePort(true),
		LoadBalance(RoundRobin()))
	if err != nil {
		t.Fatal(err)
	}

	go s.Start()

	for i := 0; i < 9; i++ {
		_, err := net.DialTimeout("tcp", "127.0.0.1:1841", time.Second*60)
		if err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(time.Millisecond * 20)

	for i := 0; i < len(s.workLoops); i++ {
		expected := 2
		if i == 0 {
			expected = 3
		}
		assert.Equal(t, expected, int(s.workLoops[i].ConnectionCount()))
	}

	s.Stop()
}
