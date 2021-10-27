package main

import (
	"flag"
	"strconv"

	"github.com/Allenxuxu/gev/log"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/plugins/websocket"
	"github.com/Allenxuxu/gev/plugins/websocket/ws"
)

type example struct {
}

// connection lifecycle
// OnConnect() -> OnRequest() -> OnHeader() -> OnMessage() -> OnClose()

func (s *example) OnConnect(c *gev.Connection) {
	//log.Println("OnConnect: ", c.PeerAddr())
}

func (s *example) OnMessage(c *gev.Connection, data []byte) (messageType ws.MessageType, out []byte) {
	//log.Println("OnMessage: ", string(data))

	messageType = ws.MessageBinary
	out = data

	return
}

func (s *example) OnClose(c *gev.Connection) {
	//log.Println("123 OnClose", c.PeerAddr())
}

// NewWebSocketServer 创建 WebSocket Server
func NewWebSocketServer(handler websocket.WSHandler, u *ws.Upgrader, opts ...gev.Option) (server *gev.Server, err error) {
	opts = append(opts, gev.CustomProtocol(websocket.New(u)))
	return gev.NewServer(websocket.NewHandlerWrap(u, handler), opts...)
}

func main() {
	log.SetLevel(log.LevelDebug)
	var (
		port  int
		loops int
	)

	flag.IntVar(&port, "port", 1833, "server port")
	flag.IntVar(&loops, "loops", -1, "num loops")
	flag.Parse()

	handler := &example{}
	wsUpgrader := &ws.Upgrader{}
	//wsUpgrader.OnRequest = func(c *connection.Connection, uri []byte) error {
	//	log.Println("OnRequest: ", string(uri))
	//
	//	return nil
	//}

	s, err := NewWebSocketServer(handler, wsUpgrader,
		gev.Network("tcp"),
		gev.Address(":"+strconv.Itoa(port)),
		gev.NumLoops(loops))
	if err != nil {
		panic(err)
	}

	s.Start()
}
