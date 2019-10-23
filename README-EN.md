# gev

[![Github Actions](https://github.com/Allenxuxu/gev/workflows/CI/badge.svg)](https://github.com/Allenxuxu/gev/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/Allenxuxu/gev)](https://goreportcard.com/report/github.com/Allenxuxu/gev)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/a2a55fe9c0c443e198f588a6c8026cd0)](https://www.codacy.com/manual/Allenxuxu/gev?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=Allenxuxu/gev&amp;utm_campaign=Badge_Grade)
[![GoDoc](https://godoc.org/github.com/Allenxuxu/gev?status.svg)](https://godoc.org/github.com/Allenxuxu/gev)
[![LICENSE](https://img.shields.io/badge/LICENSE-MIT-blue)](https://github.com/Allenxuxu/gev/blob/master/LICENSE)
[![Code Size](https://img.shields.io/github/languages/code-size/Allenxuxu/gev.svg?style=flat)](https://img.shields.io/github/languages/code-size/Allenxuxu/gev.svg?style=flat)

`gev` is a lightweight, fast non-blocking TCP network library based on Reactor mode.

## Features

- High-performance event loop based on epoll and kqueue
- Support multi-core and multi-threading
- Dynamic expansion of read and write buffers implemented by Ring Buffer
- Asynchronous read and write
- SO_REUSEPORT port reuse support
- Support WebSocket
- Support for scheduled tasks, delayed tasks

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

### TCP

```go
package main

import (
	"log"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/ringbuffer"
)

type example struct{}

func (s *example) OnConnect(c *connection.Connection) {
	log.Println(" OnConnect ： ", c.PeerAddr())
}

func (s *example) OnMessage(c *connection.Connection, buffer *ringbuffer.RingBuffer) (out []byte) {
	log.Println("OnMessage")
	first, end := buffer.PeekAll()
	out = first
	if len(end) > 0 {
		out = append(out, end...)
	}
	buffer.RetrieveAll()
	return
}

func (s *example) OnClose(c *connection.Connection) {
	log.Println("OnClose")
}

func main() {
	handler := new(example)

	s, err := gev.NewServer(handler,
		gev.Address(":1833"),
		gev.NumLoops(2),
		gev.ReusePort(true))
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
	OnMessage(c *connection.Connection, buffer *ringbuffer.RingBuffer) []byte
	OnClose(c *connection.Connection)
}

func NewServer(handler Handler, opts ...Option) (server *Server, err error) {
```

When the message arrivals, `gev` will send data within a slice back to the client by calling OnMessage.

```go
func (s *example) OnMessage(c *connection.Connection, buffer *ringbuffer.RingBuffer) (out []byte)
```

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

```go
package main

import (
	"flag"
	"github.com/Allenxuxu/gev/ws"
	"log"
	"math/rand"
	"strconv"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
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
		if err := c.SendWebsocketData(ws.MessageText, data); err != nil {
			if e := c.CloseWebsocket(err.Error()); e != nil {
				panic(e)
			}
		}
	case 2:
		if e := c.CloseWebsocket("close"); e != nil {
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

	s, err := gev.NewWebSocketServer(handler,
		gev.Network("tcp"),
		gev.Address(":"+strconv.Itoa(port)),
		gev.NumLoops(loops))
	if err != nil {
		panic(err)
	}

	s.Start()
}
```

*WebSocketHandler* is an interface that programs must implement.

```go
type WebSocketHandler interface {
	OnConnect(c *connection.Connection)
	OnMessage(c *connection.Connection, msg []byte) (ws.MessageType, []byte)
	OnClose(c *connection.Connection)
}
```

The WebSocket-related interface is basically the same as the TCP service. The main difference is `OnMessage`.

```go
func (c *Connection) SendWebsocketData(messageType ws.MessageType, buffer []byte) error
func (c *Connection) CloseWebsocket(reason string) error
```

Because the WebSocket protocol is actually built on top of the TCP protocol, `SendWebsocketData` and `CloseWebsocket` are provided.

## Example

<details>
  <summary> echo server</summary>

```go
package main

import (
	"flag"
	"strconv"
	"log"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/ringbuffer"
)

type example struct{}

func (s *example) OnConnect(c *connection.Connection) {
	log.Println(" OnConnect ： ", c.PeerAddr())
}
func (s *example) OnMessage(c *connection.Connection, buffer *ringbuffer.RingBuffer) (out []byte) {
	//log.Println("OnMessage")
	first, end := buffer.PeekAll()
	out = first
	if len(end) > 0 {
		out = append(out, end...)
	}
	buffer.RetrieveAll()
	return
}

func (s *example) OnClose() {
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
	"github.com/Allenxuxu/ringbuffer"
	"github.com/Allenxuxu/toolkit/sync/atomic"
)

type Server struct {
	clientNum     atomic.Int64
	maxConnection int64
	server        *gev.Server
}

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

func (s *Server) Start() {
	s.server.Start()
}

func (s *Server) Stop() {
	s.server.Stop()
}

func (s *Server) OnConnect(c *connection.Connection) {
	s.clientNum.Add(1)
	log.Println(" OnConnect ： ", c.PeerAddr())

	if s.clientNum.Get() > s.maxConnection {
		_ = c.ShutdownWrite()
		log.Println("Refused connection")
		return
	}
}
func (s *Server) OnMessage(c *connection.Connection, buffer *ringbuffer.RingBuffer) (out []byte) {
	log.Println("OnMessage")
	first, end := buffer.PeekAll()
	out = first
	if len(end) > 0 {
		out = append(out, end...)
	}
	buffer.RetrieveAll()
	return
}

func (s *Server) OnClose() {
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
	"github.com/Allenxuxu/ringbuffer"
	"log"
	"sync"
	"time"
)

type Server struct {
	conn   *list.List
	mu     sync.RWMutex
	server *gev.Server
}

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

func (s *Server) Start() {
	s.server.RunEvery(1*time.Second, s.RunPush)
	s.server.Start()
}

func (s *Server) Stop() {
	s.server.Stop()
}

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

func (s *Server) OnConnect(c *connection.Connection) {
	log.Println(" OnConnect ： ", c.PeerAddr())

	s.mu.Lock()
	e := s.conn.PushBack(c)
	s.mu.Unlock()
	c.SetContext(e)
}
func (s *Server) OnMessage(c *connection.Connection, buffer *ringbuffer.RingBuffer) (out []byte) {
	log.Println("OnMessage")
	first, end := buffer.PeekAll()
	out = first
	if len(end) > 0 {
		out = append(out, end...)
	}
	buffer.RetrieveAll()
	return
}

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
	"github.com/Allenxuxu/gev/ws"
	"log"
	"math/rand"
	"strconv"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
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
		if err := c.SendWebsocketData(ws.MessageText, data); err != nil {
			if e := c.CloseWebsocket(err.Error()); e != nil {
				panic(e)
			}
		}
	case 2:
		if e := c.CloseWebsocket("close"); e != nil {
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

	s, err := gev.NewWebSocketServer(handler,
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


## References

- [evio](https://github.com/tidwall/evio)
- [muduo](https://github.com/chenshuo/muduo)
