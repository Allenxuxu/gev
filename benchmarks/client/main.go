package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"time"
)

var addr = flag.String("a", "localhost:1833", "address")
var num = flag.Int("c", 1, "connection number")
var timeOut = flag.Int("t", 2, "timeout second")
var msgLen = flag.Int("m", 1024, "message length")

var msg []byte

func main() {
	flag.Parse()
	msg = make([]byte, *msgLen)
	rand.Read(msg)

	startC := make(chan interface{})
	closeC := make(chan interface{})
	result := make(chan int64, *num)

	for i := 0; i < *num; i++ {
		conn, err := net.Dial("tcp", *addr)
		if err != nil {
			panic(err)
		}
		go handler(conn, startC, closeC, result)
	}

	// start
	close(startC)

	time.Sleep(time.Duration(*timeOut) * time.Second)
	// stop
	close(closeC)

	var totalMessagesRead int64
	for i := 0; i < *num; i++ {
		totalMessagesRead += <-result
	}

	fmt.Println(totalMessagesRead/int64(*timeOut*1024*1024), " MiB/s throughput")
}

func handler(conn net.Conn, startC chan interface{}, closeC chan interface{}, result chan int64) {
	var count int64
	buf := make([]byte, 2*(*msgLen))
	<-startC

	_, e := conn.Write(msg)
	if e != nil {
		fmt.Println("Error to send message because of ", e.Error())
	}

	for {
		select {
		case <-closeC:
			result <- count
			conn.Close()
			return
		default:
			n, err := conn.Read(buf)
			if n > 0 {
				count += int64(n)
			}
			if err != nil {
				fmt.Print("Error to read message because of ", err)
				result <- count
				conn.Close()
				return
			}

			_, err = conn.Write(buf[:n])
			if err != nil {
				fmt.Println("Error to send message because of ", e.Error())
			}
		}
	}
}
