package gev

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/connector"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/toolkit/sync"
	"github.com/Allenxuxu/toolkit/sync/atomic"
)

type example struct {
	Count     atomic.Int64
	JustCount atomic.Int64
}

func (s *example) OnConnect(c *connection.Connection) {
	s.Count.Add(1)
	s.JustCount.Add(1)
	log.Info(" OnConnect ： ", c.PeerAddr())
}

func (s *example) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out []byte) {
	log.Info("OnMessage")

	//out = data
	msg := append([]byte{}, data...)
	if err := c.Send(msg); err != nil {
		panic(err)
	}
	return
}

func (s *example) OnClose(c *connection.Connection) {
	s.Count.Add(-1)
}

func TestServer_Start(t *testing.T) {
	handler := new(example)

	s, err := NewServer(handler,
		Network("tcp"),
		Address("127.0.0.1:1831"),
		NumLoops(8),
		ReusePort(true))
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(time.Second * 1)
		sw := sync.WaitGroupWrapper{}
		for i := 0; i < 1; i++ {
			sw.AddAndRun(func() {
				startClient(s.opts.Network, s.opts.Address)
			})
		}

		sw.Wait()
		s.Stop()
	}()

	s.Start()
}

func startClient(network, addr string) {
	rand.Seed(time.Now().UnixNano())
	c, err := net.Dial(network, addr)
	if err != nil {
		panic(err)
	}
	defer c.Close()
	rd := bufio.NewReader(c)
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 8
	start := time.Now()
	for time.Since(start) < duration {
		log.Info(time.Since(start), duration)
		sz := rand.Int()%(1024) + 1
		data := make([]byte, sz)
		log.Info("read start")
		if _, err := rand.Read(data); err != nil {
			panic(err)
		}
		log.Info("write start")
		if n, err := c.Write(data); err != nil {
			panic(err)
		} else {
			log.Info("writed", n)
		}
		data2 := make([]byte, len(data))
		log.Info("read full")
		if _, err := io.ReadFull(rd, data2); err != nil {
			panic(err)
		}
		if string(data) != string(data2) {
			panic("mismatch")
		}
	}
}

func TestServer_StopWithClient(t *testing.T) {
	handler := new(example)

	s, err := NewServer(handler,
		Network("tcp"),
		Address("127.0.0.1:1831"),
		NumLoops(8),
		ReusePort(true))
	if err != nil {
		t.Fatal(err)
	}

	go s.Start()

	cb := new(clientCallback)
	time.Sleep(time.Second)
	var success, failed atomic.Int64

	connector, err := connector.NewConnector(connector.NumLoops(8))
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Stop()
	go connector.Start()

	wg := &sync.WaitGroupWrapper{}
	for i := 0; i < 100; i++ {
		wg.AddAndRun(func() {
			conn, err := connector.Dial("tcp", "127.0.0.1:1831", cb, nil, 0)
			if err != nil {
				failed.Add(1)
				log.Info("error", err)
				return
			}
			success.Add(1)
			if err := conn.Close(); err != nil {
				panic(err)
			}
		})
	}

	wg.Wait()
	time.Sleep(time.Second * 1)
	log.Infof("Success: %d Failed: %d\n", success, failed)

	count := handler.Count.Get()
	if count != 0 {
		t.Fatal(count)
	}

	count = handler.JustCount.Get()
	if count != 100 {
		t.Fatal(count)
	}

	s.Stop()
}

func TestServer_StopAndSendWithClient(t *testing.T) {
	handler := new(example)

	s, err := NewServer(handler,
		Network("tcp"),
		Address("127.0.0.1:1831"),
		NumLoops(8),
		ReusePort(true))
	if err != nil {
		t.Fatal(err)
	}

	go s.Start()
	cb := new(clientCallback)
	time.Sleep(time.Second * 1)
	var success, failed atomic.Int64
	wg := &sync.WaitGroupWrapper{}

	connector, err := connector.NewConnector()
	if err != nil {
		t.Fatal(err)
	}

	defer connector.Stop()
	go connector.Start()

	log.Info("start handling")
	time.Sleep(time.Second * 2)
	for i := 0; i < 100; i++ {
		wg.AddAndRun(func() {
			conn, err := connector.DialWithTimeout(time.Second*5, "tcp", "127.0.0.1:1831", cb, nil, 0)
			if err != nil {
				failed.Add(1)
				log.Info("error", err)
				return
			}

			err = conn.Send([]byte("data_test"))
			if err != nil {
				panic(err)
			}
			// waiting for callback executed
			time.Sleep(time.Second)
			if err := conn.Close(); err != nil {
				panic(err)
			}
			success.Add(1)
		})
	}

	wg.Wait()
	log.Infof("Success: %d Failed: %d\n", success, failed)

	time.Sleep(time.Second * 1)
	count := handler.Count.Get()
	if count != 0 {
		t.Fatal(count)
	}
	if cb.reqCount.Get() != 100 {
		t.Fatal(cb.reqCount.Get())
	}

	s.Stop()
}

type clientCallback struct {
	reqCount atomic.Int64
}

func (cc *clientCallback) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out []byte) {
	//	log.Info("client OnMessage", string(data))
	cc.reqCount.Add(1)
	return
}

func (cc *clientCallback) OnClose(c *connection.Connection) {
	//	log.Info("client OnClose")
}

func ExampleServer_RunAfter() {
	handler := new(example)

	s, err := NewServer(handler,
		Network("tcp"),
		Address(":1833"),
		NumLoops(8),
		ReusePort(true))
	if err != nil {
		panic(err)
	}

	go s.Start()
	defer s.Stop()

	s.RunAfter(time.Second, func() {
		fmt.Println("RunAfter")
	})

	time.Sleep(2500 * time.Millisecond)

	// Output:
	// RunAfter
}

func ExampleServer_RunEvery() {
	handler := new(example)

	s, err := NewServer(handler,
		Network("tcp"),
		Address(":1833"),
		NumLoops(8),
		ReusePort(true))
	if err != nil {
		panic(err)
	}

	go s.Start()
	defer s.Stop()

	t := s.RunEvery(time.Second, func() {
		fmt.Println("EveryFunc")
	})

	time.Sleep(4500 * time.Millisecond)
	t.Stop()
	time.Sleep(4500 * time.Millisecond)

	// Output:
	// EveryFunc
	// EveryFunc
	// EveryFunc
	// EveryFunc
}

func TestServer_Stop(t *testing.T) {
	log.SetLevel(log.LevelDebug)
	handler := new(example)

	s, err := NewServer(handler,
		Network("tcp"),
		Address(":1832"),
		NumLoops(8),
		ReusePort(true))
	if err != nil {
		t.Fatal(err)
	}

	go s.Start()

	time.Sleep(time.Second)
	var success, failed atomic.Int64
	wg := &sync.WaitGroupWrapper{}
	for i := 0; i < 100; i++ {
		wg.AddAndRun(func() {
			conn, err := net.DialTimeout("tcp", "127.0.0.1:1832", time.Second*60)
			if err != nil {
				failed.Add(1)
				log.Error(err)
				return
			}
			success.Add(1)
			if err := conn.Close(); err != nil {
				panic(err)
			}
		})
	}

	wg.Wait()
	log.Infof("Success: %d Failed: %d\n", success, failed)

	time.Sleep(time.Second)
	count := handler.Count.Get()
	if count != 0 {
		t.Fatal(count)
	}

	s.Stop()
}

type example1 struct {
	Count atomic.Int64
}

func (s *example1) OnConnect(c *connection.Connection) {
	s.Count.Add(1)
	_ = c.Send([]byte("hello gev"))
	//log.Println(" OnConnect ： ", c.PeerAddr())
}

func (s *example1) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out []byte) {
	//log.Println("OnMessage")

	//out = data
	if err := c.Send(data); err != nil {
		panic(err)
	}
	return
}

func (s *example1) OnClose(c *connection.Connection) {
	s.Count.Add(-1)
	//log.Println("OnClose")
}

func TestServer_Stop1(t *testing.T) {
	log.SetLevel(log.LevelDebug)
	handler := new(example1)

	s, err := NewServer(handler,
		Network("tcp"),
		Address(":1833"),
		NumLoops(8),
		ReusePort(true))
	if err != nil {
		t.Fatal(err)
	}

	go s.Start()

	time.Sleep(time.Second)
	var success, failed atomic.Int64
	wg := &sync.WaitGroupWrapper{}
	for i := 0; i < 100; i++ {
		wg.AddAndRun(func() {
			conn, err := net.DialTimeout("tcp", "127.0.0.1:1833", time.Second*60)
			if err != nil {
				failed.Add(1)
				log.Error(err)
				return
			}
			success.Add(1)
			if err := conn.Close(); err != nil {
				panic(err)
			}
		})
	}

	wg.Wait()
	log.Infof("Success: %d Failed: %d\n", success, failed)

	time.Sleep(time.Second)
	count := handler.Count.Get()
	if count != 0 {
		t.Fatal(count)
	}

	s.Stop()
}
