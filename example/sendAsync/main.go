package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"strconv"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/log"
	"github.com/gobwas/pool/pbytes"
)

type example struct{}

func (s *example) OnConnect(c *connection.Connection) {
	log.Info(" OnConnect ： ", c.PeerAddr())
}
func (s *example) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out interface{}) {
	log.Info("OnMessage ", string(data))

	b := pbytes.Get(0, 10)
	b = append(b, []byte("1234\n")...)

	_ = c.Send(b, connection.SendInLoop(func(i interface{}) {
		log.Info("put []byte ")
		pbytes.Put(i.([]byte))
	}))
	return
}

func (s *example) OnClose(c *connection.Connection) {
	log.Info("OnClose")
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

	s.Start()
}
