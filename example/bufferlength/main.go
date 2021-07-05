package main

import (
	"container/list"
	"log"
	"sync"
	"time"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
)

const clientsKey = "demo_push_message_key"

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
		if c.WriteBufferLength() > 1024*10 {
			log.Printf("write buffer length > 1024*10")
			continue
		}
		_ = c.Send([]byte("hello\n"))
	}
}

// OnConnect callback
func (s *Server) OnConnect(c *connection.Connection) {
	log.Println(" OnConnect ： ", c.PeerAddr())

	s.mu.Lock()
	e := s.conn.PushBack(c)
	s.mu.Unlock()
	c.Set(clientsKey, e)
}

// OnMessage callback
func (s *Server) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out interface{}) {
	log.Printf("OnMessage, read buffer len %d, write buffer len %d \n", c.ReadBufferLength(), c.WriteBufferLength())

	out = data
	return
}

// OnClose callback
func (s *Server) OnClose(c *connection.Connection) {
	log.Println("OnClose")
	v, ok := c.Get(clientsKey)
	if !ok {
		log.Println("OnClose : get key fail")
		return
	}

	s.mu.Lock()
	s.conn.Remove(v.(*list.Element))
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
