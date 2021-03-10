package connection

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/Allenxuxu/gev/metrics"

	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/gev/poller"
	"github.com/Allenxuxu/ringbuffer"
	"github.com/Allenxuxu/ringbuffer/pool"
	"github.com/Allenxuxu/toolkit/sync/atomic"
	"github.com/RussellLuo/timingwheel"
	"github.com/gobwas/pool/pbytes"
	"golang.org/x/sys/unix"
)

type CallBack interface {
	OnMessage(c *Connection, ctx interface{}, data []byte) []byte
	OnClose(c *Connection)
}

// Connection TCP 连接
type Connection struct {
	fd           int
	connected    atomic.Bool
	outBuffer    *ringbuffer.RingBuffer // write buffer
	outBufferLen atomic.Int64
	inBuffer     *ringbuffer.RingBuffer // read buffer
	inBufferLen  atomic.Int64
	callBack     CallBack
	loop         *eventloop.EventLoop
	peerAddr     string
	ctx          interface{}
	KeyValueContext

	idleTime    time.Duration
	activeTime  atomic.Int64
	timingWheel *timingwheel.TimingWheel

	protocol Protocol
}

var ErrConnectionClosed = errors.New("connection closed")

// New 创建 Connection
func New(fd int, loop *eventloop.EventLoop, sa unix.Sockaddr, protocol Protocol, tw *timingwheel.TimingWheel, idleTime time.Duration, callBack CallBack) *Connection {
	conn := &Connection{
		fd:          fd,
		peerAddr:    sockAddrToString(sa),
		outBuffer:   pool.Get(),
		inBuffer:    pool.Get(),
		callBack:    callBack,
		loop:        loop,
		idleTime:    idleTime,
		timingWheel: tw,
		protocol:    protocol,
	}
	conn.connected.Set(true)

	if conn.idleTime > 0 {
		_ = conn.activeTime.Swap(time.Now().Unix())
		conn.timingWheel.AfterFunc(conn.idleTime, conn.closeTimeoutConn())
	}

	return conn
}

func (c *Connection) closeTimeoutConn() func() {
	return func() {
		now := time.Now()
		intervals := now.Sub(time.Unix(c.activeTime.Get(), 0))
		if intervals >= c.idleTime {
			_ = c.Close()
		} else {
			c.timingWheel.AfterFunc(c.idleTime-intervals, c.closeTimeoutConn())
		}
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
		return ErrConnectionClosed
	}

	c.loop.QueueInLoop(func() {
		c.sendInLoop(c.protocol.Packet(c, buffer))
	})
	return nil
}

// Close 关闭连接
func (c *Connection) Close() error {
	if !c.connected.Get() {
		return ErrConnectionClosed
	}

	c.loop.QueueInLoop(func() {
		c.handleClose(c.fd)
	})
	return nil
}

// ShutdownWrite 关闭可写端，等待读取完接收缓冲区所有数据
func (c *Connection) ShutdownWrite() error {
	c.connected.Set(false)
	return unix.Shutdown(c.fd, unix.SHUT_WR)
}

// ReadBufferLength read buffer 当前积压的数据长度
func (c *Connection) ReadBufferLength() int64 {
	return c.inBufferLen.Get()
}

// WriteBufferLength write buffer 当前积压的数据长度
func (c *Connection) WriteBufferLength() int64 {
	return c.outBufferLen.Get()
}

// HandleEvent 内部使用，event loop 回调
func (c *Connection) HandleEvent(fd int, events poller.Event) {
	now := time.Now()
	if c.idleTime > 0 {
		_ = c.activeTime.Swap(now.Unix())
	}

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

	c.inBufferLen.Swap(int64(c.inBuffer.Length()))
	c.outBufferLen.Swap(int64(c.outBuffer.Length()))

	if metrics.Enable.Get() {
		metrics.ConnBufferCap.WithLabelValues("in", c.PeerAddr()).Set(float64(c.inBuffer.Capacity()))
		metrics.ConnBufferLen.WithLabelValues("in", c.PeerAddr()).Set(float64(c.inBuffer.Length()))

		metrics.ConnBufferCap.WithLabelValues("out", c.PeerAddr()).Set(float64(c.outBuffer.Capacity()))
		metrics.ConnBufferLen.WithLabelValues("out", c.PeerAddr()).Set(float64(c.outBuffer.Length()))

		var action string
		if events&poller.EventWrite != 0 {
			action = "write"
		} else if events&poller.EventRead != 0 {
			action = "read"
		}
		if len(action) > 0 {
			metrics.ConnHandlerDuration.WithLabelValues(action).Set(float64(time.Since(now).Microseconds()))
		}
	}
}

func (c *Connection) handlerProtocol(buffer *ringbuffer.RingBuffer) []byte {
	// 在调用方函数里归还
	out := pbytes.GetCap(1024)
	ctx, receivedData := c.protocol.UnPacket(c, buffer)
	for ctx != nil || len(receivedData) != 0 {
		sendData := c.callBack.OnMessage(c, ctx, receivedData)
		if len(sendData) > 0 {
			out = append(out, c.protocol.Packet(c, sendData)...)
		}

		ctx, receivedData = c.protocol.UnPacket(c, buffer)
	}
	return out
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

		pbytes.Put(out)
	} else {
		_, _ = c.inBuffer.Write(buf[:n])
		out := c.handlerProtocol(c.inBuffer)
		if len(out) != 0 {
			c.sendInLoop(out)
		}

		pbytes.Put(out)
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
		if err := c.loop.EnableRead(fd); err != nil {
			log.Error("[EnableRead]", err)
		}
	}
}

func (c *Connection) handleClose(fd int) {
	if c.connected.Get() {
		c.connected.Set(false)
		c.loop.DeleteFdInLoop(fd)

		c.callBack.OnClose(c)
		if err := unix.Close(fd); err != nil {
			log.Error("[close fd]", err)
		}

		pool.Put(c.inBuffer)
		pool.Put(c.outBuffer)
	}
}

func (c *Connection) sendInLoop(data []byte) {
	if c.outBuffer.Length() > 0 {
		_, _ = c.outBuffer.Write(data)
	} else {
		n, err := unix.Write(c.fd, data)
		if err != nil && err != unix.EAGAIN {
			c.handleClose(c.fd)
			return
		}

		if n <= 0 {
			_, _ = c.outBuffer.Write(data)
		} else if n < len(data) {
			_, _ = c.outBuffer.Write(data[n:])
		}

		if c.outBuffer.Length() > 0 {
			_ = c.loop.EnableReadWrite(c.fd)
		}
	}
}

func sockAddrToString(sa unix.Sockaddr) string {
	switch sa := (sa).(type) {
	case *unix.SockaddrInet4:
		return net.JoinHostPort(net.IP(sa.Addr[:]).String(), strconv.Itoa(sa.Port))
	case *unix.SockaddrInet6:
		return net.JoinHostPort(net.IP(sa.Addr[:]).String(), strconv.Itoa(sa.Port))
	default:
		return fmt.Sprintf("(unknown - %T)", sa)
	}
}
