package gev

import (
	"errors"
	"net"
	"runtime"
	"syscall"
	"time"

	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/toolkit/sync"
	"github.com/Allenxuxu/toolkit/sync/atomic"
	"github.com/RussellLuo/timingwheel"
	"github.com/libp2p/go-reuseport"
	"golang.org/x/sys/unix"
)

type Connector struct {
	workLoops   []*eventloop.EventLoop
	opts        *Options
	timingWheel *timingwheel.TimingWheel
	running     atomic.Bool
}

func connect(network, address string) (int, unix.Sockaddr, error) {
	addr, err := reuseport.ResolveAddr(network, address)
	if err != nil {
		return 0, nil, err
	}

	var sa unix.Sockaddr

	// net dial.go
	var domain, typ int
	switch ra := addr.(type) {
	case *net.TCPAddr:
		domain = unix.AF_INET
		typ = unix.SOCK_STREAM
		ipaddr := ra.IP.To4()
		if len(ipaddr) == net.IPv4len {
			addr := &unix.SockaddrInet4{Port: ra.Port}
			copy(addr.Addr[:], ipaddr)
			sa = addr
		} else if len(ipaddr) == net.IPv6len {
			addr := &unix.SockaddrInet6{Port: ra.Port}
			copy(addr.Addr[:], ipaddr)
			sa = addr
		}
	case *net.UnixAddr:
		domain = unix.AF_UNIX
		typ = unix.SOCK_STREAM
		sa = &unix.SockaddrUnix{Name: ra.Name}

	default:
		return 0, nil, errors.New("unsupported network/address type")
	}

	fd, err := unix.Socket(domain, typ, unix.PROT_NONE)
	if err != nil {
		log.Error("unix-socket err:", err)
		return 0, nil, err
	}

	if err = unix.SetNonblock(fd, true); err != nil {
		log.Error("unix-setnonblock err:", err)
		_ = unix.Close(fd)
		return 0, nil, err
	}

	err = unix.Connect(fd, sa)
	if err != nil && err != unix.EINPROGRESS {
		_ = unix.Close(fd)
		return 0, nil, err
	} else if err != nil {
		err = nil
	}

	l := time.After(time.Second * 5)
	check := func() (unix.Sockaddr, error) {
		var n int
		for {
			select {
			case <-l:
				err = errors.New("timeout")
				_ = unix.Close(fd)
				return nil, err

			default:
				wFdSet := &unix.FdSet{}
				wFdSet.Set(fd)

				n, err = unix.Select(fd+1, wFdSet, wFdSet, nil, &unix.Timeval{Sec: 1, Usec: 0})
				if err != nil {
					return nil, err
				}

				if n > 0 {
					nerr, err := unix.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_ERROR)
					if err != nil {
						return nil, err
					}
					switch err := unix.Errno(nerr); err {
					case unix.EINPROGRESS, unix.EALREADY, unix.EINTR:
					case unix.EISCONN:
						return nil, nil
					case unix.Errno(0):
						if sa, err := unix.Getpeername(fd); err == nil {
							return sa, nil
						}
						return nil, err
					}

					runtime.KeepAlive(fd)
				}
			}
		}
	}

	sa, err = check()
	if err != nil {
		_ = unix.Close(fd)
		return 0, nil, err
	}

	return fd, sa, nil
}

func (c *Connector) NewConn(callback connection.CallBack, network, address string, idleTime time.Duration) (*connection.Connection, error) {
	if callback == nil {
		return nil, errors.New("callback is nil")
	}

	fd, sa, err := connect(network, address)
	if err != nil {
		return nil, err
	}

	loop := c.opts.Strategy(c.workLoops)
	conn := connection.New(fd, loop, sa, c.opts.Protocol, c.timingWheel, idleTime, callback)
	loop.QueueInLoop(func() {
		if err := loop.AddSocketAndEnableRead(fd, conn); err != nil {
			log.Error("[AddSocketAndEnableRead]", err)
		}
	})

	return conn, nil
}

func NewConnector(opts ...Option) (connector *Connector, err error) {
	connector = new(Connector)
	connector.opts = newOptions(opts...)

	connector.timingWheel = timingwheel.NewTimingWheel(connector.opts.tick, connector.opts.wheelSize)
	if connector.opts.NumLoops <= 0 {
		connector.opts.NumLoops = runtime.NumCPU()
	}

	wloops := make([]*eventloop.EventLoop, connector.opts.NumLoops)
	for i := 0; i < connector.opts.NumLoops; i++ {
		l, err := eventloop.New()
		if err != nil {
			for j := 0; j < i; j++ {
				_ = wloops[j].Stop()
			}
			return nil, err
		}
		wloops[i] = l
	}

	connector.workLoops = wloops
	return
}

func (c *Connector) Start() {
	sw := sync.WaitGroupWrapper{}
	c.timingWheel.Start()

	length := len(c.workLoops)
	for i := 0; i < length; i++ {
		sw.AddAndRun(c.workLoops[i].RunLoop)
	}

	c.running.Set(true)
	sw.Wait()
}

func (c *Connector) Stop() {
	if c.running.Get() {
		c.running.Set(false)

		c.timingWheel.Stop()

		for k := range c.workLoops {
			if err := c.workLoops[k].Stop(); err != nil {
				log.Error(err)
			}
		}
	}
}

func (c *Connector) Options() Options {
	return *c.opts
}
