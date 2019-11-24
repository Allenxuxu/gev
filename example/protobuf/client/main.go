package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"

	pb "github.com/Allenxuxu/gev/example/protobuf/proto"
	"github.com/Allenxuxu/gev/plugins/protobuf"
	"github.com/golang/protobuf/proto"
)

func main() {
	conn, e := net.Dial("tcp", ":1833")
	if e != nil {
		log.Fatal(e)
	}
	defer conn.Close()

	var buffer []byte
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Text to send: ")
		text, _ := reader.ReadString('\n')
		name := text[:len(text)-1]

		switch rand.Int() % 2 {
		case 0:
			msg := &pb.Msg1{
				Name: name,
				Id:   1,
			}

			data, err := proto.Marshal(msg)
			if err != nil {
				panic(err)
			}
			buffer = protobuf.PackMessage("msg1", data)
		case 1:
			msg := &pb.Msg2{
				Name:  name,
				Alias: "big " + name,
				Id:    2,
			}

			data, err := proto.Marshal(msg)
			if err != nil {
				panic(err)
			}
			buffer = protobuf.PackMessage("msg2", data)
		}

		_, err := conn.Write(buffer)
		if err != nil {
			panic(err)
		}
	}
}
