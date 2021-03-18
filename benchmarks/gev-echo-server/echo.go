package main

import (
	"flag"
	"net/http"
	"strconv"

	"github.com/Allenxuxu/gev/log"

	_ "net/http/pprof"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
)

type example struct {
}

func (s *example) OnConnect(c *connection.Connection) {}
func (s *example) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out []byte) {

	out = data

	//msg := append([]byte{}, data...)
	//go func() {
	//	if err := c.Send(msg); err != nil {
	//		//log.Errorf("send error :%v", err)
	//	}
	//}()
	return
}

func (s *example) OnClose(c *connection.Connection) {
	//log.Error("onclose ")
}

func main() {
	log.SetLevel(log.LevelDebug)
	go func() {
		if err := http.ListenAndServe(":6089", nil); err != nil {
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
