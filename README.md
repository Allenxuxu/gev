# gev

[![Github Actions](https://github.com/Allenxuxu/gev/workflows/CI/badge.svg)](https://github.com/Allenxuxu/gev/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/Allenxuxu/gev)](https://goreportcard.com/report/github.com/Allenxuxu/gev)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/a2a55fe9c0c443e198f588a6c8026cd0)](https://www.codacy.com/manual/Allenxuxu/gev?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=Allenxuxu/gev&amp;utm_campaign=Badge_Grade)
[![GoDoc](https://godoc.org/github.com/Allenxuxu/gev?status.svg)](https://godoc.org/github.com/Allenxuxu/gev)
[![LICENSE](https://img.shields.io/badge/LICENSE-MIT-blue)](https://github.com/Allenxuxu/gev/blob/master/LICENSE)
[![Code Size](https://img.shields.io/github/languages/code-size/Allenxuxu/gev.svg?style=flat)](https://img.shields.io/github/languages/code-size/Allenxuxu/gev.svg?style=flat)
[![Sourcegraph](https://sourcegraph.com/github.com/Allenxuxu/gev/-/badge.svg)](https://sourcegraph.com/github.com/Allenxuxu/gev?badge)

#### [中文](README-ZH.md) | English

`gev` is a lightweight, fast non-blocking TCP network library / websocket server based on Reactor mode. 

Support custom protocols to quickly and easily build high-performance servers.

## Features

- High-performance event loop based on epoll and kqueue
- Support multi-core and multi-threading
- Dynamic expansion of read and write buffers implemented by Ring Buffer
- Asynchronous read and write
- SO_REUSEPORT port reuse support
- Automatically clean up idle connections
- Support WebSocket/Protobuf, custom protocols
- Support for scheduled tasks, delayed tasks
- High performance websocket server

## Network model

`gev` uses only a few goroutines, one of them listens for connections and the others (work coroutines) handle read and write events of connected clients. The count of work coroutines is configurable, which is the core number of host CPUs by default.

<div align=center>
<img src="benchmarks/out/reactor.png" height="300"/>
</div>

## Performance Test

<details>
  <summary> 📈 Test chart </summary>

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
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"time"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/log"
	"github.com/Allenxuxu/toolkit/sync/atomic"
)

type example struct {
	Count atomic.Int64
}

func (s *example) OnConnect(c *gev.Connection) {
	s.Count.Add(1)
	//log.Println(" OnConnect ： ", c.PeerAddr())
}
func (s *example) OnMessage(c *gev.Connection, ctx interface{}, data []byte) (out interface{}) {
	//log.Println("OnMessage")
	out = data
	return
}

func (s *example) OnClose(c *gev.Connection) {
	s.Count.Add(-1)
	//log.Println("OnClose")
}

func main() {
	go func() {
		if err := http.ListenAndServe(":6060", nil); err != nil {
			panic(err)
		}
	}()

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
		gev.MetricsServer("", ":9091"),
	)
	if err != nil {
		panic(err)
	}

	s.RunEvery(time.Second*2, func() {
		log.Info("connections :", handler.Count.Get())
	})

	s.Start()
}
```

*Handler* is an interface that programs must implement.

```go
type CallBack interface {
	OnMessage(c *Connection, ctx interface{}, data []byte) interface{}
	OnClose(c *Connection)
}

type Handler interface {
	CallBack
	OnConnect(c *Connection)
}
```

OnMessage will be called back when a complete data frame arrives.Users can get the data, process the business logic, and return the data that needs to be sent.

When there is data coming, gev does not call back OnMessage immediately, but instead calls back an UnPacket function.Probably the execution logic is as follows:

```go
ctx, receivedData := c.protocol.UnPacket(c, buffer)
for ctx != nil || len(receivedData) != 0 {
	sendData := c.callBack.OnMessage(c, ctx, receivedData)
	if sendData != nil {
		*tmpBuffer = append(*tmpBuffer, c.protocol.Packet(c, sendData)...)
	}

	ctx, receivedData = c.protocol.UnPacket(c, buffer)
}
```

![protocol](benchmarks/out/protocol.png)

The UnPacket function will check whether the data in the ringbuffer is a complete data frame. If it is, the data will be unpacked and return the payload data. If it is not a complete data frame, it will return directly.

The return value of UnPacket `(interface{}, []byte)` will be passed in as a call to OnMessage `ctx interface{}, data []byte` and callback.Ctx is designed to pass special information generated when parsing data frames in the UnPacket function (which is required for complex data frame protocols), and data is used to pass payload data.

```go
type Protocol interface {
	UnPacket(c *Connection, buffer *ringbuffer.RingBuffer) (interface{}, []byte)
	Packet(c *Connection, data interface{}) []byte
}

type DefaultProtocol struct{}

func (d *DefaultProtocol) UnPacket(c *Connection, buffer *ringbuffer.RingBuffer) (interface{}, []byte) {
	s, e := buffer.PeekAll()
	if len(e) > 0 {
		size := len(s) + len(e)
		userBuffer := *c.UserBuffer()
		if size > cap(userBuffer) {
			userBuffer = make([]byte, size)
			*c.UserBuffer() = userBuffer
		}

		copy(userBuffer, s)
		copy(userBuffer[len(s):], e)

		return nil, userBuffer
	} else {
		buffer.RetrieveAll()

		return nil, s
	}
}

func (d *DefaultProtocol) Packet(c *Connection, data interface{}) []byte {
	return data.([]byte)
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
func (c *Connection) Send(data interface{}, opts ...ConnectionOption) error
```

*ShutdownWrite* works for reverting connected status to false and closing connection.

Check out the example [Maximum connections](example/maxconnection/main.go) for a detailed.

```go
func (c *Connection) ShutdownWrite() error
```

[RingBuffer](https://github.com/Allenxuxu/ringbuffer) is a dynamical expansion implementation of circular buffer.

### WebSocket

The WebSocket protocol is built on top of the TCP protocol, so gev doesn't need to be built in, but instead provides support in the form of plugins, in the plugins/websocket directory.

<details>
  <summary> code </summary>

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

</details>

The detailed implementation can be viewed by the [plugin](plugins/websocket). The source code can be viewed using the [websocket example](example/websocket).

## Example

- [Echo Server](example/echo)
- [Automatically clean up idle connections](example/idleconnection)
- [Maximum connections](example/maxconnection)
- [Server timing push](example/pushmessage)
- [WebSocket](example/websocket)
- [Protobuf](example/protobuf)
- [...](example)

## Buy me a coffee

<img src="https://raw.githubusercontent.com/Allenxuxu/doc/master/alipay.jpeg" width = "200" height="300" />
<img src="https://raw.githubusercontent.com/Allenxuxu/doc/master/wechat.jpeg" width = "200" height="300" />

**Paypal**: [Paypal/AllenXuxu](https://www.paypal.me/AllenXuxu)

## Thanks

Thanks JetBrains for the free open source license

<a href="https://www.jetbrains.com/?from=gev" target="_blank">
	<img src="https://raw.githubusercontent.com/Allenxuxu/doc/master/jetbrains.png" width = "260" align=center />
</a>

## References

- [evio](https://github.com/tidwall/evio)
- [muduo](https://github.com/chenshuo/muduo)
