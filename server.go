package gev

import (
	"errors"
	"runtime"
	"time"

	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/gev/listener"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/toolkit/sync"
	"github.com/RussellLuo/timingwheel"
	"golang.org/x/sys/unix"
)

// Handler Server 注册接口
type Handler interface {
	OnConnect(c *connection.Connection)
	OnMessage(c *connection.Connection, ctx interface{}, data []byte) []byte
	OnClose(c *connection.Connection)
}

// Server gev Server
type Server struct {
	loop          *eventloop.EventLoop
	workLoops     []*eventloop.EventLoop
	nextLoopIndex int
	callback      Handler

	timingWheel *timingwheel.TimingWheel
	opts        *Options
}

// NewServer 创建 Server
func NewServer(handler Handler, opts ...Option) (server *Server, err error) {
	if handler == nil {
		return nil, errors.New("handler is nil")
	}
	options := newOptions(opts...)
	server = new(Server)
	server.callback = handler
	server.opts = options
	server.timingWheel = timingwheel.NewTimingWheel(server.opts.tick, server.opts.wheelSize)
	server.loop, err = eventloop.New()
	if err != nil {
		_ = server.loop.Stop()
		return nil, err
	}

	l, err := listener.New(server.opts.Network, server.opts.Address, options.ReusePort, server.loop, server.handleNewConnection)
	if err != nil {
		return nil, err
	}
	if err = server.loop.AddSocketAndEnableRead(l.Fd(), l); err != nil {
		return nil, err
	}

	if server.opts.NumLoops <= 0 {
		server.opts.NumLoops = runtime.NumCPU()
	}

	wloops := make([]*eventloop.EventLoop, server.opts.NumLoops)
	for i := 0; i < server.opts.NumLoops; i++ {
		l, err := eventloop.New()
		if err != nil {
			for j := 0; j < i; j++ {
				_ = wloops[j].Stop()
			}
			return nil, err
		}
		wloops[i] = l
	}
	server.workLoops = wloops

	return
}

// RunAfter 延时任务
func (s *Server) RunAfter(d time.Duration, f func()) *timingwheel.Timer {
	return s.timingWheel.AfterFunc(d, f)
}

// RunEvery 定时任务
func (s *Server) RunEvery(d time.Duration, f func()) *timingwheel.Timer {
	return s.timingWheel.ScheduleFunc(&everyScheduler{Interval: d}, f)
}

func (s *Server) nextLoop() *eventloop.EventLoop {
	// TODO 更多的负载方式
	loop := s.workLoops[s.nextLoopIndex]
	s.nextLoopIndex = (s.nextLoopIndex + 1) % len(s.workLoops)
	return loop
}

func (s *Server) handleNewConnection(fd int, sa *unix.Sockaddr) {
	loop := s.nextLoop()

	c := connection.New(fd, loop, sa, s.opts.Protocol, s.timingWheel, s.opts.IdleTime, s.callback.OnMessage, s.callback.OnClose)

	s.callback.OnConnect(c)
	if err := loop.AddSocketAndEnableRead(fd, c); err != nil {
		log.Error("[AddSocketAndEnableRead]", err)
	}
}

// Start 启动 Server
func (s *Server) Start() {
	sw := sync.WaitGroupWrapper{}
	s.timingWheel.Start()

	length := len(s.workLoops)
	for i := 0; i < length; i++ {
		sw.AddAndRun(s.workLoops[i].RunLoop)
	}

	sw.AddAndRun(s.loop.RunLoop)
	sw.Wait()
}

// Stop 关闭 Server
func (s *Server) Stop() {
	s.timingWheel.Stop()
	if err := s.loop.Stop(); err != nil {
		log.Error(err)
	}

	for k := range s.workLoops {
		if err := s.workLoops[k].Stop(); err != nil {
			log.Error(err)
		}
	}
}

// Options 返回 options
func (s *Server) Options() Options {
	return *s.opts
}
