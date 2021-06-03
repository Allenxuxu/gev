package connector

import (
	"errors"
	"fmt"
	"net"
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

var ErrDialTimeout = errors.New("i/o timeout")

type connectionSocketState uint8

const (
	connectingConnectionSocketState connectionSocketState = iota + 1
	connectedConnectionSocketState
	disconnectedConnectionSocketState
)

type Connection struct {
	state connectionSocketState
	loop  *eventloop.EventLoop
	*connection.Connection
	timeout time.Duration
	result  chan error
	fd      int

	protocol connection.Protocol
	tw       *timingwheel.TimingWheel
	idleTime time.Duration
	callBack connection.CallBack
}

func newConnection(
	network, address string,
	loop *eventloop.EventLoop,
	timeout time.Duration,
	protocol connection.Protocol,
	tw *timingwheel.TimingWheel,
	idleTime time.Duration,
	callBack connection.CallBack) (*Connection, error) {

	fd, err := unixOpenConnect(network, address)
	if err != nil {
		if fd > 0 {
			_ = unix.Close(fd)
		}
		return nil, err
	}

	connectResult := make(chan error)

	conn := &Connection{
		state:    connectingConnectionSocketState,
		loop:     loop,
		timeout:  timeout,
		result:   connectResult,
		fd:       fd,
		protocol: protocol,
		tw:       tw,
		idleTime: idleTime,
		callBack: callBack,
	}

	loop.QueueInLoop(func() {
		if err := loop.AddSocketAndEnableRead(fd, conn); err != nil {
			log.Info("[AddSocketAndEnableRead]", err)
		}

		if err := loop.EnableReadWrite(fd); err != nil {
			log.Info("[EnableReadWrite] error ", err)
		}
	})

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	if timeout > 0 {
		ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(timeout))
		defer cancel()
	} else {
		ctx = context.Background()
	}

	close(connectResult)
	select {
	case e := <-connectResult:
		if e != nil {
			conn.state = disconnectedConnectionSocketState
			return nil, e
		}

		return conn, nil
	case <-ctx.Done():
		conn.state = disconnectedConnectionSocketState
		return nil, ErrDialTimeout
	}
}

func (c *Connection) HandleEvent(fd int, events poller.Event) {
	if c.state == connectingConnectionSocketState {
		if events == poller.EventWrite {
			if err := checkConn(fd); err != nil {
				c.closeUnconnected()
				c.result <- err
				return
			}

			sa, err := unix.Getpeername(fd)
			if err != nil {
				c.closeUnconnected()
				c.result <- err
				return
			}

			c.Connection = connection.New(c.fd, c.loop, sa, c.protocol, c.tw, c.idleTime, c.callBack)
			c.state = connectedConnectionSocketState
			c.result <- nil
			c.Connection.HandleEvent(fd, events)
			return
		}

		c.closeUnconnected()
		c.result <- fmt.Errorf("wrong_event %v", events)
	} else if c.state == connectedConnectionSocketState {
		c.Connection.HandleEvent(fd, events)
	}
}

func (c *Connection) closeUnconnected() {
	c.loop.DeleteFdInLoop(c.fd)
	_ = unix.Close(c.fd)
}

func (c *Connection) Close() error {
	return c.Connection.Close()
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

func unixOpenConnect(network, address string) (int, error) {
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
		return fd, err
	}

	if err = unix.SetNonblock(fd, true); err != nil {
		log.Error("unix-setnonblock err:", err)
		return fd, err
	}

	err = unix.Connect(fd, sa)
	if err != nil && err != unix.EINPROGRESS {
		return fd, err
	} else if err != nil {
		err = nil
	}

	return fd, nil
}
