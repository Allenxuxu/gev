package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/net/websocket"
)

var (
	addr    = flag.String("a", "localhost:1833", "address")
	num     = flag.Int("c", 1, "connection number")
	timeOut = flag.Int("t", 2, "timeout second")
	msgLen  = flag.Int("m", 1024, "message length")

	msg []byte
)

func main() {
	flag.Parse()
	fmt.Printf("*** %d connections, %d seconds, %d byte packets ***\n", *num, *timeOut, *msgLen)

	msg = make([]byte, *msgLen)
	rand.Read(msg)
	startC := make(chan interface{})
	closeC := make(chan interface{})
	result := make(chan int64, *num)
	req := make(chan int64, *num)

	for i := 0; i < *num; i++ {
		go startWebSocketClient(startC, closeC, result, req)
	}

	// start
	close(startC)
	time.Sleep(time.Duration(*timeOut) * time.Second)
	// stop
	close(closeC)

	var totalMessagesRead, reqCount int64
	for i := 0; i < *num; i++ {
		totalMessagesRead += <-result
		reqCount += <-req
	}

	fmt.Println(totalMessagesRead/int64(*timeOut*1024*1024), " MiB/s throughput")
	fmt.Println(reqCount/int64(*timeOut), " qps")
}

func startWebSocketClient(startC chan interface{}, closeC chan interface{}, result, req chan int64) {
	var count, reqCount int64
	buf := make([]byte, 2*(*msgLen))

	address := "ws://" + *addr
	c, err := websocket.Dial(address, "", address)
	if err != nil {
		panic(err)
	}
	c.MaxPayloadBytes = *msgLen * 2
	<-startC

	if n, err := c.Write(msg); err != nil || n != len(msg) {
		panic(err)
	}

	for {
		select {
		case <-closeC:
			result <- count
			req <- reqCount
			c.Close()
			return
		default:
			n, err := c.Read(buf)
			if err != nil || n != len(msg) {
				fmt.Printf("read error %v  %d", err, n)
				panic(errors.New("read error"))
			}
			if !bytes.Equal(msg, buf[:n]) {
				panic("mismatch")
			}

			count += int64(n)
			reqCount++

			_, err = c.Write(msg)
			if err != nil {
				fmt.Println("Error to send message because of ", err.Error())
			}
		}
	}
}
