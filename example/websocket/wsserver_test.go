package main

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/plugins/websocket/ws"
	"github.com/Allenxuxu/gev/plugins/websocket/ws/util"
	"github.com/Allenxuxu/toolkit/sync"
	"golang.org/x/net/websocket"
)

type wsExample struct{}

func (s *wsExample) OnConnect(c *connection.Connection) {
	//log.Println(" OnConnect ： ", c.PeerAddr())
}
func (s *wsExample) OnMessage(c *connection.Connection, data []byte) (messageType ws.MessageType, out []byte) {
	messageType = ws.MessageText

	switch rand.Int() % 3 {
	case 0:
		out = data
	case 1:
		msg, err := util.PackData(ws.MessageText, data)
		if err != nil {
			panic(err)
		}
		if err := c.Send(msg); err != nil {
			msg, err := util.PackCloseData(err.Error())
			if err != nil {
				panic(err)
			}
			if e := c.Send(msg); e != nil {
				panic(e)
			}
		}
	case 2:
		msg, err := util.PackCloseData("close")
		if err != nil {
			panic(err)
		}
		if e := c.Send(msg); e != nil {
			panic(e)
		}
	}
	return
}

func (s *wsExample) OnClose(c *connection.Connection) {
	//log.Println("OnClose")
}

func TestWebSocketServer_Start(t *testing.T) {
	handler := new(wsExample)

	s, err := NewWebSocketServer(handler, &ws.Upgrader{},
		gev.Address(":1834"),
		gev.NumLoops(8),
		gev.ReusePort(true))
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(time.Second)
		sw := sync.WaitGroupWrapper{}
		for i := 0; i < 100; i++ {
			sw.AddAndRun(func() {
				startWebSocketClient(s.Options().Address)
			})
		}

		sw.Wait()
		s.Stop()
	}()

	s.Start()
}

func startWebSocketClient(addr string) {
	rand.Seed(time.Now().UnixNano())
	addr = "ws://localhost" + addr
	c, err := websocket.Dial(addr, "", addr)
	if err != nil {
		panic(err)
	}
	defer c.Close()
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 8
	start := time.Now()
	for time.Since(start) < duration {
		sz := rand.Int()%(1024*3) + 1
		data := make([]byte, sz)
		if _, err := rand.Read(data); err != nil {
			panic(err)
		}
		if n, err := c.Write(data); err != nil || n != len(data) {
			panic(err)
		}

		data2 := make([]byte, len(data))
		if n, err := c.Read(data2); err != nil || n != len(data) {
			if err != io.EOF {
				panic(err)
			} else {
				return
			}
		}
		if !bytes.Equal(data, data2) {
			panic("mismatch")
		}
	}
}
