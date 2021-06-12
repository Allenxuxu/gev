package gev

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/eventloop"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/toolkit/sync"
	"github.com/stretchr/testify/assert"
)

type example2 struct {
}

func (s *example2) OnConnect(c *connection.Connection) {
	log.Info(" OnConnect ： ", c.PeerAddr())
	if err := c.Close(); err != nil {
		panic(err)
	}
}

func (s *example2) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out []byte) {
	log.Info("OnMessage")

	return
}

func (s *example2) OnClose(c *connection.Connection) {
	log.Info("OnClose")
}

func TestConnClose(t *testing.T) {
	log.SetLevel(log.LevelDebug)
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

func (s *example3) OnConnect(c *connection.Connection) {
	//log.Info(" OnConnect ： ", c.PeerAddr())
}

func (s *example3) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out []byte) {
	//log.Info("OnMessage")

	return
}

func (s *example3) OnClose(c *connection.Connection) {
	//log.Info("OnClose")
}
func TestIdleTime(t *testing.T) {
	log.SetLevel(log.LevelDebug)
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
	for i := 0; i < 100; i++ {
		wg.AddAndRun(func() {
			conn, err := net.DialTimeout("tcp", "127.0.0.1:1830", time.Second*60)
			if err != nil {
				log.Error(err)
				return
			}

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

func TestConnLoadBalanceLeastConnection(t *testing.T) {
	handler := new(example3)

	s, err := NewServer(handler,
		Network("tcp"),
		Address(":1840"),
		NumLoops(4),
		ReusePort(true),
		LoadBalance(eventloop.LeastConnection()))
	if err != nil {
		t.Fatal(err)
	}

	go s.Start()

	for i := 0; i < 200; i++ {
		_, err := net.DialTimeout("tcp", "127.0.0.1:1840", time.Second*60)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Millisecond * 20)
	}

	time.Sleep(time.Millisecond * 20)

	for i := 0; i < len(s.workLoops); i++ {
		assert.Equal(t, int64(50), s.workLoops[i].ConnectionCount())
	}

	s.Stop()
}

func TestConnLoadBalanceRoundRobin(t *testing.T) {
	handler := new(example3)

	s, err := NewServer(handler,
		Network("tcp"),
		Address(":1841"),
		NumLoops(4),
		ReusePort(true),
		LoadBalance(eventloop.RoundRobin()))
	if err != nil {
		t.Fatal(err)
	}

	go s.Start()

	for i := 0; i < 9; i++ {
		_, err := net.DialTimeout("tcp", "127.0.0.1:1841", time.Second*60)
		if err != nil {
			t.Fatal(err)
		}
	}
	time.Sleep(time.Millisecond * 20)

	for i := 0; i < len(s.workLoops); i++ {
		expected := 2
		if i == 0 {
			expected = 3
		}
		assert.Equal(t, expected, int(s.workLoops[i].ConnectionCount()))
	}

	s.Stop()
}
