package gev

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLeastConnection(t *testing.T) {
	var (
		loops []*EventLoop
		n     = 100
		min   int64
	)

	for i := 0; i < n; i++ {
		l := &EventLoop{}
		connCount := int64(rand.Intn(n))
		l.ConnCunt.Swap(connCount)
		loops = append(loops, l)

		if connCount < min {
			min = connCount
		}
	}

	strategy := LeastConnection()
	for i := 0; i < n; i++ {
		l := strategy(loops)
		assert.Equal(t, min, l.ConnectionCount())
	}

}

func TestRoundRobin(t *testing.T) {
	var (
		loops []*EventLoop
		n     = 100
	)

	for i := 0; i < n; i++ {
		l := &EventLoop{}
		l.ConnCunt.Swap(int64(i))
		loops = append(loops, l)
	}

	strategy := RoundRobin()

	for i := 0; i < n; i++ {
		l := strategy(loops)
		assert.Equal(t, int64(i), l.ConnectionCount())
	}
}
