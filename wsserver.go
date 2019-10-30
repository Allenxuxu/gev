package gev

import (
	"github.com/Allenxuxu/gev/plugins/websocket"
	"github.com/Allenxuxu/gev/ws"
)

// NewWebSocketServer 创建 WebSocket Server
func NewWebSocketServer(handler websocket.WebSocketHandler, u *ws.Upgrader, opts ...Option) (server *Server, err error) {
	opts = append(opts, Protocol(websocket.New(u)))
	return NewServer(websocket.NewHandlerWrap(u, handler), opts...)
}
