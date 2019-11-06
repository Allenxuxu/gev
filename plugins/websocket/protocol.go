package websocket

import (
	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/gev/plugins/websocket/ws"
	"github.com/Allenxuxu/ringbuffer"
)

// Protocol websocket
type Protocol struct {
	upgrade *ws.Upgrader
}

// New 创建 websocket Protocol
func New(u *ws.Upgrader) *Protocol {
	return &Protocol{upgrade: u}
}

// UnPacket 解析 websocket 协议，返回 header ，payload
func (p *Protocol) UnPacket(c *connection.Connection, buffer *ringbuffer.RingBuffer) (ctx interface{}, out []byte) {
	upgraded := c.Context()
	if upgraded == nil {
		var err error
		out, _, err = p.upgrade.Upgrade(buffer)
		if err != nil {
			log.Error("Websocket Upgrade :", err)
			return
		}
		c.SetContext(true)
	} else {
		header, err := ws.VirtualReadHeader(buffer)
		if err != nil {
			if err != ws.ErrHeaderNotReady {
				log.Error(err)
			}
			return
		}
		if buffer.VirtualLength() >= int(header.Length) {
			buffer.VirtualFlush()

			payload := make([]byte, int(header.Length))
			_, _ = buffer.Read(payload)

			if header.Masked {
				ws.Cipher(payload, header.Mask, 0)
			}

			ctx = &header
			out = payload
		} else {
			buffer.VirtualRevert()
		}
	}
	return
}

// Packet 直接返回
func (p *Protocol) Packet(c *connection.Connection, data []byte) []byte {
	return data
}
