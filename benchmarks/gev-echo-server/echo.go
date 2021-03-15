package main

import (
	"flag"
	"net/http"
	"strconv"

	_ "net/http/pprof"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
)

type example struct {
}

func (s *example) OnConnect(c *connection.Connection) {}
func (s *example) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out []byte) {
	out = data
	return
}

func (s *example) OnClose(c *connection.Connection) {}

func main() {
	go func() {
		if err := http.ListenAndServe(":6061", nil); err != nil {
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
	)
	if err != nil {
		panic(err)
	}

	s.Start()
}
