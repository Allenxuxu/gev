package gev

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Allenxuxu/gev/eventloop"
)

func TestLeastConnection(t *testing.T) {
	var (
		loops []*eventloop.EventLoop
		n     = 100
		min   int64
	)

	for i := 0; i < n; i++ {
		l := &eventloop.EventLoop{}
		connCount := int64(rand.Intn(n))
		l.ConnCunt.Swap(connCount)
		loops = append(loops, l)

		if connCount < min {
			min = connCount
		}
	}

	strategy := eventloop.LeastConnection()
	for i := 0; i < n; i++ {
		l := strategy(loops)
		assert.Equal(t, min, l.ConnectionCount())
	}

}

func TestRoundRobin(t *testing.T) {
	var (
		loops []*eventloop.EventLoop
		n     = 100
	)

	for i := 0; i < n; i++ {
		l := &eventloop.EventLoop{}
		l.ConnCunt.Swap(int64(i))
		loops = append(loops, l)
	}

	strategy := eventloop.RoundRobin()

	for i := 0; i < n; i++ {
		l := strategy(loops)
		assert.Equal(t, int64(i), l.ConnectionCount())
	}
}
