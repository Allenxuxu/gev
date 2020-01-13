// +build darwin netbsd freebsd openbsd dragonfly

package poller

import (
	"errors"
	"sync"

	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/toolkit/sync/atomic"
	"golang.org/x/sys/unix"
)

// Poller Kqueue封装
type Poller struct {
	fd       int
	running  atomic.Bool
	waitDone chan struct{}
	sockets  sync.Map // [fd]events
}

// Create 创建Poller
func Create() (*Poller, error) {
	fd, err := unix.Kqueue()
	if err != nil {
		return nil, err
	}
	_, err = unix.Kevent(fd, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Flags:  unix.EV_ADD | unix.EV_CLEAR,
	}}, nil, nil)
	if err != nil {
		return nil, err
	}

	return &Poller{
		fd:       fd,
		waitDone: make(chan struct{}),
	}, nil
}

// Wake 唤醒 kqueue
func (p *Poller) Wake() error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Fflags: unix.NOTE_TRIGGER,
	}}, nil, nil)
	return err
}

// Close 关闭 kqueue
func (p *Poller) Close() (err error) {
	if !p.running.Get() {
		return ErrClosed
	}

	p.running.Set(false)
	if err = p.Wake(); err != nil {
		return
	}

	<-p.waitDone
	_ = unix.Close(p.fd)
	return
}

// AddRead 注册fd到kqueue并注册可读事件
func (p *Poller) AddRead(fd int) error {
	p.sockets.Store(fd, EventRead)

	kEvents := p.kEvents(EventNone, EventRead, fd)
	_, err := unix.Kevent(p.fd, kEvents, nil, nil)
	return err
}

// Del 从kqueue删除fd
func (p *Poller) Del(fd int) error {
	v, ok := p.sockets.Load(fd)
	if !ok {
		return errors.New("sync map load error")
	}

	kEvents := p.kEvents(v.(Event), EventNone, fd)
	_, err := unix.Kevent(p.fd, kEvents, nil, nil)
	if err != nil {
		p.sockets.Delete(fd)
	}
	return err
}

// EnableReadWrite 修改fd注册事件为可读可写事件
func (p *Poller) EnableReadWrite(fd int) error {
	oldEvents, ok := p.sockets.Load(fd)
	if !ok {
		return errors.New("sync map load error")
	}

	newEvents := EventWrite | EventRead
	kEvents := p.kEvents(oldEvents.(Event), newEvents, fd)
	_, err := unix.Kevent(p.fd, kEvents, nil, nil)
	if err != nil {
		p.sockets.Store(fd, newEvents)
	}
	return err
}

// EnableRead 修改fd注册事件为可读事件
func (p *Poller) EnableRead(fd int) error {
	oldEvents, ok := p.sockets.Load(fd)
	if !ok {
		return errors.New("sync map load error")
	}

	newEvents := EventRead
	kEvents := p.kEvents(oldEvents.(Event), newEvents, fd)
	_, err := unix.Kevent(p.fd, kEvents, nil, nil)
	if err != nil {
		p.sockets.Store(fd, newEvents)
	}
	return err
}

func (p *Poller) kEvents(old Event, new Event, fd int) (ret []unix.Kevent_t) {
	if new&EventRead != 0 {
		if old&EventRead == 0 {
			ret = append(ret, unix.Kevent_t{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_READ})
		}
	} else {
		if old&EventRead != 0 {
			ret = append(ret, unix.Kevent_t{Ident: uint64(fd), Flags: unix.EV_DELETE, Filter: unix.EVFILT_READ})
		}
	}

	if new&EventWrite != 0 {
		if old&EventWrite == 0 {
			ret = append(ret, unix.Kevent_t{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE})
		}
	} else {
		if old&EventWrite != 0 {
			ret = append(ret, unix.Kevent_t{Ident: uint64(fd), Flags: unix.EV_DELETE, Filter: unix.EVFILT_WRITE})
		}
	}
	return
}

// Poll 启动 kqueue 循环
func (p *Poller) Poll(handler func(fd int, event Event)) {
	defer func() {
		close(p.waitDone)
	}()

	events := make([]unix.Kevent_t, waitEventsBegin)
	var wake bool
	p.running.Set(true)
	for {
		n, err := unix.Kevent(p.fd, nil, events, nil)
		if err != nil && err != unix.EINTR {
			log.Error("EpollWait: ", err)
			continue
		}

		for i := 0; i < n; i++ {
			fd := int(events[i].Ident)
			if fd != 0 {
				var rEvents Event
				if (events[i].Flags&unix.EV_ERROR != 0) || (events[i].Flags&unix.EV_EOF != 0) {
					rEvents |= EventErr
				}
				if events[i].Filter == unix.EVFILT_WRITE {
					rEvents |= EventWrite
				}
				if events[i].Filter == unix.EVFILT_READ {
					rEvents |= EventRead
				}

				handler(fd, rEvents)
			} else {
				wake = true
			}
		}

		if wake {
			handler(-1, 0)
			wake = false
			if !p.running.Get() {
				return
			}
		}
		if n == len(events) {
			events = make([]unix.Kevent_t, n*2)
		}
	}
}
