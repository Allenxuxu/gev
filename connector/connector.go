package connector

import (
	"errors"
	"runtime"
	"time"

	"golang.org/x/net/context"

	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/toolkit/sync"
	"github.com/Allenxuxu/toolkit/sync/atomic"
	"github.com/RussellLuo/timingwheel"
)

type Connector struct {
	workLoops   []*eventloop.EventLoop
	opts        *Options
	timingWheel *timingwheel.TimingWheel
	running     atomic.Bool
}

func (c *Connector) Dial(network, address string, callback connection.CallBack, protocol connection.Protocol, idleTime time.Duration) (*Connection, error) {
	return c.DialWithTimeout(0, network, address, callback, protocol, idleTime)
}

func (c *Connector) DialWithTimeout(timeout time.Duration, network, address string, callback connection.CallBack, protocol connection.Protocol, idleTime time.Duration) (*Connection, error) {
	if callback == nil {
		return nil, errors.New("callback is nil")
	}

	if protocol == nil {
		protocol = &connection.DefaultProtocol{}
	}

	loop := c.opts.Strategy(c.workLoops)

	dialCtx := context.Background()
	if timeout > 0 {
		subCtx, cancel := context.WithDeadline(dialCtx, time.Now().Add(timeout))
		dialCtx = subCtx
		defer cancel()
	}

	for {
		select {
		case <-dialCtx.Done():
			return nil, ErrDialTimeout
		default:
			conn, err := newConnection(dialCtx, network, address, loop, protocol, c.timingWheel, idleTime, callback)
			if err != nil {
				time.Sleep(time.Millisecond * 100)
				continue
			}

			return conn, nil
		}

	}
}

func NewConnector(opts ...Option) (connector *Connector, err error) {
	connector = new(Connector)
	connector.opts = newOptions(opts...)

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

func (c *Connector) Options() Options {
	return *c.opts
}
