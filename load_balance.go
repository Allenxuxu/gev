package gev

import "github.com/Allenxuxu/gev/eventloop"

type BalanceStrategy func([]*eventloop.EventLoop) *eventloop.EventLoop

func RoundRobin() BalanceStrategy {
	var nextLoopIndex int
	return func(loops []*eventloop.EventLoop) *eventloop.EventLoop {
		l := loops[nextLoopIndex]
		nextLoopIndex = (nextLoopIndex + 1) % len(loops)
		return l
	}
}

func LeastConnection() BalanceStrategy {
	return func(loops []*eventloop.EventLoop) *eventloop.EventLoop {
		l := loops[0]

		for i := 1; i < len(loops); i++ {
			if loops[i].ConnectionCount() < l.ConnectionCount() {
				l = loops[i]
			}
		}

		return l
	}
}
