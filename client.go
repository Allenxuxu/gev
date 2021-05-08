package gev

import (
	"errors"
	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/toolkit/sync"
	"github.com/Allenxuxu/toolkit/sync/atomic"
	"github.com/RussellLuo/timingwheel"
	"golang.org/x/sys/unix"
	"net"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type Client struct {
	*connection.Connection

	opts        *Options
	timingWheel *timingwheel.TimingWheel
	loop        *eventloop.EventLoop
	running     atomic.Bool
}

func IsSet(s *unix.FdSet, fd uintptr) bool {
	bits := 8 * unsafe.Sizeof(s.Bits[0])
	if fd >= bits*uintptr(len(s.Bits)) {
		panic("fdset: fd out of range")
	}
	n := fd / bits
	m := fd % bits
	return s.Bits[n]&(1<<m) != 0
}

func NewClientConnection(callback connection.CallBack, opts ...Option) (client *Client, err error) {
	if callback == nil {
		return nil, errors.New("callback is nil")
	}

	client = new(Client)
	options := newOptions(opts...)
	client.opts = options
	client.timingWheel = timingwheel.NewTimingWheel(options.tick, options.wheelSize)

	loop, err := eventloop.New()
	if err != nil {
		return nil, err
	}
	client.loop = loop

	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, unix.PROT_NONE)
	if err != nil {
		log.Error("unix-socket err:", err)
		return
	}

	ip_str := options.Address[:strings.Index(options.Address, ":")]
	ipaddr := net.ParseIP(ip_str).To4()
	var sa unix.Sockaddr
	if len(ipaddr) == net.IPv4len {
		addr := &unix.SockaddrInet4{Port: 1831}
		copy(addr.Addr[:], net.ParseIP("127.0.0.1").To4())
		sa = addr
	} else if len(ipaddr) == net.IPv6len {
		addr := &unix.SockaddrInet6{Port: 1831}
		copy(addr.Addr[:], net.ParseIP("127.0.0.1").To16())
		sa = addr
	} else if ipaddr == nil {
		addr := &unix.SockaddrUnix{Name: options.Address}
		sa = addr
	}

	if err = unix.SetNonblock(fd, true); err != nil {
		log.Error("unix-setnonblock err:", err)
		_ = unix.Close(fd)
		return
	}

	err = unix.Connect(fd, sa)
	if err != nil && err != unix.EINPROGRESS {
		_ = unix.Close(fd)
		return
	} else if err != nil {
		err = nil
	}

	l := time.After(time.Second * 5)
	check := func() error {
		var n int
		for {
			select {
			case <-l:
				err = errors.New("timeout")
				_ = unix.Close(fd)
				return err

			default:
				wFdSet := &unix.FdSet{}
				wFdSet.Set(fd)

				n, err = unix.Select(fd+1, wFdSet, wFdSet, nil, &unix.Timeval{Sec: 1, Usec: 0})
				if err != nil {
					return err
				}

				if n > 0 {
					if IsSet(wFdSet, uintptr(fd)) {
						nerr, err := unix.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_ERROR)
						if err != nil {
							return err
						}
						switch err := unix.Errno(nerr); err {
						case unix.EINPROGRESS, unix.EALREADY, unix.EINTR:
						case unix.EISCONN:
							return nil
						case unix.Errno(0):
							if _, err := unix.Getpeername(fd); err == nil {
								return nil
							}
							return err
						}

						runtime.KeepAlive(fd)
					}
				}

			}
		}
	}

	err = check()
	if err != nil {
		return
	}

	client.Connection = connection.New(fd, loop, nil, options.Protocol, client.timingWheel, options.IdleTime, callback)
	loop.QueueInLoop(func() {
		if err := loop.AddSocketAndEnableRead(fd, client.Connection); err != nil {
			log.Info("[AddSocketAndEnableRead]", err)
		}
	})
	return
}

func (c *Client) Close() error {
	return c.Connection.Close()
}

func (c *Client) Start() {
	sw := sync.WaitGroupWrapper{}
	c.timingWheel.Start()

	var running atomic.Bool

	sw.AddAndRun(c.loop.RunLoop)
	running.Set(true)
	sw.Wait()
}

func (c *Client) Stop() {
	if c.running.Get() {
		c.running.Set(false)

		c.timingWheel.Stop()
		if err := c.loop.Stop(); err != nil {
			log.Error(err)
		}
	}

}

func (c *Client) Options() Options {
	return *c.opts
}
