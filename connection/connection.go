package connection

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/gev/poller"
	"github.com/Allenxuxu/ringbuffer"
	"github.com/Allenxuxu/toolkit/sync/atomic"
	"golang.org/x/sys/unix"
)

// ReadCallback 数据可读回调函数
type ReadCallback func(c *Connection, ctx interface{}, data []byte) []byte

// CloseCallback 关闭回调函数
type CloseCallback func(c *Connection)

// Connection TCP 连接
type Connection struct {
	fd            int
	connected     atomic.Bool
	outBuffer     *ringbuffer.RingBuffer // write buffer
	inBuffer      *ringbuffer.RingBuffer // read buffer
	readCallback  ReadCallback
	closeCallback CloseCallback
	loop          *eventloop.EventLoop
	peerAddr      string
	ctx           interface{}

	dataPacket Protocol
}

// New 创建 Connection
func New(fd int, loop *eventloop.EventLoop, sa *unix.Sockaddr, dataPacket Protocol, readCb ReadCallback, closeCb CloseCallback) *Connection {
	conn := &Connection{
		fd:            fd,
		peerAddr:      sockaddrToString(sa),
		outBuffer:     ringbuffer.New(1024),
		inBuffer:      ringbuffer.New(1024),
		readCallback:  readCb,
		closeCallback: closeCb,
		loop:          loop,
		dataPacket:    dataPacket,
	}
	conn.connected.Set(true)

	return conn
}

// Context 获取 Context
func (c *Connection) Context() interface{} {
	return c.ctx
}

// SetContext 设置 Context
func (c *Connection) SetContext(ctx interface{}) {
	c.ctx = ctx
}

// PeerAddr 获取客户端地址信息
func (c *Connection) PeerAddr() string {
	return c.peerAddr
}

// Connected 是否已连接
func (c *Connection) Connected() bool {
	return c.connected.Get()
}

// Send 用来在非 loop 协程发送
func (c *Connection) Send(buffer []byte) error {
	if !c.connected.Get() {
		return errors.New("connection closed")
	}

	c.loop.QueueInLoop(func() {
		c.sendInLoop(c.dataPacket.Packet(c, buffer))
	})
	return nil
}

// ShutdownWrite 关闭可写端，等待读取完接收缓冲区所有数据
func (c *Connection) ShutdownWrite() error {
	c.connected.Set(false)
	return unix.Shutdown(c.fd, unix.SHUT_WR)
}

// HandleEvent 内部使用，event loop 回调
func (c *Connection) HandleEvent(fd int, events poller.Event) {
	if events&poller.EventErr != 0 {
		c.handleClose(fd)
		return
	}

	if c.outBuffer.Length() != 0 {
		if events&poller.EventWrite != 0 {
			c.handleWrite(fd)
		}
	} else if events&poller.EventRead != 0 {
		c.handleRead(fd)
	}
}

func (c *Connection) handlerProtocol(buffer *ringbuffer.RingBuffer) []byte {
	ctx, receivedData := c.dataPacket.UnPacket(c, buffer)
	if ctx != nil || len(receivedData) != 0 {
		sendData := c.readCallback(c, ctx, receivedData)
		if len(sendData) > 0 {
			return c.dataPacket.Packet(c, sendData)
		}
	}
	return nil
}

func (c *Connection) handleRead(fd int) {
	// TODO 避免这次内存拷贝
	buf := c.loop.PacketBuf()
	n, err := unix.Read(c.fd, buf)
	if n == 0 || err != nil {
		if err != unix.EAGAIN {
			c.handleClose(fd)
		}
		return
	}

	if c.inBuffer.Length() == 0 {
		buffer := ringbuffer.NewWithData(buf[:n])
		out := c.handlerProtocol(buffer)

		if buffer.Length() > 0 {
			first, _ := buffer.PeekAll()
			_, _ = c.inBuffer.Write(first)
		}
		if len(out) != 0 {
			c.sendInLoop(out)
		}
	} else {
		_, _ = c.inBuffer.Write(buf[:n])
		out := c.handlerProtocol(c.inBuffer)
		if len(out) != 0 {
			c.sendInLoop(out)
		}
	}
}

func (c *Connection) handleWrite(fd int) {
	first, end := c.outBuffer.PeekAll()
	n, err := unix.Write(c.fd, first)
	if err != nil {
		if err == unix.EAGAIN {
			return
		}
		c.handleClose(fd)
		return
	}
	c.outBuffer.Retrieve(n)

	if n == len(first) && len(end) > 0 {
		n, err = unix.Write(c.fd, end)
		if err != nil {
			if err == unix.EAGAIN {
				return
			}
			c.handleClose(fd)
			return
		}
		c.outBuffer.Retrieve(n)
	}

	if c.outBuffer.Length() == 0 {
		_ = c.loop.EnableRead(fd)
	}
}

func (c *Connection) handleClose(fd int) {
	c.connected.Set(false)
	_ = unix.Close(fd)
	c.loop.DeleteFdInLoop(fd)

	c.closeCallback(c)
}

func (c *Connection) sendInLoop(data []byte) {
	if c.outBuffer.Length() > 0 {
		_, _ = c.outBuffer.Write(data)
	} else {
		n, err := unix.Write(c.fd, data)
		if err != nil {
			if err == unix.EAGAIN {
				return
			}
			c.handleClose(c.fd)
			return
		}
		if n == 0 {
			_, _ = c.outBuffer.Write(data)
		} else if n < len(data) {
			_, _ = c.outBuffer.Write(data[n:])
		}

		if c.outBuffer.Length() > 0 {
			_ = c.loop.EnableReadWrite(c.fd)
		}
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
