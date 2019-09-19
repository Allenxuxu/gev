package listener

import (
	"errors"
	"net"
	"os"

	"github.com/Allenxuxu/gev/poller"
	"golang.org/x/sys/unix"
)

// HandleConnFunc 处理新连接
type HandleConnFunc func(fd int, sa *unix.Sockaddr)

// Listener 监听TCP连接
type Listener struct {
	file    *os.File
	fd      int
	handleC HandleConnFunc
}

// New 创建Listener
func New(network, addr string, handlerConn HandleConnFunc) (*Listener, error) {
	listener, err := net.Listen(network, addr)
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
		file:    file,
		fd:      fd,
		handleC: handlerConn}, nil
}

// HandleEvent 内部使用，供 event loop 回调处理事件
func (l *Listener) HandleEvent(fd int, events poller.Event) {
	if events&poller.EventRead != 0 {
		nfd, sa, err := unix.Accept(fd)
		if err != nil {
			//TODO 错误处理
			if err != unix.EAGAIN {
				panic("accept: " + err.Error())
			}
			return
		}
		if err := unix.SetNonblock(nfd, true); err != nil {
			panic(err)
		}

		l.handleC(nfd, &sa)
	} else {
		panic("listener unexpect events")
	}
}

// Fd Listener fd
func (l *Listener) Fd() int {
	return l.fd
}
