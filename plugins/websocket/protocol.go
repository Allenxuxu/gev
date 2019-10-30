package websocket

import (
	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/ws"
	"github.com/Allenxuxu/ringbuffer"
	"log"
)

type Protocol struct {
	upgrade *ws.Upgrader
}

func New(u *ws.Upgrader) *Protocol {
	return &Protocol{upgrade: u}
}

func (p *Protocol) UnPacket(c *connection.Connection, buffer *ringbuffer.RingBuffer) (ctx interface{}, out []byte) {
	if !c.Upgraded {
		var err error
		out, _, err = p.upgrade.Upgrade(buffer)
		if err != nil {
			log.Println("Websocket Upgrade :", err)
			return
		}
		c.Upgraded = true
	} else {
		header, err := ws.VirtualReadHeader(buffer)
		if err != nil {
			log.Println(err)
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

func (p *Protocol) Packet(c *connection.Connection, data []byte) []byte {
	return data
}
