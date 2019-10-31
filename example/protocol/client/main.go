package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func Packet(data []byte) []byte {
	buffer := make([]byte, 4+len(data))
	// 将buffer前面四个字节设置为包长度，大端序
	binary.BigEndian.PutUint32(buffer, uint32(len(data)))
	copy(buffer[4:], data)
	return buffer
}

func UnPacket(c net.Conn) ([]byte, error) {
	var header = make([]byte, 4)

	_, err := io.ReadFull(c, header)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(header)
	contentByte := make([]byte, length)
	_, e := io.ReadFull(c, contentByte) //读取内容
	if e != nil {
		return nil, e
	}

	return contentByte, nil
}

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

		buffer := Packet([]byte(text))
		_, err := conn.Write(buffer)
		if err != nil {
			panic(err)
		}

		// listen for reply
		msg, err := UnPacket(conn)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Message from server (len %d) : %s", len(msg), string(msg))
	}
}
