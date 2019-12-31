package listener

import (
	"errors"
	"net"
	"os"

	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/gev/poller"
	reuseport "github.com/libp2p/go-reuseport"
	"golang.org/x/sys/unix"
)

// HandleConnFunc 处理新连接
type HandleConnFunc func(fd int, sa *unix.Sockaddr)

// Listener 监听TCP连接
type Listener struct {
	file     *os.File
	fd       int
	handleC  HandleConnFunc
	listener net.Listener
	loop     *eventloop.EventLoop
}

// New 创建Listener
func New(network, addr string, reusePort bool, loop *eventloop.EventLoop, handlerConn HandleConnFunc) (*Listener, error) {
	var listener net.Listener
	var err error
	if reusePort {
		listener, err = reuseport.Listen(network, addr)
	} else {
		listener, err = net.Listen(network, addr)
	}
	if err != nil {
		return nil, err
	}

	l, ok := listener.(*net.TCPListener)
	if !ok {
		return nil, errors.New("could not get file descriptor")
	}

	file, err := l.File()
	if err != nil {
		return nil, err
	}
	fd := int(file.Fd())
	if err = unix.SetNonblock(fd, true); err != nil {
		return nil, err
	}

	return &Listener{
		file:     file,
		fd:       fd,
		handleC:  handlerConn,
		listener: listener,
		loop:     loop}, nil
}

// HandleEvent 内部使用，供 event loop 回调处理事件
func (l *Listener) HandleEvent(fd int, events poller.Event) {
	if events&poller.EventRead != 0 {
		nfd, sa, err := unix.Accept(fd)
		if err != nil {
			if err != unix.EAGAIN {
				log.Error("accept:", err)
			}
			return
		}
		if err := unix.SetNonblock(nfd, true); err != nil {
			_ = unix.Close(nfd)
			log.Error("set nonblock:", err)
			return
		}

		l.handleC(nfd, &sa)
	}
}

// Close listener
func (l *Listener) Close() error {
	l.loop.QueueInLoop(func() {
		l.loop.DeleteFdInLoop(l.fd)
		if err := l.listener.Close(); err != nil {
			log.Error("[Listener] close error: ", err)
		}
	})

	return nil
}

// Fd Listener fd
func (l *Listener) Fd() int {
	return l.fd
}
