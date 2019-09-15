// +build linux

package eventloop

import (
	"log"

	"github.com/Allenxuxu/gev/poller"
	"github.com/Allenxuxu/toolkit/sync/spinlock"
)

// Socket ...
type Socket interface {
	HandleEvent(fd int, events uint32)
}

// EventLoop 事件循环
type EventLoop struct {
	poll    *poller.Poller
	sockets map[int]Socket
	packet  []byte

	pendingFunc []func()
	mu          spinlock.SpinLock
	mapMu       spinlock.SpinLock
}

func New() (*EventLoop, error) {
	p, err := poller.Create()
	if err != nil {
		return nil, err
	}

	return &EventLoop{
		poll:    p,
		sockets: make(map[int]Socket),
		packet:  make([]byte, 0xFFFF),
	}, nil
}

func (l *EventLoop) PacketBuf() []byte {
	return l.packet
}

func (l *EventLoop) DeleteFdInLoop(fd int) {
	_ = l.poll.Del(fd)
	l.mapMu.Lock()
	delete(l.sockets, fd)
	l.mapMu.Unlock()
}

func (l *EventLoop) AddSocketAndEnableRead(fd int, s Socket) error {
	var err error
	l.mapMu.Lock()
	l.sockets[fd] = s
	l.mapMu.Unlock()

	if err = l.poll.AddRead(fd); err != nil {
		delete(l.sockets, fd)
		return err
	}
	return nil
}

func (l *EventLoop) EnableReadWrite(fd int) error {
	return l.poll.EnableReadWrite(fd)
}

func (l *EventLoop) EnableRead(fd int) error {
	return l.poll.EnableRead(fd)
}

func (l *EventLoop) RunLoop() {
	l.poll.Poll(l.handlerEvent)
}

func (l *EventLoop) Stop() error {
	return l.poll.Close()
}

func (l *EventLoop) QueueInLoop(f func()) {
	l.mu.Lock()
	l.pendingFunc = append(l.pendingFunc, f)
	l.mu.Unlock()

	if err := l.poll.Wake(); err != nil {
		log.Println("QueueInLoop Wake loop, ", err)
	}
}

func (l *EventLoop) handlerEvent(fd int, events uint32) {
	if fd != -1 {
		l.mapMu.Lock()
		s, ok := l.sockets[fd]
		l.mapMu.Unlock()
		if ok {
			s.HandleEvent(fd, events)
		} else {
			//TODO
			panic("conn not find")
		}
	} else {
		l.doPendingFunc()
	}
}

func (l *EventLoop) doPendingFunc() {
	l.mu.Lock()
	pf := l.pendingFunc
	l.pendingFunc = nil
	l.mu.Unlock()

	length := len(pf)
	for i := 0; i < length; i++ {
		pf[i]()
	}
}
