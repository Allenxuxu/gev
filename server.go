// +build linux

package gev

import (
	"fmt"
	"net"
	"runtime"
	"strconv"

	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/gev/listener"
	"github.com/Allenxuxu/ringbuffer"
	"github.com/Allenxuxu/toolkit/sync"
	"golang.org/x/sys/unix"
)

type Handler interface {
	OnConnect(c *connection.Connection)
	OnMessage(c *connection.Connection, buffer *ringbuffer.RingBuffer) []byte
	OnClose()
}

type Server struct {
	loop          *eventloop.EventLoop
	workLoops     []*eventloop.EventLoop
	nextLoopIndex int
	callback      Handler

	opts *Options
}

func NewServer(handler Handler, opts ...Option) (server *Server, err error) {
	options := newOptions(opts...)
	server = new(Server)
	server.callback = handler
	server.opts = options
	server.loop, err = eventloop.New()
	if err != nil {
		_ = server.loop.Stop()
		return nil, err
	}

	l, err := listener.New(server.opts.Network, server.opts.Address, server.handleNewConnection)
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

func (s *Server) nextLoop() *eventloop.EventLoop {
	// TODO 更多的负载方式
	loop := s.workLoops[s.nextLoopIndex]
	s.nextLoopIndex = (s.nextLoopIndex + 1) % len(s.workLoops)
	return loop
}

func (s *Server) handleNewConnection(fd int, sa *unix.Sockaddr) {
	loop := s.nextLoop()

	c := connection.New(fd, loop, s.callback.OnMessage, s.callback.OnClose)
	c.SetPeerAddr(sockaddrToString(sa))

	_ = loop.AddSocketAndEnableRead(fd, c)

	s.callback.OnConnect(c)
}

func (s *Server) Start() {
	sw := sync.WaitGroupWrapper{}

	length := len(s.workLoops)
	for i := 0; i < length; i++ {
		sw.AddAndRun(s.workLoops[i].RunLoop)
	}

	sw.AddAndRun(s.loop.RunLoop)
	sw.Wait()
}

func (s *Server) Stop() {
	_ = s.loop.Stop()

	for k := range s.workLoops {
		_ = s.workLoops[k].Stop()
	}
}

func sockaddrToString(sa *unix.Sockaddr) string {
	switch sa := (*sa).(type) {
	case *unix.SockaddrInet4:
		return net.JoinHostPort(net.IP(sa.Addr[:]).String(), strconv.Itoa(sa.Port))
	case *unix.SockaddrInet6:
		return net.JoinHostPort(net.IP(sa.Addr[:]).String(), strconv.Itoa(sa.Port))
	default:
		return fmt.Sprintf("(unknown - %T)", sa)
	}
}
