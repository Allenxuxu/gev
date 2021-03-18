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

	fmt.Printf("*** %d connections, %d seconds, %d byte packets ***\n", *num, *timeOut, *msgLen)

	msg = make([]byte, *msgLen)
	rand.Read(msg)

	startC := make(chan interface{})
	closeC := make(chan interface{})
	result := make(chan int64, *num)
	req := make(chan int64, *num)

	for i := 0; i < *num; i++ {
		conn, err := net.Dial("tcp", *addr)
		if err != nil {
			panic(err)
		}
		go handler(conn, startC, closeC, result, req)
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

func handler(conn net.Conn, startC chan interface{}, closeC chan interface{}, result, req chan int64) {
	var count, reqCount int64
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
			req <- reqCount
			conn.Close()
			return
		default:
			n, err := conn.Read(buf)
			if n > 0 {
				count += int64(n)
				reqCount++
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
