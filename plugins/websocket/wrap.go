package websocket

import (
	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/gev/plugins/websocket/ws"
	"github.com/Allenxuxu/gev/plugins/websocket/ws/util"
)

// WSHandler WebSocket Server 注册接口
type WSHandler interface {
	OnConnect(c *connection.Connection)
	OnMessage(c *connection.Connection, msg []byte) (ws.MessageType, []byte)
	OnClose(c *connection.Connection)
}

// HandlerWrap gev Handler wrap
type HandlerWrap struct {
	wsHandler WSHandler
	Upgrade   *ws.Upgrader
}

// NewHandlerWrap websocket handler wrap
func NewHandlerWrap(u *ws.Upgrader, wsHandler WSHandler) *HandlerWrap {
	return &HandlerWrap{
		wsHandler: wsHandler,
		Upgrade:   u,
	}
}

// OnConnect wrap
func (s *HandlerWrap) OnConnect(c *connection.Connection) {
	s.wsHandler.OnConnect(c)
}

// OnMessage wrap
func (s *HandlerWrap) OnMessage(c *connection.Connection, ctx interface{}, payload []byte) []byte {
	header, ok := ctx.(*ws.Header)
	if !ok && len(payload) != 0 { // 升级协议 握手
		return payload
	}

	if ok {
		if header.OpCode.IsControl() {
			var (
				out []byte
				err error
			)
			switch header.OpCode {
			case ws.OpClose:
				out, err = util.HandleClose(header, payload)
				if err != nil {
					log.Error(err)
				}
				_ = c.ShutdownWrite()
			case ws.OpPing:
				out, err = util.HandlePing(payload)
				if err != nil {
					log.Error(err)
				}
			case ws.OpPong:
				out, err = util.HandlePong(payload)
				if err != nil {
					log.Error(err)
				}
			}
			return out
		}

		messageType, out := s.wsHandler.OnMessage(c, payload)
		if len(out) > 0 {
			var frame *ws.Frame
			switch messageType {
			case ws.MessageBinary:
				frame = ws.NewBinaryFrame(out)
			case ws.MessageText:
				frame = ws.NewTextFrame(out)
			}
			var err error
			out, err = ws.FrameToBytes(frame)
			if err != nil {
				log.Error(err)
			}

			return out
		}
	}
	return nil
}

// OnClose wrap
func (s *HandlerWrap) OnClose(c *connection.Connection) {
	s.wsHandler.OnClose(c)
}
