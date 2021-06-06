package connector

import (
	"testing"
	"time"

	"github.com/Allenxuxu/gev/connection"
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
	panic("implement me")
}

func init() {
	var err error

	if dialer, err = NewConnector(); err != nil {
		panic(err)
	}

	go dialer.Start()
}

func TestConnection_ListenerNotExist(t *testing.T) {
	cb := new(exampleCallback)
	_, err := dialer.DialWithTimeout(time.Second*5, "tcp", "127.0.0.1:1830", cb, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
}
