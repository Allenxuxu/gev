// Copyright 2019 Xu Xu. All rights reserved.
// Copyright 2017 Joshua J Baker. All rights reserved.

// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Allenxuxu/eviop"
	"github.com/Allenxuxu/ringbuffer"
)

func main() {
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

	var events eviop.Events
	events.NumLoops = loops
	events.Serving = func(srv eviop.Server) (action eviop.Action) {
		log.Printf("echo server started on port %d (loops: %d)", port, srv.NumLoops)
		if reuseport {
			log.Printf("reuseport")
		}
		return
	}
	events.Data = func(c *eviop.Conn, in *ringbuffer.RingBuffer) (out []byte, action eviop.Action) {
		first, end := in.PeekAll()
		if trace {
			log.Printf("%s", strings.TrimSpace(string(first)+string(end)))
		}
		out = first
		if len(end) > 0 {
			out = append(out, end...)
		}
		in.RetrieveAll()
		return
	}
	scheme := "tcp"
	if udp {
		scheme = "udp"
	}
	log.Fatal(eviop.Serve(events, time.Second*10, fmt.Sprintf("%s://:%d?reuseport=%t", scheme, port, reuseport)))
}
