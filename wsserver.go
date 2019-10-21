package gev

import (
	"errors"
	"log"
	"runtime"

	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/gev/listener"
	"github.com/Allenxuxu/gev/ws"
	"github.com/Allenxuxu/gev/ws/handler"
	"github.com/Allenxuxu/ringbuffer"
	"github.com/RussellLuo/timingwheel"
)

// WebSocketHandler WebSocket Server 注册接口
type WebSocketHandler interface {
	OnConnect(c *connection.Connection)
	OnMessage(c *connection.Connection, msg []byte) (ws.MessageType, []byte)
	OnClose(c *connection.Connection)
}

type handlerWrap struct {
	wsHandler WebSocketHandler
	Upgrade   *ws.Upgrader
}

func newHandlerWrap(u *ws.Upgrader, wsHandler WebSocketHandler) *handlerWrap {
	return &handlerWrap{
		wsHandler: wsHandler,
		Upgrade:   u,
	}
}

func (s *handlerWrap) OnConnect(c *connection.Connection) {
	s.wsHandler.OnConnect(c)
}

func (s *handlerWrap) OnMessage(c *connection.Connection, buffer *ringbuffer.RingBuffer) (out []byte) {
	if !c.Upgraded {
		var err error
		out, _, err = s.Upgrade.Upgrade(buffer)
		if err != nil {
			log.Println("Websocket Upgrade :", err)
			return
		}
		c.Upgraded = true
		return
	}
	return handler.HandleWebSocket(c, buffer, s.wsHandler.OnMessage)
}

func (s *handlerWrap) OnClose(c *connection.Connection) {
	s.wsHandler.OnClose(c)
}

// NewWebSocketServer 创建 WebSocket Server
func NewWebSocketServer(handler WebSocketHandler, opts ...Option) (server *Server, err error) {
	if handler == nil {
		return nil, errors.New("handler is nil")
	}

	options := newOptions(opts...)
	server = new(Server)
	server.callback = newHandlerWrap(options.Upgrade, handler)
	server.opts = options
	server.timingWheel = timingwheel.NewTimingWheel(server.opts.tick, server.opts.wheelSize)
	server.loop, err = eventloop.New()
	if err != nil {
		_ = server.loop.Stop()
		return nil, err
	}

	l, err := listener.New(server.opts.Network, server.opts.Address, options.ReusePort, server.handleNewConnection)
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
