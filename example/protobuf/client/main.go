package main

import (
	"bufio"
	"fmt"
	"log"
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

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Text to send: ")
		text, _ := reader.ReadString('\n')
		name := text[:len(text)-1]

		user := &pb.User{
			Name: name,
			Id:   1,
		}

		data, err := proto.Marshal(user)
		if err != nil {
			panic(err)
		}

		buffer := protobuf.PackMessage("test123", data)

		_, err = conn.Write(buffer)
		if err != nil {
			panic(err)
		}

	}
}
