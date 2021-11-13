// +build !windows

package gev

import (
	"errors"
	"net"
	"os"

	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/gev/poller"
	"github.com/libp2p/go-reuseport"
	"golang.org/x/sys/unix"
)

// handleConnFunc 处理新连接
type handleConnFunc func(fd int, sa unix.Sockaddr)

// listener 监听TCP连接
type listener struct {
	file     *os.File
	fd       int
	handleC  handleConnFunc
	listener net.Listener
	loop     *eventloop.EventLoop
}

// newListener 创建Listener
func newListener(network, addr string, reusePort bool, handlerConn handleConnFunc) (*listener, error) {
	var ls net.Listener
	var err error
	if reusePort {
		ls, err = reuseport.Listen(network, addr)
	} else {
		ls, err = net.Listen(network, addr)
	}
	if err != nil {
		return nil, err
	}

	l, ok := ls.(*net.TCPListener)
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

	loop, err := eventloop.New()
	if err != nil {
		return nil, err
	}

	listener := &listener{
		file:     file,
		fd:       fd,
		handleC:  handlerConn,
		listener: ls,
		loop:     loop,
	}
	if err = loop.AddSocketAndEnableRead(fd, listener); err != nil {
		return nil, err
	}

	return listener, nil
}

func (l *listener) Run() {
	l.loop.Run()
}

// HandleEvent 内部使用，供 event loop 回调处理事件
func (l *listener) HandleEvent(fd int, events poller.Event) {
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

		l.handleC(nfd, sa)
	}
}

func (l *listener) Close() error {
	return l.listener.Close()
}

func (l *listener) Stop() error {
	return l.loop.Stop()
}
