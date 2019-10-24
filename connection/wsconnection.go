package connection

import (
	"errors"

	"github.com/Allenxuxu/gev/ws"
)

// SendWebsocketData 发送 websocket message
func (c *Connection) SendWebsocketData(messageType ws.MessageType, buffer []byte) error {
	if !c.connected.Get() {
		return errors.New("connection closed")
	}

	var frame *ws.Frame
	switch messageType {
	case ws.MessageBinary:
		frame = ws.NewBinaryFrame(buffer)
	case ws.MessageText:
		frame = ws.NewTextFrame(buffer)
	}
	data, err := ws.FrameToBytes(frame)
	if err != nil {
		return err
	}

	c.loop.QueueInLoop(func() {
		c.sendInLoop(data)
	})
	return nil
}

// CloseWebsocket 关闭 websocket 连接
func (c *Connection) CloseWebsocket(reason string) error {
	if !c.connected.Get() {
		return errors.New("connection closed")
	}

	data, err := ws.FrameToBytes(ws.NewCloseFrame(ws.NewCloseFrameBody(ws.StatusNormalClosure, reason)))
	if err != nil {
		return err
	}

	c.loop.QueueInLoop(func() {
		c.sendInLoop(data)
		_ = c.ShutdownWrite()
	})
	return nil
}
