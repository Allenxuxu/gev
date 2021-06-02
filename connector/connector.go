package connector

import (
	"errors"
	"runtime"
	"time"

	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/toolkit/sync"
	"github.com/Allenxuxu/toolkit/sync/atomic"
	"github.com/RussellLuo/timingwheel"
)

type Connector struct {
	workLoops   []*eventloop.EventLoop
	opts        *ConnectorOptions
	timingWheel *timingwheel.TimingWheel
	running     atomic.Bool
}

func (c *Connector) Dial(callback connection.CallBack, network, address string, idleTime time.Duration) (*Connection, error) {
	return c.DialWithTimeout(0, callback, network, address, idleTime)
}

func (c *Connector) DialWithTimeout(timeout time.Duration, callback connection.CallBack, network, address string, idleTime time.Duration) (*Connection, error) {
	if callback == nil {
		return nil, errors.New("callback is nil")
	}

	loop := c.opts.Strategy(c.workLoops)

	conn, err := newConnection(network, address, loop, timeout, c.opts.Protocol, c.timingWheel, idleTime, callback)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func NewConnector(opts ...ConnectorOption) (connector *Connector, err error) {
	connector = new(Connector)
	connector.opts = newConnectorOptions(opts...)

	connector.timingWheel = timingwheel.NewTimingWheel(connector.opts.tick, connector.opts.wheelSize)
	if connector.opts.NumLoops <= 0 {
		connector.opts.NumLoops = runtime.NumCPU()
	}

	wloops := make([]*eventloop.EventLoop, connector.opts.NumLoops)
	for i := 0; i < connector.opts.NumLoops; i++ {
		l, err := eventloop.New()
		if err != nil {
			for j := 0; j < i; j++ {
				_ = wloops[j].Stop()
			}
			return nil, err
		}
		wloops[i] = l
	}

	connector.workLoops = wloops
	return
}

func (c *Connector) Start() {
	sw := sync.WaitGroupWrapper{}
	c.timingWheel.Start()

	length := len(c.workLoops)
	for i := 0; i < length; i++ {
		sw.AddAndRun(c.workLoops[i].RunLoop)
	}

	c.running.Set(true)
	sw.Wait()
}

func (c *Connector) Stop() {
	if c.running.Get() {
		c.running.Set(false)

		c.timingWheel.Stop()

		for k := range c.workLoops {
			if err := c.workLoops[k].Stop(); err != nil {
				log.Error(err)
			}
		}
	}
}

func (c *Connector) Options() ConnectorOptions {
	return *c.opts
}
