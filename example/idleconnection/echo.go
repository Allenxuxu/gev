package main

import (
	"flag"
	"strconv"
	"time"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/log"
)

type example struct {
}

func (s *example) OnConnect(c *connection.Connection) {
	log.Info(" OnConnect ： ", c.PeerAddr())
}
func (s *example) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out []byte) {
	log.Infof("OnMessage from : %s", c.PeerAddr())
	out = data
	return
}

func (s *example) OnClose(c *connection.Connection) {
	log.Info("OnClose: ", c.PeerAddr())
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
		gev.IdleTime(5*time.Second))
	if err != nil {
		panic(err)
	}

	s.Start()
}
