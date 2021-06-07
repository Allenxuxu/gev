package connector

import (
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/context"

	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/gev/poller"
	"github.com/RussellLuo/timingwheel"
	"github.com/libp2p/go-reuseport"
	"golang.org/x/sys/unix"
)

var (
	ErrDialTimeout      = errors.New("i/o timeout")
	ErrConnectionHandle = errors.New("cannot handle connection")
	ErrInvalidArguments = errors.New("invalid arguments")
)

type connectionSocketState uint8

const (
	connectingConnectionSocketState connectionSocketState = iota + 1
	connectedConnectionSocketState
	disconnectedConnectionSocketState
)

type Connection struct {
	state   connectionSocketState
	stateMu sync.Mutex

	loop *eventloop.EventLoop
	*connection.Connection
	ctx      context.Context
	result   chan error
	fd       int
	protocol connection.Protocol
	tw       *timingwheel.TimingWheel
	idleTime time.Duration
	callBack connection.CallBack
}

func newConnection(
	ctx context.Context,
	network, address string,
	loop *eventloop.EventLoop,
	protocol connection.Protocol,
	tw *timingwheel.TimingWheel,
	idleTime time.Duration,
	callBack connection.CallBack) (*Connection, error) {

	connectResult := make(chan error)

	conn := &Connection{
		state:    connectingConnectionSocketState,
		loop:     loop,
		ctx:      ctx,
		result:   connectResult,
		protocol: protocol,
		tw:       tw,
		idleTime: idleTime,
		callBack: callBack,
	}

	fd, err := unixOpenConnect(network, address)
	conn.fd = fd
	switch err {
	case unix.EINPROGRESS, unix.EALREADY, unix.EINTR:
		conn.state = connectingConnectionSocketState
	case nil, syscall.EISCONN:
		runtime.KeepAlive(fd)
		conn.state = connectedConnectionSocketState
		if err := checkConn(fd); err != nil {
			conn.closeUnconnected()
			return nil, fmt.Errorf("checkConn err: %v", err)
		}

		sa, err := unix.Getpeername(fd)
		if err != nil {
			conn.closeUnconnected()
			return nil, fmt.Errorf("getPeerName err: %v", err)
		}

		conn.Connection = connection.New(fd, loop, sa, protocol, tw, idleTime, callBack)

	default:
		return nil, err
	}

	loop.QueueInLoop(func() {
		if err := loop.AddSocketAndEnableRead(fd, conn); err != nil {
			log.Info("[AddSocketAndEnableRead]", fd, err)
			connectResult <- err
			return
		}

		if err := loop.EnableReadWrite(fd); err != nil {
			log.Info("[EnableReadWrite] error ", fd, err)
			connectResult <- err
		}
	})

	if conn.state == connectedConnectionSocketState {
		return conn, nil
	}

	defer close(connectResult)

	select {
	case e := <-connectResult:
		if e != nil {
			return nil, e
		}

		return conn, nil
	case <-ctx.Done():
		conn.stateMu.Lock()
		defer conn.stateMu.Unlock()

		switch conn.state {
		case connectingConnectionSocketState:
			conn.state = disconnectedConnectionSocketState
			conn.closeUnconnected()
			return nil, ErrDialTimeout
		case connectedConnectionSocketState:
			return conn, nil
		default:
			return nil, ErrDialTimeout
		}
	}
}

func parseError(errorNo unix.Errno) error {
	switch errorNo {
	case unix.EINVAL:
		return ErrInvalidArguments
	default:
		return errors.New(unix.ErrnoName(errorNo))
	}
}

func (c *Connection) HandleEvent(fd int, events poller.Event) {
	if c.state == connectingConnectionSocketState {
		c.stateMu.Lock()
		defer c.stateMu.Unlock()

		if c.state != connectingConnectionSocketState {
			return
		}

		if events&poller.EventErr != 0 {
			c.state = disconnectedConnectionSocketState
			c.closeUnconnected()
			c.result <- ErrConnectionHandle
		} else if events&poller.EventWrite != 0 {
			if err := checkConn(fd); err != nil {
				c.closeUnconnected()
				c.result <- err
				return
			}

			sa, err := unix.Getpeername(fd)
			if err != nil {
				c.closeUnconnected()
				c.result <- parseError(err.(unix.Errno))
				return
			}

			c.Connection = connection.New(c.fd, c.loop, sa, c.protocol, c.tw, c.idleTime, c.callBack)
			c.state = connectedConnectionSocketState
			c.result <- nil
			c.Connection.HandleEvent(fd, events)
		} else {
			c.state = disconnectedConnectionSocketState
			c.closeUnconnected()

			c.result <- fmt.Errorf("wrong_event %v", events)
		}
	} else if c.state == connectedConnectionSocketState {
		c.Connection.HandleEvent(fd, events)
	}
}

func (c *Connection) closeUnconnected() {
	c.loop.DeleteFdInLoop(c.fd)
	_ = unix.Close(c.fd)
}

func (c *Connection) Close() error {
	if err := c.Connection.Close(); err != nil {
		return err
	}

	return nil
}

func checkConn(fd int) error {
	nerr, err := unix.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_ERROR)
	if err != nil {
		return err
	}

	unixError := unix.Errno(nerr)
	if unixError != unix.Errno(0) {
		return err
	}

	return nil
}

func unixOpenConnect(network, address string) (fd int, err error) {
	defer func() {
		if fd > 0 {
			switch err {
			case unix.EINPROGRESS, unix.EALREADY, unix.EINTR:
			default:
				_ = unix.Close(fd)
			}
		}
	}()

	addr, err := reuseport.ResolveAddr(network, address)
	if err != nil {
		return
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

	fd, err = unix.Socket(domain, typ, unix.PROT_NONE)
	if err != nil {
		return
	}

	if fd == 0 {
		err = errors.New("wrong fd value")
		return
	}

	if err = unix.SetNonblock(fd, true); err != nil {
		err = fmt.Errorf("SetNonblock error: %v", err)
		return
	}

	err = unix.Connect(fd, sa)
	return
}
