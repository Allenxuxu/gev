// +build linux

package listener

import (
	"errors"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

type HandleConnFunc func(fd int, sa *unix.Sockaddr)

type Listener struct {
	file    *os.File
	handleC HandleConnFunc
}

func New(network, addr string, handlerConn HandleConnFunc) (*Listener, error) {
	listener, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}

	netl, ok := listener.(*net.TCPListener)
	if !ok {
		return nil, errors.New("could not get file descriptor")
	}

	file, err := netl.File()
	if err != nil {
		return nil, err
	}
	if err = unix.SetNonblock(int(file.Fd()), true); err != nil {
		return nil, err
	}

	return &Listener{
		file:    file,
		handleC: handlerConn}, nil
}

func (l *Listener) HandleEvent(fd int, events uint32) {
	if events&(unix.EPOLLIN|unix.EPOLLPRI|unix.EPOLLRDHUP) != 0 {
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

func (l *Listener) Fd() int {
	return int(l.file.Fd())
}
