// +build darwin netbsd freebsd openbsd dragonfly

package poller

import (
	"github.com/Allenxuxu/toolkit/sync/atomic"
	"golang.org/x/sys/unix"
)

type Poller struct {
	fd       int
	running  atomic.Bool
	waitDone chan struct{}
}

func Create() (*Poller, error) {
	fd, err := unix.Kqueue()
	if err != nil {
		panic(err)
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

func (p *Poller) Wake() error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Fflags: unix.NOTE_TRIGGER,
	}}, nil, nil)
	return err
}

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

func (p *Poller) AddRead(fd int) error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_READ},
	}, nil, nil)
	return err
}

func (p *Poller) Del(fd int) error {
	return nil
}

func (p *Poller) EnableReadWrite(fd int) error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		//{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_READ}, // TODO 调用时所有 fd 已经 AddRead
		{Ident: uint64(fd), Flags: unix.EV_ADD, Filter: unix.EVFILT_WRITE},
	}, nil, nil)
	return err
}

func (p *Poller) EnableRead(fd int) error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{
		{Ident: uint64(fd), Flags: unix.EV_DELETE, Filter: unix.EVFILT_WRITE}, // TODO 调用时所有 fd 已经 EnableReadWrite
	}, nil, nil)
	return err
}

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
			panic("EpollWait: " + err.Error())
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
				if !p.running.Get() {
					return
				}
				wake = true
			}
		}

		if wake {
			handler(-1, 0)
			wake = false
		}
		if n == len(events) {
			events = make([]unix.Kevent_t, n*2)
		}
	}
}
