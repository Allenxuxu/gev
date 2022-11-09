// +build windows

package gev

import (
	"errors"
	"io"
	"net"
	stdsync "sync"
	"time"

	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/gev/poller"
	"github.com/Allenxuxu/ringbuffer"
	"github.com/Allenxuxu/toolkit/sync"
	"github.com/Allenxuxu/toolkit/sync/atomic"
	"github.com/RussellLuo/timingwheel"
)

// Handler Server 注册接口
type Handler interface {
	CallBack
	OnConnect(c *Connection)
}

// Server gev Server
type Server struct {
	listener    net.Listener
	callback    Handler
	connections []*Connection

	timingWheel *timingwheel.TimingWheel
	opts        *Options
	running     atomic.Bool
	dying       chan struct{}
}

// NewServer 创建 Server
func NewServer(handler Handler, opts ...Option) (server *Server, err error) {
	if handler == nil {
		return nil, errors.New("handler is nil")
	}
	options := newOptions(opts...)
	server = new(Server)
	server.dying = make(chan struct{})
	server.callback = handler
	server.opts = options
	server.timingWheel = timingwheel.NewTimingWheel(server.opts.tick, server.opts.wheelSize)

	server.listener, err = net.Listen(server.opts.Network, server.opts.Address)
	if err != nil {
		return nil, err
	}

	return
}

// RunAfter 延时任务
func (s *Server) RunAfter(d time.Duration, f func()) *timingwheel.Timer {
	return s.timingWheel.AfterFunc(d, f)
}

// RunEvery 定时任务
func (s *Server) RunEvery(d time.Duration, f func()) *timingwheel.Timer {
	return s.timingWheel.ScheduleFunc(&everyScheduler{Interval: d}, f)
}

// Start 启动 Server
func (s *Server) Start() {
	sw := sync.WaitGroupWrapper{}
	s.timingWheel.Start()

	sw.AddAndRun(func() {
		for {
			select {
			case <-s.dying:
				return

			default:
				conn, err := s.listener.Accept()
				if err != nil {
					log.Errorf("accept error: %v", err)
					continue
				}

				connection := NewConnection(conn, s.opts.Protocol, s.timingWheel, s.opts.IdleTime, s.callback)
				s.connections = append(s.connections, connection)

				s.callback.OnConnect(connection)

				sw.AddAndRun(func() {
					connection.readLoop()
				})
				sw.AddAndRun(func() {
					connection.writeLoop()
				})
			}
		}
	})

	s.running.Set(true)

	log.Infof("server run in windows")
	sw.Wait()
}

// Stop 关闭 Server
func (s *Server) Stop() {
	if s.running.Get() {
		close(s.dying)
		s.running.Set(false)

		s.timingWheel.Stop()
		if err := s.listener.Close(); err != nil {
			log.Error(err)
		}

		for _, c := range s.connections {
			c.Close()
		}
	}
}

// Options 返回 options
func (s *Server) Options() Options {
	return *s.opts
}

// connection

type CallBack interface {
	OnMessage(c *Connection, ctx interface{}, data []byte) interface{}
	OnClose(c *Connection)
}

// Connection TCP 连接
type Connection struct {
	conn         net.Conn
	connected    atomic.Bool
	dying        chan struct{}
	userBuffer   *[]byte
	buffer       *ringbuffer.RingBuffer
	outBuffer    *ringbuffer.RingBuffer // write buffer
	inBuffer     *ringbuffer.RingBuffer // read buffer
	outBufferLen atomic.Int64
	inBufferLen  atomic.Int64
	callBack     CallBack
	ctx          interface{}
	KeyValueContext

	mu         stdsync.Mutex
	taskQueueW []func()
	taskQueueR []func()

	idleTime    time.Duration
	activeTime  atomic.Int64
	timingWheel *timingwheel.TimingWheel

	protocol Protocol
}

var ErrConnectionClosed = errors.New("connection closed")

// NewConnection 创建 Connection
func NewConnection(
	conn net.Conn,

	protocol Protocol,
	tw *timingwheel.TimingWheel,
	idleTime time.Duration,
	callBack CallBack) *Connection {

	userBuffer := make([]byte, 4096)

	connection := &Connection{
		conn:        conn,
		dying:       make(chan struct{}),
		outBuffer:   ringbuffer.GetFromPool(),
		inBuffer:    ringbuffer.GetFromPool(),
		callBack:    callBack,
		idleTime:    idleTime,
		timingWheel: tw,
		protocol:    protocol,
		buffer:      ringbuffer.New(0),
		taskQueueW:  make([]func(), 0, 1024),
		taskQueueR:  make([]func(), 0, 1024),
		userBuffer:  &userBuffer,
	}
	connection.connected.Set(true)

	if connection.idleTime > 0 {
		_ = connection.activeTime.Swap(time.Now().Unix())
		connection.timingWheel.AfterFunc(connection.idleTime, connection.closeTimeoutConn())
	}

	return connection
}

func (c *Connection) UserBuffer() *[]byte {
	return c.userBuffer
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
	return c.conn.RemoteAddr().String()
}

// Connected 是否已连接
func (c *Connection) Connected() bool {
	return c.connected.Get()
}

// Send 用来在非 loop 协程发送
func (c *Connection) Send(data interface{}, opts ...ConnectionOption) error {
	if !c.connected.Get() {
		return ErrConnectionClosed
	}

	opt := ConnectionOptions{}
	for _, o := range opts {
		o(&opt)
	}

	f := func() {
		if c.connected.Get() {
			c.sendInLoop(c.protocol.Packet(c, data))

			if opt.sendInLoopFinish != nil {
				opt.sendInLoopFinish(data)
			}
		}
	}

	c.mu.Lock()
	c.taskQueueW = append(c.taskQueueW, f)
	c.mu.Unlock()

	return nil
}

// Close 关闭连接
func (c *Connection) Close() error {
	if c.connected.Get() {
		log.Info("Close ", c.PeerAddr())

		close(c.dying)
		c.connected.Set(false)
		c.callBack.OnClose(c)

		return c.conn.Close()
	}

	return nil
}

// ShutdownWrite 关闭可写端，等待读取完接收缓冲区所有数据
func (c *Connection) ShutdownWrite() error {
	log.Info("ShutdownWrite ", c.PeerAddr())

	//return nil
	return c.Close()
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

}

func (c *Connection) readLoop() {
	buf := make([]byte, 0, 66635)

	for {
		select {
		case <-c.dying:
			return

		default:
			//c.conn.SetReadDeadline(time.Now().Add(time.Second))
			n, err := c.conn.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Info("read error: ", err)
				}

				c.Close()
				return
			}

			_, _ = c.inBuffer.Write(buf[:n])
			buf = buf[:0]
			c.handlerProtocol(&buf, c.inBuffer)

			if len(buf) != 0 {
				tmp := make([]byte, len(buf))
				copy(tmp, buf)
				_ = c.Send(tmp)
			}
			buf = buf[:cap(buf)]

			if c.idleTime > 0 {
				_ = c.activeTime.Swap(time.Now().Unix())
			}
			c.inBufferLen.Swap(int64(c.inBuffer.Length()))
		}
	}
}

func (c *Connection) writeLoop() {
	for {
		select {
		case <-c.dying:
			return

		default:
			c.doPendingFunc()

			if c.outBuffer.IsEmpty() {
				continue
			}

			c.conn.SetWriteDeadline(time.Now().Add(time.Second))

			first, end := c.outBuffer.PeekAll()
			n, err := c.conn.Write(first)
			if err != nil {
				log.Error("Write error: ", err)

				c.Close()
				return
			}
			c.outBuffer.Retrieve(n)

			if n == len(first) && len(end) > 0 {
				n, err = c.conn.Write(end)
				if err != nil {
					log.Error("Write error: ", err)

					c.Close()
					return
				}
				c.outBuffer.Retrieve(n)
			}

			if c.idleTime > 0 {
				_ = c.activeTime.Swap(time.Now().Unix())
			}

			c.outBufferLen.Swap(int64(c.outBuffer.Length()))
		}
	}
}

func (c *Connection) doPendingFunc() {
	c.mu.Lock()
	c.taskQueueW, c.taskQueueR = c.taskQueueR, c.taskQueueW
	c.mu.Unlock()

	length := len(c.taskQueueR)
	for i := 0; i < length; i++ {
		c.taskQueueR[i]()
	}

	c.taskQueueR = c.taskQueueR[:0]
}

func (c *Connection) sendInLoop(data []byte) (closed bool) {
	if !c.outBuffer.IsEmpty() {
		_, _ = c.outBuffer.Write(data)
	} else {
		n, err := c.conn.Write(data)
		if err != nil {
			log.Error("Write error: ", err)

			c.Close()

			return true
		}

		if n <= 0 {
			_, _ = c.outBuffer.Write(data)
		} else if n < len(data) {
			_, _ = c.outBuffer.Write(data[n:])
		}
	}

	return false
}

func (c *Connection) handlerProtocol(tmpBuffer *[]byte, buffer *ringbuffer.RingBuffer) {
	ctx, receivedData := c.protocol.UnPacket(c, buffer)
	for ctx != nil || len(receivedData) != 0 {
		sendData := c.callBack.OnMessage(c, ctx, receivedData)
		if sendData != nil {
			*tmpBuffer = append(*tmpBuffer, c.protocol.Packet(c, sendData)...)
		}

		ctx, receivedData = c.protocol.UnPacket(c, buffer)
	}
}

func (c *Connection) closeTimeoutConn() func() {
	return func() {
		now := time.Now()
		intervals := now.Sub(time.Unix(c.activeTime.Get(), 0))
		if intervals >= c.idleTime {
			log.Info("closeTimeoutConn ", c.conn.RemoteAddr())
			_ = c.Close()
		} else {
			c.timingWheel.AfterFunc(c.idleTime-intervals, c.closeTimeoutConn())
		}
	}
}
