package connector

import (
	"testing"
	"time"

	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/log"
)

var (
	dialer *Connector
)

type exampleCallback struct {
}

func (e exampleCallback) OnMessage(c *connection.Connection, ctx interface{}, data []byte) []byte {
	panic("implement me")
}

func (e exampleCallback) OnClose(c *connection.Connection) {
	panic("implement ")
}

func init() {
	var err error

	if dialer, err = NewConnector(); err != nil {
		panic(err)
	}

	go dialer.Start()
	time.Sleep(time.Second * 3)
}

func TestConnection_ListenerNotExist(t *testing.T) {
	cb := new(exampleCallback)
	_, err := dialer.Dial("tcp", "127.0.0.1:1830", cb, nil, 0)
	if err != ErrConnectionHandle {
		t.Fatal("error is not connection handle", err)
	}
	log.Info(err)
}
