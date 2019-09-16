package listener

import (
	"errors"
	"net"
	"os"

	"github.com/Allenxuxu/gev/poller"
	"golang.org/x/sys/unix"
)

type HandleConnFunc func(fd int, sa *unix.Sockaddr)

type Listener struct {
	file    *os.File
	fd      int
	handleC HandleConnFunc
}

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

func (l *Listener) Fd() int {
	return l.fd
}
