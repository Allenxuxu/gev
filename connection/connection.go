// +build linux

package connection

import (
	"log"

	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/ringbuffer"
	"golang.org/x/sys/unix"
)

type ReadCallback func(c *Connection, buffer *ringbuffer.RingBuffer) []byte
type CloseCallback func()

type Connection struct {
	fd        int
	outBuffer *ringbuffer.RingBuffer // write buffer
	inBuffer  *ringbuffer.RingBuffer // read buffer

	readCallback  ReadCallback
	closeCallback CloseCallback
	loop          *eventloop.EventLoop
	peerAddr      string
}

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

func (c *Connection) SetPeerAddr(addr string) {
	c.peerAddr = addr
}

func (c *Connection) PeerAddr() string {
	return c.peerAddr
}

// Send 用来在非 loop 协程发送
func (c *Connection) Send(buffer []byte) {
	c.loop.QueueInLoop(func() {
		c.sendInLoop(buffer)
	})
}

// HandleEvent 内部使用，eventloop 回调
func (c *Connection) HandleEvent(fd int, events uint32) {
	switch {
	case events&(unix.EPOLLIN|unix.EPOLLPRI|unix.EPOLLRDHUP) != 0:
		c.handleRead(fd)
	case events&unix.EPOLLOUT != 0:
		c.handleWrite(fd)
	case events&unix.EPOLLERR != 0:
		c.handleError(fd)
	case ((events & unix.POLLHUP) != 0) && ((events & unix.POLLIN) == 0):
		c.handleClose(fd)
	default:
		log.Println("unexcept events")
	}
}

func (c *Connection) handleRead(fd int) {
	// TODO 避免这次内存拷贝
	buf := *c.loop.PacketBuf()
	n, err := unix.Read(c.fd, buf)
	if n == 0 || err != nil {
		if err != unix.EAGAIN {
			c.handleClose(fd)
			//panic(err)
		}
		log.Println(n, err)
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
		return
	}
	c.outBuffer.Retrieve(n)

	if n == len(first) && len(end) > 0 {
		n, err = unix.Write(c.fd, end)
		if err != nil {
			return
		}
		c.outBuffer.Retrieve(n)
	}

	if c.outBuffer.Length() == 0 {
		_ = c.loop.EnableRead(fd)
	}
}

func (c *Connection) handleClose(fd int) {
	_ = unix.Close(fd)
	c.loop.DeleteFdInLoop(fd)

	c.closeCallback()
}

func (c *Connection) handleError(fd int) {
	c.handleClose(fd)
}

func (c *Connection) sendInLoop(data []byte) {
	if c.outBuffer.Length() != 0 {
		_, _ = c.outBuffer.Write(data)
	} else {
		n, err := unix.Write(c.fd, data)
		if n == 0 || err != nil {
			_, _ = c.outBuffer.Write(data)
		} else if n < len(data) {
			_, _ = c.outBuffer.Write(data[n:])
		}
	}

	if c.outBuffer.Length() > 0 {
		_ = c.loop.EnableReadWrite(c.fd)
	}
}
