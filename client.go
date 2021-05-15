package gev

import (
	"errors"
	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/toolkit/sync"
	"github.com/Allenxuxu/toolkit/sync/atomic"
	"github.com/RussellLuo/timingwheel"
	"github.com/libp2p/go-reuseport"
	"golang.org/x/sys/unix"
	"net"
	"runtime"
	"syscall"
	"time"
)

type Connector struct {
	workLoops   []*eventloop.EventLoop
	opts        *Options
	timingWheel *timingwheel.TimingWheel
	running     atomic.Bool
}

func connect(network, address string) (int, error) {
	addr, err := reuseport.ResolveAddr(network, address)
	if err != nil {
		return 0, err
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
		return 0, errors.New("unsupported network/address type")
	}

	fd, err := unix.Socket(domain, typ, unix.PROT_NONE)
	if err != nil {
		log.Error("unix-socket err:", err)
		return 0, err
	}

	if err = unix.SetNonblock(fd, true); err != nil {
		log.Error("unix-setnonblock err:", err)
		_ = unix.Close(fd)
		return 0, err
	}

	err = unix.Connect(fd, sa)
	if err != nil && err != unix.EINPROGRESS {
		_ = unix.Close(fd)
		return 0, err
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

	err = check()
	if err != nil {
		_ = unix.Close(fd)
		return 0, err
	}

	return fd, nil
}

func (c *Connector) NewConn(callback connection.CallBack, network, address string, idleTime time.Duration) (*connection.Connection, error) {
	if callback == nil {
		return nil, errors.New("callback is nil")
	}

	fd, err := connect(network, address)
	if err != nil {
		return nil, err
	}

	loop := c.opts.Strategy(c.workLoops)
	conn := connection.New(fd, loop, nil, c.opts.Protocol, c.timingWheel, idleTime, callback)
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
