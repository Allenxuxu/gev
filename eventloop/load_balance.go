package eventloop

type LoadBalanceStrategy func([]*EventLoop) *EventLoop

func RoundRobin() LoadBalanceStrategy {
	var nextLoopIndex int
	return func(loops []*EventLoop) *EventLoop {
		l := loops[nextLoopIndex]
		nextLoopIndex = (nextLoopIndex + 1) % len(loops)
		return l
	}
}

func LeastConnection() LoadBalanceStrategy {
	return func(loops []*EventLoop) *EventLoop {
		l := loops[0]

		for i := 1; i < len(loops); i++ {
			if loops[i].ConnectionCount() < l.ConnectionCount() {
				l = loops[i]
			}
		}

		return l
	}
}
