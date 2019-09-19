package connection

import (
	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/gev/poller"
	"github.com/Allenxuxu/ringbuffer"
	"golang.org/x/sys/unix"
)

// ReadCallback 数据可读回调函数
type ReadCallback func(c *Connection, buffer *ringbuffer.RingBuffer) []byte

// CloseCallback 关闭回调函数
type CloseCallback func()

// Connection TCP 连接
type Connection struct {
	fd        int
	outBuffer *ringbuffer.RingBuffer // write buffer
	inBuffer  *ringbuffer.RingBuffer // read buffer

	readCallback  ReadCallback
	closeCallback CloseCallback
	loop          *eventloop.EventLoop
	peerAddr      string
	ctx           interface{}
}

// 创建 Connection
func New(fd int, loop *eventloop.EventLoop, readCb ReadCallback, closeCb CloseCallback) *Connection {
	return &Connection{
		fd:            fd,
		outBuffer:     ringbuffer.New(1024),
		inBuffer:      ringbuffer.New(1024),
		readCallback:  readCb,
		closeCallback: closeCb,
		loop:          loop,
	}
}

// Context 获取 Context
func (c *Connection) Context() interface{} {
	return c.ctx
}

// SetContext 设置 Context
func (c *Connection) SetContext(ctx interface{}) {
	c.ctx = ctx
}

// SetPeerAddr 内部使用，设置客户端地址信息
func (c *Connection) SetPeerAddr(addr string) {
	c.peerAddr = addr
}

// PeerAddr 获取客户端地址信息
func (c *Connection) PeerAddr() string {
	return c.peerAddr
}

// Send 用来在非 loop 协程发送
func (c *Connection) Send(buffer []byte) {
	c.loop.QueueInLoop(func() {
		c.sendInLoop(buffer)
	})
}

// HandleEvent 内部使用，event loop 回调
func (c *Connection) HandleEvent(fd int, events poller.Event) {
	if events&poller.EventErr != 0 {
		c.handleClose(fd)
		return
	}

	//log.Println(fd, debug(events), c.inBuffer.Capacity(), c.inBuffer.Length(), c.outBuffer.Capacity(), c.outBuffer.Length())
	if c.outBuffer.Length() != 0 {
		if events&poller.EventWrite != 0 {
			c.handleWrite(fd)
		}
	} else if events&poller.EventRead != 0 {
		c.handleRead(fd)
	}
}

//func debug(events poller.Event) string {
//	var ret string
//	if events&poller.EventErr != 0 {
//		ret += "EventErr "
//	}
//	if events&poller.EventRead != 0 {
//		ret += "EventRead "
//	}
//	if events&poller.EventWrite != 0 {
//		ret += "EventWrite "
//	}
//
//	return ret
//}

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
	_, _ = c.inBuffer.Write(buf[:n])

	out := c.readCallback(c, c.inBuffer)
	if len(out) != 0 {
		c.sendInLoop(out)
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
	//log.Println("close ", fd)
	_ = unix.Close(fd)
	c.loop.DeleteFdInLoop(fd)

	c.closeCallback()
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
