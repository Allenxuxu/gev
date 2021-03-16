package eventloop

import (
	"unsafe"

	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/gev/poller"
	"github.com/Allenxuxu/toolkit/sync/atomic"
	"github.com/Allenxuxu/toolkit/sync/spinlock"
)

// Socket 接口
type Socket interface {
	HandleEvent(fd int, events poller.Event)
	Close() error
}

// EventLoop 事件循环
type EventLoop struct {
	eventLoopLocal
	// nolint
	// Prevents false sharing on widespread platforms with
	// 128 mod (cache line size) = 0 .
	pad [128 - unsafe.Sizeof(eventLoopLocal{})%128]byte
}

// nolint
type eventLoopLocal struct {
	needWake    atomic.Bool
	poll        *poller.Poller
	mu          spinlock.SpinLock
	sockets     map[int]Socket
	packet      []byte
	pendingFunc []func()
	UserBuffer  *[]byte
}

// New 创建一个 EventLoop
func New() (*EventLoop, error) {
	p, err := poller.Create()
	if err != nil {
		return nil, err
	}

	userBuffer := make([]byte, 1024)
	return &EventLoop{
		eventLoopLocal: eventLoopLocal{
			poll:       p,
			packet:     make([]byte, 0xFFFF),
			sockets:    make(map[int]Socket),
			UserBuffer: &userBuffer,
		},
	}, nil
}

// PacketBuf 内部使用，临时缓冲区
func (l *EventLoop) PacketBuf() []byte {
	return l.packet
}

// DeleteFdInLoop 删除 fd
func (l *EventLoop) DeleteFdInLoop(fd int) {
	if err := l.poll.Del(fd); err != nil {
		log.Error("[DeleteFdInLoop]", err)
	}
	delete(l.sockets, fd)
}

// AddSocketAndEnableRead 增加 Socket 到时间循环中，并注册可读事件
func (l *EventLoop) AddSocketAndEnableRead(fd int, s Socket) error {
	var err error
	l.sockets[fd] = s

	if err = l.poll.AddRead(fd); err != nil {
		delete(l.sockets, fd)
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
	l.needWake.Set(true)
	l.poll.Poll(l.handlerEvent)
}

// Stop 关闭事件循环
func (l *EventLoop) Stop() error {
	l.QueueInLoop(func() {
		for _, v := range l.sockets {
			if err := v.Close(); err != nil {
				log.Error(err)
			}
		}
		l.sockets = nil
	})

	return l.poll.Close()
}

// QueueInLoop 添加 func 到事件循环中执行
func (l *EventLoop) QueueInLoop(f func()) {
	l.mu.Lock()
	l.pendingFunc = append(l.pendingFunc, f)
	l.mu.Unlock()

	// ToDo csp
	l.needWake.Set(false)
	if err := l.poll.Wake(); err != nil {
		log.Error("QueueInLoop Wake loop, ", err)
	}

}

func (l *EventLoop) handlerEvent(fd int, events poller.Event) {
	if fd != -1 {
		s, ok := l.sockets[fd]
		if ok {
			s.HandleEvent(fd, events)
		}
	} else {
		l.needWake.Set(true)
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
