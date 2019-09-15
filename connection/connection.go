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
	if ((events & unix.POLLHUP) != 0) && ((events & unix.POLLIN) == 0) {
		c.handleClose(fd)
	}
	if events&unix.EPOLLERR != 0 {
		//log.Println("epollerr", fd)
		c.handleError(fd)
	}

	if events&unix.EPOLLOUT != 0 {
		c.handleWrite(fd)
	} else if events&(unix.EPOLLIN|unix.EPOLLPRI|unix.EPOLLRDHUP) != 0 {
		c.handleRead(fd)
	}
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
	log.Println(c.inBuffer.Capacity(), c.inBuffer.Length(), c.outBuffer.Capacity(), c.outBuffer.Length())
}

func (c *Connection) handleError(fd int) {
	//err := unix.Close(fd)
	//if err != nil {
	//	panic(err)
	//}
	//c.loop.DeleteFdInLoop(fd)
}

func (c *Connection) sendInLoop(data []byte) {
	if c.outBuffer.Length() > 0 {
		_, _ = c.outBuffer.Write(data)
	} else {
		n, err := unix.Write(c.fd, data)
		log.Println("write ", n, err)
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
	}

	if c.outBuffer.Length() > 0 {
		_ = c.loop.EnableReadWrite(c.fd)
	}
}
