// +build !windows

package gev

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/Allenxuxu/toolkit/sync"

	"github.com/Allenxuxu/gev/log"
)

type example2 struct {
}

func (s *example2) OnConnect(c *Connection) {
	log.Info(" OnConnect ： ", c.PeerAddr())
	if err := c.Close(); err != nil {
		panic(err)
	}
}

func (s *example2) OnMessage(c *Connection, ctx interface{}, data []byte) (out interface{}) {
	log.Info("OnMessage")

	return
}

func (s *example2) OnClose(c *Connection) {
	log.Info("OnClose")
}

func TestConnClose(t *testing.T) {
	handler := new(example2)

	s, err := NewServer(handler,
		Network("tcp"),
		Address(":1843"),
		NumLoops(8),
		ReusePort(true))
	if err != nil {
		t.Fatal(err)
	}

	go s.Start()

	conn, err := net.DialTimeout("tcp", "127.0.0.1:1843", time.Second*60)
	if err != nil {
		log.Error(err)
		return
	}

	buf := make([]byte, 10)
	n, err := conn.Read(buf)
	if n != 0 || err != io.EOF {
		t.Fatal()
	}

	s.Stop()
}

type example3 struct {
}

func (s *example3) OnConnect(c *Connection) {
	// log.Info(" OnConnect ： ", c.PeerAddr())
}

func (s *example3) OnMessage(c *Connection, ctx interface{}, data []byte) (out interface{}) {
	// log.Info("OnMessage")

	return
}

func (s *example3) OnClose(c *Connection) {
	// log.Info("OnClose")
}
func TestIdleTime(t *testing.T) {
	handler := new(example3)

	s, err := NewServer(handler,
		Network("tcp"),
		Address(":1830"),
		NumLoops(8),
		ReusePort(true),
		IdleTime(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}

	go s.Start()

	start := time.Now()
	wg := &sync.WaitGroupWrapper{}
	log.Infof("Server start")

	for i := 0; i < 100; i++ {
		wg.AddAndRun(func() {
			conn, err := net.DialTimeout("tcp", "127.0.0.1:1830", time.Second*3)
			if err != nil {
				log.Error(err)
				return
			}

			log.Infof("Client conn success %v", conn.LocalAddr())

			buf := make([]byte, 10)
			n, err := conn.Read(buf)
			if n != 0 || err != io.EOF {
				t.Fatal()
			}
		})
	}
	wg.Wait()

	et := time.Since(start)
	if et.Seconds() > 4 || et.Seconds() < 3 {
		t.Fatal(et.Seconds())
	}

	s.Stop()
}
