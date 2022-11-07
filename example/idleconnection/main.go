package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"time"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/log"
)

type example struct {
}

func (s *example) OnConnect(c *gev.Connection) {
	log.Info(" OnConnect ： ", c.PeerAddr())
}
func (s *example) OnMessage(c *gev.Connection, ctx interface{}, data []byte) (out interface{}) {
	log.Infof("OnMessage from : %s", c.PeerAddr())
	out = data
	return
}

func (s *example) OnClose(c *gev.Connection) {
	log.Info("OnClose: ", c.PeerAddr())
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
		gev.IdleTime(5*time.Second),
		gev.MetricsServer("", ":9091"),
	)
	if err != nil {
		panic(err)
	}

	s.Start()
}
