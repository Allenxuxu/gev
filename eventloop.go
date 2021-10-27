package gev

import (
	"unsafe"

	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/gev/poller"
	"github.com/Allenxuxu/toolkit/sync/atomic"
	"github.com/Allenxuxu/toolkit/sync/spinlock"
)

var (
	DefaultPacketSize    = 65536
	DefaultBufferSize    = 4096
	DefaultTaskQueueSize = 1024
)

type handler interface {
	handleEvent(fd int, events poller.Event)
}

// Socket 接口
type Socket interface {
	handler

	Close() error
}

// eventLoop 事件循环
type eventLoop struct {
	eventLoopLocal
	// nolint
	// Prevents false sharing on widespread platforms with
	// 128 mod (cache line size) = 0 .
	pad [128 - unsafe.Sizeof(eventLoopLocal{})%128]byte
}

// nolint
type eventLoopLocal struct {
	ConnCunt   atomic.Int64
	needWake   *atomic.Bool
	poll       *poller.Poller
	mu         spinlock.SpinLock
	sockets    map[int]Socket
	packet     []byte
	taskQueueW []func()
	taskQueueR []func()

	UserBuffer *[]byte
}

// newEventLoop 创建一个 eventLoop
func newEventLoop() (*eventLoop, error) {
	p, err := poller.Create()
	if err != nil {
		return nil, err
	}

	userBuffer := make([]byte, DefaultBufferSize)
	return &eventLoop{
		eventLoopLocal: eventLoopLocal{
			poll:       p,
			packet:     make([]byte, DefaultPacketSize),
			sockets:    make(map[int]Socket),
			UserBuffer: &userBuffer,
			needWake:   atomic.New(true),
			taskQueueW: make([]func(), 0, DefaultTaskQueueSize),
			taskQueueR: make([]func(), 0, DefaultTaskQueueSize),
		},
	}, nil
}

// PacketBuf 内部使用，临时缓冲区
func (l *eventLoop) PacketBuf() []byte {
	return l.packet
}

func (l *eventLoop) ConnectionCount() int64 {
	return l.ConnCunt.Get()
}

// deleteFdInLoop 删除 fd
func (l *eventLoop) deleteFdInLoop(fd int) {
	if err := l.poll.Del(fd); err != nil {
		log.Error("[deleteFdInLoop]", err)
	}
	delete(l.sockets, fd)
	l.ConnCunt.Add(-1)
}

// addSocketAndEnableRead 增加 Socket 到时间循环中，并注册可读事件
func (l *eventLoop) addSocketAndEnableRead(fd int, s Socket) error {
	l.sockets[fd] = s
	if err := l.poll.AddRead(fd); err != nil {
		delete(l.sockets, fd)
		return err
	}

	l.ConnCunt.Add(1)
	return nil
}

// enableReadWrite 注册可读可写事件
func (l *eventLoop) enableReadWrite(fd int) error {
	return l.poll.EnableReadWrite(fd)
}

// enableRead 只注册可写事件
func (l *eventLoop) enableRead(fd int) error {
	return l.poll.EnableRead(fd)
}

// runLoop 启动事件循环
func (l *eventLoop) runLoop() {
	l.poll.Poll(l.handlerEvent)
}

// stop 关闭事件循环
func (l *eventLoop) stop() error {
	l.queueInLoop(func() {
		for _, v := range l.sockets {
			if err := v.Close(); err != nil {
				log.Error(err)
			}
		}
		l.sockets = nil
	})

	_ = l.ConnCunt.Swap(0)
	return l.poll.Close()
}

// queueInLoop 添加 func 到事件循环中执行
func (l *eventLoop) queueInLoop(f func()) {
	l.mu.Lock()
	l.taskQueueW = append(l.taskQueueW, f)
	l.mu.Unlock()

	if l.needWake.CompareAndSwap(true, false) {
		if err := l.poll.Wake(); err != nil {
			log.Error("queueInLoop Wake loop, ", err)
		}
	}
}

func (l *eventLoop) handlerEvent(fd int, events poller.Event) {
	if fd != -1 {
		s, ok := l.sockets[fd]
		if ok {
			s.handleEvent(fd, events)
		}
	} else {
		l.needWake.Set(true)
		l.doPendingFunc()
	}
}

func (l *eventLoop) doPendingFunc() {
	l.mu.Lock()
	l.taskQueueW, l.taskQueueR = l.taskQueueR, l.taskQueueW
	l.mu.Unlock()

	length := len(l.taskQueueR)
	for i := 0; i < length; i++ {
		l.taskQueueR[i]()
	}

	l.taskQueueR = l.taskQueueR[:0]
}
