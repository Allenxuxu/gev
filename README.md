# gev

[![Github Actions](https://github.com/Allenxuxu/gev/workflows/CI/badge.svg)](https://github.com/Allenxuxu/gev/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/Allenxuxu/gev)](https://goreportcard.com/report/github.com/Allenxuxu/gev)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/a2a55fe9c0c443e198f588a6c8026cd0)](https://www.codacy.com/manual/Allenxuxu/gev?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=Allenxuxu/gev&amp;utm_campaign=Badge_Grade)
[![GoDoc](https://godoc.org/github.com/Allenxuxu/gev?status.svg)](https://godoc.org/github.com/Allenxuxu/gev)
[![LICENSE](https://img.shields.io/badge/LICENSE-MIT-blue)](https://github.com/Allenxuxu/gev/blob/master/LICENSE)
[![Code Size](https://img.shields.io/github/languages/code-size/Allenxuxu/gev.svg?style=flat)](https://img.shields.io/github/languages/code-size/Allenxuxu/gev.svg?style=flat)

#### [中文](README-ZH.md) | English

`gev` is a lightweight, fast non-blocking TCP network library based on Reactor mode. 

Support custom protocols to quickly and easily build high-performance servers.

## Features

- High-performance event loop based on epoll and kqueue
- Support multi-core and multi-threading
- Dynamic expansion of read and write buffers implemented by Ring Buffer
- Asynchronous read and write
- SO_REUSEPORT port reuse support
- Support WebSocket/Protobuf
- Support for scheduled tasks, delayed tasks
- Support for custom protocols

## Network model

`gev` uses only a few goroutines, one of them listens for connections and the others (work coroutines) handle read and write events of connected clients. The count of work coroutines is configurable, which is the core number of host CPUs by default.

![image](benchmarks/out/reactor.png)

## Performance Test

> Test environment: Ubuntu18.04 | 4 Virtual CPUs | 4.0 GiB

### Throughput Test

limit GOMAXPROCS=1（Single thread），1 work goroutine

![image](benchmarks/out/gev11.png)

limit GOMAXPROCS=4，4 work goroutine

![image](benchmarks/out/gev44.png)

### Other Test

<details>
  <summary> Speed ​​Test </summary>

Compared with the simple performance of similar libraries, the pressure measurement method is the same as the evio project.

- gnet
- eviop
- evio
- net (StdLib)

limit GOMAXPROCS=1，1 work goroutine

![image](benchmarks/out/echo-1c-1loops.png)

limit GOMAXPROCS=1，4 work goroutine

![image](benchmarks/out/echo-1c-4loops.png)

limit GOMAXPROCS=4，4 work goroutine

![image](benchmarks/out/echo-4c-4loops.png)

</details>

## Install

```bash
go get -u github.com/Allenxuxu/gev
```

## Getting start

### echo demo

```go
package main

import (
	"flag"
	"strconv"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
)

type example struct{}

func (s *example) OnConnect(c *connection.Connection) {
	//log.Println(" OnConnect ： ", c.PeerAddr())
}
func (s *example) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out []byte) {
	//log.Println("OnMessage")
	out = data
	return
}

func (s *example) OnClose(c *connection.Connection) {
	//log.Println("OnClose")
}

func main() {
	handler := new(example)
	var port int
	var loops int

	flag.IntVar(&port, "port", 1833, "server port")
	flag.IntVar(&loops, "loops", -1, "num loops")
	flag.Parse()

	s, err := gev.NewServer(handler,
		gev.Address(":"+strconv.Itoa(port)),
		gev.NumLoops(loops))
	if err != nil {
		panic(err)
	}

	s.Start()
}
```

*Handler* is an interface that programs must implement.

```go
type Handler interface {
	OnConnect(c *connection.Connection)
	OnMessage(c *connection.Connection, ctx interface{}, data []byte) []byte
	OnClose(c *connection.Connection)
}

func NewServer(handler Handler, opts ...Option) (server *Server, err error)
```

OnMessage will be called back when a complete data frame arrives.Users can get the data, process the business logic, and return the data that needs to be sent.

When there is data coming, gev does not call back OnMessage immediately, but instead calls back an UnPacket function.Probably the execution logic is as follows:

```go
ctx, receivedData := c.protocol.UnPacket(c, buffer)
	if ctx != nil || len(receivedData) != 0 {
		sendData := c.OnMessage(c, ctx, receivedData)
		if len(sendData) > 0 {
			return c.protocol.Packet(c, sendData)
		}
	}
```

![protocol](benchmarks/out/protocol.png)

The UnPacket function will check whether the data in the ringbuffer is a complete data frame. If it is, the data will be unpacked and return the payload data. If it is not a complete data frame, it will return directly.

The return value of UnPacket `(interface{}, []byte)` will be passed in as a call to OnMessage `ctx interface{}, data []byte` and callback.Ctx is designed to pass special information generated when parsing data frames in the UnPacket function (which is required for complex data frame protocols), and data is used to pass payload data.

```go
type Protocol interface {
	UnPacket(c *Connection, buffer *ringbuffer.RingBuffer) (interface{}, []byte)
	Packet(c *Connection, data []byte) []byte
}

type DefaultProtocol struct{}

func (d *DefaultProtocol) UnPacket(c *Connection, buffer *ringbuffer.RingBuffer) (interface{}, []byte) {
	ret := buffer.Bytes()
	buffer.RetrieveAll()
	return nil, ret
}

func (d *DefaultProtocol) Packet(c *Connection, data []byte) []byte {
	return data
}
```

As above, gev provides a default Protocol implementation that will fetch all data in the receive buffer ( ringbuffer ).In actual use, there is usually a data frame protocol of its own, and gev can be set in the form of a plug-in: it is set by variable parameters when creating Server.

```go
s, err := gev.NewServer(handler,gev.Protocol(&ExampleProtocol{}))
```

Check out the example [Protocol](example/protocol) for a detailed.

There is also a *Send* method that can be used for sending data. But *Send* puts the data to Event-Loop and invokes it to send the data rather than sending data by itself immediately.

Check out the example [Server timing push](example/pushmessage/main.go) for a detailed.

```go
func (c *Connection) Send(buffer []byte) error
```

*ShutdownWrite* works for reverting connected status to false and closing connection.

Check out the example [Maximum connections](example/maxconnection/main.go) for a detailed.

```go
func (c *Connection) ShutdownWrite() error
```

[RingBuffer](https://github.com/Allenxuxu/ringbuffer) is a dynamical expansion implementation of circular buffer.

### WebSocket

The WebSocket protocol is built on top of the TCP protocol, so gev doesn't need to be built in, but instead provides support in the form of plugins, in the plugins/websocket directory.

```go
type Protocol struct {
	upgrade *ws.Upgrader
}

func New(u *ws.Upgrader) *Protocol {
	return &Protocol{upgrade: u}
}

func (p *Protocol) UnPacket(c *connection.Connection, buffer *ringbuffer.RingBuffer) (ctx interface{}, out []byte) {
	upgraded := c.Context()
	if upgraded == nil {
		var err error
		out, _, err = p.upgrade.Upgrade(buffer)
		if err != nil {
			log.Println("Websocket Upgrade :", err)
			return
		}
		c.SetContext(true)
	} else {
		header, err := ws.VirtualReadHeader(buffer)
		if err != nil {
			log.Println(err)
			return
		}
		if buffer.VirtualLength() >= int(header.Length) {
			buffer.VirtualFlush()

			payload := make([]byte, int(header.Length))
			_, _ = buffer.Read(payload)

			if header.Masked {
				ws.Cipher(payload, header.Mask, 0)
			}

			ctx = &header
			out = payload
		} else {
			buffer.VirtualRevert()
		}
	}
	return
}

func (p *Protocol) Packet(c *connection.Connection, data []byte) []byte {
	return data
}
```

The detailed implementation can be viewed by the [plugin](plugins/websocket). The source code can be viewed using the [websocket example](example/websocket).

## Example

<details>
  <summary> echo server</summary>

```go
package main

import (
	"flag"
	"strconv"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
)

type example struct{}

func (s *example) OnConnect(c *connection.Connection) {
	//log.Println(" OnConnect ： ", c.PeerAddr())
}
func (s *example) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out []byte) {
	//log.Println("OnMessage")
	out = data
	return
}

func (s *example) OnClose(c *connection.Connection) {
	//log.Println("OnClose")
}

func main() {
	handler := new(example)
	var port int
	var loops int

	flag.IntVar(&port, "port", 1833, "server port")
	flag.IntVar(&loops, "loops", -1, "num loops")
	flag.Parse()

	s, err := gev.NewServer(handler,
		gev.Network("tcp"),
		gev.Address(":"+strconv.Itoa(port)),
		gev.NumLoops(loops))
	if err != nil {
		panic(err)
	}

	s.Start()
}
```

</details>

<details>
  <summary> Maximum connections </summary>

```go
package main

import (
	"log"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/toolkit/sync/atomic"
)

// Server example
type Server struct {
	clientNum     atomic.Int64
	maxConnection int64
	server        *gev.Server
}

// New server
func New(ip, port string, maxConnection int64) (*Server, error) {
	var err error
	s := new(Server)
	s.maxConnection = maxConnection
	s.server, err = gev.NewServer(s,
		gev.Address(ip+":"+port))
	if err != nil {
		return nil, err
	}

	return s, nil
}

// Start server
func (s *Server) Start() {
	s.server.Start()
}

// Stop server
func (s *Server) Stop() {
	s.server.Stop()
}

// OnConnect callback
func (s *Server) OnConnect(c *connection.Connection) {
	s.clientNum.Add(1)
	log.Println(" OnConnect ： ", c.PeerAddr())

	if s.clientNum.Get() > s.maxConnection {
		_ = c.ShutdownWrite()
		log.Println("Refused connection")
		return
	}
}

// OnMessage callback
func (s *Server) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out []byte) {
	log.Println("OnMessage")
	out = data
	return
}

// OnClose callback
func (s *Server) OnClose(c *connection.Connection) {
	s.clientNum.Add(-1)
	log.Println("OnClose")
}

func main() {
	s, err := New("", "1833", 1)
	if err != nil {
		panic(err)
	}
	defer s.Stop()

	s.Start()
}
```

</details>

<details>
  <summary> Server timing push </summary>

```go
package main

import (
	"container/list"
	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
	"log"
	"sync"
	"time"
)

// Server example
type Server struct {
	conn   *list.List
	mu     sync.RWMutex
	server *gev.Server
}

// New server
func New(ip, port string) (*Server, error) {
	var err error
	s := new(Server)
	s.conn = list.New()
	s.server, err = gev.NewServer(s,
		gev.Address(ip+":"+port))
	if err != nil {
		return nil, err
	}

	return s, nil
}

// Start server
func (s *Server) Start() {
	s.server.RunEvery(1*time.Second, s.RunPush)
	s.server.Start()
}

// Stop server
func (s *Server) Stop() {
	s.server.Stop()
}

// RunPush push message
func (s *Server) RunPush() {
	var next *list.Element

	s.mu.RLock()
	defer s.mu.RUnlock()

	for e := s.conn.Front(); e != nil; e = next {
		next = e.Next()

		c := e.Value.(*connection.Connection)
		_ = c.Send([]byte("hello\n"))
	}
}

// OnConnect callback
func (s *Server) OnConnect(c *connection.Connection) {
	log.Println(" OnConnect ： ", c.PeerAddr())

	s.mu.Lock()
	e := s.conn.PushBack(c)
	s.mu.Unlock()
	c.SetContext(e)
}

// OnMessage callback
func (s *Server) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out []byte) {
	log.Println("OnMessage")
	out = data
	return
}

// OnClose callback
func (s *Server) OnClose(c *connection.Connection) {
	log.Println("OnClose")
	e := c.Context().(*list.Element)

	s.mu.Lock()
	s.conn.Remove(e)
	s.mu.Unlock()
}

func main() {
	s, err := New("", "1833")
	if err != nil {
		panic(err)
	}
	defer s.Stop()

	s.Start()
}
```

</details>

<details>
  <summary> WebSocket </summary>

```go
package main

import (
	"flag"
	"log"
	"math/rand"
	"strconv"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/plugins/websocket/ws"
	"github.com/Allenxuxu/gev/plugins/websocket/ws/util"
)

type example struct{}

func (s *example) OnConnect(c *connection.Connection) {
	log.Println(" OnConnect ： ", c.PeerAddr())
}
func (s *example) OnMessage(c *connection.Connection, data []byte) (messageType ws.MessageType, out []byte) {
	log.Println("OnMessage:", string(data))
	messageType = ws.MessageBinary
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

func (s *example) OnClose(c *connection.Connection) {
	log.Println("OnClose")
}

func main() {
	handler := new(example)
	var port int
	var loops int

	flag.IntVar(&port, "port", 1833, "server port")
	flag.IntVar(&loops, "loops", -1, "num loops")
	flag.Parse()

	s, err := NewWebSocketServer(handler, &ws.Upgrader{},
		gev.Network("tcp"),
		gev.Address(":"+strconv.Itoa(port)),
		gev.NumLoops(loops))
	if err != nil {
		panic(err)
	}

	s.Start()
}
```

```go
package main

import (
	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/plugins/websocket"
	"github.com/Allenxuxu/gev/plugins/websocket/ws"
)

// NewWebSocketServer 创建 WebSocket Server
func NewWebSocketServer(handler websocket.WebSocketHandler, u *ws.Upgrader, opts ...gev.Option) (server *gev.Server, err error) {
	opts = append(opts, gev.Protocol(websocket.New(u)))
	return gev.NewServer(websocket.NewHandlerWrap(u, handler), opts...)
}
```

</details>

<details>
  <summary> protobuf </summary>

```go
package main

import (
	"flag"
	"log"
	"strconv"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
	pb "github.com/Allenxuxu/gev/example/protobuf/proto"
	"github.com/Allenxuxu/gev/plugins/protobuf"
	"github.com/golang/protobuf/proto"
)

type example struct{}

func (s *example) OnConnect(c *connection.Connection) {
	log.Println(" OnConnect ： ", c.PeerAddr())
}
func (s *example) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out []byte) {
	msgType := ctx.(string)

	switch msgType {
	case "msg1":
		msg := &pb.Msg1{}
		if err := proto.Unmarshal(data, msg); err != nil {
			log.Println(err)
		}
		log.Println(msgType, msg)
	case "msg2":
		msg := &pb.Msg2{}
		if err := proto.Unmarshal(data, msg); err != nil {
			log.Println(err)
		}
		log.Println(msgType, msg)
	default:
		log.Println("unknown msg type")
	}

	return
}

func (s *example) OnClose(c *connection.Connection) {
	log.Println("OnClose")
}

func main() {
	handler := new(example)
	var port int
	var loops int

	flag.IntVar(&port, "port", 1833, "server port")
	flag.IntVar(&loops, "loops", -1, "num loops")
	flag.Parse()

	s, err := gev.NewServer(handler,
		gev.Network("tcp"),
		gev.Address(":"+strconv.Itoa(port)),
		gev.NumLoops(loops),
		gev.Protocol(&protobuf.Protocol{}))
	if err != nil {
		panic(err)
	}

	log.Println("server start")
	s.Start()
}
```

```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"

	pb "github.com/Allenxuxu/gev/example/protobuf/proto"
	"github.com/Allenxuxu/gev/plugins/protobuf"
	"github.com/golang/protobuf/proto"
)

func main() {
	conn, e := net.Dial("tcp", ":1833")
	if e != nil {
		log.Fatal(e)
	}
	defer conn.Close()

	var buffer []byte
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Text to send: ")
		text, _ := reader.ReadString('\n')
		name := text[:len(text)-1]

		switch rand.Int() % 2 {
		case 0:
			msg := &pb.Msg1{
				Name: name,
				Id:   1,
			}

			data, err := proto.Marshal(msg)
			if err != nil {
				panic(err)
			}
			buffer = protobuf.PackMessage("msg1", data)
		case 1:
			msg := &pb.Msg2{
				Name:  name,
				Alias: "big " + name,
				Id:    2,
			}

			data, err := proto.Marshal(msg)
			if err != nil {
				panic(err)
			}
			buffer = protobuf.PackMessage("msg2", data)
		}

		_, err := conn.Write(buffer)
		if err != nil {
			panic(err)
		}
	}
}
```

</details>


## Thanks

Thanks JetBrains for the free open source license

<a href="https://www.jetbrains.com/?from=gev" target="_blank">
	<img src="https://raw.githubusercontent.com/Allenxuxu/doc/master/jetbrains.png" width = "260" align=center />
</a>

## References

- [evio](https://github.com/tidwall/evio)
- [muduo](https://github.com/chenshuo/muduo)
