package main

import (
	"log"
	"os"
	"os/exec"
	"testing"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
	pb "github.com/Allenxuxu/gev/example/protobuf/proto"
	"github.com/Allenxuxu/gev/plugins/protobuf"
	"github.com/golang/protobuf/proto"
)

type example struct{}

func (s *example) OnConnect(c *connection.Connection) {
	log.Println(" OnConnect ： ", c.PeerAddr())
}
func (s *example) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out []byte) {
	msgType := ctx.(string)

	switch msgType {
	case "msg1":
		msg := &pb.Msg1{}
		if err := proto.Unmarshal(data, msg); err != nil {
			log.Println(err)
		}
		log.Println(msgType, msg)
	case "msg2":
		msg := &pb.Msg2{}
		if err := proto.Unmarshal(data, msg); err != nil {
			log.Println(err)
		}
		log.Println(msgType, msg)
	default:
		log.Println("unknown msg type")
	}

	return
}

func (s *example) OnClose(c *connection.Connection) {
	log.Println("OnClose")
}

func TestPlugin(t *testing.T) {
	// no plugin
	var defaultProtocol connection.DefaultProtocol
	handler := new(example)
	s, err := gev.NewServer(handler,
		gev.Network("tcp"),
		gev.Address(":1831"),
		gev.NumLoops(1))
	if err != nil {
		t.Fatal(err)
	}
	if s.Options().Protocol.String() != defaultProtocol.String() {
		t.Fatal()
	}

	s.Stop()

	// plugin
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", "/tmp/plugin.so", "./plugin.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv("GEV_PLUGIN", "/tmp/plugin.so"); err != nil {
		t.Fatal(err)
	}

	handler = new(example)
	s, err = gev.NewServer(handler,
		gev.Network("tcp"),
		gev.Address(":1133"),
		gev.NumLoops(1))
	if err != nil {
		t.Fatal(err)
	}

	if s.Options().Protocol.String() != protobuf.New().String() {
		t.Fatal()
	}

	s.Stop()
}
