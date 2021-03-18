package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/panjf2000/gnet"
)

type echoServer struct {
	*gnet.EventServer
}

func (es *echoServer) OnInitComplete(srv gnet.Server) (action gnet.Action) {
	return
}

func (es *echoServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	// Echo synchronously.
	out = frame

	//// Echo asynchronously.
	//data := append([]byte{}, frame...)
	//go func() {
	//	//time.Sleep(time.Second)
	//	c.AsyncWrite(data)
	//}()
	return
}

func main() {
	go func() {
		if err := http.ListenAndServe(":6062", nil); err != nil {
			panic(err)
		}
	}()

	var port int
	var loops int
	var udp bool
	var trace bool
	var reuseport bool

	flag.IntVar(&port, "port", 5000, "server port")
	flag.BoolVar(&udp, "udp", false, "listen on udp")
	flag.BoolVar(&reuseport, "reuseport", false, "reuseport (SO_REUSEPORT)")
	flag.BoolVar(&trace, "trace", false, "print packets to console")
	flag.IntVar(&loops, "loops", -1, "num loops")
	flag.Parse()

	echo := new(echoServer)
	log.Fatal(gnet.Serve(echo, fmt.Sprintf("tcp://:%d", port), gnet.WithNumEventLoop(loops), gnet.WithReusePort(reuseport)))
}
