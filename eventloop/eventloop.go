package eventloop

import (
	"log"
	"sync"

	"github.com/Allenxuxu/gev/poller"
	"github.com/Allenxuxu/toolkit/sync/spinlock"
)

// Socket 接口
type Socket interface {
	HandleEvent(fd int, events poller.Event)
}

// EventLoop 事件循环
type EventLoop struct {
	poll    *poller.Poller
	sockets sync.Map
	packet  []byte

	pendingFunc []func()
	mu          spinlock.SpinLock
}

// New 创建一个 EventLoop
func New() (*EventLoop, error) {
	p, err := poller.Create()
	if err != nil {
		return nil, err
	}

	return &EventLoop{
		poll:   p,
		packet: make([]byte, 0xFFFF),
	}, nil
}

// PacketBuf 内部使用，临时缓冲区
func (l *EventLoop) PacketBuf() []byte {
	return l.packet
}

// DeleteFdInLoop 删除 fd
func (l *EventLoop) DeleteFdInLoop(fd int) {
	_ = l.poll.Del(fd)
	l.sockets.Delete(fd)
}

// AddSocketAndEnableRead 增加 Socket 到时间循环中，并注册可读事件
func (l *EventLoop) AddSocketAndEnableRead(fd int, s Socket) error {
	var err error
	l.sockets.Store(fd, s)

	if err = l.poll.AddRead(fd); err != nil {
		l.sockets.Delete(fd)
		return err
	}
	return nil
}

// EnableReadWrite 注册可读可写事件
func (l *EventLoop) EnableReadWrite(fd int) error {
	return l.poll.EnableReadWrite(fd)
}

// EnableRead 只注册可写事件
func (l *EventLoop) EnableRead(fd int) error {
	return l.poll.EnableRead(fd)
}

// RunLoop 启动事件循环
func (l *EventLoop) RunLoop() {
	l.poll.Poll(l.handlerEvent)
}

// Stop 关闭事件循环
func (l *EventLoop) Stop() error {
	return l.poll.Close()
}

// QueueInLoop 添加 func 到事件循环中执行
func (l *EventLoop) QueueInLoop(f func()) {
	l.mu.Lock()
	l.pendingFunc = append(l.pendingFunc, f)
	l.mu.Unlock()

	if err := l.poll.Wake(); err != nil {
		log.Println("QueueInLoop Wake loop, ", err)
	}
}

func (l *EventLoop) handlerEvent(fd int, events poller.Event) {
	if fd != -1 {
		s, ok := l.sockets.Load(fd)
		if ok {
			s.(Socket).HandleEvent(fd, events)
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
